import data from "./support/state.json" assert { type: "json" };
import { Engine } from "../../pkg/flipt_engine_wasm.js";

describe("Evaluation", () => {
  let engine;

  it("should evaluate a feature", async () => {
    engine = new Engine("default", data);
    const result = engine.evaluate_boolean({
      flag_key: "feature_1",
      entity_id: "user_1",
      context: {},
    });

    expect(result).toEqual({
      name: "feature_1",
      enabled: true,
      reason: "default",
    });
  });
});
