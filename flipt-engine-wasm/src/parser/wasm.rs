use fliptevaluation::{error::Error, models::source, parser::Parser};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
extern "C" {
    pub fn fetch(s: &str) -> String;
}

pub struct WasmParser {}

impl WasmParser {
    pub fn new() -> Self {
        WasmParser {}
    }
}

impl Parser for WasmParser {
    fn parse(&self, namespace: &str) -> Result<source::Document, Error> {
        let response = fetch(&namespace);

        let document: source::Document = match serde_json::from_str(&response) {
            Ok(document) => document,
            Err(e) => return Err(Error::InvalidJSON(e)),
        };

        Ok(document)
    }
}
