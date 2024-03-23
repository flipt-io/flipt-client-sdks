use chrono::{DateTime, Utc};
use serde::Serialize;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{Duration, SystemTime, SystemTimeError};

pub mod error;
pub mod models;
pub mod parser;
pub mod store;

use crate::error::Error;
use crate::models::common;
use crate::models::flipt;
use crate::parser::Parser;
use crate::store::{Snapshot, Store};

const DEFAULT_PERCENT: f32 = 100.0;
const DEFAULT_TOTAL_BUCKET_NUMBER: u32 = 1000;
const DEFAULT_PERCENT_MULTIPIER: f32 = DEFAULT_TOTAL_BUCKET_NUMBER as f32 / DEFAULT_PERCENT;

pub struct Evaluator<P, S>
where
    P: Parser + Send,
    S: Store + Send,
{
    namespace: String,
    parser: P,
    store: S,
    mtx: Arc<RwLock<i32>>,
}

#[repr(C)]
pub struct EvaluationRequest {
    pub flag_key: String,
    pub entity_id: String,
    pub context: HashMap<String, String>,
}

#[derive(Serialize)]
pub struct VariantEvaluationResponse {
    pub r#match: bool,
    pub segment_keys: Vec<String>,
    pub reason: common::EvaluationReason,
    pub flag_key: String,
    pub variant_key: String,
    pub variant_attachment: String,
    pub request_duration_millis: f64,
    pub timestamp: DateTime<Utc>,
}

#[derive(Serialize)]
pub struct BooleanEvaluationResponse {
    pub enabled: bool,
    pub flag_key: String,
    pub reason: common::EvaluationReason,
    pub request_duration_millis: f64,
    pub timestamp: DateTime<Utc>,
}

#[derive(Serialize)]
pub struct ErrorEvaluationResponse {
    pub flag_key: String,
    pub namespace_key: String,
    pub reason: common::ErrorEvaluationReason,
}

#[derive(Serialize)]
pub struct BatchEvaluationResponse {
    pub responses: Vec<EvaluationResponse>,
    pub request_duration_millis: f64,
}

#[derive(Serialize)]
pub struct EvaluationResponse {
    pub r#type: common::ResponseType,
    pub boolean_evaluation_response: Option<BooleanEvaluationResponse>,
    pub variant_evaluation_response: Option<VariantEvaluationResponse>,
    pub error_evaluation_response: Option<ErrorEvaluationResponse>,
}

impl Default for VariantEvaluationResponse {
    fn default() -> Self {
        Self {
            r#match: false,
            segment_keys: Vec::new(),
            reason: common::EvaluationReason::Unknown,
            flag_key: String::from(""),
            variant_key: String::from(""),
            variant_attachment: String::from(""),
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
            reason: common::EvaluationReason::Unknown,
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
            reason: common::ErrorEvaluationReason::Unknown,
        }
    }
}

type ListFlagsResult = std::result::Result<Vec<flipt::Flag>, Error>;

type VariantEvaluationResult<T> = std::result::Result<T, Error>;

type BooleanEvaluationResult<T> = std::result::Result<T, Error>;

impl<P> Evaluator<P, Snapshot>
where
    P: Parser + Send,
{
    pub fn new_snapshot_evaluator(namespace: String, parser: P) -> Result<Self, Error> {
        let snap = Snapshot::build(&namespace, &parser)?;
        Ok(Evaluator::new(namespace, parser, snap))
    }

    pub fn replace_snapshot(&mut self) {
        match Snapshot::build(&self.namespace, &self.parser) {
            Ok(s) => {
                self.replace_store(s);
            }
            Err(_) => {
                // TODO: log::error!("error building snapshot: {}", e);
            }
        };
    }
}

