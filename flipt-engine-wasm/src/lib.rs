extern crate console_error_panic_hook;
use wasm_bindgen::prelude::*;

use fliptevaluation::{
    batch_evaluation, boolean_evaluation, error::Error, models::source, store::Snapshot,
    store::Store, variant_evaluation,
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
    pub fn new(namespace: &str) -> Self {
        console_error_panic_hook::set_once();
        let store = Snapshot::empty(namespace);

        Self {
            namespace: namespace.to_string(),
            store,
        }
    }

    pub fn snapshot(&mut self, data: JsValue) -> Result<(), JsValue> {
        let doc: source::Document = match serde_wasm_bindgen::from_value(data) {
            Ok(document) => document,
            Err(e) => {
                return Err(JsValue::from(e.to_string()));
            }
        };

        let store = match Snapshot::build(&self.namespace, doc) {
            Ok(s) => s,
            Err(e) => {
                return Err(JsValue::from(e.to_string()));
            }
        };

        self.store = store;
        Ok(())
    }

    pub fn evaluate_boolean(&self, request: JsValue) -> Result<JsValue, JsValue> {
        let result: Result<fliptevaluation::BooleanEvaluationResponse, Error> =
            match serde_wasm_bindgen::from_value(request) {
                Ok(req) => boolean_evaluation(&self.store, &self.namespace, &req),
                Err(e) => Err(Error::InvalidRequest(e.to_string())),
            };

        let response = JsResponse::from(result);

        Ok(serde_wasm_bindgen::to_value(&response)?)
    }

    pub fn evaluate_variant(&self, request: JsValue) -> Result<JsValue, JsValue> {
        let result: Result<fliptevaluation::VariantEvaluationResponse, Error> =
            match serde_wasm_bindgen::from_value(request) {
                Ok(req) => variant_evaluation(&self.store, &self.namespace, &req),
                Err(e) => Err(Error::InvalidRequest(e.to_string())),
            };

        let response = JsResponse::from(result);

        Ok(serde_wasm_bindgen::to_value(&response)?)
    }

    pub fn evaluate_batch(&self, request: JsValue) -> Result<JsValue, JsValue> {
        let result: Result<fliptevaluation::BatchEvaluationResponse, Error> =
            match serde_wasm_bindgen::from_value(request) {
                Ok(req) => batch_evaluation(&self.store, &self.namespace, req),
                Err(e) => Err(Error::InvalidRequest(e.to_string())),
            };

        let response = JsResponse::from(result);

        Ok(serde_wasm_bindgen::to_value(&response)?)
    }

    pub fn list_flags(&self) -> Result<JsValue, JsValue> {
        let flags = self.store.list_flags(&self.namespace);
        let response = JsResponse::from(Ok(flags));
        Ok(serde_wasm_bindgen::to_value(&response)?)
    }
}
