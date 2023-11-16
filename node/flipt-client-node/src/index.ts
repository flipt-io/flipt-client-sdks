import * as ffi from 'ffi-napi';
import { alloc, allocCString, types } from 'ref-napi';
import { BooleanResult, EvaluationRequest, VariantResult } from './models';

const engineLib = ffi.Library(
  process.env.ENGINE_LIB_PATH || 'libengine.dylib',
  {
    initialize_engine: ['void *', ['char **']],
    variant: ['string', ['void *', 'string']],
    boolean: ['string', ['void *', 'string']],
    destroy_engine: ['void', ['void *']]
  }
);

export class FliptEvaluationClient {
  private namespace: string;
  private engine: types.void;

  public constructor(namespace?: string) {
    const engine = engineLib.initialize_engine(
      alloc('char *', Buffer.concat([allocCString(namespace ?? 'default')]))
    );
    this.namespace = namespace ?? 'default';
    this.engine = engine;
  }

  public variant(
    flag_key: string,
    entity_id: string,
    context: {}
  ): VariantResult {
    const evaluation_request: EvaluationRequest = {
      namespace_key: this.namespace,
      flag_key: flag_key,
      entity_id: entity_id,
      context: JSON.stringify(context)
    };

    const response = Buffer.from(
      engineLib.variant(
        this.engine,
        allocCString(JSON.stringify(evaluation_request))
      )
    ).toString('utf-8');
    const variantResult: VariantResult = JSON.parse(response);

    return variantResult;
  }

  public boolean(
    flag_key: string,
    entity_id: string,
    context: {}
  ): BooleanResult {
    const evaluation_request: EvaluationRequest = {
      namespace_key: this.namespace,
      flag_key: flag_key,
      entity_id: entity_id,
      context: JSON.stringify(context)
    };

    const response = Buffer.from(
      engineLib.boolean(
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
