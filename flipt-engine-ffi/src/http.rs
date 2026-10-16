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
use tokio::sync::{mpsc, watch, Notify};
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

pub struct HTTPFetcher {
    http_client: ClientWithMiddleware,
    base_url: String,
    environment: String,
    namespace: String,
    auth_receiver: watch::Receiver<HeaderMap>,
    auth_sender: Option<watch::Sender<HeaderMap>>,
    etag: Option<String>,
    reference: Option<String>,
    request_timeout: Duration,
    update_interval: Duration,
    mode: FetchMode,
}

impl Clone for HTTPFetcher {
    fn clone(&self) -> Self {
        Self {
            http_client: self.http_client.clone(),
            base_url: self.base_url.clone(),
            environment: self.environment.clone(),
            namespace: self.namespace.clone(),
            auth_receiver: self.auth_receiver.clone(),
            auth_sender: None,
            etag: self.etag.clone(),
            reference: self.reference.clone(),
            request_timeout: self.request_timeout,
            update_interval: self.update_interval,
            mode: self.mode.clone(),
        }
    }
}

pub struct HTTPFetcherBuilder {
    base_url: String,
    environment: Option<String>,
    namespace: Option<String>,
    authentication: HeaderMap,
    reference: Option<String>,
    request_timeout: Duration,
    update_interval: Duration,
    mode: FetchMode,
    tls_config: Option<TlsConfig>,
}

#[derive(Deserialize)]
struct StreamChunk {
    result: source::Document,
}

static USER_AGENT: &str = concat!(env!("CARGO_PKG_NAME"), "/", env!("CARGO_PKG_VERSION"),);

/// Default overall request timeout for polling and streaming. Used as the
/// builder's initial `request_timeout` so a stalled HTTP response (connection
/// established, body never completes) cannot block the refresher forever.
/// `connect_timeout` does not cover this case. See #1926.
///
/// In streaming mode this bounds only the wait for response headers on each
/// reconnect; under `ErrorStrategy::Fail` a reconnect that gets no response
/// within this timeout replaces the snapshot with an error.
const DEFAULT_REQUEST_TIMEOUT: Duration = Duration::from_secs(30);

/// Bounds for the streaming reconnect backoff, mirroring the
/// `RetryTransientMiddleware` retry bounds used for polling.
const STREAM_BACKOFF_MIN: Duration = Duration::from_secs(1);
const STREAM_BACKOFF_MAX: Duration = Duration::from_secs(30);

/// Reconnect delay after `failures` consecutive stream failures.
/// Mirrors the retry policy: `min(30s, 1s * 2^(failures-1))` with Full jitter
/// (uniform in `[0, interval]`), so a fast terminal error (e.g. 404, which the
/// retry middleware does not retry) cannot spin the reconnect loop.
fn stream_backoff(failures: u32, jitter_state: &mut u64) -> Duration {
    let shift = failures.saturating_sub(1).min(5);
    let interval = STREAM_BACKOFF_MIN
        .saturating_mul(1 << shift)
        .min(STREAM_BACKOFF_MAX);
    full_jitter(interval, jitter_state)
}

/// Uniform delay in `[0, interval]`. A small splitmix64 stepper instead of a
/// dependency: backoff spreading needs no cryptographic randomness. Seeded
/// once per streaming task so separate processes diverge.
fn full_jitter(interval: Duration, state: &mut u64) -> Duration {
    *state = state.wrapping_add(0x9E37_79B9_7F4A_7C15);
    let mut z = *state;
    z = (z ^ (z >> 30)).wrapping_mul(0xBF58_476D_1CE4_E5B9);
    z = (z ^ (z >> 27)).wrapping_mul(0x94D0_49BB_1331_11EB);
    let r = z ^ (z >> 31);
    interval.mul_f64(r as f64 / u64::MAX as f64)
}

