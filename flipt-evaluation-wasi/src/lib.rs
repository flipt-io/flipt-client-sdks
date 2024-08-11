use std::cell::RefCell;

use exports::flipt::evaluation::evaluator::{Guest, GuestSnapshot};
use fliptevaluation::models::source;

wit_bindgen::generate!({
    world: "flipt:evaluation/host",
});

struct GuestEvaluator;

impl Guest for GuestEvaluator {
    type Snapshot = Snapshot;
}

export!(GuestEvaluator);

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

    fn evaluate_variant(&self, request: String) -> Option<String> {
        let request: fliptevaluation::EvaluationRequest = serde_json::from_str(&request).unwrap();
        let response = fliptevaluation::variant_evaluation(
            &*self.store.borrow(),
            &self.namespace.borrow(),
            &request,
        );
        match response {
            Ok(r) => Some(serde_json::to_string(&r).unwrap()),
            Err(_e) => None,
        }
    }

    fn evaluate_boolean(&self, request: String) -> Option<String> {
        let request: fliptevaluation::EvaluationRequest = serde_json::from_str(&request).unwrap();
        let response = fliptevaluation::boolean_evaluation(
            &*self.store.borrow(),
            &self.namespace.borrow(),
            &request,
        );
        match response {
            Ok(r) => Some(serde_json::to_string(&r).unwrap()),
            Err(_e) => None,
        }
    }

    fn evaluate_batch(&self, _requests: String) -> Option<String> {
        todo!()
    }
}
