use serde::Deserialize;

use super::flipt;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Document {
    pub namespace: Namespace,
    pub flags: Vec<Flag>,
}

impl Default for Document {
    fn default() -> Self {
        Self {
            namespace: Namespace {
                key: "".into(),
                name: None,
            },
            flags: Vec::new(),
        }
    }
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Namespace {
    pub key: String,
    pub name: Option<String>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Flag {
    pub key: String,
    pub name: String,
    pub r#type: Option<flipt::FlagType>,
    pub description: Option<String>,
    pub enabled: bool,
    pub rules: Option<Vec<Rule>>,
    pub rollouts: Option<Vec<Rollout>>,
    pub default_variant: Option<Variant>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Variant {
    pub id: String,
    pub key: String,
    pub attachment: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Rule {
    pub distributions: Vec<Distribution>,
    pub segments: Option<Vec<Segment>>,
    pub segment_operator: flipt::SegmentOperator,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Distribution {
    pub variant_key: String,
    pub rollout: f32,
    pub variant_attachment: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Rollout {
    pub description: Option<String>,
    pub segment: Option<SegmentRule>,
    pub threshold: Option<Threshold>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SegmentRule {
    pub segment_operator: Option<flipt::SegmentOperator>,
    pub value: bool,
    pub segments: Vec<Segment>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Threshold {
    pub percentage: f32,
    pub value: bool,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Segment {
    pub key: String,
    pub match_type: flipt::SegmentMatchType,
    pub constraints: Vec<SegmentConstraint>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SegmentConstraint {
    pub r#type: flipt::ConstraintComparisonType,
    pub property: String,
    pub operator: String,
    pub value: String,
}

#[cfg(test)]
mod tests {
    use crate::models::flipt::ConstraintComparisonType;

    use super::SegmentConstraint;

    #[test]
    fn test_deserialize_constraint_comparison_type_other_value() {
        let json = r#"{"type":"OTHER_CONSTRAINT_COMPARISON_TYPE","property":"this","operator":"eq","value":"something"}"#;
        let constraint: SegmentConstraint = serde_json::from_str(json).unwrap();

        assert_eq!(ConstraintComparisonType::Unknown, constraint.r#type);
    }
}
