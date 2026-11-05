pub mod evaluator;
pub mod http;
pub mod tls;
use crate::tls::TlsConfig;
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
use reqwest::header::HeaderMap;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::collections::HashMap;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::{sync_channel, SyncSender, TrySendError};
use std::sync::{Arc, Mutex, OnceLock, RwLock};
use std::thread::{JoinHandle, ThreadId};
use std::time::Duration;
use thiserror::Error;
use tokio::runtime::Builder;
use tokio::runtime::Handle;
use tokio::runtime::Runtime;
use tokio::sync::{watch, Notify};

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

/// Called by the engine after it applies new flag state. `user_data` is the
/// pointer the host passed at registration, returned untouched.
/// Nullable in C: passing NULL clears the registered callback.
pub type StateUpdateCallback = Option<extern "C" fn(user_data: *mut c_void)>;

#[derive(Clone, Copy)]
struct RegisteredCallback {
    callback: extern "C" fn(user_data: *mut c_void),
    // Stored as usize so the slot is Send; the engine never dereferences it.
    user_data: usize,
}

/// Delivers state-update callbacks on a dedicated thread, so a slow or
/// blocking host callback never stalls the tokio runtime that fetches state.
struct StateUpdateDispatcher {
    // Held for the duration of each callback invocation, so replacing or
    // clearing the callback waits for any in-flight call to finish.
    slot: Arc<Mutex<Option<RegisteredCallback>>>,
    // Set by destroy_engine without taking the slot lock, so it also works
    // from inside a callback. Checked under the lock before every call.
    stopped: Arc<AtomicBool>,
    thread: Option<JoinHandle<()>>,
    thread_id: ThreadId,
}

impl StateUpdateDispatcher {
    /// Returns the dispatcher and the sender the fetch task signals on. The
    /// dispatcher thread exits once every sender is dropped.
    fn spawn() -> (Self, SyncSender<()>) {
        // Capacity 1 coalesces bursts: while a callback runs, any number of
        // updates collapse into a single follow-up call.
        let (tx, rx) = sync_channel::<()>(1);
        let slot: Arc<Mutex<Option<RegisteredCallback>>> = Arc::new(Mutex::new(None));
        let thread_slot = slot.clone();
        let stopped = Arc::new(AtomicBool::new(false));
        let thread_stopped = stopped.clone();

        let thread = std::thread::Builder::new()
            .name("flipt-engine-callback".into())
            .spawn(move || {
                while rx.recv().is_ok() {
                    let guard = match thread_slot.lock() {
                        Ok(g) => g,
                        Err(p) => p.into_inner(),
                    };
                    // A signal buffered before destroy_engine must not reach
                    // a host that has already torn down its user_data.
                    if thread_stopped.load(Ordering::SeqCst) {
                        break;
                    }
                    if let Some(registered) = *guard {
                        // No catch_unwind: a panic cannot cross the extern "C"
                        // boundary (it aborts), so hosts must not unwind.
                        (registered.callback)(registered.user_data as *mut c_void);
                    }
                }
            })
            .expect("failed to spawn callback thread");

        let thread_id = thread.thread().id();
        (
            Self {
                slot,
                stopped,
                thread: Some(thread),
                thread_id,
            },
            tx,
        )
    }

    /// Guarantees no further callback starts. Safe to call from inside one.
    fn stop(&self) {
        self.stopped.store(true, Ordering::SeqCst);
    }

    fn on_dispatcher_thread(&self) -> bool {
        std::thread::current().id() == self.thread_id
    }

    fn set(&self, registered: Option<RegisteredCallback>) -> Result<(), Error> {
        // The dispatcher holds the slot lock while a callback runs, so
        // changing it from inside a callback would deadlock.
        if self.on_dispatcher_thread() {
            return Err(Error::Internal(
                "cannot change state update callback from within the callback".into(),
            ));
        }
        let mut guard = match self.slot.lock() {
            Ok(g) => g,
            Err(p) => p.into_inner(),
        };
        *guard = registered;
        Ok(())
    }

