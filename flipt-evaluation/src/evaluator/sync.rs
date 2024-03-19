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
    pub fn new_snapshot_evaluator(namespace: String, parser: P) -> Result<Self, Error> {
        let snap = Snapshot::build(&namespace, &parser)?;
        Ok(Evaluator::new(namespace, parser, snap))
    }

    pub fn replace_snapshot(&mut self) {
        match Snapshot::build(&self.namespace, &self.parser) {
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
    pub fn new(namespace: String, parser: P, store_impl: S) -> Self {
        Self {
            namespace,
            parser,
            store: store_impl,
            mtx: Arc::new(RwLock::new(0)),
        }
    }

    pub fn replace_store(&mut self, store_impl: S) {
        let _w_lock = self.mtx.write().unwrap();
        self.store = store_impl;
    }

    pub fn list_flags(&self) -> ListFlagsResult {
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
    ) -> VariantEvaluationResult<VariantEvaluationResponse> {
        let _r_lock = self.mtx.read().unwrap();
        let flag = match self
            .store
            .get_flag(&self.namespace, &evaluation_request.flag_key)
        {
            Some(f) => {
                if f.r#type != common::FlagType::Variant {
                    return Err(Error::InvalidRequest(format!(
                        "{} is not a variant flag",
                        &evaluation_request.flag_key,
                    )));
                }
                f
            }
            None => {
                return Err(Error::InvalidRequest(format!(
                    "failed to get flag information {}/{}",
                    &self.namespace, &evaluation_request.flag_key,
                )));
            }
        };

        self.variant_evaluation(&flag, evaluation_request)
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> BooleanEvaluationResult<BooleanEvaluationResponse> {
        let _r_lock = self.mtx.read().unwrap();
        let flag = match self
            .store
            .get_flag(&self.namespace, &evaluation_request.flag_key)
        {
            Some(f) => {
                if f.r#type != common::FlagType::Boolean {
                    return Err(Error::InvalidRequest(format!(
                        "{} is not a boolean flag",
                        &evaluation_request.flag_key,
                    )));
                }
                f
            }
            None => {
                return Err(Error::InvalidRequest(format!(
                    "failed to get flag information {}/{}",
                    &self.namespace, &evaluation_request.flag_key,
                )));
            }
        };

        self.boolean_evaluation(&flag, evaluation_request)
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
                    let boolean_evaluation = self.boolean_evaluation(&flag, &request)?;
                    evaluation_responses.push(EvaluationResponse {
                        r#type: common::ResponseType::Boolean,
                        boolean_evaluation_response: Some(boolean_evaluation),
                        variant_evaluation_response: None,
                        error_evaluation_response: None,
                    });
                }
                common::FlagType::Variant => {
                    let variant_evaluation = self.variant_evaluation(&flag, &request)?;
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
