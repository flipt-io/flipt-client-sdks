use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;

use reqwest::header::{self, HeaderMap};
use serde::Deserialize;

use fliptevaluation::error::Error;
use fliptevaluation::models::source;

#[derive(Debug, Clone, Default, Deserialize)]
#[cfg_attr(test, derive(PartialEq))]
#[serde(rename_all = "snake_case")]
#[non_exhaustive]
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
pub trait Fetcher: Send {
    fn fetch(&mut self) -> Result<Option<source::Document>, Error>;
}

#[derive(Clone)]
pub struct HTTPFetcher {
    namespace: String,
    http_client: reqwest::blocking::Client,
    base_url: String,
    authentication: HeaderMap,
    reference: Option<String>,
    etag: Option<String>,
    update_interval: Duration,
    running: Arc<AtomicBool>,
}

pub struct HTTPFetcherBuilder {
    namespace: String,
    base_url: String,
    authentication: HeaderMap,
    reference: Option<String>,
    update_interval: Duration,
}

impl HTTPFetcherBuilder {
    pub fn new(base_url: &str, namespace: &str) -> Self {
        Self {
            namespace: namespace.to_string(),
            base_url: base_url.to_string(),
            authentication: HeaderMap::new(),
            reference: None,
            update_interval: Duration::from_secs(120),
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

    pub fn update_interval(mut self, update_interval: Duration) -> Self {
        self.update_interval = update_interval;
        self
    }

    pub fn build(self) -> HTTPFetcher {
        HTTPFetcher {
            namespace: self.namespace,
            http_client: reqwest::blocking::Client::builder()
                .timeout(std::time::Duration::from_secs(10))
                .build()
                .unwrap(),
            base_url: self.base_url,
            authentication: self.authentication,
            reference: self.reference,
            etag: None,
            update_interval: self.update_interval,
            running: Arc::new(AtomicBool::new(false)),
        }
    }
}

impl HTTPFetcher {
    fn url(&self) -> String {
        match &self.reference {
            Some(reference) => {
                format!(
                    "{}/internal/v1/evaluation/snapshot/namespace/{}?reference={}",
                    self.base_url, self.namespace, reference,
                )
            }
            None => {
                format!(
                    "{}/internal/v1/evaluation/snapshot/namespace/{}",
                    self.base_url, self.namespace
                )
            }
        }
    }

    pub fn start(&mut self) -> mpsc::Receiver<Result<source::Document, Error>> {
        let running = Arc::new(AtomicBool::new(true));
        let (tx, rx) = mpsc::channel(100);
        let mut fetcher = self.clone();

        tokio::spawn(async move {
            while running.load(Ordering::Relaxed) {
                match fetcher.fetch() {
                    Ok(doc) => {
                        if let Some(doc) = doc {
                            tx.send(Ok(doc)).await.unwrap();
                        }
                    }
                    Err(e) => {
                        tx.send(Err(e)).await.unwrap();
                    }
                }

                tokio::time::sleep(fetcher.update_interval).await;
            }
        });
        rx
    }

    pub fn stop(&mut self) {
        self.running.store(false, Ordering::Relaxed);
    }
}

impl Fetcher for HTTPFetcher {
    fn fetch(&mut self) -> Result<Option<source::Document>, Error> {
        let mut headers = HeaderMap::new();
        headers.insert(
            reqwest::header::ACCEPT,
            reqwest::header::HeaderValue::from_static("application/json"),
        );
        // version (or higher) that we can accept from the server
        headers.insert(
            "X-Flipt-Accept-Server-Version",
            reqwest::header::HeaderValue::from_static("1.47.0"),
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

        let url = self.url();
        let response = match self.http_client.get(url).headers(headers).send() {
            Ok(resp) => match resp.error_for_status() {
                Ok(resp) => resp,
                Err(e) => return Err(Error::Server(format!("response: {}", e))),
            },
            Err(e) => return Err(Error::Server(format!("failed to make request: {}", e))),
        };

        match response.status() {
            // check if we have a 304 response
            reqwest::StatusCode::NOT_MODIFIED => Ok(None),
            reqwest::StatusCode::OK => {
                // check if we have a new etag
                if let Some(etag) = response.headers().get(reqwest::header::ETAG) {
                    self.etag = Some(etag.to_str().unwrap().to_string());
                }

                let response_text = match response.text() {
                    Ok(t) => t,
                    Err(e) => {
                        return Err(Error::Server(format!("failed to get response body: {}", e)))
                    }
                };

                match serde_json::from_str(&response_text) {
                    Ok(doc) => Ok(Some(doc)),
                    Err(e) => Err(Error::InvalidJSON(e.to_string())),
                }
            }
            _ => Err(Error::Server(format!(
                "unexpected http response: {} {}",
                response.status(),
                response.text().unwrap_or("".to_string())
            ))),
        }
    }
}

#[cfg(test)]
mod tests {
    use crate::http::Authentication;
    use crate::http::Fetcher;
    use crate::http::HTTPFetcherBuilder;

    #[test]
    fn test_http_fetch() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_header("etag", "etag")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url, "default")
            .authentication(Authentication::None)
            .build();

        let result = fetcher.fetch();

        assert!(result.is_ok());
        mock.assert();

        assert_eq!(fetcher.etag, Some("etag".to_string()));
    }

    #[test]
    fn test_http_fetch_not_modified() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("if-none-match", "etag")
            .with_status(304)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url, "default")
            .authentication(Authentication::None)
            .build();

        fetcher.etag = Some("etag".to_string());

        let result = fetcher.fetch();

        assert!(result.is_ok());
        assert!(result.unwrap().is_none());

        mock.assert();
    }

    #[test]
    fn test_http_fetch_error() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .with_status(500)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url, "default")
            .authentication(Authentication::None)
            .build();

        let result = fetcher.fetch();

        assert!(!result.is_ok());
        mock.assert();
    }

    #[test]
    fn test_http_fetch_token_auth() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("authorization", "Bearer foo")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url, "default")
            .authentication(Authentication::ClientToken("foo".to_string()))
            .build();

        let result = fetcher.fetch();

        assert!(result.is_ok());
        mock.assert();
    }

    #[test]
    fn test_http_fetch_jwt_auth() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("authorization", "JWT foo")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url, "default")
            .authentication(Authentication::JwtToken("foo".to_string()))
            .build();

        let result = fetcher.fetch();

        assert!(result.is_ok());
        mock.assert();
    }

    #[test]
    fn test_http_fetch_jwt_auth_failure() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("authorization", "JWT foo")
            .with_status(401)
            .with_header("content-type", "application/json")
            .with_body(r#"{"code":16,"message":"request was not authenticated","details":[]}"#)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url, "default")
            .authentication(Authentication::JwtToken("foo".to_string()))
            .build();

        let result = fetcher.fetch();

        assert!(result.is_err());
        mock.assert();
    }

    #[test]
    fn test_http_fetcher_url() {
        use super::HTTPFetcherBuilder;

        let fetcher = HTTPFetcherBuilder::new("http://localhost:8080", "default")
            .authentication(Authentication::with_client_token("secret".into()))
            .reference("ref")
            .build();

        assert_eq!(
            fetcher.url(),
            "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default?reference=ref"
        );

        let fetcher = HTTPFetcherBuilder::new("http://localhost:8080", "default")
            .authentication(Authentication::with_client_token("secret".into()))
            .build();

        assert_eq!(
            fetcher.url(),
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
