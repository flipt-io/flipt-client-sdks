/**
 * Pure JavaScript snapshot store.
 *
 * Parses the Flipt snapshot JSON (camelCase, same format the fetcher downloads)
 * and indexes it for fast lookups during evaluation.
 *
 * Mirrors the Rust Snapshot::build() + Store trait.
 */

// ── Source types (camelCase JSON from Flipt server) ─────────────────────────

interface SourceDocument {
  namespace: { key: string };
  flags: SourceFlag[];
}

interface SourceFlag {
  key: string;
  name: string;
  type?: string;
  description?: string;
  enabled: boolean;
  rules?: SourceRule[];
  rollouts?: SourceRollout[];
  defaultVariant?: SourceVariant;
}

interface SourceVariant {
  id: string;
  key: string;
  attachment?: string;
}

interface SourceRule {
  distributions: SourceDistribution[];
  segments?: SourceSegment[];
  segmentOperator: string;
}

interface SourceDistribution {
  variantKey: string;
  rollout: number;
  variantAttachment?: string;
}

interface SourceRollout {
  segment?: SourceSegmentRule;
  threshold?: SourceThreshold;
}

interface SourceSegmentRule {
  segmentOperator?: string;
  value: boolean;
  segments: SourceSegment[];
}

interface SourceThreshold {
  percentage: number;
  value: boolean;
}

interface SourceSegment {
  key: string;
  matchType: string;
  constraints: SourceConstraint[];
}

interface SourceConstraint {
  type: string;
  property: string;
  operator: string;
  value: string;
}

// ── Evaluation types (indexed for fast lookup) ──────────────────────────────

export interface EvalFlag {
  key: string;
  enabled: boolean;
  type: string; // 'BOOLEAN_FLAG_TYPE' | 'VARIANT_FLAG_TYPE'
  description?: string;
  defaultVariant?: { id: string; key: string; attachment?: string };
}

export interface EvalRule {
  id: string;
  flagKey: string;
  rank: number;
  segments: Record<string, EvalSegment>;
  segmentOperator: string;
}

export interface EvalDistribution {
  ruleId: string;
  rollout: number;
  variantKey: string;
  variantAttachment?: string;
}

export interface EvalRollout {
  rank: number;
  rolloutType: string; // 'SEGMENT' | 'THRESHOLD'
  segment?: EvalRolloutSegment;
  threshold?: { percentage: number; value: boolean };
}

export interface EvalRolloutSegment {
  value: boolean;
  segmentOperator: string;
  segments: Record<string, EvalSegment>;
}

export interface EvalSegment {
  segmentKey: string;
  matchType: string; // 'ALL_SEGMENT_MATCH_TYPE' | 'ANY_SEGMENT_MATCH_TYPE'
  constraints: EvalConstraint[];
}

export interface EvalConstraint {
  type: string;
  property: string;
  operator: string;
  value: string;
}

// ── JsStore ─────────────────────────────────────────────────────────────────

export class JsStore {
  private namespaceKey: string;
  private flags: Map<string, EvalFlag>;
  private evalRules: Map<string, EvalRule[]>;
  private evalDistributions: Map<string, EvalDistribution[]>;
  private evalRollouts: Map<string, EvalRollout[]>;

  constructor(namespace: string) {
    this.namespaceKey = namespace;
    this.flags = new Map();
    this.evalRules = new Map();
    this.evalDistributions = new Map();
    this.evalRollouts = new Map();
  }

