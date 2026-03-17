/**
 * Pure JavaScript flag evaluator.
 *
 * Ports the Rust evaluation logic from flipt-evaluation/src/lib.rs to TypeScript.
 * Supports boolean, variant, and batch evaluation with constraint matching,
 * segment operators, rollouts, distributions, and CRC32-based bucketing.
 */

import { crc32 } from './js-crc32';
import type { JsStore, EvalConstraint, EvalDistribution } from './js-store';

const DEFAULT_TOTAL_BUCKET_NUMBER = 1000;
const DEFAULT_PERCENT = 100.0;
const DEFAULT_PERCENT_MULTIPLIER = DEFAULT_TOTAL_BUCKET_NUMBER / DEFAULT_PERCENT;

// ── Response types ───────────────────────────────────────────────────────────

export interface JsVariantResponse {
  match: boolean;
  segmentKeys: string[];
  reason: string;
  flagKey: string;
  variantKey: string;
  variantAttachment: string | null;
  requestDurationMillis: number;
  timestamp: string;
}

export interface JsBooleanResponse {
  enabled: boolean;
  flagKey: string;
  reason: string;
  requestDurationMillis: number;
  timestamp: string;
  segmentKeys: string[];
}

export interface JsErrorResponse {
  flagKey: string;
  namespaceKey: string;
  reason: string;
}

export interface JsEvaluationResponse {
  type: string;
  booleanEvaluationResponse?: JsBooleanResponse;
  variantEvaluationResponse?: JsVariantResponse;
  errorEvaluationResponse?: JsErrorResponse;
}

export interface JsBatchResponse {
  responses: JsEvaluationResponse[];
  requestDurationMillis: number;
}

// ── Evaluate functions ───────────────────────────────────────────────────────

export function evaluateBoolean(
  store: JsStore,
  namespace: string,
  flagKey: string,
  entityId: string,
  context: Record<string, string>
): JsBooleanResponse {
  const start = performance.now();
  let lastRank = 0;

  const flag = store.getFlag(namespace, flagKey);
  if (!flag) {
    throw new Error(`failed to get flag information ${namespace}/${flagKey}`);
  }

  if (flag.type !== 'BOOLEAN_FLAG_TYPE') {
    throw new Error(`${flagKey} is not a boolean flag`);
  }

  const rollouts = store.getEvaluationRollouts(namespace, flagKey);
  if (!rollouts) {
    throw new Error(
      `error getting evaluation rollouts for namespace ${namespace} and flag ${flagKey}`
    );
  }

  for (const rollout of rollouts) {
    if (rollout.rank < lastRank) {
      throw new Error(`rollout rank: ${rollout.rank} detected out of order`);
    }
    lastRank = rollout.rank;

    if (rollout.threshold) {
      const normalizedValue =
        crc32(`${entityId}${flagKey}`) % 100;

      if (normalizedValue < rollout.threshold.percentage) {
        return {
          enabled: rollout.threshold.value,
          flagKey: flag.key,
          reason: 'MATCH_EVALUATION_REASON',
          requestDurationMillis: performance.now() - start,
          timestamp: new Date().toISOString(),
          segmentKeys: []
        };
      }
    } else if (rollout.segment) {
      let segmentMatches = 0;
      const segments: string[] = [];

      for (const segData of Object.values(rollout.segment.segments)) {
        if (
          matchesConstraints(
            context,
            segData.constraints,
            segData.matchType,
            entityId
          )
        ) {
          segments.push(segData.segmentKey);
          segmentMatches++;
        }
      }

      if (rollout.segment.segmentOperator === 'OR_SEGMENT_OPERATOR') {
        if (segmentMatches < 1) continue;
      } else if (
        rollout.segment.segmentOperator === 'AND_SEGMENT_OPERATOR' &&
        Object.keys(rollout.segment.segments).length !== segmentMatches
      ) {
        continue;
      }

      return {
        enabled: rollout.segment.value,
        flagKey: flag.key,
        reason: 'MATCH_EVALUATION_REASON',
        requestDurationMillis: performance.now() - start,
        timestamp: new Date().toISOString(),
        segmentKeys: segments
      };
    }
  }

  return {
    enabled: flag.enabled,
    flagKey: flag.key,
    reason: 'DEFAULT_EVALUATION_REASON',
    requestDurationMillis: performance.now() - start,
    timestamp: new Date().toISOString(),
    segmentKeys: []
  };
}

