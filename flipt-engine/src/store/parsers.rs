use snafu::{prelude::*, Whatever};
use std::env;

use crate::models::source;

pub trait Parser {
    fn parse(&self, namespace: String) -> Result<source::Document, Whatever>;
}

pub struct HTTPParser {
    http_client: reqwest::blocking::Client,
    http_url: String,
}

impl HTTPParser {
    pub fn new() -> Self {
        // We will allow the following line to panic when an error is encountered.
        let http_client = reqwest::blocking::Client::builder()
            .timeout(std::time::Duration::from_secs(10))
            .build()
            .unwrap();

        let http_url =
            env::var("FLIPT_REMOTE_URL").unwrap_or(String::from("http://localhost:8080"));

        Self {
            http_client,
            http_url,
        }
    }
}

impl Parser for HTTPParser {
    fn parse(&self, namespace: String) -> Result<source::Document, Whatever> {
        let response = match self
            .http_client
            .get(format!(
                "{}/internal/v1/evaluation/snapshot/namespace/{}",
                self.http_url, namespace
            ))
            .send()
        {
            Ok(resp) => resp,
            Err(e) => whatever!("failed to make request: err {}", e),
        };

        let response_text = match response.text() {
            Ok(t) => t,
            Err(e) => whatever!("failed to get response body: err {}", e),
        };

        let document: source::Document = match serde_json::from_str(&response_text) {
            Ok(doc) => doc,
            Err(e) => whatever!("failed to deserialize text into document: err {}", e),
        };

        Ok(document)
    }
}

#[cfg(test)]
use std::fs;
#[cfg(test)]
use std::path::PathBuf;

#[cfg(test)]
pub struct TestParser {
    path: Option<String>,
}

#[cfg(test)]
impl TestParser {
    pub fn new() -> Self {
        Self { path: None }
    }
}

#[cfg(test)]
impl Parser for TestParser {
    fn parse(&self, _: String) -> Result<source::Document, Whatever> {
        let f = match &self.path {
            Some(path) => path.to_owned(),
            None => {
                let mut d = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
                d.push("src/testdata/state.json");
                d.display().to_string()
            }
        };

        let state = fs::read_to_string(f).expect("file should have read correctly");

        let document: source::Document = match serde_json::from_str(&state) {
            Ok(document) => document,
            Err(e) => whatever!("failed to deserialize text into document: err {}", e),
        };

        Ok(document)
    }
}
