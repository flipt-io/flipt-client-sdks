import * as ffi from 'ffi-napi';
import { Pointer, alloc, allocCString } from 'ref-napi';
import * as os from 'os';
import {
  AuthenticationStrategy,
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
  private engine: Pointer<unknown>;

  public constructor(
    namespace?: string,
    engine_opts: EngineOpts<AuthenticationStrategy> = {
      url: 'http://localhost:8080',
      update_interval: 120,
      reference: ''
    }
  ) {
    const buf = Buffer.concat([allocCString(namespace ?? 'default')]);
    const engine = engineLib.initialize_engine(
      alloc('char *', buf as unknown as string),
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

    const response = engineLib.evaluate_variant(
      this.engine,
      allocCString(JSON.stringify(evaluation_request))
    );

    if (response === null) {
      throw new Error('Failed to evaluate variant');
    }

    return JSON.parse(Buffer.from(response).toString('utf-8'));
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

    const response = engineLib.evaluate_boolean(
      this.engine,
      allocCString(JSON.stringify(evaluation_request))
    );

    if (response === null) {
      throw new Error('Failed to evaluate boolean');
    }

    return JSON.parse(Buffer.from(response).toString('utf-8'));
  }

  public close() {
    engineLib.destroy_engine(this.engine);
  }
}
