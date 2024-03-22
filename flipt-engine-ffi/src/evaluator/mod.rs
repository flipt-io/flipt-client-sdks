use std::{
    sync::{Arc, RwLock},
    time::SystemTime,
};

use fliptevaluation::{
    boolean_evaluation,
    error::Error,
    get_duration_millis,
    models::{common, flipt, source::Document},
    parser::Parser,
    store::{Snapshot, Store},
    variant_evaluation, BatchEvaluationResponse, BooleanEvaluationResponse,
    ErrorEvaluationResponse, EvaluationRequest, EvaluationResponse, VariantEvaluationResponse,
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
}

impl<P> Evaluator<P, Snapshot>
where
    P: Parser + Send,
{
    pub fn new_snapshot_evaluator(namespace: &str, parser: P) -> Result<Self, Error> {
        let doc = parser.parse(namespace)?;
        let snap = Snapshot::build(namespace, doc)?;
        Ok(Evaluator::new(namespace, parser, snap))
    }

    pub fn replace_snapshot(&mut self) {
        let doc = match self.parser.parse(&self.namespace) {
            Ok(d) => d,
            Err(_) => {
                // TODO: log::error!("error parsing document: {}"", e);
                Document::default()
            }
        };

        match Snapshot::build(&self.namespace, doc) {
            Ok(s) => {
                self.replace_store(s);
            }
            Err(_) => {
                // TODO: log::error!("error building snapshot: {}", e);
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
        }
    }

    pub fn replace_store(&mut self, store_impl: S) {
        let _w_lock = self.mtx.write().unwrap();
        self.store = store_impl;
    }

    pub fn list_flags(&self) -> Result<Vec<flipt::Flag>, Error> {
        let _r_lock = self.mtx.read().unwrap();
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
        variant_evaluation(&self.namespace, evaluation_request, &self.store)
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        boolean_evaluation(&self.namespace, evaluation_request, &self.store)
    }

    pub fn batch(
        &self,
        requests: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        let now = SystemTime::now();

        let mut evaluation_responses: Vec<EvaluationResponse> = Vec::new();
        for request in requests {
            let flag = match self.store.get_flag(&self.namespace, &request.flag_key) {
                Some(f) => f,
                None => {
                    evaluation_responses.push(EvaluationResponse {
                        r#type: common::ResponseType::Error,
                        boolean_evaluation_response: None,
                        variant_evaluation_response: None,
                        error_evaluation_response: Some(ErrorEvaluationResponse {
                            flag_key: request.flag_key,
                            namespace_key: self.namespace.clone(),
                            reason: common::ErrorEvaluationReason::NotFound,
                        }),
                    });
                    continue;
                }
            };

            match flag.r#type {
                common::FlagType::Boolean => {
                    let boolean_evaluation =
                        boolean_evaluation(&self.namespace, &request, &self.store)?;
                    evaluation_responses.push(EvaluationResponse {
                        r#type: common::ResponseType::Boolean,
                        boolean_evaluation_response: Some(boolean_evaluation),
                        variant_evaluation_response: None,
                        error_evaluation_response: None,
                    });
                }
                common::FlagType::Variant => {
                    let variant_evaluation =
                        variant_evaluation(&self.namespace, &request, &self.store)?;
                    evaluation_responses.push(EvaluationResponse {
                        r#type: common::ResponseType::Variant,
                        boolean_evaluation_response: None,
                        variant_evaluation_response: Some(variant_evaluation),
                        error_evaluation_response: None,
                    });
                }
            }
        }

        Ok(BatchEvaluationResponse {
            responses: evaluation_responses,
            request_duration_millis: get_duration_millis(now.elapsed())?,
        })
    }
}
