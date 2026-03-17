// @ts-nocheck
import { describe, it, expect, beforeEach } from '@jest/globals';
import { crc32 } from '../src/core/engines/js-crc32';
import { JsStore } from '../src/core/engines/js-store';
import {
  evaluateBoolean,
  evaluateVariant,
  evaluateBatch
} from '../src/core/engines/js-evaluator';
import { JsLocalEngine } from '../src/core/engines/js-local-engine';

// ── Fixture: same structure as flipt-evaluation/src/testdata/state.json ──────

const STATE_FIXTURE = {
  namespace: { key: 'default' },
  flags: [
    {
      key: 'flag1',
      name: 'flag1',
      enabled: true,
      type: 'VARIANT_FLAG_TYPE',
      rules: [
        {
          segments: [
            {
              key: 'segment1',
              matchType: 'ANY_SEGMENT_MATCH_TYPE',
              constraints: [
                {
                  type: 'STRING_CONSTRAINT_COMPARISON_TYPE',
                  property: 'fizz',
                  operator: 'eq',
                  value: 'buzz'
                }
              ]
            }
          ],
          segmentOperator: 'OR_SEGMENT_OPERATOR',
          distributions: [
            {
              variantKey: 'variant1',
              variantAttachment: '',
              rollout: 100
            }
          ]
        }
      ],
      rollouts: []
    },
    {
      key: 'flag_boolean',
      name: 'flag_boolean',
      enabled: true,
      type: 'BOOLEAN_FLAG_TYPE',
      rules: [],
      rollouts: [
        {
          type: 'SEGMENT_ROLLOUT_TYPE',
          segment: {
            value: true,
            segmentOperator: 'OR_SEGMENT_OPERATOR',
            segments: [
              {
                key: 'segment1',
                matchType: 'ANY_SEGMENT_MATCH_TYPE',
                constraints: [
                  {
                    type: 'STRING_CONSTRAINT_COMPARISON_TYPE',
                    property: 'fizz',
                    operator: 'eq',
                    value: 'buzz'
                  }
                ]
              }
            ]
          }
        },
        {
          type: 'THRESHOLD_ROLLOUT_TYPE',
          threshold: {
            percentage: 50,
            value: true
          }
        }
      ]
    },
    {
      key: 'flag_disabled',
      name: 'flag_disabled',
      enabled: false,
      type: 'VARIANT_FLAG_TYPE',
      rules: [],
      rollouts: []
    },
    {
      key: 'flag_no_rules',
      name: 'flag_no_rules',
      enabled: true,
      type: 'VARIANT_FLAG_TYPE',
      rules: [],
      rollouts: []
    },
    {
      key: 'flag_default_variant',
      name: 'flag_default_variant',
      enabled: true,
      type: 'VARIANT_FLAG_TYPE',
      defaultVariant: { id: 'dv-1', key: 'default_var', attachment: '{"x":1}' },
      rules: [],
      rollouts: []
    },
    {
      key: 'flag_multi_dist',
      name: 'flag_multi_dist',
      enabled: true,
      type: 'VARIANT_FLAG_TYPE',
      rules: [
        {
          segments: [
            {
              key: 'everyone',
              matchType: 'ANY_SEGMENT_MATCH_TYPE',
              constraints: []
            }
          ],
          segmentOperator: 'OR_SEGMENT_OPERATOR',
          distributions: [
            { variantKey: 'control', rollout: 50 },
            { variantKey: 'treatment', rollout: 50 }
          ]
        }
      ],
      rollouts: []
    },
    {
      key: 'flag_bool_default',
      name: 'flag_bool_default',
      enabled: false,
      type: 'BOOLEAN_FLAG_TYPE',
      rules: [],
      rollouts: []
    },
    {
      key: 'flag_and_segments',
      name: 'flag_and_segments',
      enabled: true,
      type: 'VARIANT_FLAG_TYPE',
      rules: [
        {
          segments: [
            {
              key: 'seg_a',
              matchType: 'ALL_SEGMENT_MATCH_TYPE',
              constraints: [
                {
                  type: 'STRING_CONSTRAINT_COMPARISON_TYPE',
                  property: 'plan',
                  operator: 'eq',
                  value: 'premium'
                }
              ]
            },
            {
              key: 'seg_b',
              matchType: 'ALL_SEGMENT_MATCH_TYPE',
              constraints: [
                {
                  type: 'STRING_CONSTRAINT_COMPARISON_TYPE',
                  property: 'region',
                  operator: 'eq',
                  value: 'us'
                }
              ]
            }
          ],
          segmentOperator: 'AND_SEGMENT_OPERATOR',
          distributions: [
            { variantKey: 'premium_us', rollout: 100 }
          ]
        }
      ],
      rollouts: []
    }
  ]
};

