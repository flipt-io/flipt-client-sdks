import * as ffi from 'ffi-napi';
import { alloc, allocCString, types } from 'ref-napi';
import * as os from 'os';
import {
  BooleanResult,
  EngineOpts,
  EvaluationRequest,
  VariantResult
} from './models';

let libfile =
  os.platform() === 'darwin' ? 'libfliptengine.dylib' : 'libfliptengine.so';

const engineLib = ffi.Library(
  process.env.FLIPT_ENGINE_LIB_PATH || `../ext/${libfile}`,
  {
    initialize_engine: ['void *', ['char **', 'string']],
    evaluate_variant: ['string', ['void *', 'string']],
    evaluate_boolean: ['string', ['void *', 'string']],
    destroy_engine: ['void', ['void *']]
  }
);

export class FliptEvaluationClient {
  private namespace: string;
  private engine: types.void;

  public constructor(
    namespace?: string,
    engine_opts: EngineOpts = {
      url: 'http://localhost:8080',
      update_interval: 120,
      auth_token: ''
    }
  ) {
    const engine = engineLib.initialize_engine(
      alloc('char *', Buffer.concat([allocCString(namespace ?? 'default')])),
      allocCString(JSON.stringify(engine_opts))
    );
    this.namespace = namespace ?? 'default';
    this.engine = engine;
  }

  public evaluateVariant(
    flag_key: string,
    entity_id: string,
    context: {}
  ): VariantResult {
    const evaluation_request: EvaluationRequest = {
      namespace_key: this.namespace,
      flag_key: flag_key,
      entity_id: entity_id,
      context
    };

    const response = Buffer.from(
      engineLib.evaluate_variant(
        this.engine,
        allocCString(JSON.stringify(evaluation_request))
      )
    ).toString('utf-8');
    const variantResult: VariantResult = JSON.parse(response);

    return variantResult;
  }

  public evaluateBoolean(
    flag_key: string,
    entity_id: string,
    context: {}
  ): BooleanResult {
    const evaluation_request: EvaluationRequest = {
      namespace_key: this.namespace,
      flag_key: flag_key,
      entity_id: entity_id,
      context
    };

    const response = Buffer.from(
      engineLib.evaluate_boolean(
        this.engine,
        allocCString(JSON.stringify(evaluation_request))
      )
    ).toString('utf-8');
    const booleanResult: BooleanResult = JSON.parse(response);

    return booleanResult;
  }

  public freeEngine() {
    engineLib.destroy_engine(this.engine);
  }
}
