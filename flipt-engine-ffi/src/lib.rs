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
use http::{Authentication, ErrorStrategy, FetchMode, HTTPFetcher, HTTPFetcherBuilder};
use libc::c_void;
use log::{debug, error, trace, warn};
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::collections::HashMap;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, OnceLock, RwLock};
use std::time::Duration;
use thiserror::Error;
use tokio::runtime::Builder;
use tokio::runtime::Handle;
use tokio::runtime::Runtime;
use tokio::sync::Notify;

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
    environment: Option<String>,
    namespace: Option<String>,
    url: Option<String>,
    authentication: Option<Authentication>,
    request_timeout: Option<u64>,
    update_interval: Option<u64>,
    fetch_mode: Option<FetchMode>,
    reference: Option<String>,
    error_strategy: Option<ErrorStrategy>,
    snapshot: Option<String>,
    tls_config: Option<TlsConfig>,
}

#[derive(Deserialize, Debug, PartialEq)]
pub struct TlsConfig {
    /// Path to custom CA certificate file (PEM format)
    ca_cert_file: Option<String>,
    /// Raw CA certificate content (PEM format)
    ca_cert_data: Option<String>,
    /// Skip certificate verification (insecure - for development only)
    insecure_skip_verify: Option<bool>,
    /// Client certificate file for mutual TLS (PEM format)
    client_cert_file: Option<String>,
    /// Client key file for mutual TLS (PEM format)
    client_key_file: Option<String>,
    /// Raw client certificate content (PEM format)
    client_cert_data: Option<String>,
    /// Raw client key content (PEM format)
    client_key_data: Option<String>,
}

impl Default for EngineOpts {
    fn default() -> Self {
        Self {
            environment: Some("default".into()),
            namespace: Some("default".into()),
            url: Some("http://localhost:8080".into()),
            authentication: None,
            request_timeout: None,
            update_interval: Some(120),
            reference: None,
            fetch_mode: Some(FetchMode::default()),
            error_strategy: Some(ErrorStrategy::Fail),
            snapshot: None,
            tls_config: None,
        }
    }
}

pub struct Engine {
    evaluator: Arc<RwLock<Evaluator<snapshot::Snapshot>>>,
    stop_signal: Arc<AtomicBool>,
    fetcher_handle: Option<tokio::task::JoinHandle<()>>,
    stop_notify: Arc<Notify>,
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

fn init_logging() {
    use std::sync::Once;
    static INIT: Once = Once::new();

    INIT.call_once(|| {
        let _ = env_logger::Builder::from_env("FLIPT_ENGINE_LOG").try_init();
    });
}

impl Engine {
    pub fn new(
        mut fetcher: HTTPFetcher,
        evaluator: Evaluator<snapshot::Snapshot>,
        error_strategy: ErrorStrategy,
        initial_snapshot: snapshot::Snapshot,
    ) -> Self {
        let stop_signal = Arc::new(AtomicBool::new(false));
        let stop_signal_clone = stop_signal.clone();

        let stop_notify = Arc::new(Notify::new());
        let stop_notify_clone = stop_notify.clone();

        let evaluator = Arc::new(RwLock::new(evaluator));
        let evaluator_clone = evaluator.clone();

        let handle = get_or_create_runtime();

        // Set the initial snapshot
        if let Ok(mut lock) = evaluator.write() {
            lock.replace_snapshot(Ok(initial_snapshot));
        }

        // Block on initial fetch
        handle.block_on(async {
            match fetcher.initial_fetch().await {
                Ok(doc) => {
                    debug!("initial fetch succeeded");
                    let snap = snapshot::Snapshot::build(doc);
                    if let Ok(mut lock) = evaluator.write() {
                        lock.replace_snapshot(Ok(snap));
                    }
                }
                Err(err) => {
                    warn!("initial fetch failed: {err:?}");
                    if error_strategy == ErrorStrategy::Fail {
                        if let Ok(mut lock) = evaluator.write() {
                            lock.replace_snapshot(Err(err));
                        }
                    }
                }
            }
        });

        // Spawn the continuous polling task and store the JoinHandle
        let fetcher_handle = handle.spawn(async move {
            let mut rx = fetcher.start(stop_signal_clone, stop_notify_clone);
            while let Some(res) = rx.recv().await {
                match res {
                    Ok(doc) => {
                        debug!("fetch succeeded");
                        let snap = snapshot::Snapshot::build(doc);
                        if let Ok(mut lock) = evaluator_clone.write() {
                            lock.replace_snapshot(Ok(snap));
                        }
                    }
                    Err(err) => {
                        warn!("fetch failed: {err:?}");
                        if error_strategy == ErrorStrategy::Fail {
                            if let Ok(mut lock) = evaluator_clone.write() {
                                lock.replace_snapshot(Err(err));
                            }
                        }
                    }
                }
            }
        });

        Self {
            evaluator,
            stop_signal,
            fetcher_handle: Some(fetcher_handle),
            stop_notify,
        }
    }

