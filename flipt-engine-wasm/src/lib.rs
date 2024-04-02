extern crate console_error_panic_hook;
use wasm_bindgen::prelude::*;

use fliptevaluation::{
    boolean_evaluation, error::Error, models::source, store::Snapshot, variant_evaluation,
};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
pub struct JsResponse<T>
where
    T: Serialize,
{
    status: Status,
    result: Option<T>,
    error_message: Option<String>,
}

#[derive(Serialize, Deserialize)]
enum Status {
    #[serde(rename = "success")]
    Success,
    #[serde(rename = "failure")]
    Failure,
}

#[wasm_bindgen]
pub struct Engine {
    namespace: String,
    store: Snapshot,
}

impl<T> From<Result<T, Error>> for JsResponse<T>
where
    T: Serialize,
{
    fn from(value: Result<T, Error>) -> Self {
        match value {
            Ok(result) => JsResponse {
                status: Status::Success,
                result: Some(result),
                error_message: None,
            },
            Err(e) => JsResponse {
                status: Status::Failure,
                result: None,
                error_message: Some(e.to_string()),
            },
        }
    }
}

#[wasm_bindgen]
impl Engine {
    #[wasm_bindgen(constructor)]
    pub fn new(namespace: &str, data: JsValue) -> Self {
        console_error_panic_hook::set_once();
        let doc: source::Document = match serde_wasm_bindgen::from_value(data) {
            Ok(document) => document,
            Err(e) => {
                panic!("Invalid JSON: {}", e);
            }
        };

        let store = match Snapshot::build(namespace, doc) {
            Ok(s) => s,
            Err(e) => {
                panic!("Error building snapshot: {}", e);
            }
        };

        Self {
            namespace: namespace.to_string(),
            store,
        }
    }

    pub fn snapshot(&mut self, data: JsValue) {
        let doc: source::Document = match serde_wasm_bindgen::from_value(data) {
            Ok(document) => document,
            Err(e) => {
                panic!("Invalid JSON: {}", e);
            }
        };

        let store = match Snapshot::build(&self.namespace, doc) {
            Ok(s) => s,
            Err(e) => {
                panic!("Error building snapshot: {}", e);
            }
        };

        self.store = store;
    }

    pub fn evaluate_boolean(&self, request: JsValue) -> Result<JsValue, JsValue> {
        let req: fliptevaluation::EvaluationRequest = match serde_wasm_bindgen::from_value(request)
        {
            Ok(r) => r,
            Err(e) => {
                panic!("Invalid JSON: {}", e);
            }
        };

        let result = boolean_evaluation(&self.store, &self.namespace, &req);

        let response = JsResponse::from(result);

        Ok(serde_wasm_bindgen::to_value(&response)?)
    }

    pub fn evaluate_variant(&self, request: JsValue) -> Result<JsValue, JsValue> {
        let req: fliptevaluation::EvaluationRequest = match serde_wasm_bindgen::from_value(request)
        {
            Ok(r) => r,
            Err(e) => {
                panic!("Invalid JSON: {}", e);
            }
        };

        let result = variant_evaluation(&self.store, &self.namespace, &req);

        let response = JsResponse::from(result);

        Ok(serde_wasm_bindgen::to_value(&response)?)
    }
}
