use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc};
use std::thread;
use std::time::Duration;

use futures_util::stream::StreamExt;
use reqwest::header::{self, HeaderMap};
use reqwest::Response;
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

#[derive(Clone)]
enum FetchMode {
    Polling,
    Streaming,
}

#[derive(Clone)]
pub struct HTTPFetcher {
    http_client: reqwest::Client,
    url: String,
    authentication: HeaderMap,
    etag: Option<String>,
    update_interval: Duration,
    running: Arc<AtomicBool>,
    mode: FetchMode,
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
        let url = match &self.reference {
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
        };

        HTTPFetcher {
            http_client: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(10))
                .build()
                .unwrap(),
            url,
            authentication: self.authentication,
            etag: None,
            update_interval: self.update_interval,
            running: Arc::new(AtomicBool::new(false)),
            mode: FetchMode::Polling,
        }
    }
}

impl HTTPFetcher {
    pub fn start(&mut self) -> mpsc::Receiver<Result<source::Document, Error>> {
        self.running = Arc::new(AtomicBool::new(true));
        let (tx, rx) = mpsc::channel();
        let mut fetcher = self.clone();

        let mode = fetcher.mode.clone();
        let running = self.running.clone();

        thread::spawn(move || async move {
            while running.load(Ordering::Relaxed) {
                match mode {
                    FetchMode::Polling => {
                        match fetcher.fetch().await {
                            Ok(resp) => {
                                // if no response then continue
                                let ok = match resp {
                                    Some(resp) => resp,
                                    None => continue,
                                };

                                let text = match ok.text().await {
                                    Ok(text) => text,
                                    Err(e) => {
                                        tx.send(Err(Error::Server(format!(
                                            "failed to read response text: {}",
                                            e
                                        ))))
                                        .unwrap();
                                        continue;
                                    }
                                };

                                let doc: source::Document = match serde_json::from_str(&text) {
                                    Ok(doc) => doc,
                                    Err(e) => {
                                        tx.send(Err(Error::Server(format!(
                                            "failed to parse response text: {}",
                                            e
                                        ))))
                                        .unwrap();
                                        continue;
                                    }
                                };

                                tx.send(Ok(doc)).unwrap();
                            }
                            Err(e) => {
                                tx.send(Err(e)).unwrap();
                            }
                        }
                        thread::sleep(fetcher.update_interval);
                    }

                    FetchMode::Streaming => {
                        match fetcher.fetch_stream().await {
                            Ok(resp) => {
                                // if no response then continue
                                let ok = match resp {
                                    Some(resp) => resp,
                                    None => continue,
                                };

                                let mut stream = ok.bytes_stream();
                                let mut buffer = Vec::new();

                                while let Some(chunk) = stream.next().await {
                                    for byte in chunk.unwrap() {
                                        if byte == b'\n' {
                                            let text = String::from_utf8_lossy(&buffer);
                                            let doc: source::Document =
                                                serde_json::from_str(&text).unwrap();
                                            tx.send(Ok(doc)).unwrap();
                                            buffer.clear();
                                        } else {
                                            buffer.push(byte);
                                        }
                                    }
                                }
                            }
                            Err(e) => {
                                tx.send(Err(e)).unwrap();
                                // Optionally, revert to polling mode on stream error
                                // fetcher.mode = FetchMode::Polling;
                            }
                        }
                    }
                }
            }
        });
        rx
    }

    pub fn stop(&mut self) {
        self.running.store(false, Ordering::Relaxed);
    }

    fn build_headers(&self) -> HeaderMap {
        let mut headers = HeaderMap::new();
        headers.insert(
            reqwest::header::ACCEPT,
            reqwest::header::HeaderValue::from_static("application/json"),
        );
        headers.append(
            reqwest::header::ACCEPT,
            reqwest::header::HeaderValue::from_static(
                "application/vnd.flipt.io.snapshot.stream+json",
            ),
        );

        headers.insert(
            "X-Flipt-Accept-Server-Version",
            reqwest::header::HeaderValue::from_static("1.47.0"),
        );

        if let Some(etag) = &self.etag {
            headers.insert(
                reqwest::header::IF_NONE_MATCH,
                reqwest::header::HeaderValue::from_str(etag).unwrap(),
            );
        }

        // Add authentication headers
        for (key, value) in self.authentication.iter() {
            headers.insert(key, value.clone());
        }

        headers
    }

    async fn fetch(&mut self) -> Result<Option<Response>, Error> {
        let headers = self.build_headers();

        let response = self
            .http_client
            .get(self.url.clone())
            .headers(headers)
            .send()
            .await
            .map_err(|e| {
                self.stop();
                Error::Server(format!("failed to make request: {}", e))
            })?
            .error_for_status()
            .map_err(|e| {
                self.stop();
                Error::Server(format!("response: {}", e))
            })?;

        match response.status() {
            reqwest::StatusCode::NOT_MODIFIED => Ok(None),

            reqwest::StatusCode::SEE_OTHER => {
                if let Some(location) = response.headers().get(reqwest::header::LOCATION) {
                    self.url = location.to_str().unwrap().to_string();
                    self.mode = FetchMode::Streaming;
                }
                Ok(Some(response))
            }

            reqwest::StatusCode::OK => {
                // check if we have a new etag
                if let Some(etag) = response.headers().get(reqwest::header::ETAG) {
                    self.etag = Some(etag.to_str().unwrap().to_string());
                }

                Ok(Some(response))
            }

            _ => Err(Error::Server(format!(
                "unexpected http response: {} {}",
                response.status(),
                response.text().await.unwrap_or("".to_string())
            ))),
        }
    }

    async fn fetch_stream(&mut self) -> Result<Option<Response>, Error> {
        self.http_client
            .get(self.url.clone())
            .headers(self.build_headers())
            .send()
            .await
            .map_err(|e| {
                self.stop();
                Error::Server(format!("failed to make request: {}", e))
            })?
            .error_for_status()
            .map_err(|e| {
                self.stop();
                Error::Server(format!("response: {}", e))
            })
            .map(Some)
    }
}

#[cfg(test)]
mod tests {
    use crate::http::Authentication;
    use crate::http::HTTPFetcherBuilder;

    #[tokio::test]
    async fn test_http_fetch() {
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

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        mock.assert();

        assert_eq!(fetcher.etag, Some("etag".to_string()));
    }

    #[tokio::test]
    async fn test_http_fetch_not_modified() {
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

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        assert!(result.unwrap().is_none());

        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_error() {
        let mut server = mockito::Server::new();
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .with_status(500)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url, "default")
            .authentication(Authentication::None)
            .build();

        let result = fetcher.fetch().await;

        assert!(!result.is_ok());
        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_token_auth() {
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

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_jwt_auth() {
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

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_jwt_auth_failure() {
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

        let result = fetcher.fetch().await;

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
            fetcher.url,
            "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default?reference=ref"
        );

        let fetcher = HTTPFetcherBuilder::new("http://localhost:8080", "default")
            .authentication(Authentication::with_client_token("secret".into()))
            .build();

        assert_eq!(
            fetcher.url,
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
