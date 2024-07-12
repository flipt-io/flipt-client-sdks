use reqwest::header::{self, HeaderMap};
use serde::Deserialize;

use fliptevaluation::error::Error;
use fliptevaluation::models::source;
use fliptevaluation::parser::Parser;

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
    etag: Option<String>,
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
            etag: None,
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
    fn parse(&mut self, namespace: &str) -> Result<Option<source::Document>, Error> {
        let mut headers = HeaderMap::new();
        headers.insert(
            reqwest::header::CONTENT_TYPE,
            reqwest::header::HeaderValue::from_static("application/json"),
        );
        headers.insert(
            reqwest::header::ACCEPT,
            reqwest::header::HeaderValue::from_static("application/json"),
        );
        // version (or higher) that we can accept from the server
        headers.insert(
            "X-Flipt-Accept-Server-Version",
            reqwest::header::HeaderValue::from_static("1.38.0"),
        );

        // add etag / if-none-match header if we have one
        if let Some(etag) = &self.etag {
            headers.insert(
                reqwest::header::IF_NONE_MATCH,
                reqwest::header::HeaderValue::from_str(etag).unwrap(),
            );
        }

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

        // check if we have a 304 response
        if response.status() == reqwest::StatusCode::NOT_MODIFIED {
            return Ok(None);
        }

        // check if we have a new etag
        if let Some(etag) = response.headers().get(reqwest::header::ETAG) {
            self.etag = Some(etag.to_str().unwrap().to_string());
        }

        let response_text = match response.text() {
            Ok(t) => t,
            Err(e) => return Err(Error::Server(format!("failed to get response body: {}", e))),
        };

        let document: source::Document = match serde_json::from_str(&response_text) {
            Ok(doc) => doc,
            Err(e) => return Err(Error::InvalidJSON(e.to_string())),
        };

        Ok(Some(document))
    }
}

#[cfg(test)]
mod tests {
    use crate::parser::http::Authentication;
    use crate::parser::http::HTTPParserBuilder;
    use fliptevaluation::parser::Parser;

    #[test]
    fn test_parse() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create();

        let url = server.url();
        let mut parser = HTTPParserBuilder::new(&url)
            .authentication(Authentication::None)
            .build();

        let result = parser.parse("default");

        assert!(result.is_ok());
        mock.assert();
    }

    #[test]
    fn test_parse_not_modified() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("if-none-match", "etag")
            .with_status(304)
            .create();

        let url = server.url();
        let mut parser = HTTPParserBuilder::new(&url)
            .authentication(Authentication::None)
            .build();

        parser.etag = Some("etag".to_string());

        let result = parser.parse("default");

        assert!(result.is_ok());
        assert!(result.unwrap().is_none());

        mock.assert();
    }

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
