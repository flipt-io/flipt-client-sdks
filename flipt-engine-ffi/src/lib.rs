pub mod evaluator;
pub mod http;

use base64::prelude::BASE64_STANDARD;
use base64::Engine as Base64Engine;
use evaluator::Evaluator;
use fliptevaluation::error::Error;
use fliptevaluation::models::{flipt, snapshot};
use fliptevaluation::{
    BatchEvaluationResponse, BooleanEvaluationResponse, EvaluationRequest,
    VariantEvaluationResponse,
};
use http::{
    Authentication, ErrorStrategy, FetchMode, FetcherHandle, HTTPFetcher, HTTPFetcherBuilder,
};
use libc::c_void;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::collections::HashMap;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::str::Utf8Error;
use std::sync::{Arc, OnceLock};
use std::time::Duration;
use thiserror::Error;
use tokio::runtime::Builder;
use tokio::runtime::Handle;
use tokio::runtime::Runtime;
use tokio::sync::Mutex as TokioMutex;
use tokio::sync::RwLock;

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
#[non_exhaustive]
enum Status {
    #[serde(rename = "success")]
    Success,
    #[serde(rename = "failure")]
    Failure,
}

#[derive(Error, Debug, Clone)]
#[non_exhaustive]
enum FFIError {
    #[error("error engine null pointer")]
    NullPointer,
}