// ── CRC32 ────────────────────────────────────────────────────────────────────

describe('crc32', () => {
  it('returns a 32-bit unsigned integer', () => {
    const result = crc32('hello');
    expect(result).toBeGreaterThanOrEqual(0);
    expect(result).toBeLessThanOrEqual(0xffffffff);
  });

  it('returns known CRC32 IEEE values', () => {
    // Known CRC32 values (IEEE polynomial)
    expect(crc32('')).toBe(0);
    expect(crc32('123456789')).toBe(0xcbf43926);
  });

  it('is deterministic', () => {
    expect(crc32('testkey')).toBe(crc32('testkey'));
  });

  it('produces different values for different inputs', () => {
    expect(crc32('abc')).not.toBe(crc32('xyz'));
  });
});

// ── JsStore ──────────────────────────────────────────────────────────────────

describe('JsStore', () => {
  let store: JsStore;

  beforeEach(() => {
    store = new JsStore('default');
    store.loadSnapshot(STATE_FIXTURE);
  });

  it('getFlag returns existing flags', () => {
    const flag = store.getFlag('default', 'flag1');
    expect(flag).toBeDefined();
    expect(flag!.key).toBe('flag1');
    expect(flag!.type).toBe('VARIANT_FLAG_TYPE');
    expect(flag!.enabled).toBe(true);
  });

  it('getFlag returns undefined for wrong namespace', () => {
    expect(store.getFlag('other', 'flag1')).toBeUndefined();
  });

  it('getFlag returns undefined for non-existent flag', () => {
    expect(store.getFlag('default', 'nope')).toBeUndefined();
  });

  it('listFlags returns all flags', () => {
    const flags = store.listFlags('default');
    expect(flags).toBeDefined();
    expect(flags!.length).toBe(STATE_FIXTURE.flags.length);
  });

  it('listFlags returns undefined for wrong namespace', () => {
    expect(store.listFlags('other')).toBeUndefined();
  });

  it('getEvaluationRules returns rules for variant flag', () => {
    const rules = store.getEvaluationRules('default', 'flag1');
    expect(rules).toBeDefined();
    expect(rules!.length).toBe(1);
    expect(rules![0].rank).toBe(1);
    expect(rules![0].segmentOperator).toBe('OR_SEGMENT_OPERATOR');
    expect(Object.keys(rules![0].segments)).toContain('segment1');
  });

  it('getEvaluationDistributions returns distributions', () => {
    const rules = store.getEvaluationRules('default', 'flag1');
    const dists = store.getEvaluationDistributions('default', rules![0].id);
    expect(dists).toBeDefined();
    expect(dists!.length).toBe(1);
    expect(dists![0].variantKey).toBe('variant1');
    expect(dists![0].rollout).toBe(100);
  });

  it('getEvaluationRollouts returns rollouts for boolean flag', () => {
    const rollouts = store.getEvaluationRollouts('default', 'flag_boolean');
    expect(rollouts).toBeDefined();
    expect(rollouts!.length).toBe(2);
    expect(rollouts![0].rolloutType).toBe('SEGMENT');
    expect(rollouts![1].rolloutType).toBe('THRESHOLD');
    expect(rollouts![1].threshold!.percentage).toBe(50);
  });

  it('loadSnapshot clears previous state', () => {
    store.loadSnapshot({ namespace: { key: 'default' }, flags: [] });
    expect(store.listFlags('default')!.length).toBe(0);
    expect(store.getFlag('default', 'flag1')).toBeUndefined();
  });

  it('handles defaultVariant on flags', () => {
    const flag = store.getFlag('default', 'flag_default_variant');
    expect(flag!.defaultVariant).toBeDefined();
    expect(flag!.defaultVariant!.key).toBe('default_var');
    expect(flag!.defaultVariant!.attachment).toBe('{"x":1}');
  });
});

