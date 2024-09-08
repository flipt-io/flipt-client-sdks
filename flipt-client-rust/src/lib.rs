use chrono::{DateTime, Utc};
pub use fliptevaluation::http::Authentication;
pub use fliptevaluation::models::flipt::EvaluationReason;
pub use fliptevaluation::EvaluationRequest;

use fliptevaluation::BatchEvaluationResponse;
use fliptevaluation::{evaluator, http, store};
use reqwest::header::HeaderMap;
use std::sync::{Arc, Mutex};
use std::time::Duration;

pub struct BooleanEvaluationResponse {
    pub enabled: bool,
    pub flag_key: String,
    pub reason: EvaluationReason,
    pub request_duration: Duration,
    pub timestamp: DateTime<Utc>,
}

impl From<fliptevaluation::BooleanEvaluationResponse> for BooleanEvaluationResponse {
    fn from(response: fliptevaluation::BooleanEvaluationResponse) -> Self {
        Self {
            enabled: response.enabled,
            flag_key: response.flag_key,
            reason: response.reason,
            request_duration: Duration::from_millis(response.request_duration_millis as u64),
            timestamp: response.timestamp,
        }
    }
}

pub struct VariantEvaluationResponse {
    pub r#match: bool,
    pub flag_key: String,
    pub reason: EvaluationReason,
    pub variant_key: Option<String>,
    pub variant_attachment: Option<String>,
    pub request_duration: Duration,
    pub timestamp: DateTime<Utc>,
}

impl From<fliptevaluation::VariantEvaluationResponse> for VariantEvaluationResponse {
    fn from(response: fliptevaluation::VariantEvaluationResponse) -> Self {
        Self {
            r#match: response.r#match,
            flag_key: response.flag_key,
            reason: response.reason,
            variant_key: match response.variant_key.is_empty() {
                true => None,
                false => Some(response.variant_key),
            },
            variant_attachment: match response.variant_attachment.is_empty() {
                true => None,
                false => Some(response.variant_attachment),
            },
            request_duration: Duration::from_millis(response.request_duration_millis as u64),
            timestamp: response.timestamp,
        }
    }
}

pub struct FliptEvaluationClientBuilder {
    base_url: String,
    namespace: String,
    update_interval: Duration,
    authentication: HeaderMap,
}

impl FliptEvaluationClientBuilder {
    pub fn new(base_url: &str) -> Self {
        Self {
            base_url: base_url.to_string(),
            namespace: "default".to_string(),
            update_interval: Duration::from_secs(120),
            authentication: HeaderMap::new(),
        }
    }

    pub fn namespace(mut self, namespace: &str) -> Self {
        self.namespace = namespace.to_string();
        self
    }

    pub fn authentication(mut self, authentication: Authentication) -> Self {
        self.authentication = HeaderMap::from(authentication);
        self
    }

    pub fn update_interval(mut self, update_interval: Duration) -> Self {
        self.update_interval = update_interval;
        self
    }

    pub fn build(self) -> Result<FliptEvaluationClient, Box<dyn std::error::Error>> {
        let evaluator = Arc::new(Mutex::new(
            evaluator::Evaluator::new_snapshot_evaluator(
                &self.namespace,
                http::HTTPParserBuilder::new(&self.base_url).build(),
            )
            .map_err(|e| e.to_string())?,
        ));

        let client = FliptEvaluationClient {
            update_interval: self.update_interval,
            evaluator,
        };

        client.update();
        Ok(client)
    }
}

pub struct FliptEvaluationClient {
    update_interval: Duration,
    evaluator: Arc<Mutex<evaluator::Evaluator<http::HTTPParser, store::Snapshot>>>,
}

impl FliptEvaluationClient {
    fn update(&self) {
        let evaluator = self.evaluator.clone();
        let update_interval = self.update_interval;
        std::thread::spawn(move || loop {
            std::thread::sleep(update_interval);
            let mut lock = evaluator.lock().unwrap();
            lock.replace_snapshot();
        });
    }

    pub async fn variant_evaluation(
        &self,
        request: &EvaluationRequest,
    ) -> Result<VariantEvaluationResponse, Box<dyn std::error::Error>> {
        let evaluator = self.evaluator.lock().unwrap();
        Ok(VariantEvaluationResponse::from(evaluator.variant(request)?))
    }

    pub async fn boolean_evaluation(
        &self,
        request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Box<dyn std::error::Error>> {
        let evaluator = self.evaluator.lock().unwrap();
        Ok(BooleanEvaluationResponse::from(evaluator.boolean(request)?))
    }

    pub async fn batch_evaluation(
        &self,
        requests: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Box<dyn std::error::Error>> {
        let evaluator = self.evaluator.lock().unwrap();
        Ok(evaluator.batch(requests)?)
    }
}
