pub mod evaluator;
pub mod http;

use evaluator::Evaluator;
use fliptevaluation::error::Error;
use fliptevaluation::models::flipt;
use fliptevaluation::store::Snapshot;
use fliptevaluation::{
    BatchEvaluationResponse, BooleanEvaluationResponse, EvaluationRequest,
    VariantEvaluationResponse,
};
use http::{Authentication, HTTPFetcher, HTTPFetcherBuilder};
use libc::c_void;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::collections::HashMap;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::{Arc, Mutex};
use std::time::Duration;
use thiserror::Error;
use tokio::runtime::Runtime;
use tokio::task::JoinHandle;

#[derive(Deserialize)]
struct FFIEvaluationRequest {
    flag_key: String,
    entity_id: String,
    context: Option<Map<String, Value>>,
}

#[derive(Serialize)]
struct FFIResponse<T>
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

#[derive(Error, Debug, Clone)]
enum FFIError {
    #[error("error engine null pointer")]
    NullPointer,
}

impl<T> From<Result<T, Error>> for FFIResponse<T>
where
    T: Serialize,
{
    fn from(value: Result<T, Error>) -> Self {
        match value {
            Ok(result) => FFIResponse {
                status: Status::Success,
                result: Some(result),
                error_message: None,
            },
            Err(e) => FFIResponse {
                status: Status::Failure,
                result: None,
                error_message: Some(e.to_string()),
            },
        }
    }
}

fn result_to_json_ptr<T: Serialize>(result: Result<T, Error>) -> *mut c_char {
    let ffi_response: FFIResponse<T> = result.into();
    let json_string = serde_json::to_string(&ffi_response).unwrap();
    CString::new(json_string).unwrap().into_raw()
}

#[derive(Deserialize)]
pub struct EngineOpts {
    url: Option<String>,
    authentication: Option<Authentication>,
    update_interval: Option<u64>,
    reference: Option<String>,
}

impl Default for EngineOpts {
    fn default() -> Self {
        Self {
            url: Some("http://localhost:8080".into()),
            authentication: None,
            update_interval: Some(120),
            reference: None,
        }
    }
}

pub struct Engine {
    fetcher: HTTPFetcher,
    evaluator: Arc<Mutex<Evaluator<Snapshot>>>,
    runtime: Runtime,
    _update_handle: JoinHandle<()>,
}

impl Engine {
    pub fn new(mut fetcher: HTTPFetcher, evaluator: Evaluator<Snapshot>) -> Self {
        let runtime = Runtime::new().expect("Failed to create Tokio runtime");
        let evaluator = Arc::new(Mutex::new(evaluator));
        let evaluator_clone = evaluator.clone();

        let mut rx = runtime.block_on(async { fetcher.start() });

        let update_handle = runtime.spawn(async move {
            while let Some(res) = rx.recv().await {
                match res {
                    Ok(snapshot) => {
                        evaluator_clone
                            .lock()
                            .unwrap()
                            .replace_snapshot(Ok(snapshot));
                    }
                    Err(e) => {
                        // likely means the engine is shutting down
                        println!("error receiving snapshot: {}", e);
                        break;
                    }
                }
            }
        });

        Self {
            fetcher,
            evaluator,
            runtime,
            _update_handle: update_handle,
        }
    }

