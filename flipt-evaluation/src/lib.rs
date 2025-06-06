use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use web_time::Instant;

pub mod error;
pub mod models;
pub mod store;

use crate::error::Error;
use crate::models::flipt;
use crate::store::Store;

const DEFAULT_PERCENT: f32 = 100.0;
const DEFAULT_TOTAL_BUCKET_NUMBER: u32 = 1000;
const DEFAULT_PERCENT_MULTIPIER: f32 = DEFAULT_TOTAL_BUCKET_NUMBER as f32 / DEFAULT_PERCENT;

#[repr(C)]
#[derive(Deserialize, Clone, PartialEq, Debug, Serialize)]
pub struct EvaluationRequest {
    pub flag_key: String,
    pub entity_id: String,
    pub context: HashMap<String, String>,
}

#[derive(Serialize, Debug)]
pub struct VariantEvaluationResponse {
    pub r#match: bool,
    pub segment_keys: Vec<String>,
    pub reason: flipt::EvaluationReason,
    pub flag_key: String,
    pub variant_key: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub variant_attachment: Option<String>,
    pub request_duration_millis: f64,
    pub timestamp: DateTime<Utc>,
}

#[derive(Serialize, Debug)]
pub struct BooleanEvaluationResponse {
    pub enabled: bool,
    pub flag_key: String,
    pub reason: flipt::EvaluationReason,
    pub request_duration_millis: f64,
    pub timestamp: DateTime<Utc>,
}

#[derive(Serialize, Debug)]
pub struct ErrorEvaluationResponse {
    pub flag_key: String,
    pub namespace_key: String,
    pub reason: flipt::ErrorEvaluationReason,
}

#[derive(Serialize, Debug)]
pub struct BatchEvaluationResponse {
    pub responses: Vec<EvaluationResponse>,
    pub request_duration_millis: f64,
}

#[derive(Serialize, Debug)]
pub struct EvaluationResponse {
    pub r#type: flipt::ResponseType,
    pub boolean_evaluation_response: Option<BooleanEvaluationResponse>,
    pub variant_evaluation_response: Option<VariantEvaluationResponse>,
    pub error_evaluation_response: Option<ErrorEvaluationResponse>,
}

impl Default for VariantEvaluationResponse {
    fn default() -> Self {
        Self {
            r#match: false,
            segment_keys: vec![],
            reason: flipt::EvaluationReason::Unknown,
            flag_key: String::from(""),
            variant_key: String::from(""),
            variant_attachment: None,
            request_duration_millis: 0.0,
            timestamp: chrono::offset::Utc::now(),
        }
    }
}

impl Default for BooleanEvaluationResponse {
    fn default() -> Self {
        Self {
            enabled: false,
            flag_key: String::from(""),
            reason: flipt::EvaluationReason::Unknown,
            request_duration_millis: 0.0,
            timestamp: chrono::offset::Utc::now(),
        }
    }
}

impl Default for ErrorEvaluationResponse {
    fn default() -> Self {
        Self {
            flag_key: String::from(""),
            namespace_key: String::from(""),
            reason: flipt::ErrorEvaluationReason::Unknown,
        }
    }
}