    /// Joins the dispatcher thread. The caller must have stopped the fetch
    /// task first so the sender is dropped and the thread can exit.
    fn join(mut self) {
        if let Some(thread) = self.thread.take() {
            if self.on_dispatcher_thread() {
                // destroy_engine called from inside a callback: the thread
                // exits on its own once this call returns.
                return;
            }
            let _ = thread.join();
        }
    }
}

pub struct Engine {
    evaluator: Arc<RwLock<Evaluator<snapshot::Snapshot>>>,
    stop_signal: Arc<AtomicBool>,
    fetcher_handle: Option<tokio::task::JoinHandle<()>>,
    stop_notify: Arc<Notify>,
    auth_sender: Option<watch::Sender<HeaderMap>>,
    dispatcher: StateUpdateDispatcher,
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
        let auth_sender = fetcher.take_auth_sender();
        let stop_signal = Arc::new(AtomicBool::new(false));
        let stop_signal_clone = stop_signal.clone();

        let stop_notify = Arc::new(Notify::new());
        let stop_notify_clone = stop_notify.clone();

        let evaluator = Arc::new(RwLock::new(evaluator));
        let evaluator_clone = evaluator.clone();

        let handle = get_or_create_runtime();

        let (dispatcher, update_tx) = StateUpdateDispatcher::spawn();

        // Set the initial snapshot
        if let Ok(mut lock) = evaluator.write() {
            lock.replace_snapshot(Ok(initial_snapshot));
        }

        // Block on initial fetch
        handle.block_on(async {
            match fetcher.initial_fetch().await {
                Ok(doc) => {
                    log::debug!("initial fetch succeeded");
                    let snap = snapshot::Snapshot::build(doc);
                    if let Ok(mut lock) = evaluator.write() {
                        lock.replace_snapshot(Ok(snap));
                    }
                }
                Err(err) => {
                    log::warn!("initial fetch failed: {err:?}");
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
                        log::debug!("fetch succeeded");
                        let snap = snapshot::Snapshot::build(doc);
                        // The write lock is released before signalling, so a
                        // callback can evaluate flags without deadlocking.
                        let changed = match evaluator_clone.write() {
                            Ok(mut lock) => lock.replace_snapshot(Ok(snap)),
                            Err(_) => false,
                        };
                        if changed {
                            match update_tx.try_send(()) {
                                Ok(()) | Err(TrySendError::Full(())) => {}
                                Err(TrySendError::Disconnected(())) => {
                                    log::warn!("state update dispatcher is gone");
                                }
                            }
                        }
                    }
                    Err(err) => {
                        log::warn!("fetch failed: {err:?}");
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
            auth_sender,
            dispatcher,
        }
    }

    /// Registers, replaces, or (with `None`) clears the state update callback.
    /// Once this returns, no call to a previously registered callback is in
    /// flight.
    fn set_state_update_callback(
        &self,
        callback: StateUpdateCallback,
        user_data: *mut c_void,
    ) -> Result<(), Error> {
        self.dispatcher
            .set(callback.map(|callback| RegisteredCallback {
                callback,
                user_data: user_data as usize,
            }))
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
        log::trace!(
            "initialize_engine_ffi called: opts ptr=0x{:x}",
            opts as usize
        );
        let ptr = _initialize_engine(opts);
        log::trace!(
            "initialize_engine_ffi returning engine ptr=0x{:x}",
            ptr as usize
        );
        ptr
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in initialize_engine_ffi: {e:?}");
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
        log::trace!("initialize_engine called: opts ptr=0x{:x}", opts as usize);
        let ptr = _initialize_engine(opts);
        log::trace!(
            "initialize_engine returning engine ptr=0x{:x}",
            ptr as usize
        );
        ptr
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in initialize_engine: {e:?}");
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
        log::trace!(
            "get_snapshot_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _get_snapshot(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in get_snapshot_ffi: {e:?}");
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
        log::trace!(
            "get_snapshot called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _get_snapshot(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in get_snapshot: {e:?}");
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
        log::trace!(
            "evaluate_variant_ffi called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_variant(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in evaluate_variant_ffi: {e:?}");
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
        log::trace!(
            "evaluate_variant called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_variant(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in evaluate_variant: {e:?}");
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
        log::trace!(
            "evaluate_boolean_ffi called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_boolean(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in evaluate_boolean_ffi: {e:?}");
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
        log::trace!(
            "evaluate_boolean called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            evaluation_request as usize
        );
        _evaluate_boolean(engine_ptr, evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in evaluate_boolean: {e:?}");
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
        log::trace!(
            "evaluate_batch_ffi called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            batch_evaluation_request as usize
        );
        _evaluate_batch(engine_ptr, batch_evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in evaluate_batch_ffi: {e:?}");
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
        log::trace!(
            "evaluate_batch called: engine ptr=0x{:x}, req ptr=0x{:x}",
            engine_ptr as usize,
            batch_evaluation_request as usize
        );
        _evaluate_batch(engine_ptr, batch_evaluation_request)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in evaluate_batch: {e:?}");
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
        log::trace!(
            "list_flags_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _list_flags(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in list_flags_ffi: {e:?}");
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
        log::trace!("list_flags called: engine ptr=0x{:x}", engine_ptr as usize);
        _list_flags(engine_ptr)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in list_flags: {e:?}");
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
        log::trace!(
            "destroy_engine_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _destroy_engine(engine_ptr)
        // No return value
    }) {
        Ok(_) => (),
        Err(e) => {
            log::error!("PANIC in destroy_engine_ffi: {e:?}");
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
        log::trace!(
            "destroy_engine called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _destroy_engine(engine_ptr)
        // No return value
    }) {
        Ok(_) => (),
        Err(e) => {
            log::error!("PANIC in destroy_engine: {e:?}");
        }
    }
}

/// # Safety
///
/// This function will update the authentication for the engine.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn update_authentication_ffi(
    engine_ptr: *mut c_void,
    auth_json: *const c_char,
) -> *const c_char {
    match std::panic::catch_unwind(|| {
        log::trace!(
            "update_authentication_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _update_authentication(engine_ptr, auth_json)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in update_authentication_ffi: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in update_authentication_ffi".to_string(),
            )))
        }
    }
}

/// # Safety
///
/// This function will update the authentication for the engine.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn update_authentication(
    engine_ptr: *mut c_void,
    auth_json: *const c_char,
) -> *const c_char {
    match std::panic::catch_unwind(|| {
        log::trace!(
            "update_authentication called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _update_authentication(engine_ptr, auth_json)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in update_authentication: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in update_authentication".to_string(),
            )))
        }
    }
}