/// Sleep for `delay`, returning early if stopped. The caller (`start()`'s
/// loop) owns the stop-vs-reconnect decision, so no return value is needed.
async fn sleep_or_stop(stop_signal: &Arc<AtomicBool>, stop_notify: &Arc<Notify>, delay: Duration) {
    // Same ordering as in `handle_streaming`: register first so a stop
    // arriving between the flag check and the select! is not lost.
    let notified = stop_notify.notified();
    tokio::pin!(notified);
    notified.as_mut().enable();
    if stop_signal.load(Ordering::Relaxed) {
        return;
    }
    tokio::select! {
        _ = tokio::time::sleep(delay) => {},
        _ = &mut notified => {},
    }
}

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
            // Build complete error chain
            let mut error_chain = Vec::new();
            error_chain.push(error.to_string());

            let mut current_error: Option<&dyn StdError> = error.source();
            while let Some(source) = current_error {
                // Use Debug format for complete error information
                // This ensures we capture all error details, even for types with poor Display implementations
                error_chain.push(format!("{source:?}"));
                current_error = source.source();
            }

            let error_details = if error_chain.len() > 1 {
                format!(
                    "{} (caused by: {})",
                    error_chain[0],
                    error_chain[1..].join(" -> ")
                )
            } else {
                error_chain[0].clone()
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
            // Default to an overall request timeout so a stalled response cannot
            // block the refresher forever (#1926). Applies to polling and to the
            // streaming header wait. Callers can override via `request_timeout`.
            request_timeout: DEFAULT_REQUEST_TIMEOUT,
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
        if !request_timeout.is_zero() {
            self.request_timeout = request_timeout;
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
                client_builder = client_builder.timeout(self.request_timeout);
            }
            FetchMode::Streaming => {
                client_builder = client_builder
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

        let (auth_sender, auth_receiver) = watch::channel(self.authentication);

        Ok(HTTPFetcher {
            base_url: self.base_url,
            environment: self.environment.unwrap_or("default".to_string()),
            namespace: self.namespace.unwrap_or("default".to_string()),
            http_client: ClientBuilder::new(client)
                .with(RetryTransientMiddleware::new_with_policy(retry_policy))
                .with(LoggingMiddleware::default())
                .build(),
            auth_receiver,
            auth_sender: Some(auth_sender),
            etag: None,
            reference: self.reference,
            request_timeout: self.request_timeout,
            update_interval: self.update_interval,
            mode: self.mode,
        })
    }
}

type FetchResult = Result<source::Document, Error>;

/// What one `handle_streaming` episode did, so `start()` can manage the
/// reconnect backoff.
enum StreamOutcome {
    /// Stop was requested; exit the task.
    Stopped,
    /// Episode ended; back off, then reconnect. `had_docs` reports whether
    /// any valid document was forwarded (a productive connection reconnects
    /// quickly; a barren one backs off harder).
    Reconnect { had_docs: bool },
}