pub fn variant_evaluation(
    store: &dyn Store,
    namespace: &str,
    request: &EvaluationRequest,
) -> Result<VariantEvaluationResponse, Error> {
    let start = Instant::now();
    let mut last_rank = 0;

    let flag = store
        .get_flag(namespace, &request.flag_key)
        .ok_or_else(|| {
            Error::InvalidRequest(format!(
                "failed to get flag information {}/{}",
                namespace, &request.flag_key,
            ))
        })?;

    if !matches!(flag.r#type, flipt::FlagType::Variant) {
        return Err(Error::InvalidRequest(format!(
            "{} is not a variant flag",
            &request.flag_key,
        )));
    }

    let mut variant_evaluation_response = VariantEvaluationResponse {
        flag_key: flag.key.clone(),
        ..Default::default()
    };

    if let Some(default_variant) = &flag.default_variant {
        variant_evaluation_response.reason = flipt::EvaluationReason::Default;
        variant_evaluation_response.variant_key = default_variant.key.clone();
        variant_evaluation_response.variant_attachment = default_variant.attachment.clone();
    }

    if !flag.enabled {
        variant_evaluation_response.reason = flipt::EvaluationReason::FlagDisabled;
        variant_evaluation_response.request_duration_millis = start.elapsed().as_millis() as f64;
        return Ok(variant_evaluation_response);
    }

    let evaluation_rules = store
        .get_evaluation_rules(namespace, &request.flag_key)
        .ok_or_else(|| {
            Error::Unknown(format!(
                "error getting evaluation rules for namespace {} and flag {}",
                namespace,
                request.flag_key.clone()
            ))
        })?;

    // if no rules and flag is enabled, return default variant
    if evaluation_rules.is_empty() {
        variant_evaluation_response.request_duration_millis = start.elapsed().as_millis() as f64;
        return Ok(variant_evaluation_response);
    }

    for rule in evaluation_rules {
        if rule.rank < last_rank {
            return Err(Error::InvalidRequest(format!(
                "rule rank: {} detected out of order",
                rule.rank
            )));
        }

        last_rank = rule.rank;

        let mut segment_keys: Vec<String> = vec![];
        let mut segment_matches = 0;

        for (segment_key, segment) in &rule.segments {
            let matched = matches_constraints(
                &request.context,
                &segment.constraints,
                &segment.match_type,
                &request.entity_id,
            )?;

            if matched {
                segment_keys.push(segment_key.clone());
                segment_matches += 1;
            }
        }

        if rule.segment_operator == flipt::SegmentOperator::Or {
            if segment_matches < 1 {
                continue;
            }
        } else if rule.segment_operator == flipt::SegmentOperator::And
            && rule.segments.len() != segment_matches
        {
            continue;
        }

        variant_evaluation_response.segment_keys = segment_keys;

        let distributions = store
            .get_evaluation_distributions(namespace, &rule.id)
            .ok_or_else(|| {
                Error::Unknown(format!(
                    "error getting evaluation distributions for namespace {} and rule {}",
                    namespace,
                    rule.id.clone()
                ))
            })?;

        let mut valid_distributions: Vec<flipt::EvaluationDistribution> = vec![];
        let mut buckets: Vec<i32> = vec![];

        for distribution in distributions {
            if distribution.rollout > 0.0 {
                valid_distributions.push(distribution.clone());

                if buckets.is_empty() {
                    let bucket = (distribution.rollout * DEFAULT_PERCENT_MULTIPIER) as i32;
                    buckets.push(bucket);
                } else {
                    let bucket = buckets[buckets.len() - 1]
                        + (distribution.rollout * DEFAULT_PERCENT_MULTIPIER) as i32;
                    buckets.push(bucket);
                }
            }
        }

        // no distributions for the rule
        // match is true here because it did match the segment/rule
        if valid_distributions.is_empty() {
            variant_evaluation_response.r#match = true;
            variant_evaluation_response.reason = flipt::EvaluationReason::Match;
            variant_evaluation_response.request_duration_millis =
                start.elapsed().as_millis() as f64;
            return Ok(variant_evaluation_response);
        }

        let bucket =
            crc32fast::hash(format!("{}{}", request.flag_key, request.entity_id).as_bytes())
                % DEFAULT_TOTAL_BUCKET_NUMBER;

        buckets.sort();

        let index = match buckets.binary_search(&(bucket as i32 + 1)) {
            Ok(idx) => idx,
            Err(idx) => idx,
        };

        // if index is outside of our existing buckets then it does not match any distribution
        if index == valid_distributions.len() {
            variant_evaluation_response.r#match = false;
            variant_evaluation_response.request_duration_millis =
                start.elapsed().as_millis() as f64;
            return Ok(variant_evaluation_response);
        }

        let d = &valid_distributions[index];

        variant_evaluation_response.r#match = true;
        variant_evaluation_response.variant_key = d.variant_key.clone();
        variant_evaluation_response.variant_attachment = d.variant_attachment.clone();
        variant_evaluation_response.reason = flipt::EvaluationReason::Match;
        variant_evaluation_response.request_duration_millis = start.elapsed().as_millis() as f64;
        return Ok(variant_evaluation_response);
    }

    Ok(variant_evaluation_response)
}

