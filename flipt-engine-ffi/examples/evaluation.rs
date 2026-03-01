// cargo run --example evaluation

use fliptengine::{
    evaluator::Evaluator,
    http::{Authentication, ErrorStrategy, HTTPFetcherBuilder},
};
use fliptevaluation::{models::snapshot::Snapshot, EvaluationRequest};
use std::collections::HashMap;

fn main() {
    let mut fetcher = HTTPFetcherBuilder::new("http://localhost:8080")
        .authentication(Authentication::with_client_token("secret".into()))
        .build()
        .unwrap();

    let auth_sender = fetcher.take_auth_sender();

    let evaluator = Evaluator::new("default");

    let engine = fliptengine::Engine::new(
        fetcher,
        evaluator,
        ErrorStrategy::Fail,
        Snapshot::default(),
        auth_sender,
    );
    let mut context: HashMap<String, String> = HashMap::new();
    context.insert("fizz".into(), "buzz".into());

    std::thread::sleep(std::time::Duration::from_millis(5000));
    let variant = engine.variant(&EvaluationRequest {
        flag_key: "flag1".into(),
        entity_id: "entity".into(),
        context: context.clone(),
    });

    println!("variant key {:?}", variant.unwrap().variant_key);
}
