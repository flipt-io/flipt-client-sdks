/**
 * Pure JavaScript Evaluation Engine.
 *
 * Evaluates flags locally using a TypeScript port of the Rust evaluation logic.
 * No WebAssembly required — fully CSP-compliant, zero network latency per evaluation.
 *
 * Uses the same snapshot data that all other engines consume (fetched from the
 * Flipt /internal/v1/evaluation/snapshot endpoint).
 */

import { IEvaluationEngine } from './types';
import { JsStore } from './js-store';
import {
  evaluateVariant,
  evaluateBoolean,
  evaluateBatch
} from './js-evaluator';

export class JsLocalEngine implements IEvaluationEngine {
  private store: JsStore;
  private namespace: string;

  constructor(namespace: string) {
    this.namespace = namespace;
    this.store = new JsStore(namespace);
  }

  evaluateVariant(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): any {
    return evaluateVariant(
      this.store,
      this.namespace,
      request.flagKey,
      request.entityId,
      request.context
    );
  }

  evaluateBoolean(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): any {
    return evaluateBoolean(
      this.store,
      this.namespace,
      request.flagKey,
      request.entityId,
      request.context
    );
  }

  evaluateBatch(
    requests: Array<{
      flagKey: string;
      entityId: string;
      context: Record<string, string>;
    }>
  ): any {
    return evaluateBatch(this.store, this.namespace, requests);
  }

  listFlags(): any {
    const flags = this.store.listFlags(this.namespace);
    return flags ?? [];
  }

  snapshot(data: any): void {
    this.store.loadSnapshot(data);
  }
}