impl<T, E> From<Result<T, E>> for FFIResponse<T>
where
    T: Serialize,
    E: std::error::Error,
{
    fn from(value: Result<T, E>) -> Self {
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

fn result_to_json_ptr<T: Serialize, E: std::error::Error>(result: Result<T, E>) -> *mut c_char {
    let ffi_response: FFIResponse<T> = result.into();
    let json_string = serde_json::to_string(&ffi_response).unwrap();
    CString::new(json_string).unwrap().into_raw()
}

#[derive(Deserialize, Debug)]
pub struct EngineOpts {
    url: Option<String>,
    authentication: Option<Authentication>,
    request_timeout: Option<u64>,
    update_interval: Option<u64>,
    fetch_mode: Option<FetchMode>,
    reference: Option<String>,
    error_strategy: Option<ErrorStrategy>,
}

impl Default for EngineOpts {
    fn default() -> Self {
        Self {
            url: Some("http://localhost:8080".into()),
            authentication: None,
            request_timeout: None,
            update_interval: Some(120),
            reference: None,
            fetch_mode: Some(FetchMode::default()),
            error_strategy: Some(ErrorStrategy::Fail),
        }
    }
}

pub struct Engine {
    evaluator: Arc<RwLock<Evaluator<snapshot::Snapshot>>>,
    fetcher_handle: Arc<TokioMutex<FetcherHandle>>,
}

static RUNTIME: OnceLock<Runtime> = OnceLock::new();

fn get_or_create_runtime() -> &'static Handle {
    let runtime = RUNTIME.get_or_init(|| {
        // If we're not in a runtime, create one
        Builder::new_multi_thread()
            .thread_name("flipt-engine-ffi")
            .enable_all()
            .build()
            .expect("Failed to create runtime")
    });

    // Return either the existing runtime's handle or our created runtime's handle
    match Handle::try_current() {
        Ok(handle) => unsafe { std::mem::transmute::<&Handle, &'static Handle>(&handle) },
        Err(_) => runtime.handle(),
    }
}

impl Engine {
    pub fn new(
        namespace: &str,
        fetcher: HTTPFetcher,
        evaluator: Evaluator<snapshot::Snapshot>,
        error_strategy: ErrorStrategy,
    ) -> Self {
        let evaluator = Arc::new(RwLock::new(evaluator));
        let evaluator_clone = evaluator.clone();
        let namespace_clone = namespace.to_string();
        let fetcher_handle = Arc::new(TokioMutex::new(FetcherHandle::new(fetcher)));

        let handle = get_or_create_runtime();

        // Block on initial fetch
        handle.block_on(async {
            let mut fh = fetcher_handle.lock().await;
            match fh.fetcher_mut().initial_fetch().await {
                Ok(doc) => {
                    let snap = snapshot::Snapshot::build(&namespace_clone, doc);
                    let mut lock = evaluator_clone.write().await;
                    lock.replace_snapshot(Ok(snap));
                }
                Err(err) => {
                    if error_strategy == ErrorStrategy::Fail {
                        let mut lock = evaluator_clone.write().await;
                        lock.replace_snapshot(Err(err));
                    }
                }
            }
        });

        // Start the fetcher and spawn the background task
        let fetcher_handle_bg = fetcher_handle.clone();
        let mut rx = handle.block_on(async {
            let mut fh = fetcher_handle_bg.lock().await;
            fh.start()
        });

        let evaluator_clone_bg = evaluator.clone();
        let namespace_clone_bg = namespace_clone.clone();
        handle.spawn(async move {
            loop {
                let res = rx.recv().await;
                match res {
                    Some(Ok(doc)) => {
                        let snap = snapshot::Snapshot::build(&namespace_clone_bg, doc);
                        let mut lock = evaluator_clone_bg.write().await;
                        lock.replace_snapshot(Ok(snap));
                    }
                    Some(Err(err)) => {
                        if error_strategy == ErrorStrategy::Fail {
                            let mut lock = evaluator_clone_bg.write().await;
                            lock.replace_snapshot(Err(err));
                        }
                    }
                    None => break, // Channel closed, exit loop
                }
            }
        });

        Self {
            evaluator,
            fetcher_handle,
        }
    }

    /// Helper to lock the evaluator and run a closure, mapping lock errors to Error::Internal
    async fn with_evaluator_read_lock_async<F, R>(&self, f: F) -> Result<R, Error>
    where
        F: FnOnce(&Evaluator<snapshot::Snapshot>) -> Result<R, Error>,
    {
        let lock = self.evaluator.read().await;
        f(&lock)
    }

    async fn with_evaluator_write_lock_async<F, R>(&self, f: F) -> Result<R, Error>
    where
        F: FnOnce(&mut Evaluator<snapshot::Snapshot>) -> Result<R, Error>,
    {
        let mut lock = self.evaluator.write().await;
        f(&mut lock)
    }

    pub async fn variant_async(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<VariantEvaluationResponse, Error> {
        self.with_evaluator_read_lock_async(|lock| lock.variant(evaluation_request))
            .await
    }

    pub async fn boolean_async(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        self.with_evaluator_read_lock_async(|lock| lock.boolean(evaluation_request))
            .await
    }

    pub async fn batch_async(
        &self,
        batch_evaluation_request: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        self.with_evaluator_read_lock_async(|lock| lock.batch(batch_evaluation_request))
            .await
    }

    pub async fn list_flags_async(&self) -> Result<Vec<flipt::Flag>, Error> {
        self.with_evaluator_read_lock_async(|lock| lock.list_flags())
            .await
    }

    pub async fn set_snapshot_async(&self, snapshot: snapshot::Snapshot) -> Result<(), Error> {
        // Stop the fetcher
        {
            let mut fh = self.fetcher_handle.lock().await;
            fh.stop().await;
        }

        // Replace the snapshot
        self.with_evaluator_write_lock_async(|lock| {
            lock.replace_snapshot(Ok(snapshot));
            Ok(())
        })
        .await?;

        // Restart the fetcher
        {
            let mut fh = self.fetcher_handle.lock().await;
            fh.start();
        }

        Ok(())
    }

    pub async fn get_snapshot_async(&self) -> Result<snapshot::Snapshot, Error> {
        self.with_evaluator_read_lock_async(|lock| lock.get_snapshot())
            .await
    }
}

// Public FFI functions

/// # Safety
///
/// This function will initialize an Engine and return a pointer back to the caller.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn initialize_engine_ffi(
    namespace: *const c_char,
    opts: *const c_char,
) -> *mut c_void {
    _initialize_engine(namespace, opts)
}

/// # Safety
///
/// This function will initialize an Engine and return a pointer back to the caller.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn initialize_engine(
    namespace: *const c_char,
    opts: *const c_char,
) -> *mut c_void {
    _initialize_engine(namespace, opts)
}

/// # Safety
///
/// This function will take in a pointer to the engine and a snapshot string, and replace the in-memory snapshot.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn set_snapshot_ffi(
    engine_ptr: *mut c_void,
    snapshot: *const c_char,
) -> *const c_char {
    _set_snapshot(engine_ptr, snapshot)
}