export function evaluateVariant(
  store: JsStore,
  namespace: string,
  flagKey: string,
  entityId: string,
  context: Record<string, string>
): JsVariantResponse {
  const start = performance.now();
  let lastRank = 0;

  const flag = store.getFlag(namespace, flagKey);
  if (!flag) {
    throw new Error(`failed to get flag information ${namespace}/${flagKey}`);
  }

  if (flag.type !== 'VARIANT_FLAG_TYPE') {
    throw new Error(`${flagKey} is not a variant flag`);
  }

  const response: JsVariantResponse = {
    match: false,
    segmentKeys: [],
    reason: '',
    flagKey: flag.key,
    variantKey: '',
    variantAttachment: null,
    requestDurationMillis: 0,
    timestamp: ''
  };

  if (flag.defaultVariant) {
    response.reason = 'DEFAULT_EVALUATION_REASON';
    response.variantKey = flag.defaultVariant.key;
    response.variantAttachment = flag.defaultVariant.attachment ?? null;
  }

  if (!flag.enabled) {
    response.reason = 'FLAG_DISABLED_EVALUATION_REASON';
    response.requestDurationMillis = performance.now() - start;
    response.timestamp = new Date().toISOString();
    return response;
  }

  const rules = store.getEvaluationRules(namespace, flagKey);
  if (!rules) {
    throw new Error(
      `error getting evaluation rules for namespace ${namespace} and flag ${flagKey}`
    );
  }

  if (rules.length === 0) {
    response.requestDurationMillis = performance.now() - start;
    response.timestamp = new Date().toISOString();
    return response;
  }

  for (const rule of rules) {
    if (rule.rank < lastRank) {
      throw new Error(`rule rank: ${rule.rank} detected out of order`);
    }
    lastRank = rule.rank;

    let segmentMatches = 0;
    const segmentKeys: string[] = [];

    for (const [segKey, segment] of Object.entries(rule.segments)) {
      if (
        matchesConstraints(
          context,
          segment.constraints,
          segment.matchType,
          entityId
        )
      ) {
        segmentKeys.push(segKey);
        segmentMatches++;
      }
    }

    if (rule.segmentOperator === 'OR_SEGMENT_OPERATOR') {
      if (segmentMatches < 1) continue;
    } else if (
      rule.segmentOperator === 'AND_SEGMENT_OPERATOR' &&
      Object.keys(rule.segments).length !== segmentMatches
    ) {
      continue;
    }

    response.segmentKeys = segmentKeys;

    const distributions = store.getEvaluationDistributions(
      namespace,
      rule.id
    );
    if (!distributions) {
      throw new Error(
        `error getting evaluation distributions for namespace ${namespace} and rule ${rule.id}`
      );
    }

    const validDists: EvalDistribution[] = [];
    const buckets: number[] = [];
    let cumulativeRollout = 0;

    for (const dist of distributions) {
      if (dist.rollout > 0) {
        validDists.push(dist);
        cumulativeRollout += dist.rollout;
        buckets.push(Math.round(cumulativeRollout * DEFAULT_PERCENT_MULTIPLIER));
      }
    }

    // No distributions → match is true (matched segment/rule)
    if (validDists.length === 0) {
      response.match = true;
      response.reason = 'MATCH_EVALUATION_REASON';
      response.requestDurationMillis = performance.now() - start;
      response.timestamp = new Date().toISOString();
      return response;
    }

    const bucket = crc32(`${flagKey}${entityId}`) % DEFAULT_TOTAL_BUCKET_NUMBER;
    buckets.sort((a, b) => a - b);

    // Binary search equivalent: find first bucket >= (bucket + 1)
    const target = bucket + 1;
    let index = buckets.length;
    let lo = 0;
    let hi = buckets.length;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;
      if (buckets[mid] < target) {
        lo = mid + 1;
      } else {
        hi = mid;
      }
    }
    index = lo;

    // Outside existing buckets → no distribution match
    if (index === validDists.length) {
      response.match = false;
      response.reason = 'DEFAULT_EVALUATION_REASON';
      response.requestDurationMillis = performance.now() - start;
      response.timestamp = new Date().toISOString();
      return response;
    }

    const d = validDists[index];
    response.match = true;
    response.variantKey = d.variantKey;
    response.variantAttachment = d.variantAttachment ?? null;
    response.reason = 'MATCH_EVALUATION_REASON';
    response.requestDurationMillis = performance.now() - start;
    response.timestamp = new Date().toISOString();
    return response;
  }

  response.requestDurationMillis = performance.now() - start;
  response.timestamp = new Date().toISOString();
  return response;
}

export function evaluateBatch(
  store: JsStore,
  namespace: string,
  requests: Array<{
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }>
): JsBatchResponse {
  const start = performance.now();
  const responses: JsEvaluationResponse[] = [];

  for (const req of requests) {
    const flag = store.getFlag(namespace, req.flagKey);
    if (!flag) {
      responses.push({
        type: 'ERROR_EVALUATION_RESPONSE_TYPE',
        errorEvaluationResponse: {
          flagKey: req.flagKey,
          namespaceKey: namespace,
          reason: 'NOT_FOUND_ERROR_EVALUATION_REASON'
        }
      });
      continue;
    }

    if (flag.type === 'BOOLEAN_FLAG_TYPE') {
      const result = evaluateBoolean(
        store,
        namespace,
        req.flagKey,
        req.entityId,
        req.context
      );
      responses.push({
        type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE',
        booleanEvaluationResponse: result
      });
    } else {
      const result = evaluateVariant(
        store,
        namespace,
        req.flagKey,
        req.entityId,
        req.context
      );
      responses.push({
        type: 'VARIANT_EVALUATION_RESPONSE_TYPE',
        variantEvaluationResponse: result
      });
    }
  }

  return {
    responses,
    requestDurationMillis: performance.now() - start
  };
}

