use std::sync::{Arc, RwLock};

use fliptevaluation::{
    batch_evaluation, boolean_evaluation,
    error::Error,
    models::{flipt, snapshot},
    store::Store,
    variant_evaluation, BatchEvaluationResponse, BooleanEvaluationResponse, EvaluationRequest,
    VariantEvaluationResponse,
};

pub struct Evaluator<S>
where
    S: Store + Send,
{
    namespace: String,
    store: S,
    mtx: Arc<RwLock<i32>>,
    error: Option<Error>,
}

impl Evaluator<snapshot::Snapshot> {
    pub fn new(namespace: &str) -> Self {
        let snap = snapshot::Snapshot::empty(namespace);
        Self {
            namespace: namespace.to_string(),
            store: snap,
            mtx: Arc::new(RwLock::new(0)),
            error: None,
        }
    }

    pub fn replace_snapshot(&mut self, res: Result<snapshot::Snapshot, Error>) {
        let _w_lock = self.mtx.write().unwrap();
        match res {
            Ok(snap) => {
                self.store = snap;
                self.error = None;
            }
            Err(err) => {
                self.error = Some(err);
            }
        }
    }

    pub fn get_snapshot(&self) -> Result<snapshot::Snapshot, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if let Some(error) = &self.error {
            return Err(error.clone());
        }
        Ok(self.store.clone())
    }

    pub fn list_flags(&self) -> Result<Vec<flipt::Flag>, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if let Some(error) = &self.error {
            return Err(error.clone());
        }
        match self.store.list_flags(&self.namespace) {
            Some(f) => Ok(f),
            None => Err(Error::Unknown(format!(
                "failed to get flags for {}",
                self.namespace,
            ))),
        }
    }

    pub fn variant(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<VariantEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if let Some(error) = &self.error {
            return Err(error.clone());
        }
        variant_evaluation(&self.store, &self.namespace, evaluation_request)
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if let Some(error) = &self.error {
            return Err(error.clone());
        }
        boolean_evaluation(&self.store, &self.namespace, evaluation_request)
    }

    pub fn batch(
        &self,
        requests: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if let Some(error) = &self.error {
            return Err(error.clone());
        }
        batch_evaluation(&self.store, &self.namespace, requests)
    }
}

#[cfg(test)]
mod tests {
    use std::collections::HashMap;

    use super::*;
    use mockall::predicate::*;

    fn assert_error_response<T>(response: Result<T, Error>, expected_error: &str) {
        match response {
            Err(err) => {
                assert_eq!(expected_error, err.to_string());
            }
            Ok(_) => {
                panic!("expected error but got okay");
            }
        }
    }

    #[test]
    fn test_empty_snapshot() {
        let evaluator = Evaluator::new("namespace");

        let response = evaluator.list_flags();
        match response {
            Ok(v) => assert_eq!(0, v.len()),
            Err(_) => panic!("unexpected error"),
        };
        let requests = vec![EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("user@flipt.io"),
            context: HashMap::new(),
        }];
        let response = evaluator.batch(requests);
        assert!(response.is_ok());
        for resp in response.unwrap().responses {
            assert!(resp.error_evaluation_response.is_some());
        }
        let request = &EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("user@flipt.io"),
            context: HashMap::new(),
        };
        let response = evaluator.boolean(request);
        assert_error_response(
            response,
            "invalid request: failed to get flag information namespace/foo",
        );
        let response = evaluator.variant(request);
        assert_error_response(
            response,
            "invalid request: failed to get flag information namespace/foo",
        );
    }

    #[test]
    fn test_replace_snapshot() {
        let mut evaluator = Evaluator::new("namespace");
        let snapshot = snapshot::Snapshot::empty("namespace");
        evaluator.replace_snapshot(Ok(snapshot.clone()));
        assert_eq!(evaluator.store, snapshot);
    }

    #[test]
    fn test_replace_snapshot_error() {
        let mut evaluator = Evaluator::new("namespace");
        evaluator.replace_snapshot(Err(Error::Unknown("error".to_string())));

        let response = evaluator.list_flags();
        assert_error_response(response, "unknown error: error");

        let request = &EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("user@flipt.io"),
            context: HashMap::new(),
        };
        let response = evaluator.boolean(request);
        assert_error_response(response, "unknown error: error");

        let response = evaluator.variant(request);
        assert_error_response(response, "unknown error: error");

        let requests = vec![EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("user@flipt.io"),
            context: HashMap::new(),
        }];
        let response = evaluator.batch(requests);
        assert_error_response(response, "unknown error: error");
    }

    #[test]
    fn test_get_snapshot() {
        let mut evaluator = Evaluator::new("namespace");
        let snapshot = snapshot::Snapshot::empty("namespace");
        evaluator.replace_snapshot(Ok(snapshot.clone()));
        let result = evaluator.get_snapshot();
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), snapshot);
    }
}
