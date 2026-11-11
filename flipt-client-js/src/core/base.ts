import {
  BatchEvaluationResponse,
  BooleanEvaluationResponse,
  ErrorEvaluationResponse,
  EvaluationRequest,
  EvaluationResponse,
  Flag,
  IFetcher,
  VariantEvaluationResponse,
  ErrorStrategy,
  Response,
  VariantResult,
  BooleanResult,
  BatchResult,
  ListFlagsResult,
  Hook,
  Logger
} from './types';

import { deserialize, serialize } from './utils';

export type FliptClient = BaseFliptClient;

/** The subset of the EventSource API used by the client (browser and `eventsource` package). */
export interface StreamSource {
  readyState: number;
  onopen: ((event: any) => void) | null;
  onmessage: ((event: any) => void) | null;
  onerror: ((event: any) => void) | null;
  close(): void;
}

const noopLogger: Logger = {
  debug: () => {},
  info: () => {},
  warn: () => {},
  error: () => {}
};

/** Base delay before the stream is restarted after a permanent close. */
const STREAM_RECONNECT_BASE_DELAY_MS = 1_000;

/** Upper bound for the delay between stream restart attempts. */
const STREAM_RECONNECT_MAX_DELAY_MS = 30_000;

export abstract class BaseFliptClient {
  protected engine: any; // Type will be provided by platform implementations
  protected fetcher: IFetcher;
  protected etag?: string;
  protected errorStrategy?: ErrorStrategy;
  protected hook?: Hook;
  protected eventSource?: StreamSource;
  protected logger: Logger = noopLogger;
  private stateUpdateListeners: { fn: () => void }[] = [];
  private closed = false;
  private streamFactory?: () => StreamSource;
  private reconnectTimer?: ReturnType<typeof setTimeout>;
  private reconnectAttempts = 0;

  constructor(engine: any, fetcher: IFetcher) {
    this.engine = engine;
    this.fetcher = fetcher;
  }