impl<P, S> Evaluator<P, S>
where
    P: Parser + Send,
    S: Store + Send,
{
    pub fn new(namespace: String, parser: P, store_impl: S) -> Self {
        Self {
            namespace,
            parser,
            store: store_impl,
            mtx: Arc::new(RwLock::new(0)),
        }
    }

    pub fn replace_store(&mut self, store_impl: S) {
        let _w_lock = self.mtx.write().unwrap();
        self.store = store_impl;
    }

    pub fn list_flags(&self) -> ListFlagsResult {
        let _r_lock = self.mtx.read().unwrap();
        match self.store.list_flags(&self.namespace) {
            Some(f) => Ok(f),
            None => Err(Error::Unknown(format!(
                "failed to get flags for {}",
                self.namespace,
            ))),
        }
    }

    pub fn variant(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> VariantEvaluationResult<VariantEvaluationResponse> {
        let _r_lock = self.mtx.read().unwrap();
        let flag = match self
            .store
            .get_flag(&self.namespace, &evaluation_request.flag_key)
        {
            Some(f) => {
                if f.r#type != common::FlagType::Variant {
                    return Err(Error::InvalidRequest(format!(
                        "{} is not a variant flag",
                        &evaluation_request.flag_key,
                    )));
                }
                f
            }
            None => {
                return Err(Error::InvalidRequest(format!(
                    "failed to get flag information {}/{}",
                    &self.namespace, &evaluation_request.flag_key,
                )));
            }
        };

        self.variant_evaluation(&flag, evaluation_request)
    }

    pub fn boolean(
        &self,
        evaluation_request: &EvaluationRequest,
    ) -> BooleanEvaluationResult<BooleanEvaluationResponse> {
        let _r_lock = self.mtx.read().unwrap();
        let flag = match self
            .store
            .get_flag(&self.namespace, &evaluation_request.flag_key)
        {
            Some(f) => {
                if f.r#type != common::FlagType::Boolean {
                    return Err(Error::InvalidRequest(format!(
                        "{} is not a boolean flag",
                        &evaluation_request.flag_key,
                    )));
                }
                f
            }
            None => {
                return Err(Error::InvalidRequest(format!(
                    "failed to get flag information {}/{}",
                    &self.namespace, &evaluation_request.flag_key,
                )));
            }
        };

        self.boolean_evaluation(&flag, evaluation_request)
    }

    pub fn batch(
        &self,
        requests: Vec<EvaluationRequest>,
    ) -> Result<BatchEvaluationResponse, Error> {
        let now = SystemTime::now();

        let mut evaluation_responses: Vec<EvaluationResponse> = Vec::new();
        for request in requests {
            let flag = match self.store.get_flag(&self.namespace, &request.flag_key) {
                Some(f) => f,
                None => {
                    evaluation_responses.push(EvaluationResponse {
                        r#type: common::ResponseType::Error,
                        boolean_evaluation_response: None,
                        variant_evaluation_response: None,
                        error_evaluation_response: Some(ErrorEvaluationResponse {
                            flag_key: request.flag_key,
                            namespace_key: self.namespace.clone(),
                            reason: common::ErrorEvaluationReason::NotFound,
                        }),
                    });
                    continue;
                }
            };

            match flag.r#type {
                common::FlagType::Boolean => {
                    let boolean_evaluation = self.boolean_evaluation(&flag, &request)?;
                    evaluation_responses.push(EvaluationResponse {
                        r#type: common::ResponseType::Boolean,
                        boolean_evaluation_response: Some(boolean_evaluation),
                        variant_evaluation_response: None,
                        error_evaluation_response: None,
                    });
                }
                common::FlagType::Variant => {
                    let variant_evaluation = self.variant_evaluation(&flag, &request)?;
                    evaluation_responses.push(EvaluationResponse {
                        r#type: common::ResponseType::Variant,
                        boolean_evaluation_response: None,
                        variant_evaluation_response: Some(variant_evaluation),
                        error_evaluation_response: None,
                    });
                }
            }
        }

        Ok(BatchEvaluationResponse {
            responses: evaluation_responses,
            request_duration_millis: get_duration_millis(now.elapsed())?,
        })
    }

    fn variant_evaluation(
        &self,
        flag: &flipt::Flag,
        evaluation_request: &EvaluationRequest,
    ) -> VariantEvaluationResult<VariantEvaluationResponse> {
        let now = SystemTime::now();
        let mut last_rank = 0;

        let mut variant_evaluation_response = VariantEvaluationResponse {
            flag_key: flag.key.clone(),
            ..Default::default()
        };

        if !flag.enabled {
            variant_evaluation_response.reason = common::EvaluationReason::FlagDisabled;
            variant_evaluation_response.request_duration_millis =
                get_duration_millis(now.elapsed())?;
            return Ok(variant_evaluation_response);
        }

        let evaluation_rules = match self
            .store
            .get_evaluation_rules(&self.namespace, &evaluation_request.flag_key)
        {
            Some(rules) => rules,
            None => {
                return Err(Error::Unknown(format!(
                    "error getting evaluation rules for namespace {} and flag {}",
                    self.namespace.clone(),
                    evaluation_request.flag_key.clone()
                )));
            }
        };

        for rule in evaluation_rules {
            if rule.rank < last_rank {
                return Err(Error::InvalidRequest(format!(
                    "rule rank: {} detected out of order",
                    rule.rank
                )));
            }

            last_rank = rule.rank;

            let mut segment_keys: Vec<String> = Vec::new();
            let mut segment_matches = 0;

            for kv in &rule.segments {
                let matched = match self.matches_constraints(
                    &evaluation_request.context,
                    &kv.1.constraints,
                    &kv.1.match_type,
                    &evaluation_request.entity_id,
                ) {
                    Ok(b) => b,
                    Err(err) => return Err(err),
                };

                if matched {
                    segment_keys.push(kv.0.into());
                    segment_matches += 1;
                }
            }

            if rule.segment_operator == common::SegmentOperator::Or {
                if segment_matches < 1 {
                    continue;
                }
            } else if rule.segment_operator == common::SegmentOperator::And
                && rule.segments.len() != segment_matches
            {
                continue;
            }

            variant_evaluation_response.segment_keys = segment_keys;

            let distributions = match self
                .store
                .get_evaluation_distributions(&self.namespace, &rule.id)
            {
                Some(evaluation_distributions) => evaluation_distributions,
                None => {
                    return Err(Error::Unknown(format!(
                        "error getting evaluation distributions for namespace {} and rule {}",
                        self.namespace.clone(),
                        rule.id.clone()
                    )))
                }
            };

            let mut valid_distributions: Vec<flipt::EvaluationDistribution> = Vec::new();
            let mut buckets: Vec<i32> = Vec::new();

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
            if valid_distributions.is_empty() {
                variant_evaluation_response.r#match = true;
                variant_evaluation_response.reason = common::EvaluationReason::Match;
                variant_evaluation_response.request_duration_millis =
                    get_duration_millis(now.elapsed())?;
                return Ok(variant_evaluation_response);
            }

            let bucket = crc32fast::hash(
                format!(
                    "{}{}",
                    evaluation_request.flag_key, evaluation_request.entity_id
                )
                .as_bytes(),
            ) % DEFAULT_TOTAL_BUCKET_NUMBER;

            buckets.sort();

            let index = match buckets.binary_search(&(bucket as i32 + 1)) {
                Ok(idx) => idx,
                Err(idx) => idx,
            };

            if index == valid_distributions.len() {
                variant_evaluation_response.r#match = false;
                variant_evaluation_response.request_duration_millis =
                    get_duration_millis(now.elapsed())?;
                return Ok(variant_evaluation_response);
            }

            let d = &valid_distributions[index];

            variant_evaluation_response.r#match = true;
            variant_evaluation_response.variant_key = d.variant_key.clone();
            variant_evaluation_response.variant_attachment = d.variant_attachment.clone();
            variant_evaluation_response.reason = common::EvaluationReason::Match;
            variant_evaluation_response.request_duration_millis =
                get_duration_millis(now.elapsed())?;
            return Ok(variant_evaluation_response);
        }

        Ok(variant_evaluation_response)
    }

    fn boolean_evaluation(
        &self,
        flag: &flipt::Flag,
        evaluation_request: &EvaluationRequest,
    ) -> BooleanEvaluationResult<BooleanEvaluationResponse> {
        let now = SystemTime::now();
        let mut last_rank = 0;

        let evaluation_rollouts = match self
            .store
            .get_evaluation_rollouts(&self.namespace, &evaluation_request.flag_key)
        {
            Some(rollouts) => rollouts,
            None => {
                return Err(Error::Unknown(format!(
                    "error getting evaluation rollouts for namespace {} and flag {}",
                    self.namespace.clone(),
                    evaluation_request.flag_key.clone()
                )));
            }
        };

        for rollout in evaluation_rollouts {
            if rollout.rank < last_rank {
                return Err(Error::InvalidRequest(format!(
                    "rollout rank: {} detected out of order",
                    rollout.rank
                )));
            }

            last_rank = rollout.rank;

            if rollout.threshold.is_some() {
                let threshold = rollout.threshold.unwrap();

                let normalized_value = (crc32fast::hash(
                    format!(
                        "{}{}",
                        evaluation_request.entity_id, evaluation_request.flag_key
                    )
                    .as_bytes(),
                ) as i32
                    % 100) as f32;

                if normalized_value < threshold.percentage {
                    return Ok(BooleanEvaluationResponse {
                        enabled: threshold.value,
                        flag_key: flag.key.clone(),
                        reason: common::EvaluationReason::Match,
                        request_duration_millis: get_duration_millis(now.elapsed())?,
                        timestamp: chrono::offset::Utc::now(),
                    });
                }
            } else if rollout.segment.is_some() {
                let segment = rollout.segment.unwrap();
                let mut segment_matches = 0;

                for s in &segment.segments {
                    let matched = match self.matches_constraints(
                        &evaluation_request.context,
                        &s.1.constraints,
                        &s.1.match_type,
                        &evaluation_request.entity_id,
                    ) {
                        Ok(v) => v,
                        Err(err) => return Err(err),
                    };

                    if matched {
                        segment_matches += 1;
                    }
                }

                if segment.segment_operator == common::SegmentOperator::Or {
                    if segment_matches < 1 {
                        continue;
                    }
                } else if segment.segment_operator == common::SegmentOperator::And
                    && segment.segments.len() != segment_matches
                {
                    continue;
                }

                return Ok(BooleanEvaluationResponse {
                    enabled: segment.value,
                    flag_key: flag.key.clone(),
                    reason: common::EvaluationReason::Match,
                    request_duration_millis: get_duration_millis(now.elapsed())?,
                    timestamp: chrono::offset::Utc::now(),
                });
            }
        }

        Ok(BooleanEvaluationResponse {
            enabled: flag.enabled,
            flag_key: flag.key.clone(),
            reason: common::EvaluationReason::Default,
            request_duration_millis: get_duration_millis(now.elapsed())?,
            timestamp: chrono::offset::Utc::now(),
        })
    }

    fn matches_constraints(
        &self,
        eval_context: &HashMap<String, String>,
        constraints: &Vec<flipt::EvaluationConstraint>,
        segment_match_type: &common::SegmentMatchType,
        entity_id: &str,
    ) -> Result<bool, Error> {
        let mut constraint_matches: usize = 0;

        for constraint in constraints {
            let value = match eval_context.get(&constraint.property) {
                Some(v) => v,
                None => {
                    // If we have an entityId return dummy value which is an empty string.
                    ""
                }
            };

            let matched = match constraint.r#type {
                common::ConstraintComparisonType::String => matches_string(constraint, value),
                common::ConstraintComparisonType::Number => matches_number(constraint, value)?,
                common::ConstraintComparisonType::Boolean => matches_boolean(constraint, value)?,
                common::ConstraintComparisonType::DateTime => matches_datetime(constraint, value)?,
                common::ConstraintComparisonType::EntityId => matches_string(constraint, entity_id),
                _ => {
                    return Ok(false);
                }
            };

            if matched {
                constraint_matches += 1;

                if segment_match_type == &common::SegmentMatchType::Any {
                    break;
                } else {
                    continue;
                }
            } else if segment_match_type == &common::SegmentMatchType::All {
                break;
            } else {
                continue;
            }
        }

        let is_match = match segment_match_type {
            common::SegmentMatchType::All => constraints.len() == constraint_matches,
            common::SegmentMatchType::Any => constraints.is_empty() || constraint_matches != 0,
        };

        Ok(is_match)
    }
}

