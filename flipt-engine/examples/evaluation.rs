// cargo run --example evaluation

use fliptengine::{self, evaluator, parser};
use std::collections::HashMap;

fn main() {
    let evaluator = evaluator::Evaluator::new_snapshot_evaluator(
        vec!["default".into()],
        parser::HTTPParser::new("http://localhost:8080", Some("secret")),
    );

    let eng = fliptengine::Engine::new(evaluator, Default::default());
    let mut context: HashMap<String, String> = HashMap::new();
    context.insert("fizz".into(), "buzz".into());

    let thread = std::thread::spawn(move || loop {
        std::thread::sleep(std::time::Duration::from_millis(5000));
        let variant = eng.variant(&evaluator::EvaluationRequest {
            namespace_key: "default".into(),
            flag_key: "flag1".into(),
            entity_id: "entity".into(),
            context: context.clone(),
        });

        println!("variant key {}", variant.unwrap().variant_key);
    });

    thread.join().expect("current thread panicked");
}
