use flipt_evaluation::error::Error;
use flipt_evaluation::parser::HTTPParser;
use flipt_evaluation::store::Snapshot;
use flipt_evaluation::{
    BooleanEvaluationResponse, EvaluationRequest, Evaluator, VariantEvaluationResponse,
};
use libc::c_void;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::collections::HashMap;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::{Arc, Mutex};

#[derive(Deserialize)]
struct EvalRequest {
    namespace_key: String,
    flag_key: String,
    entity_id: String,
    context: Option<Map<String, Value>>,
}

#[derive(Serialize)]
struct EvalResponse<T>
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

impl<T> From<Result<T, Error>> for EvalResponse<T>
where
    T: Serialize,
{
    fn from(value: Result<T, Error>) -> Self {
        match value {
            Ok(result) => EvalResponse {
                status: Status::Success,
                result: Some(result),
                error_message: None,
            },
            Err(e) => EvalResponse {
                status: Status::Failure,
                result: None,
                error_message: Some(e.to_string()),
            },
        }
    }
}

fn result_to_json_ptr<T: Serialize>(result: Result<T, Error>) -> *mut c_char {
    let ffi_response: EvalResponse<T> = result.into();
    let json_string = serde_json::to_string(&ffi_response).unwrap();
    CString::new(json_string).unwrap().into_raw()
}

#[derive(Deserialize)]
pub struct EngineOpts {
    url: Option<String>,
    auth_token: Option<String>,
    update_interval: Option<u64>,
}

impl Default for EngineOpts {
    fn default() -> Self {
        Self {
            url: Some("http://localhost:8080".into()),
            auth_token: None,
            update_interval: Some(120),
        }
    }
}

pub struct Engine {
    pub opts: EngineOpts,
    pub evaluator: Arc<Mutex<Evaluator<HTTPParser, Snapshot>>>,
}

impl Engine {
    pub fn new(evaluator: Evaluator<HTTPParser, Snapshot>, opts: EngineOpts) -> Self {
        let mut engine = Self {
            opts,
            evaluator: Arc::new(Mutex::new(evaluator)),
        };

        engine.update();

        engine
    }

    fn update(&mut self) {
        let evaluator = self.evaluator.clone();
        let update_interval = self.opts.update_interval.unwrap_or(120);

        std::thread::spawn(move || loop {
            std::thread::sleep(std::time::Duration::from_secs(update_interval));
            let mut lock = evaluator.lock().unwrap();
            lock.replace_snapshot();
        });
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
}

/// # Safety
///
/// This function should not be called unless an Engine is initiated. It provides a helper
/// utility to retrieve an Engine instance for evaluation use.
unsafe fn get_engine<'a>(engine_ptr: *mut c_void) -> Result<&'a mut Engine, Error> {
    if engine_ptr.is_null() {
        Err(Error::NullPointer)
    } else {
        Ok(unsafe { &mut *(engine_ptr as *mut Engine) })
    }
}

/// # Safety
///
/// This function will initialize an Engine and return a pointer back to the caller.
#[no_mangle]
pub unsafe extern "C" fn initialize_engine(
    namespaces: *const *const c_char,
    opts: *const c_char,
) -> *mut c_void {
    let mut index = 0;
    let mut namespaces_vec = Vec::new();

    while !(*namespaces.offset(index)).is_null() {
        let c_str = CStr::from_ptr(*namespaces.offset(index));
        if let Ok(rust_str) = c_str.to_str() {
            namespaces_vec.push(rust_str.to_string());
        }

        index += 1;
    }

    // TODO(yquansah): There seems to be some issue across the FFI layer where the char** is picked up at this
    // layer to have another "empty" string in its input. For now we will filter out the empty string, but
    // should circle back to investigate what may be happening upstream.
    let namespaces_vec: Vec<String> = namespaces_vec
        .into_iter()
        .filter(|namespace| !namespace.is_empty())
        .collect();

    let engine_opts_bytes = CStr::from_ptr(opts).to_bytes();
    let bytes_str_repr = std::str::from_utf8(engine_opts_bytes).unwrap();
    let engine_opts: EngineOpts = serde_json::from_str(bytes_str_repr).unwrap_or_default();

    let http_url = engine_opts
        .url
        .to_owned()
        .unwrap_or("http://localhost:8080".into());

    let auth_token = engine_opts.auth_token.to_owned();

    let parser = HTTPParser::new(&http_url, auth_token.clone().as_deref());
    let evaluator = Evaluator::new_snapshot_evaluator(namespaces_vec, parser).unwrap();

    Box::into_raw(Box::new(Engine::new(evaluator, engine_opts))) as *mut c_void
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

unsafe fn get_evaluation_request(evaluation_request: *const c_char) -> EvaluationRequest {
    let evaluation_request_bytes = CStr::from_ptr(evaluation_request).to_bytes();
    let bytes_str_repr = std::str::from_utf8(evaluation_request_bytes).unwrap();
    let client_eval_request: EvalRequest = serde_json::from_str(bytes_str_repr).unwrap();

    let mut context_map: HashMap<String, String> = HashMap::new();
    if let Some(context_value) = client_eval_request.context {
        for (key, value) in context_value {
            if let serde_json::Value::String(val) = value {
                context_map.insert(key, val);
            }
        }
    }

    EvaluationRequest {
        namespace_key: client_eval_request.namespace_key,
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

    drop(Box::from_raw(engine_ptr as *mut Engine));
}
