import * as ffi from '@2060.io/ffi-napi';
import { Pointer, allocCString } from '@2060.io/ref-napi';
import * as os from 'os';
import {
  AuthenticationStrategy,
  BooleanResult,
  BatchResult,
  EngineOpts,
  EvaluationRequest,
  VariantResult,
  Flag,
  Result
} from './models';
import * as path from 'path';

let libfile = '';

// get absolute path to libfliptengine
switch (os.platform()) {
  case 'darwin':
    if (os.arch() === 'arm64') {
      libfile = 'darwin_arm64/libfliptengine.dylib';
      break;
    } else if (os.arch() === 'x64') {
      libfile = 'darwin_x86_64/libfliptengine.dylib';
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
  evaluate_variant: ['char **', ['void *', 'string']],
  evaluate_boolean: ['char **', ['void *', 'string']],
  evaluate_batch: ['char **', ['void *', 'string']],
  list_flags: ['char **', ['void *']],
  destroy_engine: ['void', ['void *']],
  destroy_string: ['void', ['char **']]
});

export class FliptEvaluationClient {
  private engine: Pointer<unknown>;

  /**
   * Initializes a new instance of the engine with the specified options.
   *
   * @param namespace - An optional namespace string for the engine instance. If not provided,
   * `default` namespace will be used.
   *
   * @param engine_opts - The optional options for configuring the engine. If not provided,
   *                      default values will be used.
   *
   * @returns Flipt evaluation client
   *
   * @throws Error
   * This exception is thrown if platform is not supported.
   */
  public constructor(
    namespace?: string,
    engine_opts: EngineOpts<AuthenticationStrategy> = {
      url: 'http://localhost:8080',
      update_interval: 120,
      reference: ''
    }
  ) {
    const engine = engineLib.initialize_engine(
      allocCString(namespace ?? 'default'),
      allocCString(JSON.stringify(engine_opts))
    );
    this.engine = engine;
  }

  /**
   * Performs evaluation for a variant flag.
   *
   * @param flag_key - Feature flag key. {@link EvaluationRequest.flag_key}
   * @param entity_id - Entity identifier. {@link EvaluationRequest.entity_id}
   * @param context - Context information for flag evaluation.  {@link EvaluationRequest.context}
   *
   * @returns {@link VariantResult} the result of evaluation
   *
   * @throws {@link Error}
   * This exception is thrown if unexpected error occurs.
   */
  public evaluateVariant(
    flag_key: string,
    entity_id: string,
    context: {}
  ): VariantResult {
    const evaluation_request: EvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    const response = engineLib.evaluate_variant(
      this.engine,
      allocCString(JSON.stringify(evaluation_request))
    );

    if (response === null) {
      throw new Error('Failed to evaluate variant');
    }

    return JSON.parse(this.stringify(response));
  }

  /**
   * Performs evaluation for a boolean flag.
   *
   * @param flag_key - Feature flag key. {@link EvaluationRequest.flag_key}
   * @param entity_id - Entity identifier. {@link EvaluationRequest.entity_id}
   * @param context - Context information for flag evaluation.  {@link EvaluationRequest.context}
   *
   * @returns {@link BooleanResult} the result of evaluation
   *
   * @throws {@link Error}
   * This exception is thrown if unexpected error occurs.
   */
  public evaluateBoolean(
    flag_key: string,
    entity_id: string,
    context: {}
  ): BooleanResult {
    const evaluation_request: EvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    const response = engineLib.evaluate_boolean(
      this.engine,
      allocCString(JSON.stringify(evaluation_request))
    );

    if (response === null) {
      throw new Error('Failed to evaluate boolean');
    }

    return JSON.parse(this.stringify(response));
  }

  /**
   * Performs batch evaluation for bulk of flags.
   *
   * @param requests - array of evaluation requests {@link EvaluationRequest}.
   *
   * @returns {@link BatchResult}
   *
   * @throws {@link Error}
   * This exception is thrown if unexpected error occurs.
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchResult {
    const response = engineLib.evaluate_batch(
      this.engine,
      allocCString(JSON.stringify(requests))
    );

    if (response === null) {
      throw new Error('Failed to evaluate batch');
    }

    return JSON.parse(this.stringify(response));
  }

  /**
   * Retrieves flags.
   *
   * @returns {@link Result} the list of available flags in current namespace
   */
  public listFlags(): Result<Flag[]> {
    const response = engineLib.list_flags(this.engine);
    if (response === null) {
      throw new Error('Failed to list flags');
    }

    return JSON.parse(this.stringify(response));
  }

  /**
   * Frees memory allocated for the Flipt engine.
   *
   * It should be called when client is not in use anymore.
   */
  public close() {
    engineLib.destroy_engine(this.engine);
  }

  /**
   * Get the string from ffi pointer and deallocate the data
   */
  private stringify(response: Pointer<string>): string {
    const value = Buffer.from(response.readCString()).toString('utf-8');
    engineLib.destroy_string(response);
    return value;
  }
}
