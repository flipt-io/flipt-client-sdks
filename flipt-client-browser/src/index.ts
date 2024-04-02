import { Engine } from "flipt-engine-wasm";
import { EngineOpts } from "./models";

interface IEvaluationRequest {
  flag_key: string;
  entity_id: string;
  context: object;
}

export class FliptEvaluationClient {
  private engine: Engine;

  private constructor(engine: Engine) {
    this.engine = engine;
  }

  static async init(
    namespace: string = "default",
    engine_opts: EngineOpts = {
      url: "http://localhost:8080",
      update_interval: 120,
      reference: "",
    }
  ) {
    let url = engine_opts.url ?? "http://localhost:8080";
    url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

    const headers = new Headers();
    headers.append("Accept", "application/json");
    headers.append("Content-Type", "application/json");

    if (engine_opts.authentication) {
      headers.append("Authorization", engine_opts.authentication);
    }

    const resp = await fetch(url, {
      method: "GET",
      headers,
    });

    if (!resp.ok) {
      throw new Error("Failed to fetch data");
    }

    const data = await resp.json();

    return new FliptEvaluationClient(new Engine(namespace, data));
  }

  public evaluateVariant(flag_key: string, entity_id: string, context: {}) {
    const evaluation_request: IEvaluationRequest = {
      flag_key,
      entity_id,
      context,
    };

    return this.engine.evaluate_variant(evaluation_request);
  }

  public evaluateBoolean(flag_key: string, entity_id: string, context: {}) {
    const evaluation_request: IEvaluationRequest = {
      flag_key,
      entity_id,
      context,
    };

    return this.engine.evaluate_boolean(evaluation_request);
  }
}