// ── Boolean evaluation ───────────────────────────────────────────────────────

describe('evaluateBoolean', () => {
  let store: JsStore;

  beforeEach(() => {
    store = new JsStore('default');
    store.loadSnapshot(STATE_FIXTURE);
  });

  it('matches segment rollout when context matches', () => {
    const result = evaluateBoolean(
      store, 'default', 'flag_boolean', 'user1', { fizz: 'buzz' }
    );
    expect(result.enabled).toBe(true);
    expect(result.flagKey).toBe('flag_boolean');
    expect(result.reason).toBe('MATCH_EVALUATION_REASON');
    expect(result.segmentKeys).toContain('segment1');
  });

  it('falls through to threshold rollout when segment does not match', () => {
    // CRC32 of "someuser1flag_boolean" % 100 determines threshold match
    const result = evaluateBoolean(
      store, 'default', 'flag_boolean', 'someuser1', { fizz: 'nope' }
    );
    // Either matched threshold (reason=MATCH) or default
    expect(['MATCH_EVALUATION_REASON', 'DEFAULT_EVALUATION_REASON']).toContain(
      result.reason
    );
    expect(result.flagKey).toBe('flag_boolean');
  });

  it('returns default when no rollouts match', () => {
    const result = evaluateBoolean(
      store, 'default', 'flag_bool_default', 'user1', {}
    );
    expect(result.enabled).toBe(false);
    expect(result.reason).toBe('DEFAULT_EVALUATION_REASON');
  });

  it('throws for non-boolean flag', () => {
    expect(() =>
      evaluateBoolean(store, 'default', 'flag1', 'user1', {})
    ).toThrow('is not a boolean flag');
  });

  it('throws for non-existent flag', () => {
    expect(() =>
      evaluateBoolean(store, 'default', 'nope', 'user1', {})
    ).toThrow('failed to get flag information');
  });

  it('includes timing fields', () => {
    const result = evaluateBoolean(
      store, 'default', 'flag_boolean', 'user1', { fizz: 'buzz' }
    );
    expect(typeof result.requestDurationMillis).toBe('number');
    expect(result.requestDurationMillis).toBeGreaterThanOrEqual(0);
    expect(result.timestamp).toBeTruthy();
  });
});

// ── Variant evaluation ───────────────────────────────────────────────────────

