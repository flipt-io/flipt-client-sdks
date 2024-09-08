use fliptevaluation::{evaluator, http, store};
pub use fliptevaluation::{
    BatchEvaluationResponse, BooleanEvaluationResponse, EvaluationRequest,
    VariantEvaluationResponse,
};
use std::sync::{Arc, Mutex};
use std::time::Duration;

pub struct FliptClient {
    update_interval: Duration,
    evaluator: Arc<Mutex<evaluator::Evaluator<http::HTTPParser, store::Snapshot>>>,
}

impl FliptClient {
    pub fn new(base_url: &str, namespace: &str, update_interval: Duration) -> Self {
        let evaluator = Arc::new(Mutex::new(
            evaluator::Evaluator::new_snapshot_evaluator(
                namespace,
                http::HTTPParserBuilder::new(base_url).build(),
            )
            .unwrap(),
        ));

        let client = Self {
            update_interval,
            evaluator,
        };

        client.update();
        client
    }

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
        Ok(evaluator.variant(request)?)
    }

    pub async fn boolean_evaluation(
        &self,
        request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Box<dyn std::error::Error>> {
        let evaluator = self.evaluator.lock().unwrap();
        Ok(evaluator.boolean(request)?)
    }

    pub async fn batch_evaluation(
        &self,
        requests: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Box<dyn std::error::Error>> {
        let evaluator = self.evaluator.lock().unwrap();
        Ok(evaluator.batch(requests)?)
    }
}
