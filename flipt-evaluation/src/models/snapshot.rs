use serde::Deserialize;
use serde::Serialize;
use std::collections::HashMap;

use crate::models::{flipt, source};

#[derive(Debug, PartialEq, Clone, Serialize, Deserialize)]
pub struct Snapshot {
    pub version: u32,
    pub namespace: Namespace,
}

#[derive(Debug, PartialEq, Clone, Serialize, Deserialize)]
pub struct Namespace {
    pub key: String,
    pub flags: HashMap<String, flipt::Flag>,
    pub eval_rules: HashMap<String, Vec<flipt::EvaluationRule>>,
    pub eval_rollouts: HashMap<String, Vec<flipt::EvaluationRollout>>,
    pub eval_distributions: HashMap<String, Vec<flipt::EvaluationDistribution>>,
}

impl Snapshot {
    pub fn empty(namespace: &str) -> Snapshot {
        Self {
            version: 1,
            namespace: Namespace {
                key: namespace.to_string(),
                flags: HashMap::new(),
                eval_rules: HashMap::new(),
                eval_rollouts: HashMap::new(),
                eval_distributions: HashMap::new(),
            },
        }
    }

    pub fn build(namespace: &str, doc: source::Document) -> Snapshot {
        let mut flags: HashMap<String, flipt::Flag> = HashMap::new();
        let mut eval_rules: HashMap<String, Vec<flipt::EvaluationRule>> = HashMap::new();
        let mut eval_rollouts: HashMap<String, Vec<flipt::EvaluationRollout>> = HashMap::new();
        let mut eval_dists: HashMap<String, Vec<flipt::EvaluationDistribution>> = HashMap::new();

        for flag in doc.flags {
            let f = flipt::Flag {
                key: flag.key.clone(),
                enabled: flag.enabled,
                description: flag.description,
                r#type: flag.r#type.unwrap_or(flipt::FlagType::Variant),
                default_variant: flag.default_variant.map(|v| flipt::Variant {
                    id: v.id,
                    key: v.key,
                    attachment: v.attachment,
                }),
            };

            flags.insert(f.key.clone(), f);

            // Flag Rules
            let mut eval_rules_collection: Vec<flipt::EvaluationRule> = Vec::new();

            let flag_rules = flag.rules.unwrap_or(vec![]);

            for (idx, rule) in flag_rules.into_iter().enumerate() {
                let index = idx + 1;
                let rule_id = format!("{}-{}", flag.key, index);
                let mut eval_rule = flipt::EvaluationRule {
                    id: rule_id.clone(),
                    rank: index,
                    flag_key: flag.key.clone(),
                    segments: HashMap::new(),
                    segment_operator: rule.segment_operator,
                };

                if rule.segments.is_some() {
                    let rule_segments = rule.segments.unwrap();

                    for rule_segment in rule_segments {
                        let mut eval_constraints: Vec<flipt::EvaluationConstraint> = Vec::new();
                        for constraint in rule_segment.constraints {
                            eval_constraints.push(flipt::EvaluationConstraint {
                                r#type: constraint.r#type,
                                property: constraint.property,
                                operator: constraint.operator,
                                value: constraint.value,
                            });
                        }

                        eval_rule.segments.insert(
                            rule_segment.key.clone(),
                            flipt::EvaluationSegment {
                                segment_key: rule_segment.key,
                                match_type: rule_segment.match_type,
                                constraints: eval_constraints,
                            },
                        );
                    }
                }

                let mut evaluation_distributions: Vec<flipt::EvaluationDistribution> = Vec::new();

                for distribution in rule.distributions {
                    evaluation_distributions.push(flipt::EvaluationDistribution {
                        rule_id: rule_id.clone(),
                        variant_key: distribution.variant_key,
                        variant_attachment: distribution.variant_attachment,
                        rollout: distribution.rollout,
                    })
                }

                eval_dists.insert(rule_id.clone(), evaluation_distributions);

                eval_rules_collection.push(eval_rule);
            }

            eval_rules.insert(flag.key.clone(), eval_rules_collection);

            // Flag Rollouts
            let mut eval_rollout_collection: Vec<flipt::EvaluationRollout> = Vec::new();
            let mut rollout_idx = 0;

            let flag_rollouts = flag.rollouts.unwrap_or(vec![]);

            for rollout in flag_rollouts {
                rollout_idx += 1;

                let mut evaluation_rollout: flipt::EvaluationRollout = flipt::EvaluationRollout {
                    rank: rollout_idx,
                    rollout_type: flipt::RolloutType::Unknown,
                    segment: None,
                    threshold: None,
                };

                evaluation_rollout.rank = rollout_idx;

                if rollout.threshold.is_some() {
                    let threshold = rollout.threshold.unwrap();
                    evaluation_rollout.threshold = Some(flipt::RolloutThreshold {
                        percentage: threshold.percentage,
                        value: threshold.value,
                    });

                    evaluation_rollout.rollout_type = flipt::RolloutType::Threshold;
                } else if rollout.segment.is_some() {
                    let mut evaluation_rollout_segments: HashMap<String, flipt::EvaluationSegment> =
                        HashMap::new();

                    let segment_rule = rollout.segment.unwrap();

                    for segment in segment_rule.segments {
                        let mut constraints: Vec<flipt::EvaluationConstraint> = Vec::new();
                        for constraint in segment.constraints {
                            constraints.push(flipt::EvaluationConstraint {
                                r#type: constraint.r#type,
                                property: constraint.property,
                                value: constraint.value,
                                operator: constraint.operator,
                            });
                        }

                        evaluation_rollout_segments.insert(
                            segment.key.clone(),
                            flipt::EvaluationSegment {
                                segment_key: segment.key,
                                match_type: segment.match_type.clone(),
                                constraints,
                            },
                        );
                    }

                    evaluation_rollout.rollout_type = flipt::RolloutType::Segment;
                    evaluation_rollout.segment = Some(flipt::RolloutSegment {
                        value: segment_rule.value,
                        segment_operator: segment_rule
                            .segment_operator
                            .unwrap_or(flipt::SegmentOperator::Or),
                        segments: evaluation_rollout_segments,
                    });
                }

                eval_rollout_collection.push(evaluation_rollout);
            }

            eval_rollouts.insert(flag.key.clone(), eval_rollout_collection);
        }

        Self {
            version: 1,
            namespace: Namespace {
                key: namespace.to_string(),
                flags,
                eval_rules,
                eval_rollouts,
                eval_distributions: eval_dists,
            },
        }
    }
}
