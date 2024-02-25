import * as flipt from "flipt-engine-wasm";

export function fetch_evaluation_data(namespace) {
  return {};
}

function main() {
  let engine = new flipt.Engine("default");

  let evaluationRequest = {
    entity_id: "user123",
    flag_key: "new-header",
  };

  let result = engine.boolean_evaluation(evaluationRequest);

  console.log(result);
}

main();