describe('evaluateVariant', () => {
  let store: JsStore;

  beforeEach(() => {
    store = new JsStore('default');
    store.loadSnapshot(STATE_FIXTURE);
  });

  it('matches variant with 100% distribution', () => {
    const result = evaluateVariant(
      store, 'default', 'flag1', 'user1', { fizz: 'buzz' }
    );
    expect(result.match).toBe(true);
    expect(result.variantKey).toBe('variant1');
    expect(result.reason).toBe('MATCH_EVALUATION_REASON');
    expect(result.segmentKeys).toContain('segment1');
    expect(result.flagKey).toBe('flag1');
  });

  it('does not match when context does not satisfy constraints', () => {
    const result = evaluateVariant(
      store, 'default', 'flag1', 'user1', { fizz: 'wrong' }
    );
    expect(result.match).toBe(false);
  });

  it('returns FLAG_DISABLED reason for disabled flag', () => {
    const result = evaluateVariant(
      store, 'default', 'flag_disabled', 'user1', {}
    );
    expect(result.match).toBe(false);
    expect(result.reason).toBe('FLAG_DISABLED_EVALUATION_REASON');
  });

  it('returns empty result for flag with no rules', () => {
    const result = evaluateVariant(
      store, 'default', 'flag_no_rules', 'user1', {}
    );
    expect(result.match).toBe(false);
  });

  it('returns default variant when present and no rules', () => {
    const result = evaluateVariant(
      store, 'default', 'flag_default_variant', 'user1', {}
    );
    expect(result.match).toBe(false);
    expect(result.variantKey).toBe('default_var');
    expect(result.variantAttachment).toBe('{"x":1}');
    expect(result.reason).toBe('DEFAULT_EVALUATION_REASON');
  });

  it('throws for non-variant flag', () => {
    expect(() =>
      evaluateVariant(store, 'default', 'flag_boolean', 'user1', {})
    ).toThrow('is not a variant flag');
  });

  it('throws for non-existent flag', () => {
    expect(() =>
      evaluateVariant(store, 'default', 'nope', 'user1', {})
    ).toThrow('failed to get flag information');
  });

  it('multi-distribution: deterministic bucketing always returns same variant', () => {
    const r1 = evaluateVariant(
      store, 'default', 'flag_multi_dist', 'entity-abc', {}
    );
    const r2 = evaluateVariant(
      store, 'default', 'flag_multi_dist', 'entity-abc', {}
    );
    expect(r1.match).toBe(true);
    expect(r1.variantKey).toBe(r2.variantKey);
    expect(['control', 'treatment']).toContain(r1.variantKey);
  });

  it('multi-distribution: different entities may get different variants', () => {
    const results = new Set<string>();
    // Try many entity IDs to get coverage of both variants
    for (let i = 0; i < 100; i++) {
      const r = evaluateVariant(
        store, 'default', 'flag_multi_dist', `entity-${i}`, {}
      );
      results.add(r.variantKey);
    }
    // With 50/50 split and 100 entities, both should appear
    expect(results.size).toBe(2);
    expect(results.has('control')).toBe(true);
    expect(results.has('treatment')).toBe(true);
  });

  it('AND segment operator requires all segments to match', () => {
    const result = evaluateVariant(
      store, 'default', 'flag_and_segments', 'user1',
      { plan: 'premium', region: 'us' }
    );
    expect(result.match).toBe(true);
    expect(result.variantKey).toBe('premium_us');
    expect(result.segmentKeys).toContain('seg_a');
    expect(result.segmentKeys).toContain('seg_b');
  });

  it('AND segment operator fails when one segment does not match', () => {
    const result = evaluateVariant(
      store, 'default', 'flag_and_segments', 'user1',
      { plan: 'premium', region: 'eu' }
    );
    expect(result.match).toBe(false);
  });

  it('includes timing fields', () => {
    const result = evaluateVariant(
      store, 'default', 'flag1', 'user1', { fizz: 'buzz' }
    );
    expect(typeof result.requestDurationMillis).toBe('number');
    expect(result.timestamp).toBeTruthy();
  });
});

// ── Batch evaluation ─────────────────────────────────────────────────────────

describe('evaluateBatch', () => {
  let store: JsStore;

  beforeEach(() => {
    store = new JsStore('default');
    store.loadSnapshot(STATE_FIXTURE);
  });

  it('evaluates mixed flag types in a single batch', () => {
    const result = evaluateBatch(store, 'default', [
      { flagKey: 'flag1', entityId: 'user1', context: { fizz: 'buzz' } },
      { flagKey: 'flag_boolean', entityId: 'user1', context: { fizz: 'buzz' } }
    ]);
    expect(result.responses.length).toBe(2);
    expect(result.responses[0].type).toBe('VARIANT_EVALUATION_RESPONSE_TYPE');
    expect(result.responses[0].variantEvaluationResponse!.match).toBe(true);
    expect(result.responses[1].type).toBe('BOOLEAN_EVALUATION_RESPONSE_TYPE');
    expect(result.responses[1].booleanEvaluationResponse!.enabled).toBe(true);
  });

  it('returns error response for non-existent flag', () => {
    const result = evaluateBatch(store, 'default', [
      { flagKey: 'nonexistent', entityId: 'user1', context: {} }
    ]);
    expect(result.responses.length).toBe(1);
    expect(result.responses[0].type).toBe('ERROR_EVALUATION_RESPONSE_TYPE');
    expect(result.responses[0].errorEvaluationResponse!.reason).toBe(
      'NOT_FOUND_ERROR_EVALUATION_REASON'
    );
  });

  it('includes requestDurationMillis', () => {
    const result = evaluateBatch(store, 'default', [
      { flagKey: 'flag1', entityId: 'user1', context: { fizz: 'buzz' } }
    ]);
    expect(typeof result.requestDurationMillis).toBe('number');
    expect(result.requestDurationMillis).toBeGreaterThanOrEqual(0);
  });

  it('handles empty request array', () => {
    const result = evaluateBatch(store, 'default', []);
    expect(result.responses.length).toBe(0);
  });
});