  /**
   * Build the store from a snapshot document (same JSON the fetcher returns).
   */
  loadSnapshot(doc: unknown): void {
    const d = doc as SourceDocument;

    this.flags.clear();
    this.evalRules.clear();
    this.evalDistributions.clear();
    this.evalRollouts.clear();

    if (!d || !d.flags) return;

    for (const flag of d.flags) {
      this.flags.set(flag.key, {
        key: flag.key,
        enabled: flag.enabled,
        type: flag.type ?? 'VARIANT_FLAG_TYPE',
        description: flag.description,
        defaultVariant: flag.defaultVariant
          ? {
              id: flag.defaultVariant.id,
              key: flag.defaultVariant.key,
              attachment: flag.defaultVariant.attachment
            }
          : undefined
      });

      // ── Rules + Distributions ───────────────────────────────────────
      const rules: EvalRule[] = [];
      const flagRules = flag.rules ?? [];

      for (let idx = 0; idx < flagRules.length; idx++) {
        const rule = flagRules[idx];
        const rank = idx + 1;
        const ruleId = `${flag.key}-${rank}`;

        const segments: Record<string, EvalSegment> = {};
        for (const seg of rule.segments ?? []) {
          segments[seg.key] = {
            segmentKey: seg.key,
            matchType: seg.matchType,
            constraints: seg.constraints.map((c) => ({
              type: c.type,
              property: c.property,
              operator: c.operator,
              value: c.value
            }))
          };
        }

        rules.push({
          id: ruleId,
          flagKey: flag.key,
          rank,
          segments,
          segmentOperator: rule.segmentOperator
        });

        const dists: EvalDistribution[] = (rule.distributions ?? []).map(
          (d) => ({
            ruleId,
            variantKey: d.variantKey,
            variantAttachment: d.variantAttachment,
            rollout: d.rollout
          })
        );
        this.evalDistributions.set(ruleId, dists);
      }
      this.evalRules.set(flag.key, rules);

      // ── Rollouts ────────────────────────────────────────────────────
      const rollouts: EvalRollout[] = [];
      const flagRollouts = flag.rollouts ?? [];

      for (let i = 0; i < flagRollouts.length; i++) {
        const rollout = flagRollouts[i];
        const rank = i + 1;

        const evalRollout: EvalRollout = {
          rank,
          rolloutType: 'UNKNOWN'
        };

        if (rollout.threshold) {
          evalRollout.rolloutType = 'THRESHOLD';
          evalRollout.threshold = {
            percentage: rollout.threshold.percentage,
            value: rollout.threshold.value
          };
        } else if (rollout.segment) {
          evalRollout.rolloutType = 'SEGMENT';
          const segs: Record<string, EvalSegment> = {};
          for (const seg of rollout.segment.segments) {
            segs[seg.key] = {
              segmentKey: seg.key,
              matchType: seg.matchType,
              constraints: seg.constraints.map((c) => ({
                type: c.type,
                property: c.property,
                operator: c.operator,
                value: c.value
              }))
            };
          }
          evalRollout.segment = {
            value: rollout.segment.value,
            segmentOperator:
              rollout.segment.segmentOperator ?? 'OR_SEGMENT_OPERATOR',
            segments: segs
          };
        }

        rollouts.push(evalRollout);
      }
      this.evalRollouts.set(flag.key, rollouts);
    }
  }

  // ── Store trait methods ─────────────────────────────────────────────────

  getFlag(namespace: string, flagKey: string): EvalFlag | undefined {
    if (this.namespaceKey !== namespace) return undefined;
    return this.flags.get(flagKey);
  }

  listFlags(namespace: string): EvalFlag[] | undefined {
    if (this.namespaceKey !== namespace) return undefined;
    return Array.from(this.flags.values());
  }

  getEvaluationRules(
    namespace: string,
    flagKey: string
  ): EvalRule[] | undefined {
    if (this.namespaceKey !== namespace) return undefined;
    return this.evalRules.get(flagKey);
  }

  getEvaluationDistributions(
    namespace: string,
    ruleId: string
  ): EvalDistribution[] | undefined {
    if (this.namespaceKey !== namespace) return undefined;
    return this.evalDistributions.get(ruleId);
  }

  getEvaluationRollouts(
    namespace: string,
    flagKey: string
  ): EvalRollout[] | undefined {
    if (this.namespaceKey !== namespace) return undefined;
    return this.evalRollouts.get(flagKey);
  }
}
