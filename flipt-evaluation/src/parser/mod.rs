use reqwest::header::{self, HeaderMap};
use serde::Deserialize;

use crate::error::Error;
use crate::models::source;

pub trait Parser {
    fn parse(&self, namespace: &str) -> Result<source::Document, Error>;
}

#[derive(Debug, Clone, Default, Deserialize)]
#[cfg_attr(test, derive(PartialEq))]
#[serde(rename_all = "snake_case")]
pub enum Authentication {
    #[default]
    None,
    ClientToken(String),
    JwtToken(String),
}

impl Authentication {
    pub fn with_client_token(token: String) -> Self {
        Authentication::ClientToken(token)
    }

    pub fn with_jwt_token(token: String) -> Self {
        Authentication::JwtToken(token)
    }

    pub fn authenticate(&self) -> Option<String> {
        match self {
            Authentication::ClientToken(token) => {
                let header_format: String = format!("Bearer {}", token).parse().unwrap();
                Some(header_format)
            }
            Authentication::JwtToken(token) => {
                let header_format: String = format!("JWT {}", token).parse().unwrap();
                Some(header_format)
            }
            Authentication::None => None,
        }
    }
}

impl From<Authentication> for HeaderMap {
    fn from(value: Authentication) -> Self {
        let mut header_map = HeaderMap::new();
        match value.authenticate() {
            Some(val) => {
                header_map.insert(
                    header::AUTHORIZATION,
                    header::HeaderValue::from_str(&val).unwrap(),
                );

                header_map
            }
            None => header_map,
        }
    }
}

pub struct HTTPParser {
    http_client: reqwest::blocking::Client,
    http_url: String,
    authentication: HeaderMap,
    reference: Option<String>,
}

pub struct HTTPParserBuilder {
    http_url: String,
    authentication: HeaderMap,
    reference: Option<String>,
}

impl HTTPParserBuilder {
    pub fn new(http_url: &str) -> Self {
        Self {
            http_url: http_url.to_string(),
            authentication: HeaderMap::new(),
            reference: None,
        }
    }

    pub fn authentication(mut self, authentication: Authentication) -> Self {
        self.authentication = HeaderMap::from(authentication);
        self
    }

    pub fn reference(mut self, reference: &str) -> Self {
        self.reference = Some(reference.to_string());
        self
    }

    pub fn build(self) -> HTTPParser {
        HTTPParser {
            http_client: reqwest::blocking::Client::builder()
                .timeout(std::time::Duration::from_secs(10))
                .build()
                .unwrap(),
            http_url: self.http_url,
            authentication: self.authentication,
            reference: self.reference,
        }
    }
}

impl HTTPParser {
    fn url(&self, namespace: &str) -> String {
        match &self.reference {
            Some(reference) => {
                format!(
                    "{}/internal/v1/evaluation/snapshot/namespace/{}?reference={}",
                    self.http_url, namespace, reference,
                )
            }
            None => {
                format!(
                    "{}/internal/v1/evaluation/snapshot/namespace/{}",
                    self.http_url, namespace
                )
            }
        }
    }
}

impl Parser for HTTPParser {
    fn parse(&self, namespace: &str) -> Result<source::Document, Error> {
        let mut headers = HeaderMap::new();
        headers.insert(
            reqwest::header::CONTENT_TYPE,
            reqwest::header::HeaderValue::from_static("application/json"),
        );
        headers.insert(
            reqwest::header::ACCEPT,
            reqwest::header::HeaderValue::from_static("application/json"),
        );

        for (key, value) in self.authentication.iter() {
            headers.insert(key, value.clone());
        }

        let response = match self
            .http_client
            .get(self.url(namespace))
            .headers(headers)
            .send()
        {
            Ok(resp) => match resp.error_for_status() {
                Ok(resp) => resp,
                Err(e) => return Err(Error::Server(format!("response: {}", e))),
            },
            Err(e) => return Err(Error::Server(format!("failed to make request: {}", e))),
        };

        let response_text = match response.text() {
            Ok(t) => t,
            Err(e) => return Err(Error::Server(format!("failed to get response body: {}", e))),
        };

        let document: source::Document = match serde_json::from_str(&response_text) {
            Ok(doc) => doc,
            Err(e) => return Err(Error::InvalidJSON(e)),
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
    fn parse(&self, _: &str) -> Result<source::Document, Error> {
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
            Err(e) => return Err(Error::InvalidJSON(e)),
        };

        Ok(document)
    }
}

#[cfg(test)]
mod tests {
    use crate::parser::Authentication;

    #[test]
    fn test_http_parser_url() {
        use super::HTTPParserBuilder;

        let parser = HTTPParserBuilder::new("http://localhost:8080")
            .authentication(Authentication::with_client_token("secret".into()))
            .reference("ref")
            .build();

        assert_eq!(
            parser.url("default"),
            "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default?reference=ref"
        );

        let parser = HTTPParserBuilder::new("http://localhost:8080")
            .authentication(Authentication::with_client_token("secret".into()))
            .build();

        assert_eq!(
            parser.url("default"),
            "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default"
        );
    }

    #[test]
    fn test_deserialize_no_auth() {
        let json = r#""#;

        let unwrapped_string: Authentication = serde_json::from_str(&json).unwrap_or_default();

        assert_eq!(unwrapped_string, Authentication::None);
    }

    #[test]
    fn test_deserialize_client_token() {
        let json = r#"{"client_token":"secret"}"#;

        let unwrapped_string: Authentication = serde_json::from_str(&json).unwrap_or_default();

        assert_eq!(
            unwrapped_string,
            Authentication::ClientToken("secret".into())
        );
    }

    #[test]
    fn test_deserialize_jwt_token() {
        let json = r#"{"jwt_token":"secret"}"#;

        let unwrapped_string: Authentication = serde_json::from_str(&json).unwrap_or_default();

        assert_eq!(unwrapped_string, Authentication::JwtToken("secret".into()));
    }
}
