import * as ffi from 'ffi-napi';
import { alloc, allocCString, types } from 'ref-napi';
import * as os from 'os';
import {
  BooleanResult,
  EngineOpts,
  EvaluationRequest,
  VariantResult
} from './models';
import * as path from 'path';

let libfile = '';

// get absolute path to libfliptengine
switch (os.platform()) {
  case 'darwin':
    if (os.arch() === 'arm64') {
      libfile = 'darwin_arm64/libfliptengine.dylib';
      break;
    }
    throw new Error('Unsupported platform: ' + os.platform() + '/' + os.arch());
  case 'linux':
    if (os.arch() === 'arm64') {
      libfile = 'linux_arm64/libfliptengine.so';
      break;
    } else if (os.arch() === 'x64') {
      libfile = 'linux_x86_64/libfliptengine.so';
      break;
    }
    throw new Error('Unsupported platform: ' + os.platform() + '/' + os.arch());
  default:
    throw new Error('Unsupported platform: ' + os.platform() + '/' + os.arch());
}

libfile = path.join(__dirname, '..', 'ext', libfile);

const engineLib = ffi.Library(libfile, {
  initialize_engine: ['void *', ['char **', 'string']],
  evaluate_variant: ['string', ['void *', 'string']],
  evaluate_boolean: ['string', ['void *', 'string']],
  destroy_engine: ['void', ['void *']]
});

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

  public close() {
    engineLib.destroy_engine(this.engine);
  }
}
