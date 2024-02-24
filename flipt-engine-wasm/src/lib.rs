pub mod parser;

use crate::parser::wasm::WasmParser;
use fliptevaluation::store::Snapshot;
use fliptevaluation::{EvaluationRequest, Evaluator};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct Engine {
    evaluator: Evaluator<WasmParser, Snapshot>,
}

#[wasm_bindgen]
impl Engine {
    #[wasm_bindgen(constructor)]
    pub fn new(namespace: &str) -> Self {
        let evaluator =
            Evaluator::new_snapshot_evaluator(vec![namespace.to_string()], WasmParser::new())
                .unwrap();
        Engine { evaluator }
    }

    pub fn boolean_evaluation(&self, evaluation_request: JsValue) -> Result<JsValue, JsValue> {
        let request: EvaluationRequest = serde_wasm_bindgen::from_value(evaluation_request)
            .map_err(|e| JsValue::from_str(&format!("{}", e)))?;

        self.evaluator
            .boolean(&request)
            .map(|response| serde_wasm_bindgen::to_value(&response).unwrap())
            .map_err(|e| JsValue::from_str(&format!("{}", e)))
    }
}