// ── JsLocalEngine (integration) ─────────────────────────────────────────────

describe('JsLocalEngine', () => {
  let engine: JsLocalEngine;

  beforeEach(() => {
    engine = new JsLocalEngine('default');
    engine.snapshot(STATE_FIXTURE);
  });

  it('evaluateVariant returns match result', () => {
    const result = engine.evaluateVariant({
      flagKey: 'flag1',
      entityId: 'user1',
      context: { fizz: 'buzz' }
    });
    expect(result.match).toBe(true);
    expect(result.variantKey).toBe('variant1');
  });

  it('evaluateBoolean returns enabled', () => {
    const result = engine.evaluateBoolean({
      flagKey: 'flag_boolean',
      entityId: 'user1',
      context: { fizz: 'buzz' }
    });
    expect(result.enabled).toBe(true);
    expect(result.reason).toBe('MATCH_EVALUATION_REASON');
  });

  it('evaluateBatch returns array of responses', () => {
    const result = engine.evaluateBatch([
      { flagKey: 'flag1', entityId: 'user1', context: { fizz: 'buzz' } },
      { flagKey: 'flag_boolean', entityId: 'user1', context: { fizz: 'buzz' } }
    ]);
    expect(result.responses.length).toBe(2);
  });

  it('listFlags returns all loaded flags', () => {
    const flags = engine.listFlags();
    expect(Array.isArray(flags)).toBe(true);
    expect(flags.length).toBe(STATE_FIXTURE.flags.length);
  });

  it('listFlags returns empty before snapshot is loaded', () => {
    const fresh = new JsLocalEngine('default');
    const flags = fresh.listFlags();
    expect(Array.isArray(flags)).toBe(true);
    expect(flags.length).toBe(0);
  });

  it('snapshot replaces previous data', () => {
    engine.snapshot({
      namespace: { key: 'default' },
      flags: [
        {
          key: 'new_flag',
          name: 'new_flag',
          enabled: true,
          type: 'BOOLEAN_FLAG_TYPE',
          rules: [],
          rollouts: []
        }
      ]
    });
    const flags = engine.listFlags();
    expect(flags.length).toBe(1);
    expect(flags[0].key).toBe('new_flag');
  });

  it('isAsync is not set (sync engine)', () => {
    expect(engine.isAsync).toBeUndefined();
  });
});

// ── Constraint matching edge cases ───────────────────────────────────────────