// ── Constraint matching ──────────────────────────────────────────────────────

function matchesConstraints(
  evalContext: Record<string, string>,
  constraints: EvalConstraint[],
  segmentMatchType: string,
  entityId: string
): boolean {
  let constraintMatches = 0;

  for (const constraint of constraints) {
    const value =
      constraint.type === 'ENTITY_ID_CONSTRAINT_COMPARISON_TYPE'
        ? entityId
        : evalContext[constraint.property] ?? '';

    let matched: boolean;

    switch (constraint.type) {
      case 'STRING_CONSTRAINT_COMPARISON_TYPE':
      case 'ENTITY_ID_CONSTRAINT_COMPARISON_TYPE':
        matched = matchesString(constraint, value);
        break;
      case 'NUMBER_CONSTRAINT_COMPARISON_TYPE':
        matched = matchesNumber(constraint, value);
        break;
      case 'BOOLEAN_CONSTRAINT_COMPARISON_TYPE':
        matched = matchesBoolean(constraint, value);
        break;
      case 'DATETIME_CONSTRAINT_COMPARISON_TYPE':
        matched = matchesDatetime(constraint, value);
        break;
      default:
        return false;
    }

    if (matched) {
      constraintMatches++;
      if (segmentMatchType === 'ANY_SEGMENT_MATCH_TYPE') break;
    } else if (segmentMatchType === 'ALL_SEGMENT_MATCH_TYPE') {
      break;
    }
  }

  if (segmentMatchType === 'ALL_SEGMENT_MATCH_TYPE') {
    return constraints.length === constraintMatches;
  }
  // ANY
  return constraints.length === 0 || constraintMatches > 0;
}

// ── String matching ──────────────────────────────────────────────────────────

function oneofString(v: string, values: string): boolean {
  try {
    const arr: string[] = JSON.parse(values);
    return Array.isArray(arr) && arr.includes(v);
  } catch {
    return false;
  }
}

function matchesString(constraint: EvalConstraint, v: string): boolean {
  const op = constraint.operator;

  if (op === 'empty') return v === '';
  if (op === 'notempty') return v !== '';
  if (v === '') return false;

  const val = constraint.value;
  switch (op) {
    case 'eq':
      return v === val;
    case 'neq':
      return v !== val;
    case 'prefix':
      return v.startsWith(val);
    case 'suffix':
      return v.endsWith(val);
    case 'isoneof':
      return oneofString(v, val);
    case 'isnotoneof':
      return !oneofString(v, val);
    case 'contains':
      return v.includes(val);
    case 'notcontains':
      return !v.includes(val);
    default:
      return false;
  }
}

// ── Number matching ──────────────────────────────────────────────────────────

function oneofNumber(v: number, values: string): boolean {
  try {
    const arr: number[] = JSON.parse(values);
    return Array.isArray(arr) && arr.includes(v);
  } catch {
    return false;
  }
}

function matchesNumber(constraint: EvalConstraint, v: string): boolean {
  const op = constraint.operator;

  if (op === 'notpresent') return v === '';
  if (op === 'present') return v !== '';
  if (v === '') return false;

  const vNum = parseFloat(v);
  if (isNaN(vNum)) return false;

  if (op === 'isoneof') return oneofNumber(vNum, constraint.value);
  if (op === 'isnotoneof') return !oneofNumber(vNum, constraint.value);

  const cNum = parseFloat(constraint.value);
  if (isNaN(cNum)) return false;

  switch (op) {
    case 'eq':
      return vNum === cNum;
    case 'neq':
      return vNum !== cNum;
    case 'lt':
      return vNum < cNum;
    case 'lte':
      return vNum <= cNum;
    case 'gt':
      return vNum > cNum;
    case 'gte':
      return vNum >= cNum;
    default:
      return false;
  }
}

// ── Boolean matching ─────────────────────────────────────────────────────────

function matchesBoolean(constraint: EvalConstraint, v: string): boolean {
  const op = constraint.operator;

  if (op === 'notpresent') return v === '';
  if (op === 'present') return v !== '';
  if (v === '') return false;

  if (v !== 'true' && v !== 'false') return false;
  const vBool = v === 'true';

  switch (op) {
    case 'true':
      return vBool;
    case 'false':
      return !vBool;
    default:
      return false;
  }
}

// ── DateTime matching ────────────────────────────────────────────────────────

function matchesDatetime(constraint: EvalConstraint, v: string): boolean {
  const op = constraint.operator;

  if (op === 'notpresent') return v === '';
  if (op === 'present') return v !== '';
  if (v === '') return false;

  const d = Date.parse(v);
  if (isNaN(d)) return false;

  const cv = Date.parse(constraint.value);
  if (isNaN(cv)) return false;

  switch (op) {
    case 'eq':
      return d === cv;
    case 'neq':
      return d !== cv;
    case 'lt':
      return d < cv;
    case 'lte':
      return d <= cv;
    case 'gt':
      return d > cv;
    case 'gte':
      return d >= cv;
    default:
      return false;
  }
}
