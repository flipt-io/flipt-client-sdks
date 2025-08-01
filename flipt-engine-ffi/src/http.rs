use std::error::Error as StdError;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::Duration;

use async_trait::async_trait;
use futures::TryStreamExt;
use futures_util::stream::StreamExt;
use reqwest::header::{self, HeaderMap};
use reqwest::Response;
use reqwest_middleware::{ClientBuilder, ClientWithMiddleware, Middleware, Next};
use reqwest_retry::policies::ExponentialBackoff;
use reqwest_retry::{Jitter, RetryTransientMiddleware};
use serde::Deserialize;
use tokio::sync::{mpsc, Notify};
use tokio_util::io::StreamReader;

use fliptevaluation::error::Error;
use fliptevaluation::models::source;

use crate::tls::configure_tls;
use crate::TlsConfig;

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
                let header_format: String = format!("Bearer {token}").parse().unwrap();
                Some(header_format)
            }
            Authentication::JwtToken(token) => {
                let header_format: String = format!("JWT {token}").parse().unwrap();
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

#[derive(Debug, Clone, Default, Deserialize)]
#[cfg_attr(test, derive(PartialEq))]
#[serde(rename_all = "snake_case")]
#[non_exhaustive]
pub enum FetchMode {
    #[default]
    Polling,
    Streaming,
}

#[derive(Debug, Clone, Default, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
#[non_exhaustive]
pub enum ErrorStrategy {
    /// The default behavior: fail fast.
    #[default]
    Fail,
    /// Fallback: use the previous available Snapshot state.
    Fallback,
}

#[derive(Clone)]
pub struct HTTPFetcher {
    http_client: ClientWithMiddleware,
    base_url: String,
    environment: String,
    namespace: String,
    authentication: HeaderMap,
    etag: Option<String>,
    reference: Option<String>,
    update_interval: Duration,
    mode: FetchMode,
}

pub struct HTTPFetcherBuilder {
    base_url: String,
    environment: Option<String>,
    namespace: Option<String>,
    authentication: HeaderMap,
    reference: Option<String>,
    request_timeout: Option<Duration>,
    update_interval: Duration,
    mode: FetchMode,
    tls_config: Option<TlsConfig>,
}

#[derive(Deserialize)]
struct StreamChunk {
    result: StreamResult,
}

#[derive(Deserialize)]
struct StreamResult {
    namespaces: std::collections::HashMap<String, source::Document>,
}

static USER_AGENT: &str = concat!(env!("CARGO_PKG_NAME"), "/", env!("CARGO_PKG_VERSION"),);

/// Logging middleware that logs each request attempt
#[derive(Debug, Clone, Default)]
pub struct LoggingMiddleware {}

#[async_trait]
impl Middleware for LoggingMiddleware {
    async fn handle(
        &self,
        req: reqwest::Request,
        extensions: &mut http::Extensions,
        next: Next<'_>,
    ) -> reqwest_middleware::Result<reqwest::Response> {
        let method = req.method().clone();
        let url = req.url().clone();

        let result = next.run(req, extensions).await;

        if let Err(error) = &result {
            let error_details = if let Some(source) = StdError::source(error) {
                format!("{error} (source: {source})")
            } else {
                error.to_string()
            };

            log::warn!("error {method} {url} -> {error_details}");
        }

        result
    }
}

impl Default for HTTPFetcherBuilder {
    fn default() -> Self {
        Self::new("http://localhost:8080")
    }
}

impl HTTPFetcherBuilder {
    pub fn new(base_url: &str) -> Self {
        Self {
            base_url: base_url.to_string(),
            environment: None,
            namespace: None,
            authentication: HeaderMap::new(),
            reference: None,
            request_timeout: None,
            update_interval: Duration::from_secs(120),
            mode: FetchMode::default(),
            tls_config: None,
        }
    }

    pub fn environment(mut self, environment: &str) -> Self {
        self.environment = Some(environment.to_string());
        self
    }

    pub fn namespace(mut self, namespace: &str) -> Self {
        self.namespace = Some(namespace.to_string());
        self
    }

    pub fn authentication(mut self, authentication: Authentication) -> Self {
        self.authentication = HeaderMap::from(authentication);
        self
    }

    pub fn reference(mut self, reference: &str) -> Self {
        self.reference = Some(reference.to_string());
        self
    }

    pub fn request_timeout(mut self, request_timeout: Duration) -> Self {
        if request_timeout.as_secs() > 0 {
            self.request_timeout = Some(request_timeout);
        }
        self
    }

    pub fn update_interval(mut self, update_interval: Duration) -> Self {
        if update_interval.as_secs() > 0 {
            self.update_interval = update_interval;
        }
        self
    }

    pub fn mode(mut self, mode: FetchMode) -> Self {
        self.mode = mode;
        self
    }

    pub fn tls_config(mut self, tls_config: TlsConfig) -> Self {
        self.tls_config = Some(tls_config);
        self
    }

    pub fn build(self) -> Result<HTTPFetcher, Error> {
        let retry_policy = ExponentialBackoff::builder()
            .retry_bounds(Duration::from_secs(1), Duration::from_secs(30))
            .jitter(Jitter::Full)
            .build_with_max_retries(3);

        let mut client_builder = reqwest::Client::builder()
            .user_agent(USER_AGENT)
            .tcp_keepalive(Duration::from_secs(5))
            .pool_max_idle_per_host(10)
            .pool_idle_timeout(Duration::from_secs(60))
            .connect_timeout(Duration::from_secs(10));

        match self.mode {
            FetchMode::Polling => {
                if let Some(request_timeout) = self.request_timeout {
                    if request_timeout.as_secs() > 0 {
                        client_builder = client_builder.timeout(request_timeout);
                    }
                }
            }
            FetchMode::Streaming => {
                client_builder = client_builder
                    .http2_prior_knowledge()
                    .http2_keep_alive_interval(Duration::from_secs(5))
                    .http2_initial_stream_window_size(32 * 1024)
                    .http2_initial_connection_window_size(32 * 1024)
                    .http2_keep_alive_while_idle(true);
            }
        }

        // Apply TLS configuration if provided
        if let Some(tls_config) = &self.tls_config {
            client_builder = configure_tls(client_builder, tls_config)?;
        }

        let client = client_builder
            .build()
            .map_err(|e| Error::Internal(format!("failed to create client: {e}")))?;

        Ok(HTTPFetcher {
            base_url: self.base_url,
            environment: self.environment.unwrap_or("default".to_string()),
            namespace: self.namespace.unwrap_or("default".to_string()),
            http_client: ClientBuilder::new(client)
                .with(RetryTransientMiddleware::new_with_policy(retry_policy))
                .with(LoggingMiddleware::default())
                .build(),
            authentication: self.authentication,
            etag: None,
            reference: self.reference,
            update_interval: self.update_interval,
            mode: self.mode,
        })
    }
}

type FetchResult = Result<source::Document, Error>;

impl HTTPFetcher {
    /// Start the fetcher and return a channel to receive updates on the snapshot changes
    pub fn start(
        &mut self,
        stop_signal: Arc<AtomicBool>,
        stop_notify: Arc<Notify>,
    ) -> mpsc::Receiver<FetchResult> {
        let (tx, rx) = mpsc::channel(100);

        let mut fetcher = self.clone();
        let update_interval = fetcher.update_interval;
        // Clone for move into async
        let stop_notify_clone = stop_notify.clone();

        tokio::spawn(async move {
            while !stop_signal.load(Ordering::Relaxed) {
                match fetcher.mode {
                    FetchMode::Polling => {
                        if let Err(e) = fetcher.handle_polling(&tx).await {
                            log::warn!("error fetching polling: {e}");
                            break;
                        }
                        let mut elapsed = Duration::ZERO;
                        let interval = update_interval;
                        let check = Duration::from_millis(500);

                        while elapsed < interval {
                            if stop_signal.load(Ordering::Relaxed) {
                                return;
                            }
                            let sleep = tokio::time::sleep(check);
                            tokio::pin!(sleep);
                            tokio::select! {
                                _ = &mut sleep => {
                                    elapsed += check;
                                },
                                _ = stop_notify_clone.notified() => {
                                    return;
                                },
                            }
                        }
                    }
                    FetchMode::Streaming => {
                        if let Err(e) = fetcher.handle_streaming(&tx).await {
                            log::warn!("error fetching streaming: {e}");
                            break;
                        }
                    }
                }
            }
            // The channel will be closed when tx is dropped at the end of this function
        });

        rx
    }

    pub async fn initial_fetch(&mut self) -> FetchResult {
        let response = self.fetch().await?;

        match response {
            Some(resp) => {
                let document = resp.json::<source::Document>().await.map_err(|e| {
                    Error::InvalidJSON(format!("failed to parse response body: {e}"))
                })?;
                Ok(document)
            }
            None => Err(Error::Server("no data received from server".into())),
        }
    }

    fn build_headers(&self) -> HeaderMap {
        let mut headers = HeaderMap::new();
        headers.insert(
            reqwest::header::ACCEPT,
            reqwest::header::HeaderValue::from_static("application/json"),
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

        if !self.environment.is_empty() {
            headers.insert(
                "X-Flipt-Environment",
                reqwest::header::HeaderValue::from_str(&self.environment).unwrap(),
            );
        }

        for (key, value) in self.authentication.iter() {
            headers.insert(key, value.clone());
        }

        headers
    }

    async fn fetch(&mut self) -> Result<Option<Response>, Error> {
        let url = match &self.reference {
            Some(reference) => {
                format!(
                    "{}/internal/v1/evaluation/snapshot/namespace/{}?reference={}",
                    self.base_url, self.namespace, reference
                )
            }
            None => {
                format!(
                    "{}/internal/v1/evaluation/snapshot/namespace/{}",
                    self.base_url, self.namespace
                )
            }
        };

        log::debug!("making HTTP request to: {url}");
        match self
            .http_client
            .get(&url)
            .headers(self.build_headers())
            .send()
            .await
        {
            Ok(response) => match response.error_for_status() {
                Ok(response) => match response.status() {
                    reqwest::StatusCode::NOT_MODIFIED => Ok(None),
                    reqwest::StatusCode::OK => {
                        if let Some(etag) = response.headers().get(reqwest::header::ETAG) {
                            self.etag = Some(etag.to_str().unwrap().to_string());
                        }

                        Ok(Some(response))
                    }
                    _ => {
                        self.etag = None;
                        let status = response.status();
                        Err(Error::Server(format!("unexpected http response: {status}")))
                    }
                },
                Err(e) => {
                    self.etag = None;
                    Err(Error::Server(format!("response: {e}")))
                }
            },
            Err(e) => {
                self.etag = None;
                Err(Error::Server(format!("failed to make request: {e}")))
            }
        }
    }

    async fn fetch_stream(&mut self) -> Result<Option<Response>, Error> {
        let url = format!(
            "{}/internal/v1/evaluation/snapshots?[]namespaces={}",
            self.base_url, self.namespace
        );

        match self
            .http_client
            .get(url)
            .headers(self.build_headers())
            .send()
            .await
        {
            Ok(response) => response
                .error_for_status()
                .map_err(|e| Error::Server(format!("response: {e}")))
                .map(Some),
            Err(e) => Err(Error::Server(format!("failed to make request: {e}"))),
        }
    }

    async fn handle_polling(
        &mut self,
        sender: &mpsc::Sender<Result<source::Document, Error>>,
    ) -> Result<(), Error> {
        let result = match self.fetch().await {
            Ok(Some(resp)) => match resp.json::<source::Document>().await {
                Ok(doc) => Ok(doc),
                Err(e) => Err(Error::InvalidJSON(format!(
                    "failed to parse response body: {e}"
                ))),
            },
            Ok(None) => return Ok(()),
            Err(e) => Err(e),
        };

        sender
            .send(result)
            .await
            .map_err(|_| Error::Internal("failed to send result".into()))
    }

    async fn handle_streaming(
        &mut self,
        sender: &mpsc::Sender<Result<source::Document, Error>>,
    ) -> Result<(), Error> {
        let stream_result = self.fetch_stream().await;

        match stream_result {
            Ok(Some(resp)) => {
                let reader = StreamReader::new(resp.bytes_stream().map_err(std::io::Error::other));
                let codec = tokio_util::codec::LinesCodec::new();
                let frame_reader = tokio_util::codec::FramedRead::new(reader, codec);

                let mut stream = frame_reader
                    .into_stream()
                    .map(|frame| match frame {
                        Ok(frame) => match serde_json::from_str::<StreamChunk>(&frame) {
                            Ok(result) => Ok(result),
                            Err(e) => Err(Error::InvalidJSON(format!(
                                "failed to parse response body: {e}"
                            ))),
                        },
                        Err(e) => Err(Error::Server(format!("failed to read stream chunk: {e}"))),
                    })
                    .map(|result| match result {
                        Ok(result) => match result.result.namespaces.into_iter().next() {
                            Some((_, doc)) => Ok(doc),
                            None => {
                                let err = Error::Server("no data received from server".into());
                                Err(err)
                            }
                        },
                        Err(err) => Err(err),
                    });

                while let Some(value) = stream.next().await {
                    match sender.send(value).await {
                        Ok(_) => continue,
                        Err(e) => {
                            return Err(Error::Internal(format!(
                                "failed to send result to engine {e}"
                            )))
                        }
                    }
                }

                Ok(())
            }
            Ok(None) => Ok(()),
            Err(e) => sender
                .send(Err(e.clone()))
                .await
                .map_err(|_| Error::Internal("failed to send error".into())),
        }
    }
}

#[cfg(test)]
mod tests {
    use futures::FutureExt;
    use mockito::Server;

    use crate::http::Authentication;
    use crate::http::FetchMode;
    use crate::http::HTTPFetcherBuilder;
    use tokio::sync::mpsc;

    #[tokio::test]
    async fn test_http_fetch() {
        let mut server = Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_header("etag", "etag")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .build()
            .unwrap();

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        mock.assert();

        assert_eq!(fetcher.etag, Some("etag".to_string()));
    }

    #[tokio::test]
    async fn test_http_fetch_with_reference() {
        let mut server = Server::new_async().await;
        let mock = server
            .mock(
                "GET",
                "/internal/v1/evaluation/snapshot/namespace/default?reference=123",
            )
            .match_header("x-flipt-environment", "default")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_header("etag", "etag")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .reference("123")
            .build()
            .unwrap();

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        mock.assert();

        assert_eq!(fetcher.etag, Some("etag".to_string()));
    }

    #[tokio::test]
    async fn test_http_fetch_not_modified() {
        let mut server = mockito::Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("if-none-match", "etag")
            .match_header("x-flipt-environment", "default")
            .with_status(304)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .build()
            .unwrap();

        fetcher.etag = Some("etag".to_string());

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        assert!(result.unwrap().is_none());

        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_error() {
        let mut server = mockito::Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .with_status(500)
            .expect_at_most(4)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .build()
            .unwrap();

        let result = fetcher.fetch().await;

        assert!(result.is_err());
        assert_eq!(fetcher.etag, None);
        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_token_auth() {
        let mut server = mockito::Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .match_header("authorization", "Bearer foo")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::ClientToken("foo".to_string()))
            .build()
            .unwrap();

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_jwt_auth() {
        let mut server = mockito::Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .match_header("authorization", "JWT foo")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::JwtToken("foo".to_string()))
            .build()
            .unwrap();

        let result = fetcher.fetch().await;

        assert!(result.is_ok());
        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_jwt_auth_failure() {
        let mut server = mockito::Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .match_header("authorization", "JWT foo")
            .with_status(401)
            .with_header("content-type", "application/json")
            .with_body(r#"{"code":16,"message":"request was not authenticated","details":[]}"#)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::JwtToken("foo".to_string()))
            .build()
            .unwrap();

        let result = fetcher.fetch().await;

        assert!(result.is_err());
        mock.assert();
    }

    #[tokio::test]
    async fn test_http_fetch_stream() {
        let mut server = Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshots?[]namespaces=default")
            .match_header("x-flipt-environment", "default")
            .with_status(200)
            .with_header(
                "content-type",
                "application/json",
            )
            .with_chunked_body(|w| {
                w.write_all(b"{\"result\":{ \"namespaces\":{\"default\":{\"namespace\": {\"key\": \"default\"}, \"flags\":[]}}}}\n")?;
                w.write_all(
                    b"{\"result\":{ \"namespaces\":{\"default\":{\"namespace\": {\"key\": \"default\"}, \"flags\":[{\"key\": \"new_flag\", \"name\": \"new flag\", \"enabled\": false}]}}}}\n",
                )?;
                w.write_all(b"{\n")?;
                Ok(())
            })
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .mode(FetchMode::Streaming)
            .build()
            .unwrap();

        let (tx, mut rx) = mpsc::channel(4);
        let result = fetcher.handle_streaming(&tx).await;
        assert!(result.is_ok());
        mock.assert();
        assert!(rx.len() == 3);
        // check first result
        let result = rx
            .recv()
            .map(|r| r.expect("valid record").expect("valid doc"))
            .await;
        assert_eq!("default", result.namespace.key);
        assert_eq!(0, result.flags.len());
        // check second result
        let result = rx
            .recv()
            .map(|r| r.expect("valid record").expect("valid doc"))
            .await;
        assert_eq!("default", result.namespace.key);
        assert_eq!(1, result.flags.len());
        // check third result
        let result = rx.recv().map(|r| r.expect("valid record")).await;
        assert!(result.is_err())
    }

    #[tokio::test]
    async fn test_initial_fetch() {
        let mut server = Server::new_async().await;
        let mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .create();

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .build()
            .unwrap();

        let result = fetcher.initial_fetch().await;

        assert!(result.is_ok());
        mock.assert();
    }

    #[test]
    fn test_deserialize_no_auth() {
        let json = r#""#;

        let unwrapped_string: Authentication = serde_json::from_str(json).unwrap_or_default();

        assert_eq!(unwrapped_string, Authentication::None);
    }

    #[test]
    fn test_deserialize_client_token() {
        let json = r#"{"client_token":"secret"}"#;

        let unwrapped_string: Authentication = serde_json::from_str(json).unwrap_or_default();

        assert_eq!(
            unwrapped_string,
            Authentication::ClientToken("secret".into())
        );
    }

    #[test]
    fn test_deserialize_jwt_token() {
        let json = r#"{"jwt_token":"secret"}"#;

        let unwrapped_string: Authentication = serde_json::from_str(json).unwrap_or_default();

        assert_eq!(unwrapped_string, Authentication::JwtToken("secret".into()));
    }
}
