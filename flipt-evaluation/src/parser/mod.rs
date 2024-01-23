use reqwest::header::HeaderMap;
use serde::Deserialize;

use crate::error::Error;
use crate::models::source;

pub trait Parser {
    fn parse(&self, namespace: &str) -> Result<source::Document, Error>;
}

#[derive(Deserialize, Clone)]
pub struct Authentication {
    client_token: Option<String>,
    jwt_token: Option<String>,
}

impl Authentication {
    pub fn with_client_token(token: String) -> Self {
        Self {
            client_token: Some(token),
            jwt_token: None,
        }
    }

    pub fn with_jwt_token(token: String) -> Self {
        Self {
            client_token: None,
            jwt_token: Some(token),
        }
    }

    pub fn authenticate(&self) -> Option<String> {
        if let Some(token) = &self.client_token {
            let header_format: String = format!("Bearer {}", token).parse().unwrap();
            return Some(header_format);
        } else if let Some(token) = &self.jwt_token {
            let header_format: String = format!("JWT {}", token).parse().unwrap();
            return Some(header_format);
        }

        None
    }
}

pub struct HTTPParser {
    http_client: reqwest::blocking::Client,
    http_url: String,
    authentication: Option<String>,
    reference: Option<String>,
}

pub struct HTTPParserBuilder {
    http_url: String,
    authentication: Option<String>,
    reference: Option<String>,
}

impl HTTPParserBuilder {
    pub fn new(http_url: &str) -> Self {
        Self {
            http_url: http_url.to_string(),
            authentication: None,
            reference: None,
        }
    }

    pub fn authentication(mut self, authentication: &Authentication) -> Self {
        self.authentication = authentication.authenticate();
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

        if let Some(ref token) = self.authentication {
            headers.insert(
                reqwest::header::AUTHORIZATION,
                reqwest::header::HeaderValue::from_str(token).unwrap(),
            );
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
            .authentication(&Authentication::with_client_token("secret".into()))
            .reference("ref")
            .build();

        assert_eq!(
            parser.url("default"),
            "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default?reference=ref"
        );

        let parser = HTTPParserBuilder::new("http://localhost:8080")
            .authentication(&Authentication::with_client_token("secret".into()))
            .build();

        assert_eq!(
            parser.url("default"),
            "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default"
        );
    }
}
