use std::fmt::{self};

use serde::{Deserialize, Serialize};

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

impl fmt::Display for EvaluationReason {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{:?}", self)
    }
}

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq)]
pub enum ErrorEvaluationReason {
    #[serde(rename = "UNKNOWN_ERROR_EVALUATION_REASON")]
    Unknown,
    #[serde(rename = "NOT_FOUND_ERROR_EVALUATION_REASON")]
    NotFound,
}

impl fmt::Display for ErrorEvaluationReason {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{:?}", self)
    }
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

impl fmt::Display for ResponseType {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{:?}", self)
    }
}
