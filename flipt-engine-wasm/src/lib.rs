use fliptevaluation::error::Error;
use fliptevaluation::models::flipt::Flag;
use fliptevaluation::models::{snapshot, source};
use fliptevaluation::store::Store;
use fliptevaluation::{
    batch_evaluation, boolean_evaluation, variant_evaluation, BatchEvaluationResponse,
    BooleanEvaluationResponse, EvaluationRequest, VariantEvaluationResponse,
};
use libc::c_void;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::collections::HashMap;
use thiserror::Error;

#[derive(Deserialize)]
struct WASMEvaluationRequest {
    flag_key: String,
    entity_id: String,
    context: Option<Map<String, Value>>,
}

#[derive(Serialize)]
struct WASMResponse<T>
where
    T: Serialize,
{
    status: Status,
    result: Option<T>,
    error_message: Option<String>,
}

#[derive(Serialize)]
enum Status {
    #[serde(rename = "success")]
    Success,
    #[serde(rename = "failure")]
    Failure,
}

#[derive(Error, Debug)]
pub enum WASMError {
    #[error("Invalid JSON: {0}")]
    InvalidJson(#[from] serde_json::Error),
    #[error("Error building snapshot: {0}")]
    SnapshotBuildError(#[from] Error),
    #[error("Null pointer error")]
    NullPointer,
}

impl<T, E> From<Result<T, E>> for WASMResponse<T>
where
    T: Serialize,
    E: std::error::Error,
{
    fn from(value: Result<T, E>) -> Self {
        match value {
            Ok(result) => WASMResponse {
                status: Status::Success,
                result: Some(result),
                error_message: None,
            },
            Err(e) => WASMResponse {
                status: Status::Failure,
                result: None,
                error_message: Some(e.to_string()),
            },
        }
    }
}

fn result_to_string<T: Serialize, E: std::error::Error>(result: Result<T, E>) -> String {
    let wasm_response: WASMResponse<T> = result.into();
    serde_json::to_string(&wasm_response).unwrap()
}

pub struct Engine {
    namespace: String,
    store: snapshot::Snapshot,
}

impl Engine {
    pub fn new(namespace: &str, snapshot: &str) -> Result<Self, WASMError> {
        let doc: source::Document =
            serde_json::from_str(snapshot).map_err(WASMError::InvalidJson)?;
        let store = snapshot::Snapshot::build(doc);

        Ok(Self {
            namespace: namespace.to_string(),
            store,
        })
    }

    pub fn snapshot(&mut self, data: &str) -> Result<(), WASMError> {
        let doc: source::Document = serde_json::from_str(data).map_err(WASMError::InvalidJson)?;
        self.store = snapshot::Snapshot::build(doc);
        Ok(())
    }

    pub fn evaluate_boolean(
        &self,
        request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        boolean_evaluation(&self.store, &self.namespace, request)
    }

    pub fn evaluate_variant(
        &self,
        request: &EvaluationRequest,
    ) -> Result<VariantEvaluationResponse, Error> {
        variant_evaluation(&self.store, &self.namespace, request)
    }

    pub fn evaluate_batch(
        &self,
        request: Vec<fliptevaluation::EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        batch_evaluation(&self.store, &self.namespace, request)
    }

    pub fn list_flags(&self) -> Result<Option<Vec<Flag>>, Error> {
        Ok(self.store.list_flags(&self.namespace))
    }
}

/// # Safety
///
/// This function should not be called unless an Engine is initiated. It provides a helper
/// utility to retrieve an Engine instance for evaluation use.
unsafe fn get_engine<'a>(engine_ptr: *mut c_void) -> Result<&'a mut Engine, WASMError> {
    if engine_ptr.is_null() {
        Err(WASMError::NullPointer)
    } else {
        Ok(&mut *(engine_ptr as *mut Engine))
    }
}

/// # Safety
///
/// This function will initialize an Engine and return a pointer back to the caller.
#[no_mangle]
pub unsafe extern "C" fn initialize_engine(
    namespace_ptr: *const u8,
    namespace_len: usize,
    payload_ptr: *const u8,
    payload_len: usize,
) -> *mut c_void {
    let result = std::panic::catch_unwind(|| {
        if namespace_ptr.is_null()
            || payload_ptr.is_null()
            || namespace_len == 0
            || payload_len == 0
        {
            eprintln!("Null pointer or zero length in initialize_engine");
            return std::ptr::null_mut();
        }

        let namespace =
            match std::str::from_utf8(std::slice::from_raw_parts(namespace_ptr, namespace_len)) {
                Ok(s) => s,
                Err(e) => {
                    eprintln!("Invalid UTF-8 in namespace: {e}");
                    return std::ptr::null_mut();
                }
            };
        let payload =
            match std::str::from_utf8(std::slice::from_raw_parts(payload_ptr, payload_len)) {
                Ok(s) => s,
                Err(e) => {
                    eprintln!("Invalid UTF-8 in payload: {e}");
                    return std::ptr::null_mut();
                }
            };

        match Engine::new(namespace, payload) {
            Ok(engine) => Box::into_raw(Box::new(engine)) as *mut c_void,
            Err(e) => {
                eprintln!("Error initializing engine: {e}");
                std::ptr::null_mut()
            }
        }
    });

    result.unwrap_or_else(|_| {
        eprintln!("Panic in initialize_engine");
        std::ptr::null_mut()
    })
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a variant evaluation response.
#[no_mangle]
pub unsafe extern "C" fn evaluate_variant(
    engine_ptr: *mut c_void,
    evaluation_request_ptr: *const u8,
    evaluation_request_len: usize,
) -> u64 {
    let result = std::panic::catch_unwind(|| {
        let e = match get_engine(engine_ptr) {
            Ok(e) => e,
            Err(e) => return result_to_ptr::<VariantEvaluationResponse, _>(Err(e)),
        };

        if evaluation_request_ptr.is_null() || evaluation_request_len == 0 {
            return result_to_ptr::<VariantEvaluationResponse, _>(Err(WASMError::NullPointer));
        }

        let evaluation_request = match std::str::from_utf8(std::slice::from_raw_parts(
            evaluation_request_ptr,
            evaluation_request_len,
        )) {
            Ok(s) => s,
            Err(_) => {
                return result_to_ptr::<VariantEvaluationResponse, _>(Err(WASMError::InvalidJson(
                    serde_json::Error::io(std::io::Error::new(
                        std::io::ErrorKind::InvalidData,
                        "Invalid UTF-8 in request",
                    )),
                )))
            }
        };

        let request = match get_evaluation_request(evaluation_request) {
            Ok(req) => req,
            Err(e) => return result_to_ptr::<VariantEvaluationResponse, _>(Err(e)),
        };

        result_to_ptr(e.evaluate_variant(&request))
    });

    result.unwrap_or_else(|_| {
        eprintln!("Panic in evaluate_variant");
        0
    })
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a boolean evaluation response.
#[no_mangle]
pub unsafe extern "C" fn evaluate_boolean(
    engine_ptr: *mut c_void,
    evaluation_request_ptr: *const u8,
    evaluation_request_len: usize,
) -> u64 {
    let result = std::panic::catch_unwind(|| {
        let e = match get_engine(engine_ptr) {
            Ok(e) => e,
            Err(e) => return result_to_ptr::<BooleanEvaluationResponse, _>(Err(e)),
        };

        if evaluation_request_ptr.is_null() || evaluation_request_len == 0 {
            return result_to_ptr::<BooleanEvaluationResponse, _>(Err(WASMError::NullPointer));
        }

        let evaluation_request = match std::str::from_utf8(std::slice::from_raw_parts(
            evaluation_request_ptr,
            evaluation_request_len,
        )) {
            Ok(s) => s,
            Err(_) => {
                return result_to_ptr::<BooleanEvaluationResponse, _>(Err(WASMError::InvalidJson(
                    serde_json::Error::io(std::io::Error::new(
                        std::io::ErrorKind::InvalidData,
                        "Invalid UTF-8 in request",
                    )),
                )))
            }
        };

        let request = match get_evaluation_request(evaluation_request) {
            Ok(req) => req,
            Err(e) => return result_to_ptr::<BooleanEvaluationResponse, _>(Err(e)),
        };

        result_to_ptr(e.evaluate_boolean(&request))
    });

    result.unwrap_or_else(|_| {
        eprintln!("Panic in evaluate_boolean");
        0
    })
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a batch evaluation response.
#[no_mangle]
pub unsafe extern "C" fn evaluate_batch(
    engine_ptr: *mut c_void,
    batch_evaluation_request_ptr: *const u8,
    batch_evaluation_request_len: usize,
) -> u64 {
    let result = std::panic::catch_unwind(|| {
        let e = match get_engine(engine_ptr) {
            Ok(e) => e,
            Err(e) => return result_to_ptr::<BatchEvaluationResponse, _>(Err(e)),
        };

        if batch_evaluation_request_ptr.is_null() || batch_evaluation_request_len == 0 {
            return result_to_ptr::<BatchEvaluationResponse, _>(Err(WASMError::NullPointer));
        }

        let evaluation_requests = match std::str::from_utf8(std::slice::from_raw_parts(
            batch_evaluation_request_ptr,
            batch_evaluation_request_len,
        )) {
            Ok(s) => s,
            Err(_) => {
                return result_to_ptr::<BatchEvaluationResponse, _>(Err(WASMError::InvalidJson(
                    serde_json::Error::io(std::io::Error::new(
                        std::io::ErrorKind::InvalidData,
                        "Invalid UTF-8 in request",
                    )),
                )))
            }
        };

        let requests = match get_batch_evaluation_request(evaluation_requests) {
            Ok(reqs) => reqs,
            Err(e) => return result_to_ptr::<BatchEvaluationResponse, _>(Err(e)),
        };

        result_to_ptr(e.evaluate_batch(requests))
    });

    result.unwrap_or_else(|_| {
        eprintln!("Panic in evaluate_batch");
        0
    })
}

/// # Safety
///
/// This function will return a list of flags.
#[no_mangle]
pub unsafe extern "C" fn list_flags(engine_ptr: *mut c_void) -> u64 {
    let result = std::panic::catch_unwind(|| {
        let e = match get_engine(engine_ptr) {
            Ok(e) => e,
            Err(e) => return result_to_ptr::<Option<Vec<Flag>>, _>(Err(e)),
        };

        result_to_ptr(e.list_flags())
    });

    result.unwrap_or_else(|_| {
        eprintln!("Panic in list_flags");
        0
    })
}

/// # Safety
///
/// This function will take in a pointer to the engine and snapshot.
#[no_mangle]
pub unsafe extern "C" fn snapshot(
    engine_ptr: *mut c_void,
    snapshot_ptr: *const u8,
    snapshot_len: usize,
) -> u64 {
    let result = std::panic::catch_unwind(|| {
        let e = match get_engine(engine_ptr) {
            Ok(e) => e,
            Err(e) => return result_to_ptr::<(), _>(Err(e)),
        };

        if snapshot_ptr.is_null() || snapshot_len == 0 {
            return result_to_ptr::<(), _>(Err(WASMError::NullPointer));
        }

        let snapshot =
            match std::str::from_utf8(std::slice::from_raw_parts(snapshot_ptr, snapshot_len)) {
                Ok(s) => s,
                Err(_) => {
                    return result_to_ptr::<(), _>(Err(WASMError::InvalidJson(
                        serde_json::Error::io(std::io::Error::new(
                            std::io::ErrorKind::InvalidData,
                            "Invalid UTF-8 in snapshot",
                        )),
                    )))
                }
            };

        result_to_ptr(e.snapshot(snapshot))
    });

    result.unwrap_or_else(|_| {
        eprintln!("Panic in snapshot");
        0
    })
}

/// # Safety
///
/// This function will free the memory occupied by the engine.
#[no_mangle]
pub unsafe extern "C" fn destroy_engine(engine_ptr: *mut c_void) {
    if engine_ptr.is_null() {
        return;
    }

    drop(Box::from_raw(engine_ptr as *mut Engine));
}

/// # Safety
///
/// This function will allocate memory for the engine.
/// Returns null if allocation fails or size is 0.
#[no_mangle]
pub extern "C" fn allocate(size: usize) -> *mut c_void {
    if size == 0 {
        return std::ptr::null_mut();
    }

    let result = std::panic::catch_unwind(|| {
        let mut buf = vec![0; size];
        let ptr = buf.as_mut_ptr();
        std::mem::forget(buf);
        ptr as *mut c_void
    });

    result.unwrap_or(std::ptr::null_mut())
}

/// # Safety
///
/// This function will free the memory occupied by the engine.
/// This function will do nothing if the pointer is null or the size is 0.
#[no_mangle]
pub unsafe extern "C" fn deallocate(ptr: *mut c_void, size: usize) {
    if ptr.is_null() || size == 0 {
        return;
    }

    let _ = std::panic::catch_unwind(|| {
        let buf = Vec::from_raw_parts(ptr, size, size);
        std::mem::drop(buf);
    });
}

unsafe fn get_evaluation_request(evaluation_request: &str) -> Result<EvaluationRequest, WASMError> {
    let client_eval_request: WASMEvaluationRequest =
        serde_json::from_str(evaluation_request).map_err(WASMError::InvalidJson)?;

    let mut context_map: HashMap<String, String> = HashMap::new();
    if let Some(context_value) = client_eval_request.context {
        for (key, value) in context_value {
            if let serde_json::Value::String(val) = value {
                context_map.insert(key, val);
            }
        }
    }

    Ok(EvaluationRequest {
        flag_key: client_eval_request.flag_key,
        entity_id: client_eval_request.entity_id,
        context: context_map,
    })
}

unsafe fn get_batch_evaluation_request(
    batch_evaluation_request: &str,
) -> Result<Vec<EvaluationRequest>, WASMError> {
    let batch_eval_request: Vec<WASMEvaluationRequest> =
        serde_json::from_str(batch_evaluation_request).map_err(WASMError::InvalidJson)?;

    let mut evaluation_requests: Vec<EvaluationRequest> =
        Vec::with_capacity(batch_eval_request.len());
    for req in batch_eval_request {
        let mut context_map: HashMap<String, String> = HashMap::new();
        if let Some(context_value) = req.context {
            for (key, value) in context_value {
                if let serde_json::Value::String(val) = value {
                    context_map.insert(key, val);
                }
            }
        }

        evaluation_requests.push(EvaluationRequest {
            flag_key: req.flag_key,
            entity_id: req.entity_id,
            context: context_map,
        });
    }

    Ok(evaluation_requests)
}

/// Returns a pointer and size pair for the given string in a way compatible
/// with WebAssembly numeric types.
unsafe fn string_to_ptr(s: &str) -> (u32, u32) {
    (s.as_ptr() as u32, s.len() as u32)
}

/// Returns a pointer and size pair for the given result in a way compatible
/// with WebAssembly numeric types.
///
/// Note: This leaks the result string to the caller.
unsafe fn result_to_ptr<T: Serialize, E: std::error::Error>(result: Result<T, E>) -> u64 {
    let result = result_to_string(result);
    let (ptr, len) = string_to_ptr(&result);
    std::mem::forget(result);
    ((ptr as u64) << 32) | len as u64
}