  /**
   * Close the client and clean up resources. No state update listener is
   * called after close.
   */
  public close(): void {
    this.closed = true;
    this.stateUpdateListeners = [];
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }
    this.closeEventSource();
  }

  /**
   * Register a listener that is called after the client applies changed flag
   * state, in polling, streaming, or a manual {@link refresh}.
   *
   * Listeners are called only when evaluable state actually changes, and not
   * for the initial state, so read the flags you need once when you register.
   * Failed fetches do not call listeners. Listeners run in the order they
   * were added; an exception thrown by one is logged and does not stop the
   * others.
   *
   * @returns a function that removes the listener
   */
  public addStateUpdateListener(fn: () => void): () => void {
    if (this.closed || typeof fn !== 'function') {
      return () => {};
    }
    const entry = { fn };
    this.stateUpdateListeners.push(entry);
    return () => {
      this.stateUpdateListeners = this.stateUpdateListeners.filter(
        (l) => l !== entry
      );
    };
  }

  private notifyStateUpdate(): void {
    // Copy so listeners can add or remove listeners while being called.
    for (const listener of [...this.stateUpdateListeners]) {
      if (this.closed) {
        return;
      }
      if (!this.stateUpdateListeners.includes(listener)) {
        continue;
      }
      try {
        listener.fn();
      } catch (err) {
        this.logger.error('state update listener failed:', err);
      }
    }
  }

  /**
   * Wire up SSE handlers shared by the browser and Node clients.
   *
   * The factory is called again to open a new stream when the current one is
   * closed permanently (see {@link scheduleReconnect}).
   *
   * @param createSource creates the stream. Called once to open the stream and
   * again for every restart.
   */
  protected attachStream(createSource: () => StreamSource): void {
    this.streamFactory = createSource;
    this.openStream();
  }

  /**
   * Open the stream and attach the SSE handlers to it.
   */
  private openStream(): void {
    if (this.closed || !this.streamFactory) {
      return;
    }

    const eventSource = this.streamFactory();
    this.attachStreamHandlers(eventSource);
    this.eventSource = eventSource;
  }

  /**
   * EventSource reconnects on its own after transient errors; on every
   * (re)open we refresh so updates missed while disconnected are picked up.
   */
  private attachStreamHandlers(eventSource: StreamSource): void {
    eventSource.onopen = () => {
      this.logger.debug('sse connected');
      // The stream is healthy again, so the next failure starts over from the
      // base delay.
      this.reconnectAttempts = 0;
      this.refreshFromStream();
    };

    eventSource.onmessage = (event: { data: string }) => {
      try {
        const data = JSON.parse(event.data);
        this.logger.debug('sse message:', data);
        if (data.type === 'refetchEvaluation') {
          this.refreshFromStream();
        }
      } catch {
        this.logger.warn('sse parse error:', event.data);
      }
    };

    eventSource.onerror = (err: unknown) => {
      if (eventSource.readyState === 2 /* CLOSED */) {
        this.logger.error('sse connection closed:', err);
        if (!this.closed) {
          this.scheduleReconnect();
        }
      } else {
        this.logger.warn('sse error, reconnecting:', err);
      }
    };
  }

  /**
   * Restart the stream after it was closed permanently.
   *
   * An EventSource gives up on connections it cannot recover from by itself
   * (for example, a non-200 response), so the client opens a new one. The
   * attempts repeat with an exponential backoff until the client is closed.
   */
  private scheduleReconnect(): void {
    if (this.reconnectTimer) {
      // A restart is already scheduled.
      return;
    }

    const delay = this.reconnectDelay(this.reconnectAttempts);
    this.reconnectAttempts++;
    this.logger.warn(`sse connection closed, restarting in ${delay}ms`);

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      if (this.closed) {
        return;
      }
      this.closeEventSource();
      this.openStream();
    }, delay);
  }

  /**
   * Delay before the next restart attempt. Half of it is fixed and half is
   * random, so clients that lost the server at the same time do not all
   * reconnect at the same moment.
   *
   * The result is always in `[capped / 2, capped)`. Rounding would let a
   * random value close to 1 round up to exactly `capped`, so the value is
   * truncated instead.
   */
  private reconnectDelay(attempt: number): number {
    const capped = Math.min(
      STREAM_RECONNECT_BASE_DELAY_MS * 2 ** attempt,
      STREAM_RECONNECT_MAX_DELAY_MS
    );

    return Math.floor(capped / 2 + Math.random() * (capped / 2));
  }

  private refreshFromStream(): void {
    this.refresh().catch((err) => {
      this.logger.warn('sse refresh failed:', err);
    });
  }

  protected closeEventSource(): void {
    if (this.eventSource) {
      // Detach the handlers so a discarded source cannot trigger anything.
      this.eventSource.onopen = null;
      this.eventSource.onmessage = null;
      this.eventSource.onerror = null;
      this.eventSource.close();
      this.eventSource = undefined;
    }
  }

  /**
   * Store etag from response for next requests
   */
  protected storeEtag(resp: Response) {
    const etag = resp.headers.get('etag');
    if (etag) {
      this.etag = etag;
    }
  }

  /**
   * Returns a base64-encoded snapshot of the current flag state.
   */
  public getSnapshot(): string {
    return this.engine.get_snapshot();
  }

  /**
   * Refresh the flags snapshot
   * @returns true if the evaluable flag state changed
   */
  public async refresh(): Promise<boolean> {
    try {
      const opts = { etag: this.etag };
      const resp = await this.fetcher(opts);

      const etag = resp.headers.get('etag');
      if (this.etag && this.etag === etag) {
        return false;
      }

      let data: unknown;

      try {
        data = await resp.json();
      } catch (err) {
        // Drop the etag so the next fetch returns a body and recovers.
        this.etag = undefined;
        if (this.errorStrategy === ErrorStrategy.Fail) {
          throw err;
        }
        return false;
      }
      if (this.closed) {
        return false;
      }
      const changed = this.engine.snapshot(data) === true;
      // Store the etag only once its state is applied, or a later 304
      // would keep the client on the old state.
      this.storeEtag(resp);
      if (changed) {
        this.notifyStateUpdate();
      }
      return changed;
    } catch (error) {
      // Drop the etag so the next fetch returns a body and recovers.
      this.etag = undefined;
      if (this.errorStrategy === ErrorStrategy.Fail) {
        throw error;
      }
    }
    return false;
  }

  /**
   * Evaluate a variant flag
   */
  public evaluateVariant({
    flagKey,
    entityId,
    context
  }: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): VariantEvaluationResponse {
    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }
    const evaluationRequest: EvaluationRequest = {
      flagKey,
      entityId,
      context
    };

    this.hook?.before({ flagKey });
    const variantResult: VariantResult | null = this.engine.evaluate_variant(
      serialize(evaluationRequest)
    );

    if (variantResult === null) {
      throw new Error('Failed to evaluate variant');
    }

    if (variantResult.status === 'failure') {
      throw new Error(variantResult.errorMessage);
    }

    if (!variantResult.result) {
      throw new Error('Failed to evaluate variant');
    }

    const result = deserialize<VariantEvaluationResponse>(variantResult.result);
    this.hook?.after({
      flagKey,
      flagType: 'variant',
      value: result.variantKey,
      reason: result.reason,
      segmentKeys: result.segmentKeys
    });
    return result;
  }

  /**
   * Evaluate a boolean flag
   */
  public evaluateBoolean({
    flagKey,
    entityId,
    context
  }: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): BooleanEvaluationResponse {
    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }

    const evaluationRequest: EvaluationRequest = {
      flagKey,
      entityId,
      context
    };

    this.hook?.before({ flagKey });
    const booleanResult: BooleanResult | null = this.engine.evaluate_boolean(
      serialize(evaluationRequest)
    );

    if (booleanResult === null) {
      throw new Error('Failed to evaluate boolean');
    }

    if (booleanResult.status === 'failure') {
      throw new Error(booleanResult.errorMessage);
    }

    if (!booleanResult.result) {
      throw new Error('Failed to evaluate boolean');
    }

    const result = deserialize<BooleanEvaluationResponse>(booleanResult.result);
    this.hook?.after({
      flagKey,
      flagType: 'boolean',
      value: result.enabled.toString(),
      reason: result.reason,
      segmentKeys: result.segmentKeys
    });
    return result;
  }

  /**
   * Evaluate a batch of flag requests
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchEvaluationResponse {
    if (this.hook) {
      requests.forEach((req) => this.hook?.before({ flagKey: req.flagKey }));
    }
    const serializedRequests = requests.map(serialize);
    const batchResult: BatchResult | null =
      this.engine.evaluate_batch(serializedRequests);

    if (batchResult === null) {
      throw new Error('Failed to evaluate batch');
    }

    if (batchResult.status === 'failure') {
      throw new Error(batchResult.errorMessage);
    }

    if (!batchResult.result) {
      throw new Error('Failed to evaluate batch');
    }

    const responses = batchResult.result.responses
      .map((response): EvaluationResponse | undefined => {
        if (response.type === 'BOOLEAN_EVALUATION_RESPONSE_TYPE') {
          const booleanResponse = deserialize<BooleanEvaluationResponse>(
            // @ts-ignore
            response.boolean_evaluation_response
          );
          this.hook?.after({
            flagKey: booleanResponse.flagKey,
            flagType: 'boolean',
            value: booleanResponse.enabled.toString(),
            reason: booleanResponse.reason,
            segmentKeys: booleanResponse.segmentKeys
          });
          return {
            booleanEvaluationResponse: booleanResponse,
            type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'VARIANT_EVALUATION_RESPONSE_TYPE') {
          const variantResponse = deserialize<VariantEvaluationResponse>(
            // @ts-ignore
            response.variant_evaluation_response
          );
          this.hook?.after({
            flagKey: variantResponse.flagKey,
            flagType: 'variant',
            value: variantResponse.variantKey,
            reason: variantResponse.reason,
            segmentKeys: variantResponse.segmentKeys
          });
          return {
            variantEvaluationResponse: variantResponse,
            type: 'VARIANT_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'ERROR_EVALUATION_RESPONSE_TYPE') {
          const errorResponse = deserialize<ErrorEvaluationResponse>(
            // @ts-ignore
            response.error_evaluation_response
          );
          return {
            errorEvaluationResponse: errorResponse,
            type: 'ERROR_EVALUATION_RESPONSE_TYPE'
          };
        }
        return undefined;
      })
      .filter(
        (response): response is EvaluationResponse => response !== undefined
      );

    return {
      responses,
      requestDurationMillis: batchResult.result.requestDurationMillis
    };
  }

  public listFlags(): Flag[] {
    const listFlagsResult: ListFlagsResult | null = this.engine.list_flags();

    if (listFlagsResult === null) {
      throw new Error('Failed to list flags');
    }

    if (listFlagsResult.status === 'failure') {
      throw new Error(listFlagsResult.errorMessage);
    }

    if (!listFlagsResult.result) {
      throw new Error('Failed to list flags');
    }

    return listFlagsResult.result.map(deserialize<Flag>);
  }
}