    /// Helper to lock the evaluator for reading and run a closure, mapping lock errors to Error::Internal
    fn with_evaluator_read_lock<F, R>(&self, f: F) -> Result<R, Error>
    where
        F: FnOnce(&Evaluator<snapshot::Snapshot>) -> Result<R, Error>,
    {
        let lock = self
            .evaluator
            .read()
            .map_err(|_| Error::Internal("failed to acquire lock".to_string()))?;
        f(&lock)
    }

    pub fn variant(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<VariantEvaluationResponse, Error> {
        self.with_evaluator_read_lock(|lock| lock.variant(evaluation_request))
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        self.with_evaluator_read_lock(|lock| lock.boolean(evaluation_request))
    }

    pub fn batch(
        &self,
        batch_evaluation_request: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        self.with_evaluator_read_lock(|lock| lock.batch(batch_evaluation_request))
    }

    pub fn list_flags(&self) -> Result<Vec<flipt::Flag>, Error> {
        self.with_evaluator_read_lock(|lock| lock.list_flags())
    }

    pub fn get_snapshot(&self) -> Result<snapshot::Snapshot, Error> {
        self.with_evaluator_read_lock(|lock| lock.get_snapshot())
    }
}

// Public FFI functions

/// # Safety
///
/// This function will initialize an Engine and return a pointer back to the caller.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn initialize_engine_ffi(opts: *const c_char) -> *mut c_void {
    match std::panic::catch_unwind(|| {
        init_logging();
        trace!(
            "initialize_engine_ffi called: opts ptr=0x{:x}",
            opts as usize
        );
        let ptr = _initialize_engine(opts);
        trace!(
            "initialize_engine_ffi returning engine ptr=0x{:x}",
            ptr as usize
        );
        ptr
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in initialize_engine_ffi: {e:?}");
            std::ptr::null_mut()
        }
    }
}

/// # Safety
///
/// This function will initialize an Engine and return a pointer back to the caller.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn initialize_engine(opts: *const c_char) -> *mut c_void {
    match std::panic::catch_unwind(|| {
        init_logging();
        trace!("initialize_engine called: opts ptr=0x{:x}", opts as usize);
        let ptr = _initialize_engine(opts);
        trace!(
            "initialize_engine returning engine ptr=0x{:x}",
            ptr as usize
        );
        ptr
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in initialize_engine: {e:?}");
            std::ptr::null_mut()
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to the engine and return the current snapshot as a JSON string.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn get_snapshot_ffi(engine_ptr: *mut c_void) -> *const c_char {
    match std::panic::catch_unwind(|| {
        trace!(
            "get_snapshot_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _get_snapshot(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in get_snapshot_ffi: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in get_snapshot_ffi".to_string(),
            )))
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to the engine and return the current snapshot as a JSON string.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn get_snapshot(engine_ptr: *mut c_void) -> *const c_char {
    match std::panic::catch_unwind(|| {
        trace!(
            "get_snapshot called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _get_snapshot(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in get_snapshot: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal("panic in get_snapshot".to_string())))
        }
    }
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
    match std::panic::catch_unwind(|| {
        trace!(
            "evaluate_variant_ffi called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_variant(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in evaluate_variant_ffi: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in evaluate_variant_ffi".to_string(),
            )))
        }
    }
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
    match std::panic::catch_unwind(|| {
        trace!(
            "evaluate_variant called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_variant(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in evaluate_variant: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in evaluate_variant".to_string(),
            )))
        }
    }
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
    match std::panic::catch_unwind(|| {
        trace!(
            "evaluate_boolean_ffi called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_boolean(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in evaluate_boolean_ffi: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in evaluate_boolean_ffi".to_string(),
            )))
        }
    }
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
    match std::panic::catch_unwind(|| {
        trace!(
            "evaluate_boolean called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_boolean(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in evaluate_boolean: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in evaluate_boolean".to_string(),
            )))
        }
    }
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
    match std::panic::catch_unwind(|| {
        trace!(
            "evaluate_batch_ffi called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            batch_evaluation_request as usize
        );
        _evaluate_batch(engine_ptr, batch_evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in evaluate_batch_ffi: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in evaluate_batch_ffi".to_string(),
            )))
        }
    }
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
    match std::panic::catch_unwind(|| {
        trace!(
            "evaluate_batch called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            batch_evaluation_request as usize
        );
        _evaluate_batch(engine_ptr, batch_evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in evaluate_batch: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal("panic in evaluate_batch".to_string())))
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a list of flags.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn list_flags_ffi(engine_ptr: *mut c_void) -> *const c_char {
    match std::panic::catch_unwind(|| {
        trace!(
            "list_flags_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _list_flags(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in list_flags_ffi: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal("panic in list_flags_ffi".to_string())))
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to the engine and return a list of flags.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn list_flags(engine_ptr: *mut c_void) -> *const c_char {
    match std::panic::catch_unwind(|| {
        trace!("list_flags called: engine ptr=0x{:x}", engine_ptr as usize);
        _list_flags(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            error!("PANIC in list_flags: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal("panic in list_flags".to_string())))
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to the engine and destroy it.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn destroy_engine_ffi(engine_ptr: *mut c_void) {
    match std::panic::catch_unwind(|| {
        trace!(
            "destroy_engine_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _destroy_engine(engine_ptr)
        // No return value
    }) {
        Ok(_) => (),
        Err(e) => {
            error!("PANIC in destroy_engine_ffi: {e:?}");
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to the engine and destroy it.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn destroy_engine(engine_ptr: *mut c_void) {
    match std::panic::catch_unwind(|| {
        trace!(
            "destroy_engine called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _destroy_engine(engine_ptr)
        // No return value
    }) {
        Ok(_) => (),
        Err(e) => {
            error!("PANIC in destroy_engine: {e:?}");
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to a string and destroy it.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn destroy_string_ffi(ptr: *mut c_char) {
    match std::panic::catch_unwind(|| {
        trace!("destroy_string_ffi called: ptr=0x{:x}", ptr as usize);
        _destroy_string(ptr)
    }) {
        Ok(_) => (),
        Err(e) => {
            error!("PANIC in destroy_string_ffi: {e:?}");
        }
    }
}

/// # Safety
///
/// This function will take in a pointer to a string and destroy it.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn destroy_string(ptr: *mut c_char) {
    match std::panic::catch_unwind(|| {
        trace!("destroy_string called: ptr=0x{:x}", ptr as usize);
        _destroy_string(ptr)
    }) {
        Ok(_) => (),
        Err(e) => {
            error!("PANIC in destroy_string: {e:?}");
        }
    }
}

// Private implementation functions
unsafe extern "C" fn _initialize_engine(opts: *const c_char) -> *mut c_void {
    let result = std::panic::catch_unwind(|| {
        // Null pointer checks
        if opts.is_null() {
            return std::ptr::null_mut();
        }

        let engine_opts_bytes = CStr::from_ptr(opts).to_bytes();

        let bytes_str_repr = match std::str::from_utf8(engine_opts_bytes) {
            Ok(s) => s,
            Err(_) => return std::ptr::null_mut(),
        };

        // Safe JSON parsing with error handling
        let engine_opts: EngineOpts = serde_json::from_str(bytes_str_repr).unwrap_or_default();

        let mut fetcher_builder = HTTPFetcherBuilder::new(
            engine_opts
                .url
                .as_deref()
                .unwrap_or("http://localhost:8080"),
        );

        if let Some(environment) = engine_opts.environment {
            fetcher_builder = fetcher_builder.environment(&environment);
        }

        let namespace = engine_opts
            .namespace
            .as_deref()
            .unwrap_or("default")
            .to_string();

        fetcher_builder = fetcher_builder.namespace(&namespace);

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

        if let Some(tls_config) = engine_opts.tls_config {
            fetcher_builder = fetcher_builder.tls_config(tls_config);
        }

        let fetcher = fetcher_builder.build().unwrap_or_else(|e| {
            error!("Failed to build custom fetcher: {e}");
            HTTPFetcherBuilder::default().build().unwrap()
        });

        let evaluator = Evaluator::new(&namespace);

        // Handle initial snapshot if provided
        let initial_snapshot = engine_opts
            .snapshot
            .as_ref()
            .and_then(|snapshot_b64| {
                BASE64_STANDARD
                    .decode(snapshot_b64)
                    .ok()
                    .and_then(|decoded| serde_json::from_slice::<snapshot::Snapshot>(&decoded).ok())
            })
            .unwrap_or_default();

        let engine = Engine::new(
            fetcher,
            evaluator,
            engine_opts.error_strategy.unwrap_or_default(),
            initial_snapshot,
        );

        // Convert to raw pointer
        Box::into_raw(Box::new(engine)) as *mut c_void
    });

    result.unwrap_or(std::ptr::null_mut())
}

unsafe extern "C" fn _get_snapshot(engine_ptr: *mut c_void) -> *const c_char {
    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    let snapshot_result = e.get_snapshot();

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

    result_to_json_ptr(e.variant(&e_req))
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

    result_to_json_ptr(e.boolean(&e_req))
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

    result_to_json_ptr(e.batch(req))
}

unsafe extern "C" fn _list_flags(engine_ptr: *mut c_void) -> *const c_char {
    let res = match get_engine(engine_ptr) {
        Ok(e) => e.list_flags(),
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    result_to_json_ptr(res)
}

unsafe extern "C" fn _destroy_engine(engine_ptr: *mut c_void) {
    if engine_ptr.is_null() {
        return;
    }
    let engine = Box::from_raw(engine_ptr as *mut Engine);
    engine.stop_signal.store(true, Ordering::Relaxed);
    // Notify the fetcher task to stop
    engine.stop_notify.notify_waiters();
    // Wait for fetcher task to exit
    if let Some(handle) = engine.fetcher_handle {
        let rt = get_or_create_runtime();
        let _ = rt.block_on(handle);
    }

    // Engine is dropped here
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
    use super::*;

    #[test]
    fn test_engine_opts_with_invalid_snapshot() {
        let json = r#"{\"url\":\"http://localhost:8080\",\"snapshot\":\"eyJmb28iOiJiYXIifQ==\",\"error_strategy\":\"fallback\"}"#;

        match serde_json::from_str::<EngineOpts>(json) {
            Ok(_) => panic!("Expected error, but got Ok"),
            Err(_) => (),
        }
    }

    #[test]
    fn test_engine_opts_with_valid_snapshot() {
        let snapshot = r#"
{
    "version": 1,
    "namespace": {
        "key": "default",
        "flags": {
            "flag1": {
                "key": "flag1",
                "enabled": true,
                "type": "VARIANT_FLAG_TYPE",
                "description": "flag description"
            },
            "flag_boolean": {
                "key": "flag_boolean",
                "enabled": true,
                "type": "BOOLEAN_FLAG_TYPE",
                "description": "flag description"
            }
        },
        "eval_rules": {
            "flag1": [
                {
                    "id": "flag1-1",
                    "flag_key": "flag1",
                    "segments": {
                        "segment1": {
                            "segment_key": "segment1",
                            "match_type": "ANY_SEGMENT_MATCH_TYPE",
                            "constraints": [
                                {
                                    "type": "STRING_CONSTRAINT_COMPARISON_TYPE",
                                    "property": "fizz",
                                    "operator": "eq",
                                    "value": "buzz"
                                }
                            ]
                        }
                    },
                    "rank": 1,
                    "segment_operator": "OR_SEGMENT_OPERATOR"
                }
            ],
            "flag_boolean": []
        },
        "eval_rollouts": {
            "flag1": [],
            "flag_boolean": [
                {
                    "rollout_type": "SEGMENT_ROLLOUT_TYPE",
                    "rank": 1,
                    "segment": {
                        "value": true,
                        "segment_operator": "OR_SEGMENT_OPERATOR",
                        "segments": {
                            "segment1": {
                                "segment_key": "segment1",
                                "match_type": "ANY_SEGMENT_MATCH_TYPE",
                                "constraints": [
                                    {
                                        "type": "STRING_CONSTRAINT_COMPARISON_TYPE",
                                        "property": "fizz",
                                        "operator": "eq",
                                        "value": "buzz"
                                    }
                                ]
                            }
                        }
                    }
                },
                {
                    "rollout_type": "THRESHOLD_ROLLOUT_TYPE",
                    "rank": 2,
                    "threshold": {
                        "percentage": 50.0,
                        "value": true
                    }
                }
            ]
        },
        "eval_distributions": {
            "flag1-1": [
                {
                    "rule_id": "flag1-1",
                    "rollout": 100.0,
                    "variant_key": "variant1"
                }
            ]
        }
    }
}
"#;

        let encoded_snapshot = BASE64_STANDARD.encode(snapshot);

        let json = format!(
            r#"{{"url":"http://localhost:8080","snapshot":"{}","error_strategy":"fallback"}}"#,
            encoded_snapshot
        );

        let opts: EngineOpts = serde_json::from_str(&json).unwrap();
        assert_eq!(opts.url, Some("http://localhost:8080".to_string()));
        assert_eq!(opts.snapshot, Some(encoded_snapshot));
        assert_eq!(opts.error_strategy, Some(ErrorStrategy::Fallback));
    }

    #[test]
    fn test_engine_opts_default() {
        let opts: EngineOpts = EngineOpts::default();
        assert_eq!(opts.url, Some("http://localhost:8080".to_string()));
        assert_eq!(opts.authentication, None);
        assert_eq!(opts.request_timeout, None);
        assert_eq!(opts.update_interval, Some(120));
        assert_eq!(opts.reference, None);
        assert_eq!(opts.fetch_mode, Some(FetchMode::default()));
        assert_eq!(opts.error_strategy, Some(ErrorStrategy::Fail));
        assert_eq!(opts.snapshot, None);
        assert_eq!(opts.tls_config, None);
    }

    #[test]
    fn test_engine_opts_with_tls_config() {
        let json = r#"{
            "url": "https://localhost:8443",
            "tls_config": {
                "ca_cert_file": "/path/to/ca.crt",
                "insecure_skip_verify": true
            }
        }"#;

        let opts: EngineOpts = serde_json::from_str(json).unwrap();
        assert_eq!(opts.url, Some("https://localhost:8443".to_string()));

        let tls_config = opts.tls_config.unwrap();
        assert_eq!(tls_config.ca_cert_file, Some("/path/to/ca.crt".to_string()));
        assert_eq!(tls_config.insecure_skip_verify, Some(true));
        assert_eq!(tls_config.ca_cert_data, None);
        assert_eq!(tls_config.client_cert_file, None);
        assert_eq!(tls_config.client_key_file, None);
        assert_eq!(tls_config.client_cert_data, None);
        assert_eq!(tls_config.client_key_data, None);
    }

    #[test]
    fn test_engine_opts_with_client_certificates() {
        let json = r#"{
            "url": "https://localhost:8443",
            "tls_config": {
                "client_cert_data": "Y2VydGRhdGE=",
                "client_key_data": "a2V5ZGF0YQ=="
            }
        }"#;

        let opts: EngineOpts = serde_json::from_str(json).unwrap();
        assert_eq!(opts.url, Some("https://localhost:8443".to_string()));

        let tls_config = opts.tls_config.unwrap();
        assert_eq!(
            tls_config.client_cert_data,
            Some("Y2VydGRhdGE=".to_string())
        );
        assert_eq!(tls_config.client_key_data, Some("a2V5ZGF0YQ==".to_string()));
        assert_eq!(tls_config.insecure_skip_verify, None);
        assert_eq!(tls_config.ca_cert_file, None);
        assert_eq!(tls_config.ca_cert_data, None);
        assert_eq!(tls_config.client_cert_file, None);
        assert_eq!(tls_config.client_key_file, None);
    }
}
