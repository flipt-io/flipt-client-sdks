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
                match Snapshot::build(&self.namespace, doc) {
                    Ok(s) => {
                        self.replace_store(s, None);
                    }
                    Err(err) => {
                        // TODO: log::error!("error building snapshot: {}", e);
                        self.replace_store(Snapshot::blank(&self.namespace), Some(err))
                    }
                };
            }
            Err(err) => {
                // TODO: log::error!("error parsing document: {}"", e);
                self.replace_store(Snapshot::blank(&self.namespace), Some(err))
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
            return Err(self.error.expect("valid error"));
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
            return Err(self.error.expect("valid error"));
        }
        variant_evaluation(&self.store, &self.namespace, evaluation_request)
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> Result<BooleanEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if self.error.is_some() {
            return Err(self.error.expect("valid error"));
        }
        boolean_evaluation(&self.store, &self.namespace, evaluation_request)
    }

    pub fn batch(
        &self,
        requests: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        let _r_lock = self.mtx.read().unwrap();
        if self.error.is_some() {
            return Err(self.error.expect("valid error"));
        }
        batch_evaluation(&self.store, &self.namespace, requests)
    }
}
