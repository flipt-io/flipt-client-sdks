// cargo run --example evaluation

use fliptengine::{
    evaluator::Evaluator,
    http::{Authentication, HTTPFetcherBuilder},
};
use fliptevaluation::EvaluationRequest;
use std::collections::HashMap;

fn main() {
    let fetcher = HTTPFetcherBuilder::new("http://localhost:8080", "default")
        .authentication(Authentication::with_client_token("secret".into()))
        .build();
    let evaluator = Evaluator::new("default".into()).unwrap();

    let engine = fliptengine::Engine::new(fetcher, evaluator);
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