impl HTTPFetcher {
    /// Take the auth sender out of the fetcher. Returns `None` if already taken or if this is a clone.
    pub fn take_auth_sender(&mut self) -> Option<watch::Sender<HeaderMap>> {
        self.auth_sender.take()
    }

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
        // Consecutive failed streaming episodes; reset whenever an episode
        // forwards a valid document.
        let mut stream_failures: u32 = 0;
        // Jitter state for the reconnect backoff, seeded once per task so
        // separate processes diverge.
        let mut jitter_state: u64 = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.subsec_nanos() as u64 ^ d.as_secs())
            .unwrap_or(0x853C_9E59_7A3C_4327);

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
                        match fetcher
                            .handle_streaming(&tx, &stop_signal, &stop_notify_clone)
                            .await
                        {
                            Err(e) => {
                                log::warn!("error fetching streaming: {e}");
                                break;
                            }
                            Ok(StreamOutcome::Stopped) => return,
                            Ok(StreamOutcome::Reconnect { had_docs }) => {
                                if had_docs {
                                    stream_failures = 0;
                                }
                                stream_failures = stream_failures.saturating_add(1);
                                sleep_or_stop(
                                    &stop_signal,
                                    &stop_notify_clone,
                                    stream_backoff(stream_failures, &mut jitter_state),
                                )
                                .await;
                                if stop_signal.load(Ordering::Relaxed) {
                                    return;
                                }
                            }
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

        for (key, value) in self.auth_receiver.borrow().iter() {
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
        let mut request = self.http_client.get(&url).headers(self.build_headers());
        // Streaming mode has no client-wide timeout (it would cut long-lived
        // streams), so bound this short snapshot request on its own. Otherwise
        // the initial fetch blocks forever against a server that never responds.
        // Note: this per-request timeout applies to each retry attempt, so a
        // fully retried fetch can take ~4 x timeout + backoff, unlike the
        // streaming header wait below which bounds all retries together.
        request = request.timeout(self.request_timeout);
        match request.send().await {
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
            "{}/client/v2/environments/{}/namespaces/{}/stream",
            self.base_url, self.environment, self.namespace
        );

        let send = self
            .http_client
            .get(url)
            .headers(self.build_headers())
            .send();
        // send() resolves once response headers arrive, so this bounds only the
        // header wait, never the long-lived body. Unlike the per-attempt timeout
        // in fetch() above, this single timeout covers all retries together.
        // Flipt v2 sends no headers for a namespace without a snapshot; on
        // timeout the caller reconnects.
        let sent = tokio::time::timeout(self.request_timeout, send)
            .await
            .map_err(|_| {
                Error::Server(format!(
                    "no stream response within {:?} for env `{}` ns `{}`",
                    self.request_timeout, self.environment, self.namespace
                ))
            })?;
        match sent {
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
        stop_signal: &Arc<AtomicBool>,
        stop_notify: &Arc<Notify>,
    ) -> Result<StreamOutcome, Error> {
        // Create the Notified future before checking the flag: notify_waiters()
        // stores no permit, so a stop arriving between the check and the
        // select! below would otherwise be lost and close() would block
        // forever while waiting for response headers.
        let stop_notified = stop_notify.notified();
        tokio::pin!(stop_notified);
        stop_notified.as_mut().enable();
        if stop_signal.load(Ordering::Relaxed) {
            return Ok(StreamOutcome::Stopped);
        }

        let stream_result = tokio::select! {
            result = self.fetch_stream() => result,
            _ = &mut stop_notified => return Ok(StreamOutcome::Stopped),
        };

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
                        Ok(result) => Ok(result.result),
                        Err(err) => Err(err),
                    });

                let mut had_docs = false;
                loop {
                    let stop_notified = stop_notify.notified();
                    tokio::pin!(stop_notified);
                    stop_notified.as_mut().enable();
                    if stop_signal.load(Ordering::Relaxed) {
                        return Ok(StreamOutcome::Stopped);
                    }

                    tokio::select! {
                        value = stream.next() => {
                            match value {
                                Some(result) => {
                                    if result.is_ok() {
                                        had_docs = true;
                                    }
                                    match sender.send(result).await {
                                        Ok(_) => continue,
                                        Err(e) => {
                                            return Err(Error::Internal(format!(
                                                "failed to send result to engine {e}"
                                            )))
                                        }
                                    }
                                }
                                // The stream closed cleanly. Report it for backoff:
                                // a server that closes immediately would otherwise
                                // hot-loop. Nothing is sent, so the last snapshot
                                // is kept.
                                None => return Ok(StreamOutcome::Reconnect { had_docs }),
                            }
                        }
                        _ = &mut stop_notified => {
                            return Ok(StreamOutcome::Stopped);
                        }
                    }
                }
            }
            Ok(None) => Ok(StreamOutcome::Reconnect { had_docs: false }),
            Err(e) => {
                sender
                    .send(Err(e.clone()))
                    .await
                    .map_err(|_| Error::Internal("failed to send error".into()))?;
                // A fast terminal error (e.g. 404) is not retried by the retry
                // middleware; the caller backs off before reconnecting.
                Ok(StreamOutcome::Reconnect { had_docs: false })
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use futures::FutureExt;
    use mockito::{Matcher, Server};
    use std::sync::atomic::{AtomicBool, Ordering};
    use std::sync::Arc;

    use reqwest::header::HeaderMap;

    use crate::http::Authentication;
    use crate::http::FetchMode;
    use crate::http::HTTPFetcherBuilder;
    use crate::http::StreamOutcome;
    use tokio::sync::{mpsc, Notify};

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
            .mock("GET", "/client/v2/environments/default/namespaces/default/stream")
            .match_header("x-flipt-environment", "default")
            .with_status(200)
            .with_header(
                "content-type",
                "application/json",
            )
            .with_chunked_body(|w| {
                w.write_all(b"{\"result\":{\"namespace\": {\"key\": \"default\"}, \"flags\":[]}}\n")?;
                w.write_all(
                    b"{\"result\":{\"namespace\": {\"key\": \"default\"}, \"flags\":[{\"key\": \"new_flag\", \"name\": \"new flag\", \"enabled\": false}]}}\n",
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
        let stop_signal = Arc::new(AtomicBool::new(false));
        let stop_notify = Arc::new(Notify::new());

        let result = fetcher
            .handle_streaming(&tx, &stop_signal, &stop_notify)
            .await;
        // Two valid documents went out before the bad chunk and EOF.
        assert!(matches!(
            result,
            Ok(StreamOutcome::Reconnect { had_docs: true })
        ));
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

    #[tokio::test(flavor = "multi_thread")]
    async fn test_streaming_close_does_not_hang() {
        let mut server = Server::new_async().await;

        // Create a streaming mock that never ends (infinite stream)
        // This simulates a persistent HTTP/2 connection
        let mock = server
            .mock(
                "GET",
                "/client/v2/environments/default/namespaces/default/stream",
            )
            .match_header("x-flipt-environment", "default")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_chunked_body(|w| {
                // Send first chunk
                w.write_all(
                    b"{\"result\":{\"namespace\": {\"key\": \"default\"}, \"flags\":[]}}\n",
                )?;
                // Then block indefinitely (simulating persistent connection)
                std::thread::sleep(std::time::Duration::from_secs(1000));
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

        let stop_signal = Arc::new(AtomicBool::new(false));
        let stop_notify = Arc::new(Notify::new());

        // Start the fetcher task
        let _rx = fetcher.start(stop_signal.clone(), stop_notify.clone());

        // Give the task a moment to start and connect to the stream
        tokio::time::sleep(std::time::Duration::from_millis(500)).await;

        // Now trigger the stop signal and notify
        // This should cause the streaming loop to exit immediately
        // without hanging, even though the HTTP stream is still active
        let stop_time = std::time::Instant::now();

        stop_signal.store(true, Ordering::Relaxed);
        stop_notify.notify_waiters();

        // Wait a bit for the task to exit
        // If the fix works, this should complete quickly (< 1 second)
        // Without the fix, it would hang indefinitely waiting for the stream
        tokio::time::sleep(std::time::Duration::from_millis(200)).await;

        let elapsed = stop_time.elapsed();

        // Verify that the stop was processed quickly
        // If the streaming handler properly checks stop_signal/stop_notify,
        // this should be nearly instant
        assert!(
            elapsed < std::time::Duration::from_secs(5),
            "Streaming close took too long: {:?}. This indicates the streaming handler is not checking stop signals.",
            elapsed
        );

        mock.assert();
    }

    /// Accepts connections and never responds; returns its URL.
    async fn unresponsive_server() -> (String, tokio::task::JoinHandle<()>) {
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap());
        let handle = tokio::spawn(async move {
            let mut connections = Vec::new();
            loop {
                let (socket, _) = listener.accept().await.unwrap();
                connections.push(socket);
            }
        });
        (url, handle)
    }

    #[tokio::test]
    async fn test_streaming_stop_while_waiting_for_headers() {
        // A server that accepts the connection but never sends response headers,
        // as Flipt v2 does for a namespace without a snapshot.
        let (url, held) = unresponsive_server().await;

        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .mode(FetchMode::Streaming)
            .build()
            .unwrap();

        let stop_signal = Arc::new(AtomicBool::new(false));
        let stop_notify = Arc::new(Notify::new());
        let mut rx = fetcher.start(stop_signal.clone(), stop_notify.clone());

        tokio::time::sleep(std::time::Duration::from_millis(500)).await;
        stop_signal.store(true, Ordering::Relaxed);
        stop_notify.notify_waiters();

        // The fetcher task drops its sender when it exits.
        let exited = tokio::time::timeout(std::time::Duration::from_secs(5), rx.recv()).await;
        assert!(
            matches!(exited, Ok(None)),
            "streaming fetcher did not stop while waiting for response headers"
        );
        held.abort();
    }

    #[tokio::test]
    async fn test_streaming_initial_fetch_is_bounded() {
        let (url, held) = unresponsive_server().await;
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .mode(FetchMode::Streaming)
            .request_timeout(std::time::Duration::from_secs(1))
            .build()
            .unwrap();

        // Each attempt is bounded by request_timeout; retries add backoff.
        let result =
            tokio::time::timeout(std::time::Duration::from_secs(30), fetcher.initial_fetch()).await;
        assert!(
            matches!(result, Ok(Err(_))),
            "streaming initial fetch did not time out against an unresponsive server"
        );
        held.abort();
    }

    #[tokio::test]
    async fn test_streaming_reconnects_after_header_timeout() {
        use tokio::io::{AsyncReadExt, AsyncWriteExt};

        // First connection: never respond. Later connections: one stream chunk.
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap());
        let server = tokio::spawn(async move {
            // Hold the first connection open without responding so the initial
            // header wait times out; later connections get a snapshot.
            let (_held, _) = listener.accept().await.unwrap();
            loop {
                let (mut socket, _) = listener.accept().await.unwrap();
                let mut request = [0u8; 4096];
                let _ = socket.read(&mut request).await;
                let body = "{\"result\":{\"namespace\":{\"key\":\"default\"},\"flags\":[]}}\n";
                let response = format!(
                    "HTTP/1.1 200 OK\r\ncontent-type: application/json\r\ncontent-length: {}\r\nconnection: close\r\n\r\n{}",
                    body.len(),
                    body
                );
                let _ = socket.write_all(response.as_bytes()).await;
            }
        });

        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .mode(FetchMode::Streaming)
            .request_timeout(std::time::Duration::from_secs(1))
            .build()
            .unwrap();
        let stop_signal = Arc::new(AtomicBool::new(false));
        let stop_notify = Arc::new(Notify::new());
        let mut rx = fetcher.start(stop_signal.clone(), stop_notify.clone());

        let received = tokio::time::timeout(std::time::Duration::from_secs(10), async {
            loop {
                match rx.recv().await {
                    Some(Ok(document)) => return document,
                    Some(Err(_)) => continue, // the header timeout is reported, then retried
                    None => panic!("fetcher exited"),
                }
            }
        })
        .await;
        assert!(
            received.is_ok(),
            "streaming fetcher did not reconnect after a header timeout"
        );

        stop_signal.store(true, Ordering::Relaxed);
        stop_notify.notify_waiters();
        server.abort();
    }

    #[tokio::test]
    async fn test_streaming_fast_errors_back_off() {
        use std::sync::atomic::AtomicUsize;
        use tokio::io::{AsyncReadExt, AsyncWriteExt};

        // Every connection gets an immediate 404, which the retry middleware
        // does not retry. Without reconnect backoff this spins at
        // thousands of reconnects per second.
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", listener.local_addr().unwrap());
        let hits = Arc::new(AtomicUsize::new(0));
        let counted = hits.clone();
        let server = tokio::spawn(async move {
            loop {
                let (mut socket, _) = listener.accept().await.unwrap();
                counted.fetch_add(1, Ordering::Relaxed);
                let mut request = [0u8; 4096];
                let _ = socket.read(&mut request).await;
                let response =
                    "HTTP/1.1 404 Not Found\r\ncontent-length: 0\r\nconnection: close\r\n\r\n";
                let _ = socket.write_all(response.as_bytes()).await;
            }
        });

        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .mode(FetchMode::Streaming)
            .build()
            .unwrap();
        let stop_signal = Arc::new(AtomicBool::new(false));
        let stop_notify = Arc::new(Notify::new());
        let mut rx = fetcher.start(stop_signal.clone(), stop_notify.clone());

        // Errors must still be forwarded while the backoff slows reconnects.
        let mut errors = 0;
        let end = tokio::time::Instant::now() + std::time::Duration::from_secs(3);
        loop {
            let now = tokio::time::Instant::now();
            if now >= end {
                break;
            }
            match tokio::time::timeout(end - now, rx.recv()).await {
                Ok(Some(Err(_))) => errors += 1,
                Ok(Some(Ok(_))) => panic!("unexpected snapshot from 404 server"),
                Ok(None) => panic!("fetcher exited"),
                Err(_) => break,
            }
        }
        assert!(errors >= 1, "stream errors were not forwarded");

        let hits = hits.load(Ordering::Relaxed);
        assert!(
            hits <= 20,
            "streaming reconnects did not back off: {hits} requests in 3s"
        );

        // Close must stay prompt even mid-backoff.
        stop_signal.store(true, Ordering::Relaxed);
        stop_notify.notify_waiters();
        let exited = tokio::time::timeout(std::time::Duration::from_secs(5), rx.recv()).await;
        assert!(
            matches!(exited, Ok(None)),
            "streaming fetcher did not stop promptly during reconnect backoff"
        );
        server.abort();
    }

    #[tokio::test]
    async fn test_auth_update_via_watch_channel() {
        let mut server = Server::new_async().await;

        // First request expects no auth
        let mock_no_auth = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .match_header("authorization", Matcher::Missing)
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .expect(1)
            .create_async()
            .await;

        let url = server.url();
        let mut fetcher = HTTPFetcherBuilder::new(&url)
            .authentication(Authentication::None)
            .build()
            .unwrap();

        let auth_sender = fetcher
            .take_auth_sender()
            .expect("auth sender should be available");

        // Fetch with no auth
        let result = fetcher.fetch().await;
        assert!(result.is_ok());
        mock_no_auth.assert();

        // Now update auth via the watch channel
        let new_auth = HeaderMap::from(Authentication::ClientToken("new-token".to_string()));
        auth_sender.send(new_auth).unwrap();

        // Second request should include the new auth header
        let mock_with_auth = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .match_header("authorization", "Bearer new-token")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(r#"{"namespace": {"key": "default"}, "flags":[]}"#)
            .expect(1)
            .create_async()
            .await;

        let result = fetcher.fetch().await;
        assert!(result.is_ok());
        mock_with_auth.assert();
    }

    // Regression test for #1926: the builder must default to an overall request
    // timeout so a polling client that never sets one still cannot be blocked
    // forever by a stalled response. Previously this defaulted to `None`.
    #[test]
    fn test_full_jitter_stays_within_interval() {
        use std::time::Duration;

        let mut state: u64 = 0x1234_5678_9ABC_DEF0;
        let interval = Duration::from_secs(8);
        let mut seen_spread = false;
        for _ in 0..100 {
            let d = crate::http::full_jitter(interval, &mut state);
            assert!(d <= interval, "jittered delay exceeded interval: {d:?}");
            if d > Duration::ZERO && d < interval {
                seen_spread = true;
            }
        }
        assert!(seen_spread, "jitter never varied within the interval");
    }

    #[test]
    fn test_builder_defaults_request_timeout() {
        use crate::http::DEFAULT_REQUEST_TIMEOUT;

        let builder = HTTPFetcherBuilder::new("http://localhost:8080");
        assert_eq!(builder.request_timeout, DEFAULT_REQUEST_TIMEOUT);
    }

    // End-to-end regression test for #1926: in polling mode a stalled response
    // (headers received but the body never completes) must not block the
    // refresher forever. `connect_timeout` does not cover this, so the request
    // timeout must fire. We set an explicit short timeout to keep it fast.
    #[tokio::test(flavor = "multi_thread")]
    async fn test_polling_request_timeout_unblocks_stalled_response() {
        use std::time::{Duration, Instant};

        let mut server = Server::new_async().await;
        // Send response headers, then hold the body open so the only way out is
        // the client's overall request timeout.
        let _mock = server
            .mock("GET", "/internal/v1/evaluation/snapshot/namespace/default")
            .match_header("x-flipt-environment", "default")
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_chunked_body(|_w| {
                std::thread::sleep(Duration::from_secs(30));
                Ok(())
            })
            .create_async()
            .await;

        let request_timeout = Duration::from_secs(1);
        let mut fetcher = HTTPFetcherBuilder::new(&server.url())
            .authentication(Authentication::None)
            .request_timeout(request_timeout)
            .build()
            .unwrap();

        let start = Instant::now();
        let result = fetcher.initial_fetch().await;
        let elapsed = start.elapsed();

        assert!(
            result.is_err(),
            "stalled response should time out, not hang"
        );
        // Must actually hit the timeout, not fail instantly, and must not hang
        // far beyond the retry/timeout window.
        assert!(
            elapsed >= request_timeout,
            "fetch returned before the request timeout could fire: {elapsed:?}"
        );
        assert!(
            elapsed < Duration::from_secs(20),
            "stalled response did not time out promptly: {elapsed:?}"
        );
    }
}
