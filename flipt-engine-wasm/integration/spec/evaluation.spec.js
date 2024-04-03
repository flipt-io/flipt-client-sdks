import data from "./support/state.json" assert { type: "json" };
import { Engine } from "../../pkg/flipt_engine_wasm.js";

describe("Evaluation", () => {
  let engine;

  beforeEach(() => {
    engine = new Engine("default", data);
  });

  it("should evaluate a boolean flag", async () => {
    const response = engine.evaluate_boolean({
      flag_key: "flag_boolean",
      entity_id: "user_1",
      context: {},
    });

    expect(response).not.toBeNull();
    expect(response.status).toEqual("success");
    expect(response.result).not.toBeNull();

    const result = response.result;
    expect(result.enabled).toBe(true);
    expect(result.flag_key).toEqual("flag_boolean");
    expect(result.reason).toEqual("MATCH_EVALUATION_REASON");
  });

  it("should evaluate a multivariate flag", async () => {
    const response = engine.evaluate_variant({
      flag_key: "flag1",
      entity_id: "user_1",
      context: {
        fizz: "buzz",
      },
    });

    expect(response).not.toBeNull();
    expect(response.status).toEqual("success");
    expect(response.result).not.toBeNull();

    const result = response.result;
    expect(result.match).toBe(true);
    expect(result.flag_key).toEqual("flag1");
    expect(result.reason).toEqual("MATCH_EVALUATION_REASON");
    expect(result.variant_key).toEqual("variant1");
    expect(result.segment_keys).toEqual(["segment1"]);
  });
});