pub fn boolean_evaluation(
    store: &dyn Store,
    namespace: &str,
    request: &EvaluationRequest,
) -> Result<BooleanEvaluationResponse, Error> {
    let start = Instant::now();
    let mut last_rank = 0;

    let flag = store
        .get_flag(namespace, &request.flag_key)
        .ok_or_else(|| {
            Error::InvalidRequest(format!(
                "failed to get flag information {}/{}",
                namespace, &request.flag_key,
            ))
        })?;

    if !matches!(flag.r#type, flipt::FlagType::Boolean) {
        return Err(Error::InvalidRequest(format!(
            "{} is not a boolean flag",
            &request.flag_key,
        )));
    }

    let evaluation_rollouts = store
        .get_evaluation_rollouts(namespace, &request.flag_key)
        .ok_or_else(|| {
            Error::Unknown(format!(
                "error getting evaluation rollouts for namespace {} and flag {}",
                namespace,
                request.flag_key.clone()
            ))
        })?;

    for rollout in evaluation_rollouts {
        if rollout.rank < last_rank {
            return Err(Error::InvalidRequest(format!(
                "rollout rank: {} detected out of order",
                rollout.rank
            )));
        }

        last_rank = rollout.rank;

        if let Some(threshold) = rollout.threshold {
            let normalized_value =
                (crc32fast::hash(format!("{}{}", request.entity_id, request.flag_key).as_bytes())
                    % 100) as f32;

            if normalized_value < threshold.percentage {
                return Ok(BooleanEvaluationResponse {
                    enabled: threshold.value,
                    flag_key: flag.key.clone(),
                    reason: flipt::EvaluationReason::Match,
                    request_duration_millis: start.elapsed().as_millis() as f64,
                    timestamp: chrono::offset::Utc::now(),
                });
            }
        } else if let Some(segment) = rollout.segment {
            let mut segment_matches = 0;

            for segment_data in segment.segments.values() {
                let matched = matches_constraints(
                    &request.context,
                    &segment_data.constraints,
                    &segment_data.match_type,
                    &request.entity_id,
                )?;

                if matched {
                    segment_matches += 1;
                }
            }

            if segment.segment_operator == flipt::SegmentOperator::Or {
                if segment_matches < 1 {
                    continue;
                }
            } else if segment.segment_operator == flipt::SegmentOperator::And
                && segment.segments.len() != segment_matches
            {
                continue;
            }

            return Ok(BooleanEvaluationResponse {
                enabled: segment.value,
                flag_key: flag.key.clone(),
                reason: flipt::EvaluationReason::Match,
                request_duration_millis: start.elapsed().as_millis() as f64,
                timestamp: chrono::offset::Utc::now(),
            });
        }
    }

    Ok(BooleanEvaluationResponse {
        enabled: flag.enabled,
        flag_key: flag.key.clone(),
        reason: flipt::EvaluationReason::Default,
        request_duration_millis: start.elapsed().as_millis() as f64,
        timestamp: chrono::offset::Utc::now(),
    })
}

pub fn batch_evaluation(
    store: &dyn Store,
    namespace: &str,
    requests: Vec<EvaluationRequest>,
) -> Result<BatchEvaluationResponse, Error> {
    let start = Instant::now();

    let mut evaluation_responses: Vec<EvaluationResponse> = vec![];
    for request in requests {
        let flag = match store.get_flag(namespace, &request.flag_key) {
            Some(f) => f,
            None => {
                evaluation_responses.push(EvaluationResponse {
                    r#type: flipt::ResponseType::Error,
                    boolean_evaluation_response: None,
                    variant_evaluation_response: None,
                    error_evaluation_response: Some(ErrorEvaluationResponse {
                        flag_key: request.flag_key.clone(),
                        namespace_key: namespace.to_string(),
                        reason: flipt::ErrorEvaluationReason::NotFound,
                    }),
                });
                continue;
            }
        };

        match flag.r#type {
            flipt::FlagType::Boolean => {
                let boolean_evaluation = boolean_evaluation(store, namespace, &request)?;
                evaluation_responses.push(EvaluationResponse {
                    r#type: flipt::ResponseType::Boolean,
                    boolean_evaluation_response: Some(boolean_evaluation),
                    variant_evaluation_response: None,
                    error_evaluation_response: None,
                });
            }
            flipt::FlagType::Variant => {
                let variant_evaluation = variant_evaluation(store, namespace, &request)?;
                evaluation_responses.push(EvaluationResponse {
                    r#type: flipt::ResponseType::Variant,
                    boolean_evaluation_response: None,
                    variant_evaluation_response: Some(variant_evaluation),
                    error_evaluation_response: None,
                });
            }
        }
    }

    Ok(BatchEvaluationResponse {
        responses: evaluation_responses,
        request_duration_millis: start.elapsed().as_millis() as f64,
    })
}