fn contains_string(v: &str, values: &str) -> bool {
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
        "isoneof" => contains_string(v, value),
        "isnotoneof" => !contains_string(v, value),
        _ => false,
    }
}

fn contains_number(v: i32, values: &str) -> Result<bool, Error> {
    match serde_json::from_str::<Vec<i32>>(values) {
        Ok(values) => Ok(values.contains(&v)),
        Err(err) => Err(Error::InvalidRequest(format!(
            "error parsing numbers {}: {}",
            values, err
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
            "error parsing number {}: {}",
            v, err
        )))?,
    };

    match operator {
        "isoneof" => {
            return contains_number(v_number, &evaluation_constraint.value);
        }
        "isnotoneof" => {
            return match contains_number(v_number, &evaluation_constraint.value) {
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
            "error parsing boolean {}: {}",
            v, err
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
            "error parsing time {}: {}",
            v, e
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

fn get_duration_millis(elapsed: Result<Duration, SystemTimeError>) -> Result<f64, Error> {
    match elapsed {
        Ok(elapsed) => Ok(elapsed.as_secs_f64() * 1000.0),
        Err(e) => Err(Error::Unknown(format!("error getting duration {}", e))),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::flipt::RolloutSegment;
    use crate::parser::TestParser;
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
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("eq"),
            value: String::from("number"),
        }, "number", true),
        string_neq: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("neq"),
            value: String::from("number"),
        }, "num", true),
        string_prefix: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("prefix"),
            value: String::from("num"),
        }, "number", true),
        string_suffix: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("suffix"),
            value: String::from("ber"),
        }, "number", true),
        string_isoneof_exists: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "2", true),
        string_isoneof_absent: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "3", false),
        string_isnotoneof_exists: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "2", false),
        string_isnotoneof_absent: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::String,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from(r#"["1", "2"]"#),
        }, "3", true),
    }

    matches_datetime_tests! {
        datetime_eq: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("eq"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T15:04:05Z", true),
        datetime_neq: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("neq"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T15:03:05Z", true),
        datetime_lt: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("lt"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T14:03:05Z", true),
        datetime_gt: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("gt"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T16:03:05Z", true),
        datetime_lte: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("lte"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T15:04:05Z", true),
        datetime_gte: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::DateTime,
            property: String::from("date"),
            operator: String::from("gte"),
            value: String::from("2006-01-02T15:04:05Z"),
        }, "2006-01-02T16:03:05Z", true),

    }

    matches_number_tests! {
        number_eq: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("eq"),
            value: String::from("1"),
        }, "1", true),
        number_neq: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("neq"),
            value: String::from("1"),
        }, "0", true),
        number_lt: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("lt"),
            value: String::from("4"),
        }, "3", true),
        number_gt: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("gt"),
            value: String::from("3"),
        }, "4", true),
        number_lte: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("date"),
            operator: String::from("lte"),
            value: String::from("3"),
        }, "3", true),
        number_gte: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("date"),
            operator: String::from("gte"),
            value: String::from("3"),
        }, "4", true),
        number_isoneof_exists: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from("[1, 2]"),
        }, "2", true),
        number_isoneof_absent: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isoneof"),
            value: String::from("[1, 2]"),
        }, "3", false),
        number_isnotoneof_exists: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from("[1, 2]"),
        }, "2", false),
        number_isnotoneof_absent: (&flipt::EvaluationConstraint{
            r#type: common::ConstraintComparisonType::Number,
            property: String::from("number"),
            operator: String::from("isnotoneof"),
            value: String::from("[1, 2]"),
        }, "3", true),
    }

    #[test]
    fn test_matches_boolean_success() {
        let value_one = matches_boolean(
            &flipt::EvaluationConstraint {
                r#type: common::ConstraintComparisonType::Boolean,
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
                r#type: common::ConstraintComparisonType::Boolean,
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
                r#type: common::ConstraintComparisonType::Boolean,
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
                r#type: common::ConstraintComparisonType::Number,
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
                r#type: common::ConstraintComparisonType::Number,
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
                r#type: common::ConstraintComparisonType::String,
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
                r#type: common::ConstraintComparisonType::String,
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
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::EntityId,
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
                    segment_operator: common::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let context: HashMap<String, String> = HashMap::new();

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("user@flipt.io"),
            context,
        });
        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    // Segment Match Type ALL
    #[test]
    fn test_evaluator_match_all_no_variants_no_distributions() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("entity"),
            context,
        });
        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_all_multiple_segments() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));
        context.insert(String::from("company"), String::from("flipt"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("entity"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("boz"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("entity"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Unknown);
        assert!(v.segment_keys.is_empty());
    }

    #[test]
    fn test_evaluator_match_all_distribution_not_matched() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::Boolean,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: String::from(""),
                    rollout: 10.0,
                }])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("123"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Unknown);
    }

    #[test]
    fn test_evaluator_match_all_single_variant_distribution() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::All,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::Boolean,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: String::from(r#"{"foo": "bar"}"#),
                    rollout: 100.0,
                }])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("123"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.variant_attachment, String::from(r#"{"foo": "bar"}"#));
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
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: String::from(""),
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("2"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_all_rollout_distribution_multi_rule() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::Boolean,
                        property: String::from("premium_user"),
                        operator: String::from("true"),
                        value: String::from(""),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::All,
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
                        segment_operator: common::SegmentOperator::And,
                    },
                    flipt::EvaluationRule {
                        id: String::from("2"),
                        flag_key: String::from("foo"),
                        segments: segments_two.clone(),
                        rank: 2,
                        segment_operator: common::SegmentOperator::And,
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
                        variant_attachment: String::from(""),
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("premium_user"), String::from("true"));
        context.insert(String::from("foo"), String::from("bar"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_all_no_constraints() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::All,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: String::from(""),
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let context: HashMap<String, String> = HashMap::new();

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("10"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("01"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    // Segment Match Type ANY
    #[test]
    fn test_evaluator_match_any_no_variants_no_distributions() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::Or,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("entity"),
            context,
        });
        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_any_multiple_segments() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| Some(vec![]));

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("company"), String::from("flipt"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("entity"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);

        let mut context: HashMap<String, String> = HashMap::new();
        context.insert(String::from("bar"), String::from("boz"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("entity"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Unknown);
        assert!(v.segment_keys.is_empty());
    }

    #[test]
    fn test_evaluator_match_any_distribution_not_matched() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::Boolean,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: String::from(""),
                    rollout: 10.0,
                }])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("123"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(!v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Unknown);
    }

    #[test]
    fn test_evaluator_match_any_single_variant_distribution() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::Boolean,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![flipt::EvaluationDistribution {
                    rule_id: String::from("1"),
                    variant_key: String::from("variant1"),
                    variant_attachment: String::from(r#"{"foo": "bar"}"#),
                    rollout: 100.0,
                }])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));
        context.insert(String::from("admin"), String::from("true"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("123"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.variant_attachment, String::from(r#"{"foo": "bar"}"#));
    }

    #[test]
    fn test_evaluator_match_any_rollout_distribution() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
                        property: String::from("bar"),
                        operator: String::from("eq"),
                        value: String::from("baz"),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: String::from(""),
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("2"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_any_rollout_distribution_multi_rule() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::Boolean,
                        property: String::from("premium_user"),
                        operator: String::from("true"),
                        value: String::from(""),
                    },
                    flipt::EvaluationConstraint {
                        r#type: common::ConstraintComparisonType::String,
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
                match_type: common::SegmentMatchType::Any,
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
                        segment_operator: common::SegmentOperator::And,
                    },
                    flipt::EvaluationRule {
                        id: String::from("2"),
                        flag_key: String::from("foo"),
                        segments: segments_two.clone(),
                        rank: 2,
                        segment_operator: common::SegmentOperator::And,
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
                        variant_attachment: String::from(""),
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("premium_user"), String::from("true"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_match_any_no_constraints() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: String::from(""),
                        rollout: 50.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("10"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant1"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("01"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);

        context.insert(String::from("foo"), String::from("bar"));
        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("01"),
            context,
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    // Test cases where rollouts have a zero value
    #[test]
    fn test_evaluator_first_rollout_rule_zero() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: String::from(""),
                        rollout: 0.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 100.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant2"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_evaluator_multiple_zero_rollout_distributions() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: true,
                r#type: common::FlagType::Variant,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::String,
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
                    segment_operator: common::SegmentOperator::And,
                }])
            });

        mock_store
            .expect_get_evaluation_distributions()
            .returning(|_, _| {
                Some(vec![
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant1"),
                        variant_attachment: String::from(""),
                        rollout: 0.0,
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant2"),
                        rollout: 0.0,
                        variant_attachment: String::from(""),
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant3"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant4"),
                        rollout: 0.0,
                        variant_attachment: String::from(""),
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant5"),
                        rollout: 0.0,
                        variant_attachment: String::from(""),
                    },
                    flipt::EvaluationDistribution {
                        rule_id: String::from("1"),
                        variant_key: String::from("variant6"),
                        rollout: 50.0,
                        variant_attachment: String::from(""),
                    },
                ])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("bar"), String::from("baz"));

        let variant = evaluator.variant(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: context.clone(),
        });

        assert!(variant.is_ok());

        let v = variant.unwrap();

        assert_eq!(v.flag_key, String::from("foo"));
        assert!(v.r#match);
        assert_eq!(v.reason, common::EvaluationReason::Match);
        assert_eq!(v.variant_key, String::from("variant3"));
        assert_eq!(v.segment_keys, vec![String::from("segment1")]);
    }

    #[test]
    fn test_boolean_notpresent() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: false,
                r#type: common::FlagType::Boolean,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::Boolean,
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
                    rollout_type: common::RolloutType::Segment,
                    rank: 1,
                    segment: Some(RolloutSegment {
                        value: true,
                        segment_operator: common::SegmentOperator::Or,
                        segments: segments.clone(),
                    }),
                    threshold: None,
                }])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let boolean = evaluator.boolean(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: HashMap::new(),
        });

        assert!(boolean.is_ok());

        let b = boolean.unwrap();

        assert_eq!(b.flag_key, String::from("foo"));
        assert!(b.enabled);
        assert_eq!(b.reason, common::EvaluationReason::Match);
    }

    #[test]
    fn test_boolean_present() {
        let test_parser = TestParser::new();
        let mut mock_store = MockStore::new();

        mock_store.expect_get_flag().returning(|_, _| {
            Some(flipt::Flag {
                key: String::from("foo"),
                enabled: false,
                r#type: common::FlagType::Boolean,
            })
        });

        let mut segments: HashMap<String, flipt::EvaluationSegment> = HashMap::new();
        segments.insert(
            String::from("segment1"),
            flipt::EvaluationSegment {
                segment_key: String::from("segment1"),
                match_type: common::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: common::ConstraintComparisonType::Boolean,
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
                    rollout_type: common::RolloutType::Segment,
                    rank: 1,
                    segment: Some(RolloutSegment {
                        value: true,
                        segment_operator: common::SegmentOperator::Or,
                        segments: segments.clone(),
                    }),
                    threshold: None,
                }])
            });

        let evaluator = &Evaluator {
            namespace: "default".into(),
            parser: test_parser,
            store: mock_store,
            mtx: Arc::new(RwLock::new(0)),
        };

        let mut context: HashMap<String, String> = HashMap::new();

        context.insert(String::from("some"), String::from("baz"));

        let boolean = evaluator.boolean(&EvaluationRequest {
            flag_key: String::from("foo"),
            entity_id: String::from("1"),
            context: context,
        });

        assert!(boolean.is_ok());

        let b = boolean.unwrap();

        assert_eq!(b.flag_key, String::from("foo"));
        assert!(b.enabled);
        assert_eq!(b.reason, common::EvaluationReason::Match);
    }
}
