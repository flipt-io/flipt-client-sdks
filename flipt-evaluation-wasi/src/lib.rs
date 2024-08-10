use std::cell::RefCell;

use exports::flipt::evaluation::evaluator::{Guest, GuestSnapshot};
use flipt::evaluation::types::{
    BatchEvaluationResponse, BooleanEvaluationResponse, ErrorEvaluationResponse, EvaluationRequest,
    EvaluationResponse, EvaluationResponseType, VariantEvaluationResponse,
};
use fliptevaluation::models::source;

wit_bindgen::generate!({
    world: "flipt:evaluation/host",
});

struct GuestEvaluator;

impl Guest for GuestEvaluator {
    type Snapshot = Snapshot;
}

struct Snapshot {
    namespace: RefCell<String>,
    store: RefCell<fliptevaluation::store::Snapshot>,
}

impl GuestSnapshot for Snapshot {
    fn new(namespace: String, data: String) -> Self {
        let doc: source::Document = serde_json::from_str(&data).unwrap_or_default();
        let store = fliptevaluation::store::Snapshot::build(&namespace, doc)
            .unwrap_or(fliptevaluation::store::Snapshot::empty(&namespace));
        Self {
            namespace: RefCell::new(namespace),
            store: RefCell::new(store),
        }
    }

    fn snapshot(&self, data: String) {
        let namespace = self.namespace.borrow();
        let doc: source::Document = serde_json::from_str(&data).unwrap_or_default();
        let store = fliptevaluation::store::Snapshot::build(&namespace, doc)
            .unwrap_or(fliptevaluation::store::Snapshot::empty(&namespace));
        self.store.replace(store);
    }

    fn evaluate_variant(&self, request: EvaluationRequest) -> Option<VariantEvaluationResponse> {
        let response = fliptevaluation::variant_evaluation(
            &*self.store.borrow(),
            &self.namespace.borrow(),
            &request.into(),
        );
        Some(response.ok()?.into())
    }

    fn evaluate_boolean(&self, request: EvaluationRequest) -> Option<BooleanEvaluationResponse> {
        let response = fliptevaluation::boolean_evaluation(
            &*self.store.borrow(),
            &self.namespace.borrow(),
            &request.into(),
        );
        Some(response.ok()?.into())
    }

    fn evaluate_batch(&self, _requests: Vec<EvaluationRequest>) -> BatchEvaluationResponse {
        todo!()
    }
}

impl From<fliptevaluation::BooleanEvaluationResponse> for BooleanEvaluationResponse {
    fn from(response: fliptevaluation::BooleanEvaluationResponse) -> Self {
        Self {
            enabled: response.enabled,
            flag_key: response.flag_key,
            reason: response.reason.to_string(),
            request_duration_millis: response.request_duration_millis as f32,
            timestamp: response.timestamp.to_string(),
        }
    }
}

impl From<fliptevaluation::VariantEvaluationResponse> for VariantEvaluationResponse {
    fn from(response: fliptevaluation::VariantEvaluationResponse) -> Self {
        Self {
            match_: response.r#match,
            segment_keys: response.segment_keys,
            reason: response.reason.to_string(),
            flag_key: response.flag_key,
            variant_key: response.variant_key,
            variant_attachment: response.variant_attachment,
            request_duration_millis: response.request_duration_millis as f32,
            timestamp: response.timestamp.to_string(),
        }
    }
}

impl From<fliptevaluation::ErrorEvaluationResponse> for ErrorEvaluationResponse {
    fn from(response: fliptevaluation::ErrorEvaluationResponse) -> Self {
        Self {
            flag_key: response.flag_key,
            namespace_key: response.namespace_key,
            reason: response.reason.to_string(),
        }
    }
}

impl From<EvaluationRequest> for fliptevaluation::EvaluationRequest {
    fn from(request: EvaluationRequest) -> Self {
        Self {
            flag_key: request.flag_key,
            entity_id: request.entity_id,
            context: serde_json::from_str(&request.context).unwrap_or_default(),
        }
    }
}

impl From<fliptevaluation::models::common::ResponseType> for EvaluationResponseType {
    fn from(response: fliptevaluation::models::common::ResponseType) -> Self {
        match response {
            fliptevaluation::models::common::ResponseType::Boolean => Self::TypeBoolean,
            fliptevaluation::models::common::ResponseType::Variant => Self::TypeVariant,
            fliptevaluation::models::common::ResponseType::Error => Self::TypeError,
        }
    }
}

impl From<fliptevaluation::EvaluationResponse> for EvaluationResponse {
    fn from(response: fliptevaluation::EvaluationResponse) -> Self {
        Self {
            response_type: response.r#type.into(),
            variant_evaluation: response.variant_evaluation_response.map(|r| r.into()),
            boolean_evaluation: response.boolean_evaluation_response.map(|r| r.into()),
            error_evaluation: response.error_evaluation_response.map(|r| r.into()),
        }
    }
}
