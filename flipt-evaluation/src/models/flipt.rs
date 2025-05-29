use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct Flag {
    pub key: String,
    pub enabled: bool,
    pub r#type: FlagType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_variant: Option<Variant>,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct Variant {
    pub id: String,
    pub key: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub attachment: Option<String>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct Constraint {
    pub segment_key: String,
    pub r#type: ConstraintComparisonType,
    pub property: String,
    pub operator: String,
    pub value: String,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct EvaluationRule {
    pub id: String,
    pub flag_key: String,
    pub segments: HashMap<String, EvaluationSegment>,
    pub rank: usize,
    pub segment_operator: SegmentOperator,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct EvaluationDistribution {
    pub rule_id: String,
    pub rollout: f32,
    pub variant_key: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub variant_attachment: Option<String>,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct EvaluationRollout {
    pub rollout_type: RolloutType,
    pub rank: usize,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub segment: Option<RolloutSegment>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub threshold: Option<RolloutThreshold>,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct RolloutThreshold {
    pub percentage: f32,
    pub value: bool,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct RolloutSegment {
    pub value: bool,
    pub segment_operator: SegmentOperator,
    pub segments: HashMap<String, EvaluationSegment>,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct EvaluationSegment {
    pub segment_key: String,
    pub match_type: SegmentMatchType,
    pub constraints: Vec<EvaluationConstraint>,
}

#[derive(Clone, Debug, Serialize, PartialEq)]
pub struct EvaluationConstraint {
    pub r#type: ConstraintComparisonType,
    pub property: String,
    pub operator: String,
    pub value: String,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq)]
pub enum FlagType {
    #[serde(rename = "BOOLEAN_FLAG_TYPE")]
    Boolean,
    #[default]
    #[serde(other)]
    #[serde(rename = "VARIANT_FLAG_TYPE")]
    Variant,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq)]
pub enum SegmentOperator {
    #[serde(rename = "AND_SEGMENT_OPERATOR")]
    And,
    #[default]
    #[serde(other)]
    #[serde(rename = "OR_SEGMENT_OPERATOR")]
    Or,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq)]
pub enum SegmentMatchType {
    #[serde(rename = "ALL_SEGMENT_MATCH_TYPE")]
    All,
    #[default]
    #[serde(other)]
    #[serde(rename = "ANY_SEGMENT_MATCH_TYPE")]
    Any,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq)]
pub enum ConstraintComparisonType {
    #[serde(rename = "STRING_CONSTRAINT_COMPARISON_TYPE")]
    String,
    #[serde(rename = "NUMBER_CONSTRAINT_COMPARISON_TYPE")]
    Number,
    #[serde(rename = "BOOLEAN_CONSTRAINT_COMPARISON_TYPE")]
    Boolean,
    #[serde(rename = "DATETIME_CONSTRAINT_COMPARISON_TYPE")]
    DateTime,
    #[serde(rename = "ENTITY_ID_CONSTRAINT_COMPARISON_TYPE")]
    EntityId,
    #[default]
    #[serde(other)]
    #[serde(rename = "UNKNOWN_CONSTRAINT_COMPARISON_TYPE")]
    Unknown,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq)]
pub enum RolloutType {
    #[serde(rename = "SEGMENT_ROLLOUT_TYPE")]
    Segment,
    #[serde(rename = "THRESHOLD_ROLLOUT_TYPE")]
    Threshold,
    #[default]
    #[serde(other)]
    #[serde(rename = "UNKNOWN_ROLLOUT_TYPE")]
    Unknown,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq)]
pub enum EvaluationReason {
    #[serde(rename = "FLAG_DISABLED_EVALUATION_REASON")]
    FlagDisabled,
    #[serde(rename = "MATCH_EVALUATION_REASON")]
    Match,
    #[serde(rename = "DEFAULT_EVALUATION_REASON")]
    Default,
    #[default]
    #[serde(other)]
    #[serde(rename = "UNKNOWN_EVALUATION_REASON")]
    Unknown,
}

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq)]
pub enum ErrorEvaluationReason {
    #[serde(rename = "UNKNOWN_ERROR_EVALUATION_REASON")]
    Unknown,
    #[serde(rename = "NOT_FOUND_ERROR_EVALUATION_REASON")]
    NotFound,
}

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq)]
pub enum ResponseType {
    #[serde(rename = "VARIANT_EVALUATION_RESPONSE_TYPE")]
    Variant,
    #[serde(rename = "BOOLEAN_EVALUATION_RESPONSE_TYPE")]
    Boolean,
    #[serde(rename = "ERROR_EVALUATION_RESPONSE_TYPE")]
    Error,
}