    pub fn variant(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<VariantEvaluationResponse, Error> {
        let binding = self.evaluator.clone();
        let lock = binding.lock().unwrap();

        lock.variant(evaluation_request)
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        let binding = self.evaluator.clone();
        let lock = binding.lock().unwrap();

        lock.boolean(evaluation_request)
    }

    pub fn batch(
        &self,
        batch_evaluation_request: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        let binding = self.evaluator.clone();
        let lock = binding.lock().unwrap();

        lock.batch(batch_evaluation_request)
    }

    pub fn list_flags(&self) -> Result<Vec<flipt::Flag>, Error> {
        let binding = self.evaluator.clone();
        let lock = binding.lock().unwrap();

        lock.list_flags()
    }
}

/// # Safety
///
/// This function should not be called unless an Engine is initiated. It provides a helper
/// utility to retrieve an Engine instance for evaluation use.
unsafe fn get_engine<'a>(engine_ptr: *mut c_void) -> Result<&'a mut Engine, FFIError> {
    if engine_ptr.is_null() {
        Err(FFIError::NullPointer)
    } else {
        Ok(unsafe { &mut *(engine_ptr as *mut Engine) })
    }
}

/// # Safety
///
/// This function will initialize an Engine and return a pointer back to the caller.
#[no_mangle]
pub unsafe extern "C" fn initialize_engine(
    namespace: *const c_char,
    opts: *const c_char,
) -> *mut c_void {
    let namespace = CStr::from_ptr(namespace).to_str().unwrap();

    let engine_opts_bytes = CStr::from_ptr(opts).to_bytes();
    let bytes_str_repr = std::str::from_utf8(engine_opts_bytes).unwrap();
    let engine_opts: EngineOpts = serde_json::from_str(bytes_str_repr).unwrap_or_default();

    let http_url = engine_opts
        .url
        .to_owned()
        .unwrap_or("http://localhost:8080".into());

    let authentication = engine_opts.authentication.to_owned();
    let reference = engine_opts.reference.to_owned();
    let update_interval = engine_opts.update_interval.to_owned();

    let mut fetcher_builder = HTTPFetcherBuilder::new(&http_url, namespace);

    fetcher_builder = match update_interval {
        Some(update_interval) => {
            fetcher_builder.update_interval(Duration::from_secs(update_interval))
        }
        None => fetcher_builder,
    };

    fetcher_builder = match authentication {
        Some(authentication) => fetcher_builder.authentication(authentication),
        None => fetcher_builder,
    };

    fetcher_builder = match reference {
        Some(reference) => fetcher_builder.reference(&reference),
        None => fetcher_builder,
    };

    let fetcher = fetcher_builder.build();
    let evaluator = Evaluator::new(namespace);
    let engine = Engine::new(fetcher, evaluator);

    Box::into_raw(Box::new(engine)) as *mut c_void
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a variant evaluation response.
#[no_mangle]
pub unsafe extern "C" fn evaluate_variant(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    let e = get_engine(engine_ptr).unwrap();
    let e_req = get_evaluation_request(evaluation_request);

    result_to_json_ptr(e.variant(&e_req))
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a boolean evaluation response.
#[no_mangle]
pub unsafe extern "C" fn evaluate_boolean(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    let e = get_engine(engine_ptr).unwrap();
    let e_req = get_evaluation_request(evaluation_request);

    result_to_json_ptr(e.boolean(&e_req))
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a batch evaluation response.
#[no_mangle]
pub unsafe extern "C" fn evaluate_batch(
    engine_ptr: *mut c_void,
    batch_evaluation_request: *const c_char,
) -> *const c_char {
    let e = get_engine(engine_ptr).unwrap();
    let req = get_batch_evaluation_request(batch_evaluation_request);

    result_to_json_ptr(e.batch(req))
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a list of flags for the given namespace.
#[no_mangle]
pub unsafe extern "C" fn list_flags(engine_ptr: *mut c_void) -> *const c_char {
    let res = get_engine(engine_ptr).unwrap().list_flags();

    result_to_json_ptr(res)
}

unsafe fn get_batch_evaluation_request(
    batch_evaluation_request: *const c_char,
) -> Vec<EvaluationRequest> {
    let evaluation_request_bytes = CStr::from_ptr(batch_evaluation_request).to_bytes();
    let bytes_str_repr = std::str::from_utf8(evaluation_request_bytes).unwrap();

    let batch_eval_request: Vec<FFIEvaluationRequest> =
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

unsafe fn get_evaluation_request(evaluation_request: *const c_char) -> EvaluationRequest {
    let evaluation_request_bytes = CStr::from_ptr(evaluation_request).to_bytes();
    let bytes_str_repr = std::str::from_utf8(evaluation_request_bytes).unwrap();
    let client_eval_request: FFIEvaluationRequest = serde_json::from_str(bytes_str_repr).unwrap();

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

/// # Safety
///
/// This function will free the memory occupied by the engine.
#[no_mangle]
pub unsafe extern "C" fn destroy_engine(engine_ptr: *mut c_void) {
    if engine_ptr.is_null() {
        return;
    }

    let mut engine = Box::from_raw(engine_ptr as *mut Engine);
    engine.fetcher.stop();

    // Shutdown the runtime
    engine.runtime.shutdown_background();

    // The engine will be dropped here, cleaning up all resources
}

/// # Safety
///
/// This function will take in a pointer to the string and free the memory.
/// See Rust the safety section in CString::from_raw.
#[no_mangle]
pub unsafe extern "C" fn destroy_string(ptr: *mut c_char) {
    let _ = CString::from_raw(ptr);
}