describe('constraint matching (via evaluateVariant)', () => {
  let store: JsStore;

  function makeFixture(constraints, matchType = 'ALL_SEGMENT_MATCH_TYPE') {
    return {
      namespace: { key: 'default' },
      flags: [
        {
          key: 'test_flag',
          name: 'test_flag',
          enabled: true,
          type: 'VARIANT_FLAG_TYPE',
          rules: [
            {
              segments: [
                {
                  key: 'test_seg',
                  matchType,
                  constraints
                }
              ],
              segmentOperator: 'OR_SEGMENT_OPERATOR',
              distributions: [
                { variantKey: 'matched', rollout: 100 }
              ]
            }
          ],
          rollouts: []
        }
      ]
    };
  }

  function eval_(fixture, ctx) {
    store = new JsStore('default');
    store.loadSnapshot(fixture);
    return evaluateVariant(store, 'default', 'test_flag', 'user1', ctx);
  }

  // ── String constraints ──────────────────────────────────

  it('string: eq matches', () => {
    const f = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'color', operator: 'eq', value: 'red' }
    ]);
    expect(eval_(f, { color: 'red' }).match).toBe(true);
    expect(eval_(f, { color: 'blue' }).match).toBe(false);
  });

  it('string: neq matches', () => {
    const f = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'color', operator: 'neq', value: 'red' }
    ]);
    expect(eval_(f, { color: 'blue' }).match).toBe(true);
    expect(eval_(f, { color: 'red' }).match).toBe(false);
  });

  it('string: prefix matches', () => {
    const f = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'name', operator: 'prefix', value: 'Jo' }
    ]);
    expect(eval_(f, { name: 'John' }).match).toBe(true);
    expect(eval_(f, { name: 'Bob' }).match).toBe(false);
  });

  it('string: suffix matches', () => {
    const f = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'email', operator: 'suffix', value: '.com' }
    ]);
    expect(eval_(f, { email: 'a@b.com' }).match).toBe(true);
    expect(eval_(f, { email: 'a@b.org' }).match).toBe(false);
  });

  it('string: contains matches', () => {
    const f = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'desc', operator: 'contains', value: 'hello' }
    ]);
    expect(eval_(f, { desc: 'say hello world' }).match).toBe(true);
    expect(eval_(f, { desc: 'goodbye' }).match).toBe(false);
  });

  it('string: isoneof matches', () => {
    const f = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'tier', operator: 'isoneof', value: '["gold","silver"]' }
    ]);
    expect(eval_(f, { tier: 'gold' }).match).toBe(true);
    expect(eval_(f, { tier: 'bronze' }).match).toBe(false);
  });

  it('string: empty / notempty', () => {
    const fEmpty = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'x', operator: 'empty', value: '' }
    ]);
    expect(eval_(fEmpty, {}).match).toBe(true);
    expect(eval_(fEmpty, { x: 'val' }).match).toBe(false);

    const fNotEmpty = makeFixture([
      { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'x', operator: 'notempty', value: '' }
    ]);
    expect(eval_(fNotEmpty, { x: 'val' }).match).toBe(true);
    expect(eval_(fNotEmpty, {}).match).toBe(false);
  });

  // ── Number constraints ──────────────────────────────────

  it('number: eq/neq/lt/gt/lte/gte', () => {
    const ops = [
      { op: 'eq', ctx: '10', val: '10', expected: true },
      { op: 'eq', ctx: '11', val: '10', expected: false },
      { op: 'neq', ctx: '11', val: '10', expected: true },
      { op: 'lt', ctx: '5', val: '10', expected: true },
      { op: 'lt', ctx: '15', val: '10', expected: false },
      { op: 'lte', ctx: '10', val: '10', expected: true },
      { op: 'gt', ctx: '15', val: '10', expected: true },
      { op: 'gt', ctx: '5', val: '10', expected: false },
      { op: 'gte', ctx: '10', val: '10', expected: true }
    ];
    for (const { op, ctx, val, expected } of ops) {
      const f = makeFixture([
        { type: 'NUMBER_CONSTRAINT_COMPARISON_TYPE', property: 'age', operator: op, value: val }
      ]);
      expect(eval_(f, { age: ctx }).match).toBe(expected);
    }
  });

  it('number: present/notpresent', () => {
    const fPresent = makeFixture([
      { type: 'NUMBER_CONSTRAINT_COMPARISON_TYPE', property: 'count', operator: 'present', value: '' }
    ]);
    expect(eval_(fPresent, { count: '5' }).match).toBe(true);
    expect(eval_(fPresent, {}).match).toBe(false);

    const fNotPresent = makeFixture([
      { type: 'NUMBER_CONSTRAINT_COMPARISON_TYPE', property: 'count', operator: 'notpresent', value: '' }
    ]);
    expect(eval_(fNotPresent, {}).match).toBe(true);
    expect(eval_(fNotPresent, { count: '5' }).match).toBe(false);
  });

  it('number: isoneof', () => {
    const f = makeFixture([
      { type: 'NUMBER_CONSTRAINT_COMPARISON_TYPE', property: 'code', operator: 'isoneof', value: '[200,201,204]' }
    ]);
    expect(eval_(f, { code: '200' }).match).toBe(true);
    expect(eval_(f, { code: '404' }).match).toBe(false);
  });

  // ── Boolean constraints ─────────────────────────────────

  it('boolean: true/false operators', () => {
    const fTrue = makeFixture([
      { type: 'BOOLEAN_CONSTRAINT_COMPARISON_TYPE', property: 'active', operator: 'true', value: '' }
    ]);
    expect(eval_(fTrue, { active: 'true' }).match).toBe(true);
    expect(eval_(fTrue, { active: 'false' }).match).toBe(false);

    const fFalse = makeFixture([
      { type: 'BOOLEAN_CONSTRAINT_COMPARISON_TYPE', property: 'active', operator: 'false', value: '' }
    ]);
    expect(eval_(fFalse, { active: 'false' }).match).toBe(true);
    expect(eval_(fFalse, { active: 'true' }).match).toBe(false);
  });

  it('boolean: present/notpresent', () => {
    const fPresent = makeFixture([
      { type: 'BOOLEAN_CONSTRAINT_COMPARISON_TYPE', property: 'opt', operator: 'present', value: '' }
    ]);
    expect(eval_(fPresent, { opt: 'true' }).match).toBe(true);
    expect(eval_(fPresent, {}).match).toBe(false);
  });

  // ── Datetime constraints ────────────────────────────────

  it('datetime: lt/gt', () => {
    const fBefore = makeFixture([
      { type: 'DATETIME_CONSTRAINT_COMPARISON_TYPE', property: 'ts', operator: 'lt', value: '2025-01-01T00:00:00Z' }
    ]);
    expect(eval_(fBefore, { ts: '2024-06-01T00:00:00Z' }).match).toBe(true);
    expect(eval_(fBefore, { ts: '2025-06-01T00:00:00Z' }).match).toBe(false);

    const fAfter = makeFixture([
      { type: 'DATETIME_CONSTRAINT_COMPARISON_TYPE', property: 'ts', operator: 'gt', value: '2025-01-01T00:00:00Z' }
    ]);
    expect(eval_(fAfter, { ts: '2025-06-01T00:00:00Z' }).match).toBe(true);
    expect(eval_(fAfter, { ts: '2024-06-01T00:00:00Z' }).match).toBe(false);
  });

  it('datetime: present/notpresent', () => {
    const fPresent = makeFixture([
      { type: 'DATETIME_CONSTRAINT_COMPARISON_TYPE', property: 'ts', operator: 'present', value: '' }
    ]);
    expect(eval_(fPresent, { ts: '2025-01-01T00:00:00Z' }).match).toBe(true);
    expect(eval_(fPresent, {}).match).toBe(false);
  });

  // ── Entity ID constraint ────────────────────────────────

  it('entity_id: matches entityId parameter', () => {
    const f = makeFixture([
      { type: 'ENTITY_ID_CONSTRAINT_COMPARISON_TYPE', property: '', operator: 'eq', value: 'special-user' }
    ]);
    store = new JsStore('default');
    store.loadSnapshot(f);
    const result = evaluateVariant(store, 'default', 'test_flag', 'special-user', {});
    expect(result.match).toBe(true);

    const result2 = evaluateVariant(store, 'default', 'test_flag', 'other-user', {});
    expect(result2.match).toBe(false);
  });

  // ── ANY vs ALL segment match type ─────────────────────

  it('ANY_SEGMENT_MATCH_TYPE: matches if at least one constraint matches', () => {
    const f = makeFixture(
      [
        { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'a', operator: 'eq', value: 'x' },
        { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'b', operator: 'eq', value: 'y' }
      ],
      'ANY_SEGMENT_MATCH_TYPE'
    );
    expect(eval_(f, { a: 'x', b: 'nope' }).match).toBe(true);
    expect(eval_(f, { a: 'nope', b: 'nope' }).match).toBe(false);
  });

  it('ALL_SEGMENT_MATCH_TYPE: requires all constraints to match', () => {
    const f = makeFixture(
      [
        { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'a', operator: 'eq', value: 'x' },
        { type: 'STRING_CONSTRAINT_COMPARISON_TYPE', property: 'b', operator: 'eq', value: 'y' }
      ],
      'ALL_SEGMENT_MATCH_TYPE'
    );
    expect(eval_(f, { a: 'x', b: 'y' }).match).toBe(true);
    expect(eval_(f, { a: 'x', b: 'nope' }).match).toBe(false);
  });
});
