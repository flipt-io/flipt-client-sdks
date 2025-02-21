use fliptevaluation::error::Error;
use fliptevaluation::models::source;
use fliptevaluation::store::Snapshot;
use fliptevaluation::{
    batch_evaluation, boolean_evaluation, variant_evaluation, EvaluationRequest,
};
use libc::c_void;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::collections::HashMap;
use std::ffi::CStr;
use std::os::raw::c_char;
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

impl<T> From<Result<T, Error>> for WASMResponse<T>
where
    T: Serialize,
{
    fn from(value: Result<T, Error>) -> Self {
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

fn result_to_string<T: Serialize>(result: Result<T, Error>) -> String {
    let wasm_response: WASMResponse<T> = result.into();
    serde_json::to_string(&wasm_response).unwrap()
}

pub struct Engine {
    namespace: String,
    store: Snapshot,
}

impl Engine {
    pub fn new(namespace: &str, snapshot: &str) -> Result<Self, WASMError> {
        let doc: source::Document =
            serde_json::from_str(snapshot).map_err(WASMError::InvalidJson)?;
        let store = Snapshot::build(namespace, doc).map_err(WASMError::SnapshotBuildError)?;

        Ok(Self {
            namespace: namespace.to_string(),
            store,
        })
    }

    pub fn snapshot(&mut self, data: &str) -> Result<(), WASMError> {
        let doc: source::Document = serde_json::from_str(data).map_err(WASMError::InvalidJson)?;
        self.store =
            Snapshot::build(&self.namespace, doc).map_err(WASMError::SnapshotBuildError)?;
        Ok(())
    }

    pub fn evaluate_boolean(&self, request: &EvaluationRequest) -> Result<String, Error> {
        let result = boolean_evaluation(&self.store, &self.namespace, &request)?;
        serde_json::to_string(&result).map_err(|e| Error::InvalidJSON(e.to_string()))
    }

    pub fn evaluate_variant(&self, request: &EvaluationRequest) -> Result<String, Error> {
        let result = variant_evaluation(&self.store, &self.namespace, &request)?;
        serde_json::to_string(&result).map_err(|e| Error::InvalidJSON(e.to_string()))
    }

    pub fn evaluate_batch(
        &self,
        request: Vec<fliptevaluation::EvaluationRequest>,
    ) -> Result<String, Error> {
        let result = batch_evaluation(&self.store, &self.namespace, request)?;
        serde_json::to_string(&result).map_err(|e| Error::InvalidJSON(e.to_string()))
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
        Ok(unsafe { &mut *(engine_ptr as *mut Engine) })
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
        let namespace =
            std::str::from_utf8_unchecked(std::slice::from_raw_parts(namespace_ptr, namespace_len));
        let payload =
            std::str::from_utf8_unchecked(std::slice::from_raw_parts(payload_ptr, payload_len));

        match Engine::new(namespace, payload) {
            Ok(engine) => Box::into_raw(Box::new(engine)) as *mut c_void,
            Err(e) => {
                eprintln!("Error initializing engine: {}", e);
                std::ptr::null_mut()
            }
        }
    });

    result.unwrap_or(std::ptr::null_mut())
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
    let e = get_engine(engine_ptr).unwrap();
    let evaluation_request = unsafe {
        std::str::from_utf8_unchecked(std::slice::from_raw_parts(
            evaluation_request_ptr,
            evaluation_request_len,
        ))
    };

    let request = get_evaluation_request(evaluation_request);
    let result = result_to_string(e.evaluate_variant(&request));
    let (ptr, len) = string_to_ptr(&result);
    std::mem::forget(result); // host owns the string now and must deallocate it
    return ((ptr as u64) << 32) | len as u64;
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
    let e = get_engine(engine_ptr).unwrap();
    let evaluation_request = unsafe {
        std::str::from_utf8_unchecked(std::slice::from_raw_parts(
            evaluation_request_ptr,
            evaluation_request_len,
        ))
    };

    let request = get_evaluation_request(evaluation_request);
    let result = result_to_string(e.evaluate_boolean(&request));
    let (ptr, len) = string_to_ptr(&result);
    std::mem::forget(result); // host owns the string now and must deallocate it
    return ((ptr as u64) << 32) | len as u64;
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a batch evaluation response.
#[no_mangle]
pub unsafe extern "C" fn evaluate_batch(
    engine_ptr: *mut c_void,
    batch_evaluation_request: *const c_char,
) -> u64 {
    let e = get_engine(engine_ptr).unwrap();
    let requests = get_batch_evaluation_request(batch_evaluation_request);
    let result = result_to_string(e.evaluate_batch(requests));
    let (ptr, len) = string_to_ptr(&result);
    std::mem::forget(result); // host owns the string now and must deallocate it
    return ((ptr as u64) << 32) | len as u64;
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
#[no_mangle]
pub extern "C" fn allocate(size: usize) -> *mut c_void {
    let mut buf = vec![0; size];
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr as *mut c_void
}

/// # Safety
///
/// This function will free the memory occupied by the engine.
#[no_mangle]
pub extern "C" fn deallocate(ptr: *mut c_void, size: usize) {
    let buf = unsafe { Vec::from_raw_parts(ptr, size, size) };
    std::mem::drop(buf);
}

unsafe fn get_evaluation_request(evaluation_request: &str) -> EvaluationRequest {
    let client_eval_request: WASMEvaluationRequest =
        serde_json::from_str(evaluation_request).unwrap();

    let mut context_map: HashMap<String, String> = HashMap::new();
    if let Some(context_value) = client_eval_request.context {
        for (key, value) in context_value {
            if let serde_json::Value::String(val) = value {
                context_map.insert(key, val);
            }
        }
    }

    EvaluationRequest {
        flag_key: client_eval_request.flag_key,
        entity_id: client_eval_request.entity_id,
        context: context_map,
    }
}

unsafe fn get_batch_evaluation_request(
    batch_evaluation_request: *const c_char,
) -> Vec<EvaluationRequest> {
    let evaluation_request_bytes = CStr::from_ptr(batch_evaluation_request).to_bytes();
    let bytes_str_repr = std::str::from_utf8(evaluation_request_bytes).unwrap();

    let batch_eval_request: Vec<WASMEvaluationRequest> =
        serde_json::from_str(bytes_str_repr).unwrap();

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

    evaluation_requests
}

/// Returns a pointer and size pair for the given string in a way compatible
/// with WebAssembly numeric types.
///
/// Note: This doesn't change the ownership of the String. To intentionally
/// leak it, use [`std::mem::forget`] on the input after calling this.
unsafe fn string_to_ptr(s: &str) -> (u32, u32) {
    return (s.as_ptr() as u32, s.len() as u32);
}