/// # Safety
///
/// Registers `callback` to be called, with `user_data`, after the engine applies
/// changed flag state. Pass a null callback to clear it. Callbacks run on a
/// single engine-owned thread; bursts of updates are coalesced into one call.
/// Initial state and transitions into an error state do not fire; recovering
/// from an error does.
///
/// A callback may evaluate flags and may call `destroy_engine`, but changing
/// the callback from inside one returns an error. `user_data` must stay valid
/// until the callback is cleared or `destroy_engine` returns. The callback
/// must not unwind: a panic across this boundary aborts the process.
#[no_mangle]
#[cfg(all(target_feature = "crt-static", target_os = "linux"))]
pub unsafe extern "C" fn set_state_update_callback_ffi(
    engine_ptr: *mut c_void,
    callback: StateUpdateCallback,
    user_data: *mut c_void,
) -> *const c_char {
    match std::panic::catch_unwind(|| {
        log::trace!(
            "set_state_update_callback_ffi called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _set_state_update_callback(engine_ptr, callback, user_data)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in set_state_update_callback_ffi: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in set_state_update_callback_ffi".to_string(),
            )))
        }
    }
}

/// # Safety
///
/// Registers `callback` to be called, with `user_data`, after the engine applies
/// changed flag state. Pass a null callback to clear it. Callbacks run on a
/// single engine-owned thread; bursts of updates are coalesced into one call.
/// Initial state and transitions into an error state do not fire; recovering
/// from an error does.
///
/// A callback may evaluate flags and may call `destroy_engine`, but changing
/// the callback from inside one returns an error. `user_data` must stay valid
/// until the callback is cleared or `destroy_engine` returns. The callback
/// must not unwind: a panic across this boundary aborts the process.
#[no_mangle]
#[cfg(not(all(target_feature = "crt-static", target_os = "linux")))]
pub unsafe extern "C" fn set_state_update_callback(
    engine_ptr: *mut c_void,
    callback: StateUpdateCallback,
    user_data: *mut c_void,
) -> *const c_char {
    match std::panic::catch_unwind(|| {
        log::trace!(
            "set_state_update_callback called: engine ptr=0x{:x}",
            engine_ptr as usize
        );
        _set_state_update_callback(engine_ptr, callback, user_data)
    }) {
        Ok(ptr) => ptr,
        Err(e) => {
            log::error!("PANIC in set_state_update_callback: {e:?}");
            result_to_json_ptr::<(), _>(Err(Error::Internal(
                "panic in set_state_update_callback".to_string(),
            )))
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
        log::trace!("destroy_string_ffi called: ptr=0x{:x}", ptr as usize);
        _destroy_string(ptr)
    }) {
        Ok(_) => (),
        Err(e) => {
            log::error!("PANIC in destroy_string_ffi: {e:?}");
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
        log::trace!("destroy_string called: ptr=0x{:x}", ptr as usize);
        _destroy_string(ptr)
    }) {
        Ok(_) => (),
        Err(e) => {
            log::error!("PANIC in destroy_string: {e:?}");
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
            log::warn!("failed to build custom fetcher: {e}");
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

unsafe extern "C" fn _update_authentication(
    engine_ptr: *mut c_void,
    auth_json: *const c_char,
) -> *const c_char {
    if engine_ptr.is_null() || auth_json.is_null() {
        return result_to_json_ptr::<(), _>(Err(FFIError::NullPointer));
    }

    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    let auth_bytes = CStr::from_ptr(auth_json).to_bytes();
    let auth_str = match std::str::from_utf8(auth_bytes) {
        Ok(s) => s,
        Err(e) => {
            return result_to_json_ptr::<(), Error>(Err(Error::Internal(format!(
                "invalid UTF-8: {e}"
            ))));
        }
    };

    let auth: Authentication = match serde_json::from_str(auth_str) {
        Ok(a) => a,
        Err(e) => {
            return result_to_json_ptr::<(), Error>(Err(Error::Internal(format!(
                "invalid auth JSON: {e}"
            ))));
        }
    };

    let header_map = HeaderMap::from(auth);

    match &e.auth_sender {
        Some(sender) => match sender.send(header_map) {
            Ok(_) => result_to_json_ptr(Ok::<(), Error>(())),
            Err(_) => result_to_json_ptr::<(), Error>(Err(Error::Internal(
                "auth channel closed".to_string(),
            ))),
        },
        None => result_to_json_ptr::<(), Error>(Err(Error::Internal(
            "no auth sender available".to_string(),
        ))),
    }
}

unsafe extern "C" fn _set_state_update_callback(
    engine_ptr: *mut c_void,
    callback: StateUpdateCallback,
    user_data: *mut c_void,
) -> *const c_char {
    let e = match get_engine(engine_ptr) {
        Ok(e) => e,
        Err(e) => return result_to_json_ptr::<(), _>(Err(e)),
    };

    result_to_json_ptr(e.set_state_update_callback(callback, user_data))
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

    // The fetcher task owned the only sender, so the dispatcher thread exits
    // after any in-flight callback. Joining guarantees no callback runs once
    // destroy_engine returns.
    engine.dispatcher.stop();
    let _ = engine.dispatcher.set(None);
    engine.dispatcher.join();

    // Engine is dropped here
}

unsafe extern "C" fn _destroy_string(ptr: *mut c_char) {
    let _ = CString::from_raw(ptr);
}

// Helper functions

/// This function should not be called unless an Engine is initiated. It provides a helper
/// utility to retrieve an Engine instance for evaluation use.
unsafe fn get_engine<'a>(engine_ptr: *mut c_void) -> Result<&'a Engine, FFIError> {
    if engine_ptr.is_null() {
        Err(FFIError::NullPointer)
    } else {
        Ok(&*(engine_ptr as *const Engine))
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
    fn test_update_authentication_null_engine_ptr() {
        unsafe {
            let auth_json = CString::new(r#"{"client_token":"secret"}"#).unwrap();
            let result_ptr = _update_authentication(std::ptr::null_mut(), auth_json.as_ptr());
            assert!(!result_ptr.is_null());

            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            assert!(
                result_str.contains("failure"),
                "Expected failure response, got: {}",
                result_str
            );

            _destroy_string(result_ptr as *mut c_char);
        }
    }

    #[test]
    fn test_update_authentication_null_auth_json() {
        unsafe {
            let opts = CString::new(
                r#"{"url":"http://localhost:1","error_strategy":"fallback","update_interval":9999}"#,
            )
            .unwrap();
            let engine_ptr = _initialize_engine(opts.as_ptr());
            assert!(!engine_ptr.is_null());

            let result_ptr = _update_authentication(engine_ptr, std::ptr::null());
            assert!(!result_ptr.is_null());

            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            assert!(
                result_str.contains("failure"),
                "Expected failure response, got: {}",
                result_str
            );

            _destroy_string(result_ptr as *mut c_char);
            _destroy_engine(engine_ptr);
        }
    }

    #[test]
    fn test_update_authentication_invalid_json() {
        unsafe {
            let opts = CString::new(
                r#"{"url":"http://localhost:1","error_strategy":"fallback","update_interval":9999}"#,
            )
            .unwrap();
            let engine_ptr = _initialize_engine(opts.as_ptr());
            assert!(!engine_ptr.is_null());

            let invalid_json = CString::new("not valid json").unwrap();
            let result_ptr = _update_authentication(engine_ptr, invalid_json.as_ptr());
            assert!(!result_ptr.is_null());

            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            assert!(
                result_str.contains("failure"),
                "Expected failure response, got: {}",
                result_str
            );
            assert!(
                result_str.contains("invalid auth JSON"),
                "Expected 'invalid auth JSON' error, got: {}",
                result_str
            );

            _destroy_string(result_ptr as *mut c_char);
            _destroy_engine(engine_ptr);
        }
    }

    #[test]
    fn test_update_authentication_valid_client_token() {
        unsafe {
            let opts = CString::new(
                r#"{"url":"http://localhost:1","error_strategy":"fallback","update_interval":9999}"#,
            )
            .unwrap();
            let engine_ptr = _initialize_engine(opts.as_ptr());
            assert!(!engine_ptr.is_null());

            let auth_json = CString::new(r#"{"client_token":"new-secret"}"#).unwrap();
            let result_ptr = _update_authentication(engine_ptr, auth_json.as_ptr());
            assert!(!result_ptr.is_null());

            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            assert!(
                result_str.contains("success"),
                "Expected success response, got: {}",
                result_str
            );

            _destroy_string(result_ptr as *mut c_char);
            _destroy_engine(engine_ptr);
        }
    }

    #[test]
    fn test_update_authentication_valid_jwt_token() {
        unsafe {
            let opts = CString::new(
                r#"{"url":"http://localhost:1","error_strategy":"fallback","update_interval":9999}"#,
            )
            .unwrap();
            let engine_ptr = _initialize_engine(opts.as_ptr());
            assert!(!engine_ptr.is_null());

            let auth_json = CString::new(r#"{"jwt_token":"new-jwt"}"#).unwrap();
            let result_ptr = _update_authentication(engine_ptr, auth_json.as_ptr());
            assert!(!result_ptr.is_null());

            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            assert!(
                result_str.contains("success"),
                "Expected success response, got: {}",
                result_str
            );

            _destroy_string(result_ptr as *mut c_char);
            _destroy_engine(engine_ptr);
        }
    }

    use std::sync::atomic::AtomicUsize;
    use std::time::Instant;

    extern "C" fn count_calls(user_data: *mut c_void) {
        let counter = unsafe { &*(user_data as *const AtomicUsize) };
        counter.fetch_add(1, Ordering::SeqCst);
    }

    fn wait_for(cond: impl Fn() -> bool, timeout: Duration) -> bool {
        let deadline = Instant::now() + timeout;
        while Instant::now() < deadline {
            if cond() {
                return true;
            }
            std::thread::sleep(Duration::from_millis(20));
        }
        cond()
    }

    #[test]
    fn test_dispatcher_invokes_registered_callback() {
        let counter = AtomicUsize::new(0);
        let (dispatcher, tx) = StateUpdateDispatcher::spawn();
        dispatcher
            .set(Some(RegisteredCallback {
                callback: count_calls,
                user_data: &counter as *const AtomicUsize as usize,
            }))
            .unwrap();

        tx.send(()).unwrap();
        assert!(wait_for(
            || counter.load(Ordering::SeqCst) == 1,
            Duration::from_secs(2)
        ));

        drop(tx);
        dispatcher.join();
        assert_eq!(counter.load(Ordering::SeqCst), 1);
    }

    #[test]
    fn test_dispatcher_cleared_callback_is_not_invoked() {
        let counter = AtomicUsize::new(0);
        let (dispatcher, tx) = StateUpdateDispatcher::spawn();
        dispatcher
            .set(Some(RegisteredCallback {
                callback: count_calls,
                user_data: &counter as *const AtomicUsize as usize,
            }))
            .unwrap();
        dispatcher.set(None).unwrap();

        tx.send(()).unwrap();
        drop(tx);
        dispatcher.join();
        assert_eq!(counter.load(Ordering::SeqCst), 0);
    }

    #[test]
    fn test_dispatcher_coalesces_bursts() {
        let counter = AtomicUsize::new(0);
        let (dispatcher, tx) = StateUpdateDispatcher::spawn();
        dispatcher
            .set(Some(RegisteredCallback {
                callback: count_calls,
                user_data: &counter as *const AtomicUsize as usize,
            }))
            .unwrap();

        // Hold the slot so the dispatcher can't drain while we signal.
        let guard = dispatcher.slot.lock().unwrap();
        for _ in 0..10 {
            let _ = tx.try_send(());
        }
        drop(guard);

        drop(tx);
        dispatcher.join();
        let calls = counter.load(Ordering::SeqCst);
        assert!((1..=2).contains(&calls), "expected 1-2 calls, got {calls}");
    }

    struct ReentrantDestroy {
        calls: AtomicUsize,
        dispatcher: Mutex<Option<StateUpdateDispatcher>>,
        tx: Mutex<Option<SyncSender<()>>>,
    }

    extern "C" fn destroy_from_callback(user_data: *mut c_void) {
        let state = unsafe { &*(user_data as *const ReentrantDestroy) };
        state.calls.fetch_add(1, Ordering::SeqCst);
        // An update lands while the callback runs, then the host tears down
        // from inside the callback, mirroring _destroy_engine's sequence.
        let tx = state.tx.lock().unwrap().take().unwrap();
        tx.try_send(()).unwrap();
        drop(tx);
        let dispatcher = state.dispatcher.lock().unwrap().take().unwrap();
        dispatcher.stop();
        assert!(dispatcher.set(None).is_err());
        dispatcher.join();
    }

    #[test]
    fn test_dispatcher_no_call_after_destroy_from_callback() {
        let (dispatcher, tx) = StateUpdateDispatcher::spawn();
        let slot = dispatcher.slot.clone();
        let thread_done = dispatcher.stopped.clone();
        let state = Box::leak(Box::new(ReentrantDestroy {
            calls: AtomicUsize::new(0),
            dispatcher: Mutex::new(None),
            tx: Mutex::new(None),
        }));
        dispatcher
            .set(Some(RegisteredCallback {
                callback: destroy_from_callback,
                user_data: state as *const ReentrantDestroy as usize,
            }))
            .unwrap();
        *state.dispatcher.lock().unwrap() = Some(dispatcher);
        let first = tx.clone();
        *state.tx.lock().unwrap() = Some(tx);

        first.send(()).unwrap();
        drop(first);
        assert!(wait_for(
            || thread_done.load(Ordering::SeqCst),
            Duration::from_secs(2)
        ));
        // Give the dispatcher time to drain the buffered signal, if it would.
        std::thread::sleep(Duration::from_millis(200));
        drop(slot);
        assert_eq!(state.calls.load(Ordering::SeqCst), 1);
    }

    #[test]
    fn test_set_state_update_callback_null_engine_ptr() {
        unsafe {
            let result_ptr =
                _set_state_update_callback(std::ptr::null_mut(), None, std::ptr::null_mut());
            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            assert!(result_str.contains("failure"), "got: {result_str}");
            _destroy_string(result_ptr as *mut c_char);
        }
    }

    #[test]
    fn test_state_update_callback_fires_on_polled_change() {
        let path = "/internal/v1/evaluation/snapshot/namespace/default";
        let mut server = mockito::Server::new();
        let initial = server
            .mock("GET", path)
            .with_status(200)
            .with_body(r#"{"namespace":{"key":"default"},"flags":[]}"#)
            .create();

        let opts = CString::new(format!(
            r#"{{"url":"{}","update_interval":1}}"#,
            server.url()
        ))
        .unwrap();
        let counter = AtomicUsize::new(0);

        unsafe {
            let engine_ptr = _initialize_engine(opts.as_ptr());
            assert!(!engine_ptr.is_null());

            let result_ptr = _set_state_update_callback(
                engine_ptr,
                Some(count_calls),
                &counter as *const AtomicUsize as *mut c_void,
            );
            let result_str = CStr::from_ptr(result_ptr).to_str().unwrap();
            assert!(result_str.contains("success"), "got: {result_str}");
            _destroy_string(result_ptr as *mut c_char);

            // The first background poll returns the same (empty) state: no
            // callback. Then the server starts serving a new flag.
            std::thread::sleep(Duration::from_millis(1500));
            assert_eq!(counter.load(Ordering::SeqCst), 0);

            initial.remove();
            server
                .mock("GET", path)
                .with_status(200)
                .with_body(
                    r#"{"namespace":{"key":"default"},"flags":[{"key":"new_flag","name":"New","type":"BOOLEAN_FLAG_TYPE","enabled":true}]}"#,
                )
                .create();

            assert!(wait_for(
                || counter.load(Ordering::SeqCst) >= 1,
                Duration::from_secs(5)
            ));

            // Unchanged polls after the update must not fire again.
            std::thread::sleep(Duration::from_millis(2500));
            assert_eq!(counter.load(Ordering::SeqCst), 1);

            _destroy_engine(engine_ptr);
        }
    }

    #[test]
    fn test_state_update_callback_fires_on_streamed_change() {
        use std::io::Write;

        const EMPTY: &str = r#"{"namespace":{"key":"default"},"flags":[]}"#;
        const WITH_FLAG: &str = r#"{"namespace":{"key":"default"},"flags":[{"key":"new_flag","name":"New","type":"BOOLEAN_FLAG_TYPE","enabled":true}]}"#;

        let mut server = mockito::Server::new();
        server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .with_status(200)
            .with_body(EMPTY)
            .create();

        // The stream body advances one gated step at a time, so the test decides
        // when each chunk is sent.
        let step = Arc::new(AtomicUsize::new(0));
        let body_step = step.clone();
        server
            .mock(
                "GET",
                "/client/v2/environments/default/namespaces/default/stream",
            )
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_chunked_body(move |w| {
                let wait_step = |n: usize| {
                    wait_for(
                        || body_step.load(Ordering::SeqCst) >= n,
                        Duration::from_secs(30),
                    );
                };
                let chunk = |doc: &str| format!("{{\"result\":{doc}}}\n");

                wait_step(1);
                // Same state as the initial fetch: must not fire.
                w.write_all(chunk(EMPTY).as_bytes())?;
                w.flush()?;
                wait_step(2);
                // A real change, then a resend of it: must fire exactly once.
                w.write_all(chunk(WITH_FLAG).as_bytes())?;
                w.write_all(chunk(WITH_FLAG).as_bytes())?;
                w.flush()?;
                // Hold the stream open until the test is done.
                wait_step(3);
                Ok(())
            })
            .create();

        let opts = CString::new(format!(
            r#"{{"url":"{}","fetch_mode":"streaming"}}"#,
            server.url()
        ))
        .unwrap();
        let counter = AtomicUsize::new(0);

        unsafe {
            let engine_ptr = _initialize_engine(opts.as_ptr());
            assert!(!engine_ptr.is_null());

            let result_ptr = _set_state_update_callback(
                engine_ptr,
                Some(count_calls),
                &counter as *const AtomicUsize as *mut c_void,
            );
            _destroy_string(result_ptr as *mut c_char);

            step.store(1, Ordering::SeqCst);
            std::thread::sleep(Duration::from_millis(500));
            assert_eq!(counter.load(Ordering::SeqCst), 0);

            step.store(2, Ordering::SeqCst);
            assert!(wait_for(
                || counter.load(Ordering::SeqCst) >= 1,
                Duration::from_secs(5)
            ));

            let engine = get_engine(engine_ptr).unwrap();
            assert_eq!(engine.list_flags().unwrap().len(), 1);

            std::thread::sleep(Duration::from_millis(500));
            assert_eq!(counter.load(Ordering::SeqCst), 1);

            // Closing with the stream still open must not hang, and no
            // callback may run afterwards.
            _destroy_engine(engine_ptr);
            step.store(3, Ordering::SeqCst);
        }
        assert_eq!(counter.load(Ordering::SeqCst), 1);
    }

    #[test]
    fn test_concurrent_evaluation_shared_engine() {
        const THREADS: usize = 8;
        const ITERATIONS: usize = 500;

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
        "eval_rules": { "flag1": [], "flag_boolean": [] },
        "eval_rollouts": { "flag1": [], "flag_boolean": [] },
        "eval_distributions": {}
    }
}
"#;
        let encoded_snapshot = BASE64_STANDARD.encode(snapshot);
        let opts = CString::new(format!(
            r#"{{"url":"http://localhost:1","error_strategy":"fallback","update_interval":9999,"snapshot":"{encoded_snapshot}"}}"#
        ))
        .unwrap();

        unsafe {
            let engine_ptr = _initialize_engine(opts.as_ptr());
            assert!(!engine_ptr.is_null());

            let engine_addr = engine_ptr as usize;

            let handles: Vec<_> = (0..THREADS)
                .map(|t| {
                    std::thread::spawn(move || {
                        let ptr = engine_addr as *mut c_void;
                        let variant_req =
                            CString::new(r#"{"flag_key":"flag1","entity_id":"entity","context":{}}"#)
                                .unwrap();
                        let boolean_req = CString::new(
                            r#"{"flag_key":"flag_boolean","entity_id":"entity","context":{}}"#,
                        )
                        .unwrap();
                        let batch_req = CString::new(
                            r#"[{"flag_key":"flag1","entity_id":"entity","context":{}},{"flag_key":"flag_boolean","entity_id":"entity","context":{}}]"#,
                        )
                        .unwrap();

                        for i in 0..ITERATIONS {
                            let result_ptr = match (t + i) % 5 {
                                0 => _evaluate_variant(ptr, variant_req.as_ptr()),
                                1 => _evaluate_boolean(ptr, boolean_req.as_ptr()),
                                2 => _evaluate_batch(ptr, batch_req.as_ptr()),
                                3 => _list_flags(ptr),
                                _ => _get_snapshot(ptr),
                            };
                            assert!(!result_ptr.is_null());
                            _destroy_string(result_ptr as *mut c_char);
                        }
                    })
                })
                .collect();

            for handle in handles {
                handle.join().expect("evaluation thread panicked");
            }

            _destroy_engine(engine_ptr);
        }
    }
}
