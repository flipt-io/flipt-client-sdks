use std::sync::{Arc, RwLock};

use fliptevaluation::{
    batch_evaluation, boolean_evaluation,
    error::Error,
    models::{flipt, source::Document},
    parser::Parser,
    store::{Snapshot, Store},
    variant_evaluation, BatchEvaluationResponse, BooleanEvaluationResponse, EvaluationRequest,
    VariantEvaluationResponse,
};

pub struct Evaluator<P, S>
where
    P: Parser + Send,
    S: Store + Send,
{
    namespace: String,
    parser: P,
    store: S,
    mtx: Arc<RwLock<i32>>,
    error: Option<Error>,
}

impl<P> Evaluator<P, Snapshot>
where
    P: Parser + Send,
{
    pub fn new_snapshot_evaluator(namespace: &str, parser: P) -> Result<Self, Error> {
        let snap = Snapshot::build(namespace, Document::default())?;
        let mut e = Evaluator::new(namespace, parser, snap);
        e.replace_snapshot();
        Ok(e)
    }

    pub fn replace_snapshot(&mut self) {
        match self.parser.parse(&self.namespace) {
            Ok(doc) => {
                // if doc is none then return, nothing to do
                if doc.is_none() {
                    return;
                }
                match Snapshot::build(&self.namespace, doc.unwrap()) {
                    Ok(s) => {
                        self.replace_store(s, None);
                    }
                    Err(err) => {
                        // TODO: log::error!("error building snapshot: {}", e);
                        self.replace_store(Snapshot::empty(&self.namespace), Some(err))
                    }
                };
            }
            Err(err) => {
                // TODO: log::error!("error parsing document: {}"", e);
                self.replace_store(Snapshot::empty(&self.namespace), Some(err))
            }
        };
    }
}

impl<P, S> Evaluator<P, S>
where
    P: Parser + Send,
    S: Store + Send,
{
    pub fn new(namespace: &str, parser: P, store: S) -> Self {
        Self {
            namespace: namespace.to_string(),
            parser,
            store,
            mtx: Arc::new(RwLock::new(0)),
            error: None,
        }
    }

    pub fn replace_store(&mut self, store: S, err: Option<Error>) {
        let _w_lock = self.mtx.write().unwrap();
        self.store = store;
        self.error = err;
    }

    pub fn list_flags(&self) -> Result<Vec<flipt::Flag>, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if self.error.is_some() {
            let error = self.error.as_ref().expect("valid error");
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
        if self.error.is_some() {
            let error = self.error.as_ref().expect("valid error");
            return Err(error.clone());
        }
        variant_evaluation(&self.store, &self.namespace, evaluation_request)
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if self.error.is_some() {
            let error = self.error.as_ref().expect("valid error");
            return Err(error.clone());
        }
        boolean_evaluation(&self.store, &self.namespace, evaluation_request)
    }

    pub fn batch(
        &self,
        requests: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if self.error.is_some() {
            let error = self.error.as_ref().expect("valid error");
            return Err(error.clone());
        }
        batch_evaluation(&self.store, &self.namespace, requests)
    }
}
#[cfg(test)]
mod tests {
    use std::collections::HashMap;

    use super::*;
    use fliptevaluation::models::source;
    use fliptevaluation::parser;
    use mockall::predicate::*;
    use mockall::*;

    mock! {
        pub Parser{}
        impl parser::Parser for Parser {
            fn parse(&mut self, namespace: &str) -> Result<Option<source::Document>, Error>;
        }
    }

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
    fn test_parser_with_error() {
        let expected_error = "server error: can't connect";
        let mut parser = MockParser::new();
        parser
            .expect_parse()
            .returning(|_| Err(Error::Server("can't connect".to_owned())));
        let evaluator =
            Evaluator::new_snapshot_evaluator("namespace", parser).expect("expect valid evaluator");

        let response = evaluator.list_flags();
        assert_error_response(response, expected_error);
        let requests = vec![];
        let response = evaluator.batch(requests);
        assert_error_response(response, expected_error);
        let request = &EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("user@flipt.io"),
            context: HashMap::new(),
        };
        let response = evaluator.boolean(request);
        assert_error_response(response, expected_error);
        let response = evaluator.variant(request);
        assert_error_response(response, expected_error);
    }

    #[test]
    fn test_parser_with_empty_snapshot() {
        let mut parser = MockParser::new();
        parser
            .expect_parse()
            .returning(|_| Ok(Some(Document::default())));
        let evaluator =
            Evaluator::new_snapshot_evaluator("namespace", parser).expect("expect valid evaluator");

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
    fn test_list_flags_from_another_namespace() {
        let mut parser = MockParser::new();
        parser
            .expect_parse()
            .returning(|_| Ok(Some(Document::default())));
        let mut evaluator =
            Evaluator::new_snapshot_evaluator("namespace", parser).expect("expect valid evaluator");
        evaluator.replace_store(Snapshot::empty("other"), None);
        let response = evaluator.list_flags();
        match response {
            Err(err) => assert_eq!(
                "unknown error: failed to get flags for namespace",
                err.to_string()
            ),
            Ok(_) => panic!("unexpected error"),
        };
    }
}