/// # Safety
///
/// This function will take in a pointer to the engine and a snapshot string, and replace the in-memory snapshot.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn set_snapshot(
    engine_ptr: *mut c_void,
    snapshot: *const c_char,
) -> *const c_char {
    _set_snapshot(engine_ptr, snapshot)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return the current snapshot as a JSON string.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn get_snapshot_ffi(engine_ptr: *mut c_void) -> *const c_char {
    _get_snapshot(engine_ptr)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return the current snapshot as a JSON string.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn get_snapshot(engine_ptr: *mut c_void) -> *const c_char {
    _get_snapshot(engine_ptr)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a variant evaluation response.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn evaluate_variant_ffi(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    _evaluate_variant(engine_ptr, evaluation_request)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a variant evaluation response.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn evaluate_variant(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    _evaluate_variant(engine_ptr, evaluation_request)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a boolean evaluation response.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn evaluate_boolean_ffi(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    _evaluate_boolean(engine_ptr, evaluation_request)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a boolean evaluation response.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn evaluate_boolean(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    _evaluate_boolean(engine_ptr, evaluation_request)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a batch evaluation response.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn evaluate_batch_ffi(
    engine_ptr: *mut c_void,
    batch_evaluation_request: *const c_char,
) -> *const c_char {
    _evaluate_batch(engine_ptr, batch_evaluation_request)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a batch evaluation response.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn evaluate_batch(
    engine_ptr: *mut c_void,
    batch_evaluation_request: *const c_char,
) -> *const c_char {
    _evaluate_batch(engine_ptr, batch_evaluation_request)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a list of flags.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn list_flags_ffi(engine_ptr: *mut c_void) -> *const c_char {
    _list_flags(engine_ptr)
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a list of flags.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn list_flags(engine_ptr: *mut c_void) -> *const c_char {
    _list_flags(engine_ptr)
}

/// # Safety
///
/// This function will take in a pointer to the engine and destroy it.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn destroy_engine_ffi(engine_ptr: *mut c_void) {
    _destroy_engine(engine_ptr)
}

/// # Safety
///
/// This function will take in a pointer to the engine and destroy it.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn destroy_engine(engine_ptr: *mut c_void) {
    _destroy_engine(engine_ptr)
}

/// # Safety
///
/// This function will take in a pointer to a string and destroy it.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn destroy_string_ffi(ptr: *mut c_char) {
    _destroy_string(ptr)
}

/// # Safety
///
/// This function will take in a pointer to a string and destroy it.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn destroy_string(ptr: *mut c_char) {
    _destroy_string(ptr)
}

// Private implementation functions
unsafe extern "C" fn _initialize_engine(
    namespace: *const c_char,
    opts: *const c_char,
) -> *mut c_void {
    let result = std::panic::catch_unwind(|| {
        // Null pointer checks
        if namespace.is_null() || opts.is_null() {
            return std::ptr::null_mut();
        }

        // Safe string conversion with error handling
        let namespace = match CStr::from_ptr(namespace).to_str() {
            Ok(s) => s,
            Err(_) => return std::ptr::null_mut(),
        };

        let engine_opts_bytes = CStr::from_ptr(opts).to_bytes();

        let bytes_str_repr = match std::str::from_utf8(engine_opts_bytes) {
            Ok(s) => s,
            Err(_) => return std::ptr::null_mut(),
        };

        // Safe JSON parsing with error handling
        let engine_opts: EngineOpts = serde_json::from_str(bytes_str_repr).unwrap_or_default();

        let mut fetcher_builder = HTTPFetcherBuilder::new(
            &engine_opts
                .url
                .unwrap_or_else(|| "http://localhost:8080".into()),
            namespace,
        );

        if let Some(request_timeout) = engine_opts.request_timeout {
            fetcher_builder = fetcher_builder.request_timeout(Duration::from_secs(request_timeout));
        }

        if let Some(update_interval) = engine_opts.update_interval {
            fetcher_builder = fetcher_builder.update_interval(Duration::from_secs(update_interval));
        }

        if let Some(authentication) = engine_opts.authentication {
            fetcher_builder = fetcher_builder.authentication(authentication);
        }

        if let Some(fetch_mode) = engine_opts.fetch_mode {
            fetcher_builder = fetcher_builder.mode(fetch_mode);
        }

        if let Some(reference) = &engine_opts.reference {
            fetcher_builder = fetcher_builder.reference(reference);
        }

        let fetcher = fetcher_builder.build().unwrap();

        let evaluator = Evaluator::new(namespace);

        let engine = Engine::new(
            namespace,
            fetcher,
            evaluator,
            engine_opts.error_strategy.unwrap_or_default(),
        );

        // Convert to raw pointer
        Box::into_raw(Box::new(engine)) as *mut c_void
    });

    result.unwrap_or(std::ptr::null_mut())
}

unsafe extern "C" fn _set_snapshot(
    engine_ptr: *mut c_void,
    snapshot: *const c_char,
) -> *const c_char {
    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    let snapshot_bytes = CStr::from_ptr(snapshot).to_bytes();
    let snapshot_str = match std::str::from_utf8(snapshot_bytes) {
        Ok(s) => s,
        Err(e) => return result_to_json_ptr::<(), Utf8Error>(Err(e)),
    };

    // Base64 decode the string
    let decoded = match BASE64_STANDARD.decode(snapshot_str) {
        Ok(decoded) => decoded,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    let snap: snapshot::Snapshot = match serde_json::from_slice(&decoded) {
        Ok(snap) => snap,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    let rt = get_or_create_runtime();
    rt.block_on(async {
        match e.set_snapshot_async(snap).await {
            Ok(()) => result_to_json_ptr::<(), std::convert::Infallible>(Ok(())),
            Err(err) => result_to_json_ptr::<(), _>(Err(err)),
        }
    })
}

unsafe extern "C" fn _get_snapshot(engine_ptr: *mut c_void) -> *const c_char {
    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    let rt = get_or_create_runtime();
    let snapshot_result = rt.block_on(async { e.get_snapshot_async().await });

    let json = match &snapshot_result {
        Ok(snapshot) => match serde_json::to_string(snapshot) {
            Ok(s) => s,
            Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
        },
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    // Base64 encode the JSON string
    let encoded = BASE64_STANDARD.encode(json);

    match CString::new(encoded) {
        Ok(cstr) => cstr.into_raw(),
        Err(e) => {
            let err = format!("CString conversion failed: {e}");
            let err_json = serde_json::to_string(&FFIResponse::<()> {
                status: Status::Failure,
                result: None,
                error_message: Some(err),
            })
            .unwrap_or_else(|_| {
                "{\"status\":\"failure\",\"error_message\":\"Unknown error\"}".to_string()
            });
            CString::new(err_json).unwrap().into_raw()
        }
    }
}

unsafe extern "C" fn _evaluate_variant(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };
    let e_req = get_evaluation_request(evaluation_request);
    let rt = get_or_create_runtime();
    rt.block_on(async { result_to_json_ptr(e.variant_async(&e_req).await) })
}

unsafe extern "C" fn _evaluate_boolean(
    engine_ptr: *mut c_void,
    evaluation_request: *const c_char,
) -> *const c_char {
    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };
    let e_req = get_evaluation_request(evaluation_request);
    let rt = get_or_create_runtime();
    rt.block_on(async { result_to_json_ptr(e.boolean_async(&e_req).await) })
}

unsafe extern "C" fn _evaluate_batch(
    engine_ptr: *mut c_void,
    batch_evaluation_request: *const c_char,
) -> *const c_char {
    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };
    let req = get_batch_evaluation_request(batch_evaluation_request);
    let rt = get_or_create_runtime();
    rt.block_on(async { result_to_json_ptr(e.batch_async(req).await) })
}

unsafe extern "C" fn _list_flags(engine_ptr: *mut c_void) -> *const c_char {
    let res = match get_engine(engine_ptr) {
        Ok(e) => {
            let rt = get_or_create_runtime();
            rt.block_on(async { e.list_flags_async().await })
        }
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    result_to_json_ptr(res)
}

unsafe extern "C" fn _destroy_engine(engine_ptr: *mut c_void) {
    if engine_ptr.is_null() {
        return;
    }

    let engine = Box::from_raw(engine_ptr as *mut Engine);
    // Stop the fetcher handle if running
    let fetcher_handle = engine.fetcher_handle.clone();
    let rt = get_or_create_runtime();
    rt.block_on(async {
        let mut fh = fetcher_handle.lock().await;
        fh.stop().await;
    });
    // The engine will be dropped here, cleaning up all resources
}

unsafe extern "C" fn _destroy_string(ptr: *mut c_char) {
    let _ = CString::from_raw(ptr);
}

// Helper functions

/// This function should not be called unless an Engine is initiated. It provides a helper
/// utility to retrieve an Engine instance for evaluation use.
unsafe fn get_engine<'a>(engine_ptr: *mut c_void) -> Result<&'a mut Engine, FFIError> {
    if engine_ptr.is_null() {
        Err(FFIError::NullPointer)
    } else {
        Ok(&mut *(engine_ptr as *mut Engine))
    }
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

#[cfg(test)]
mod tests {
    use crate::{http::ErrorStrategy, EngineOpts};

    #[test]
    fn test_engine_ops_with_error_strategy() {
        let input = r#"{"url":"http://localhost:8080","update_interval":120,"authentication":null,"error_strategy":"fallback"}"#;
        let engine_opts: EngineOpts = serde_json::from_str(input).unwrap_or_default();
        assert_eq!(ErrorStrategy::Fallback, engine_opts.error_strategy.unwrap());
    }
}