fn matches_constraints(
    eval_context: &HashMap<String, String>,
    constraints: &Vec<flipt::EvaluationConstraint>,
    segment_match_type: &flipt::SegmentMatchType,
    entity_id: &str,
) -> Result<bool, Error> {
    let mut constraint_matches: usize = 0;
    for constraint in constraints {
        let value = eval_context
            .get(&constraint.property)
            .unwrap_or(&String::new())
            .to_string();

        let matched = match constraint.r#type {
            flipt::ConstraintComparisonType::String => matches_string(constraint, &value),
            flipt::ConstraintComparisonType::Number => {
                matches_number(constraint, &value).unwrap_or(false)
            }
            flipt::ConstraintComparisonType::Boolean => {
                matches_boolean(constraint, &value).unwrap_or(false)
            }
            flipt::ConstraintComparisonType::DateTime => {
                matches_datetime(constraint, &value).unwrap_or(false)
            }
            flipt::ConstraintComparisonType::EntityId => matches_string(constraint, entity_id),
            _ => {
                return Ok(false);
            }
        };

        if matched {
            constraint_matches += 1;

            if segment_match_type == &flipt::SegmentMatchType::Any {
                break;
            } else {
                continue;
            }
        } else if segment_match_type == &flipt::SegmentMatchType::All {
            break;
        } else {
            continue;
        }
    }

    let is_match = match segment_match_type {
        flipt::SegmentMatchType::All => constraints.len() == constraint_matches,
        flipt::SegmentMatchType::Any => constraints.is_empty() || constraint_matches != 0,
    };

    Ok(is_match)
}

fn oneof_string(v: &str, values: &str) -> bool {
    match serde_json::from_str::<Vec<&str>>(values) {
        Ok(values) => values.contains(&v),
        Err(_) => false,
    }
}

fn matches_string(evaluation_constraint: &flipt::EvaluationConstraint, v: &str) -> bool {
    let operator = evaluation_constraint.operator.as_str();

    match operator {
        "empty" => {
            return v.is_empty();
        }
        "notempty" => {
            return !v.is_empty();
        }
        _ => {}
    }

    if v.is_empty() {
        return false;
    }

    let value = evaluation_constraint.value.as_str();
    match operator {
        "eq" => v == value,
        "neq" => v != value,
        "prefix" => v.starts_with(value),
        "suffix" => v.ends_with(value),
        "isoneof" => oneof_string(v, value),
        "isnotoneof" => !oneof_string(v, value),
        "contains" => v.contains(value),
        "notcontains" => !v.contains(value),
        _ => false,
    }
}

fn oneof_number(v: i32, values: &str) -> Result<bool, Error> {
    match serde_json::from_str::<Vec<i32>>(values) {
        Ok(values) => Ok(values.contains(&v)),
        Err(err) => Err(Error::InvalidRequest(format!(
            "error parsing numbers {values}: {err}"
        )))?,
    }
}

fn matches_number(
    evaluation_constraint: &flipt::EvaluationConstraint,
    v: &str,
) -> Result<bool, Error> {
    let operator = evaluation_constraint.operator.as_str();

    match operator {
        "notpresent" => {
            return Ok(v.is_empty());
        }
        "present" => {
            return Ok(!v.is_empty());
        }
        _ => {}
    }

    if v.is_empty() {
        return Ok(false);
    }

    let v_number = match v.parse::<i32>() {
        Ok(v) => v,
        Err(err) => Err(Error::InvalidRequest(format!(
            "error parsing number {v}: {err}"
        )))?,
    };

    match operator {
        "isoneof" => {
            return oneof_number(v_number, &evaluation_constraint.value);
        }
        "isnotoneof" => {
            return match oneof_number(v_number, &evaluation_constraint.value) {
                Ok(v) => Ok(!v),
                Err(err) => Err(err),
            }
        }
        _ => {}
    }

    let value_number = match evaluation_constraint.value.parse::<i32>() {
        Ok(v) => v,
        Err(err) => Err(Error::InvalidRequest(format!(
            "error parsing number {}: {}",
            evaluation_constraint.value, err
        )))?,
    };

    match operator {
        "eq" => Ok(v_number == value_number),
        "neq" => Ok(v_number != value_number),
        "lt" => Ok(v_number < value_number),
        "lte" => Ok(v_number <= value_number),
        "gt" => Ok(v_number > value_number),
        "gte" => Ok(v_number >= value_number),
        _ => Ok(false),
    }
}

fn matches_boolean(
    evaluation_constraint: &flipt::EvaluationConstraint,
    v: &str,
) -> Result<bool, Error> {
    let operator = evaluation_constraint.operator.as_str();

    match operator {
        "notpresent" => {
            return Ok(v.is_empty());
        }
        "present" => {
            return Ok(!v.is_empty());
        }
        _ => {}
    }

    if v.is_empty() {
        return Ok(false);
    }

    let v_bool = match v.parse::<bool>() {
        Ok(v) => v,
        Err(err) => Err(Error::InvalidRequest(format!(
            "error parsing boolean {v}: {err}"
        )))?,
    };

    match operator {
        "true" => Ok(v_bool),
        "false" => Ok(!v_bool),
        _ => Ok(false),
    }
}

fn matches_datetime(
    evaluation_constraint: &flipt::EvaluationConstraint,
    v: &str,
) -> Result<bool, Error> {
    let operator = evaluation_constraint.operator.as_str();

    match operator {
        "notpresent" => {
            return Ok(v.is_empty());
        }
        "present" => {
            return Ok(!v.is_empty());
        }
        _ => {}
    }

    if v.is_empty() {
        return Ok(false);
    }

    let d = match DateTime::parse_from_rfc3339(v) {
        Ok(t) => t.timestamp(),
        Err(e) => Err(Error::InvalidRequest(format!(
            "error parsing time {v}: {e}"
        )))?,
    };

    let value = match DateTime::parse_from_rfc3339(&evaluation_constraint.value) {
        Ok(t) => t.timestamp(),
        Err(e) => Err(Error::InvalidRequest(format!(
            "error parsing time {}: {}",
            &evaluation_constraint.value, e
        )))?,
    };

    match operator {
        "eq" => Ok(d == value),
        "neq" => Ok(d != value),
        "lt" => Ok(d < value),
        "lte" => Ok(d <= value),
        "gt" => Ok(d > value),
        "gte" => Ok(d >= value),
        _ => Ok(false),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::flipt::RolloutSegment;
    use crate::store::MockStore;

    macro_rules! matches_string_tests {
        ($($name:ident: $value:expr,)*) => {
        $(
            #[test]
            fn $name() {
                let (first, second, expected) = $value;
                assert_eq!(expected, matches_string(first, second));
            }
        )*
        }
    }

    macro_rules! matches_datetime_tests {
        ($($name:ident: $value:expr,)*) => {
        $(
            #[test]
            fn $name() {
                let (first, second, expected) = $value;
                assert_eq!(expected, matches_datetime(first, second).unwrap());
            }
        )*
        }
    }

    macro_rules! matches_number_tests {
        ($($name:ident: $value:expr,)*) => {
        $(
            #[test]
            fn $name() {
                let (first, second, expected) = $value;
                assert_eq!(expected, matches_number(first, second).unwrap());
            }
        )*
        }
    }

    matches_string_tests! {
        string_eq: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("eq"),
            value: String::from("number"),
        }, "number", true),
        string_neq: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("neq"),
            value: String::from("number"),
        }, "num", true),
        string_prefix: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("prefix"),
            value: String::from("num"),
        }, "number", true),
        string_suffix: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("suffix"),
            value: String::from("ber"),
        }, "number", true),
        string_isoneof_exists: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "2", true),
        string_isoneof_absent: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "3", false),
        string_isnotoneof_exists: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "2", false),
        string_isnotoneof_absent: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "3", true),
        string_contains: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("contains"),
            value: String::from("num"),
        }, "number", true),
        string_contains_false: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("contains"),
            value: String::from("num"),
        }, "x", false),
        string_notcontains: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("notcontains"),
            value: String::from("num"),
        }, "x", true),
        string_notcontains_false: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("notcontains"),
            value: String::from("num"),
        }, "number", false),

    }

    matches_datetime_tests! {
        datetime_eq: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("eq"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T15:04:05Z", true),
        datetime_neq: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("neq"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T15:03:05Z", true),
        datetime_lt: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("lt"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T14:03:05Z", true),
        datetime_gt: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("gt"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T16:03:05Z", true),
        datetime_lte: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("lte"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T15:04:05Z", true),
        datetime_gte: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("gte"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T16:03:05Z", true),

    }

    matches_number_tests! {
        number_eq: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("eq"),
            value: String::from("1"),
        }, "1", true),
        number_neq: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("neq"),
            value: String::from("1"),
        }, "0", true),
        number_lt: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("lt"),
            value: String::from("4"),
        }, "3", true),
        number_gt: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("gt"),
            value: String::from("3"),
        }, "4", true),
        number_lte: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("date"),
            operator: String::from("lte"),
            value: String::from("3"),
        }, "3", true),
        number_gte: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("date"),
            operator: String::from("gte"),
            value: String::from("3"),
        }, "4", true),
        number_isoneof_exists: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from("[1, 2]"),
        }, "2", true),
        number_isoneof_absent: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from("[1, 2]"),
        }, "3", false),
        number_isnotoneof_exists: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from("[1, 2]"),
        }, "2", false),
        number_isnotoneof_absent: (&flipt::EvaluationConstraint{
            r#type: flipt::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from("[1, 2]"),
        }, "3", true),
    }

    #[test]
    fn test_matches_boolean_success() {
        let value_one = matches_boolean(
            &flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::Boolean,
                property: String::from("fizz"),
                operator: String::from("true"),
                value: "".into(),
            },
            "true",
        )
        .expect("boolean should be parsed correctly");

        assert!(value_one);

        let value_two = matches_boolean(
            &flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::Boolean,
                property: String::from("fizz"),
                operator: String::from("false"),
                value: "".into(),
            },
            "false",
        )
        .expect("boolean should be parsed correctly");

        assert!(value_two);
    }

    #[test]
    fn test_matches_boolean_failure() {
        let result = matches_boolean(
            &flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::Boolean,
                property: String::from("fizz"),
                operator: String::from("true"),
                value: "".into(),
            },
            "blah",
        );

        assert!(result.is_err());
        assert_eq!(
            result.err().unwrap().to_string(),
            "invalid request: error parsing boolean blah: provided string was not `true` or `false`"
        );
    }

    #[test]
    fn test_matches_number_failure() {
        let result_one = matches_number(
            &flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::Number,
                property: String::from("number"),
                operator: String::from("eq"),
                value: String::from("9"),
            },
            "notanumber",
        );

        assert!(result_one.is_err());
        assert_eq!(
            result_one.err().unwrap().to_string(),
            "invalid request: error parsing number notanumber: invalid digit found in string"
        );

        let result_two = matches_number(
            &flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::Number,
                property: String::from("number"),
                operator: String::from("eq"),
                value: String::from("notanumber"),
            },
            "9",
        );

        assert!(result_two.is_err());
        assert_eq!(
            result_two.err().unwrap().to_string(),
            "invalid request: error parsing number notanumber: invalid digit found in string"
        );
    }

    #[test]
    fn test_matches_datetime_failure() {
        let result_one = matches_datetime(
            &flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::String,
                property: String::from("date"),
                operator: String::from("eq"),
                value: String::from("blah"),
            },
            "2006-01-02T15:04:05Z",
        );

        assert!(result_one.is_err());
        assert_eq!(
            result_one.err().unwrap().to_string(),
            "invalid request: error parsing time blah: input contains invalid characters"
        );

        let result_two = matches_datetime(
            &flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::String,
                property: String::from("date"),
                operator: String::from("eq"),
                value: String::from("2006-01-02T15:04:05Z"),
            },
            "blah",
        );

        assert!(result_two.is_err());
        assert_eq!(
            result_two.err().unwrap().to_string(),
            "invalid request: error parsing time blah: input contains invalid characters"
        );
    }

    #[test]
    fn test_entity_id_match() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::EntityId,
                    property: String::from("entityId"),
                    operator: String::from("eq"),
                    value: String::from("user@flipt.io"),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let context: HashMap<String, String> = HashMap::new();

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("user@flipt.io"),
                context,
            },
        );
        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_flag_disabled() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: false,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::FlagDisabled);
        assert_eq!(v.variant_key, String::from(""));
        assert_eq!(v.variant_attachment, None);
    }

    #[test]
    fn test_evaluator_flag_disabled_default_variant() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: false,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: Some(flipt::Variant {
                    id: String::from("1"),
                    key: String::from("default"),
                    attachment: Some(serde_json::json!({"key": "value"}).to_string()),
                }),
            })
        });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::FlagDisabled);
        assert_eq!(v.variant_key, String::from("default"));
        assert_eq!(
            v.variant_attachment,
            Some(serde_json::json!({"key": "value"}).to_string())
        );
    }

    // Segment Match Type ALL
    #[test]
    fn test_evaluator_match_all_no_variants_no_distributions() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );
        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_all_no_distributions_default_variant() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: Some(flipt::Variant {
                    id: String::from("1"),
                    key: String::from("default"),
                    attachment: Some(serde_json::json!({"key": "value"}).to_string()),
                }),
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );
        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
        assert_eq!(v.variant_key, String::from("default"));
        assert_eq!(
            v.variant_attachment,
            Some(serde_json::json!({"key": "value"}).to_string())
        );
    }

    #[test]
    fn test_evaluator_match_all_multiple_segments() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        segments.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::String,
                    property: String::from("company"),
                    operator: String::from("eq"),
                    value: String::from("flipt"),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));
        context.insert(String::from("company"), String::from("flipt"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("boz"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Unknown);
        assert!(v.segment_keys.is_empty());
    }

    #[test]
    fn test_evaluator_match_all_distribution_not_matched() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );
        segments.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::Boolean,
                    property: String::from("admin"),
                    operator: String::from("true"),
                    value: String::from(""),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: None,
                    rollout: 10.0,
                }])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("123"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Unknown);
    }

    #[test]
    fn test_evaluator_match_all_single_variant_distribution() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );
        segments.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::Boolean,
                    property: String::from("admin"),
                    operator: String::from("true"),
                    value: String::from(""),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: Some(String::from(r#"{"foo": "bar"}"#)),
                    rollout: 100.0,
                }])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("123"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(
            v.variant_attachment,
            Some(String::from(r#"{"foo": "bar"}"#))
        );
        assert!(v
            .segment_keys
            .iter()
            .any(|segment_key| segment_key == "segment1"),);
        assert!(v
            .segment_keys
            .iter()
            .any(|segment_key| segment_key == "segment2"),);
    }

    #[test]
    fn test_evaluator_match_all_rollout_distribution() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                ])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("2"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_all_rollout_distribution_multi_rule() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::Boolean,
                        property: String::from("premium_user"),
                        operator: String::from("true"),
                        value: String::from(""),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        let mut segments_two: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments_two.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![
                    flipt::EvaluationRule {
                        id: String::from("1"),
                        flag_key: String::from("foo"),
                        segments: segments.clone(),
                        rank: 1,
                        segment_operator: flipt::SegmentOperator::And,
                    },
                    flipt::EvaluationRule {
                        id: String::from("2"),
                        flag_key: String::from("foo"),
                        segments: segments_two.clone(),
                        rank: 2,
                        segment_operator: flipt::SegmentOperator::And,
                    },
                ])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                ])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("premium_user"), String::from("true"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_all_no_constraints() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::All,
                constraints: vec![],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                ])
            });

        let context: HashMap<String, String> = HashMap::new();

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("10"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("01"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    // Segment Match Type ANY
    #[test]
    fn test_evaluator_match_any_no_variants_no_distributions() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_any_no_distributions_default_variant() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: Some(flipt::Variant {
                    id: String::from("1"),
                    key: String::from("default"),
                    attachment: Some(serde_json::json!({"key": "value"}).to_string()),
                }),
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
        assert_eq!(v.variant_key, String::from("default"));
        assert_eq!(
            v.variant_attachment,
            Some(serde_json::json!({"key": "value"}).to_string())
        );
    }

    #[test]
    fn test_evaluator_match_any_multiple_segments() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );
        segments.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::String,
                    property: String::from("company"),
                    operator: String::from("eq"),
                    value: String::from("flipt"),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("company"), String::from("flipt"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("boz"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("entity"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Unknown);
        assert!(v.segment_keys.is_empty());
    }

    #[test]
    fn test_evaluator_match_any_distribution_not_matched() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );
        segments.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::Boolean,
                    property: String::from("admin"),
                    operator: String::from("true"),
                    value: String::from(""),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: None,
                    rollout: 10.0,
                }])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("123"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Unknown);
    }

    #[test]
    fn test_evaluator_match_any_single_variant_distribution() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );
        segments.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::Boolean,
                    property: String::from("admin"),
                    operator: String::from("true"),
                    value: String::from(""),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: Some(String::from(r#"{"foo": "bar"}"#)),
                    rollout: 100.0,
                }])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("123"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(
            v.variant_attachment,
            Some(String::from(r#"{"foo": "bar"}"#))
        );
    }

    #[test]
    fn test_evaluator_match_any_rollout_distribution() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                ])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("2"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_any_rollout_distribution_multi_rule() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::Boolean,
                        property: String::from("premium_user"),
                        operator: String::from("true"),
                        value: String::from(""),
                    },
                    flipt::EvaluationConstraint {
                        r#type: flipt::ConstraintComparisonType::String,
                        property: String::from("foo"),
                        operator: String::from("eq"),
                        value: String::from("bar"),
                    },
                ],
            },
        );

        let mut segments_two: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments_two.insert(
            String::from("segment2"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment2"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![
                    flipt::EvaluationRule {
                        id: String::from("1"),
                        flag_key: String::from("foo"),
                        segments: segments.clone(),
                        rank: 1,
                        segment_operator: flipt::SegmentOperator::And,
                    },
                    flipt::EvaluationRule {
                        id: String::from("2"),
                        flag_key: String::from("foo"),
                        segments: segments_two.clone(),
                        rank: 2,
                        segment_operator: flipt::SegmentOperator::And,
                    },
                ])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                ])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("premium_user"), String::from("true"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_any_no_constraints() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                ])
            });

        let mut context: HashMap<String, String> = HashMap::new();

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("10"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("01"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        context.insert(String::from("foo"), String::from("bar"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("01"),
                context,
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    // Test cases where rollouts have a zero value
    #[test]
    fn test_evaluator_first_rollout_rule_zero() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::String,
                    property: String::from("bar"),
                    operator: String::from("eq"),
                    value: String::from("baz"),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 0.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 100.0,
                        variant_attachment: None,
                    },
                ])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_multiple_zero_rollout_distributions() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Variant,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::String,
                    property: String::from("bar"),
                    operator: String::from("eq"),
                    value: String::from("baz"),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rules()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRule {
                    id: String::from("1"),
                    flag_key: String::from("foo"),
                    segments: segments.clone(),
                    rank: 1,
                    segment_operator: flipt::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: None,
                        rollout: 0.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 0.0,
                        variant_attachment: None,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant3"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant4"),
                        rollout: 0.0,
                        variant_attachment: None,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant5"),
                        rollout: 0.0,
                        variant_attachment: None,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant6"),
                        rollout: 50.0,
                        variant_attachment: None,
                    },
                ])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));

        let variant = variant_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context: context.clone(),
            },
        );

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, flipt::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant3"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_boolean_notpresent() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: false,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Boolean,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::Boolean,
                    property: String::from("some"),
                    operator: String::from("notpresent"),
                    value: String::from(""),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rollouts()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRollout {
                    rollout_type: flipt::RolloutType::Segment,
                    rank: 1,
                    segment: Some(RolloutSegment {
                        value: true,
                        segment_operator: flipt::SegmentOperator::Or,
                        segments: segments.clone(),
                    }),
                    threshold: None,
                }])
            });

        let boolean = boolean_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context: HashMap::new(),
            },
        );

        assert!(boolean.is_ok());

        let b = boolean.unwrap();

        assert_eq!(b.flag_key, String::from("foo"));
        assert!(b.enabled);
        assert_eq!(b.reason, flipt::EvaluationReason::Match);
    }

    #[test]
    fn test_boolean_present() {
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: false,
                description: Some(String::from("foo flag")),
                r#type: flipt::FlagType::Boolean,
                default_variant: None,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::Boolean,
                    property: String::from("some"),
                    operator: String::from("present"),
                    value: String::from(""),
                }],
            },
        );

        mock_store
            .expect_get_evaluation_rollouts()
            .returning(move |_, _| {
                Some(vec![flipt::EvaluationRollout {
                    rollout_type: flipt::RolloutType::Segment,
                    rank: 1,
                    segment: Some(RolloutSegment {
                        value: true,
                        segment_operator: flipt::SegmentOperator::Or,
                        segments: segments.clone(),
                    }),
                    threshold: None,
                }])
            });

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("some"), String::from("baz"));

        let boolean = boolean_evaluation(
            &mock_store,
            "default",
            &EvaluationRequest {
                flag_key: String::from("foo"),
                entity_id: String::from("1"),
                context,
            },
        );

        assert!(boolean.is_ok());

        let b = boolean.unwrap();

        assert_eq!(b.flag_key, String::from("foo"));
        assert!(b.enabled);
        assert_eq!(b.reason, flipt::EvaluationReason::Match);
    }

    #[test]
    fn test_evaluator_matches_contraints_with_mixed_types() {
        let eval_context: HashMap<String, String> =
            HashMap::from([("fruit".into(), "apple".into())]);
        let constraints = vec![
            flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::Boolean,
                property: String::from("fruit"),
                operator: String::from("true"),
                value: String::from(""),
            },
            flipt::EvaluationConstraint {
                r#type: flipt::ConstraintComparisonType::String,
                property: String::from("fruit"),
                operator: String::from("eq"),
                value: String::from("apple"),
            },
        ];
        let result = matches_constraints(
            &eval_context,
            &constraints,
            &flipt::SegmentMatchType::Any,
            "",
        );
        assert!(result.is_ok());
        assert!(result.unwrap());
    }

    #[test]
    fn test_evaluator_matches_constraint_with_unknown_operator() {
        let eval_context: HashMap<String, String> =
            HashMap::from([("fruit".into(), "appleseed".into())]);
        let constraints = vec![flipt::EvaluationConstraint {
            r#type: flipt::ConstraintComparisonType::String,
            property: String::from("fruit"),
            operator: String::from("xunknownx"),
            value: String::from("apple"),
        }];
        let result = matches_constraints(
            &eval_context,
            &constraints,
            &flipt::SegmentMatchType::Any,
            "",
        );
        assert!(result.is_ok());
        assert!(!result.unwrap());
    }
}
