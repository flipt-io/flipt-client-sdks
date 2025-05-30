use crate::models::flipt;
use crate::models::snapshot::Snapshot;

#[cfg(test)]
use mockall::automock;

#[cfg_attr(test, automock)]
pub trait Store {
    fn list_flags(&self, namespace_key: &str) -> Option<Vec<flipt::Flag>>;
    fn get_flag(&self, namespace_key: &str, flag_key: &str) -> Option<flipt::Flag>;
    fn get_evaluation_rules(
        &self,
        namespace_key: &str,
        flag_key: &str,
    ) -> Option<Vec<flipt::EvaluationRule>>;
    fn get_evaluation_distributions(
        &self,
        namespace_key: &str,
        rule_id: &str,
    ) -> Option<Vec<flipt::EvaluationDistribution>>;
    fn get_evaluation_rollouts(
        &self,
        namespace_key: &str,
        flag_key: &str,
    ) -> Option<Vec<flipt::EvaluationRollout>>;
}

impl Store for Snapshot {
    fn list_flags(&self, namespace_key: &str) -> Option<Vec<flipt::Flag>> {
        if self.namespace.key != namespace_key {
            return None;
        }

        let flags = self.namespace.flags.values().cloned().collect();

        Some(flags)
    }

    fn get_flag(&self, namespace_key: &str, flag_key: &str) -> Option<flipt::Flag> {
        if self.namespace.key != namespace_key {
            return None;
        }

        let flag = self.namespace.flags.get(flag_key)?;

        Some(flag.clone())
    }

    fn get_evaluation_rules(
        &self,
        namespace_key: &str,
        flag_key: &str,
    ) -> Option<Vec<flipt::EvaluationRule>> {
        if self.namespace.key != namespace_key {
            return None;
        }

        let eval_rules = self.namespace.eval_rules.get(flag_key)?;

        Some(eval_rules.to_vec())
    }

    fn get_evaluation_distributions(
        &self,
        namespace_key: &str,
        rule_id: &str,
    ) -> Option<Vec<flipt::EvaluationDistribution>> {
        if self.namespace.key != namespace_key {
            return None;
        }

        let evaluation_distributions = self.namespace.eval_distributions.get(rule_id)?;

        Some(evaluation_distributions.to_vec())
    }

    fn get_evaluation_rollouts(
        &self,
        namespace_key: &str,
        flag_key: &str,
    ) -> Option<Vec<flipt::EvaluationRollout>> {
        if self.namespace.key != namespace_key {
            return None;
        }

        let eval_rollouts = self.namespace.eval_rollouts.get(flag_key)?;

        Some(eval_rollouts.to_vec())
    }
}

#[cfg(test)]
mod tests {
    use std::{fs, path::PathBuf};

    use super::{Snapshot, Store};
    use crate::{error::Error, models::flipt, models::source};

    #[cfg(test)]
    pub struct TestFetcher {
        path: Option<String>,
    }

    #[cfg(test)]
    impl TestFetcher {
        pub fn new() -> Self {
            Self { path: None }
        }
    }

    #[cfg(test)]
    impl TestFetcher {
        fn fetch(&mut self, _: &str) -> Result<Option<source::Document>, Error> {
            let f = match &self.path {
                Some(path) => path.to_owned(),
                None => {
                    let mut d = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
                    d.push("src/testdata/state.json");
                    d.display().to_string()
                }
            };

            let state = fs::read_to_string(f).expect("file should have read correctly");

            let document: source::Document = match serde_json::from_str(&state) {
                Ok(document) => document,
                Err(e) => return Err(Error::InvalidJSON(e.to_string())),
            };

            Ok(Some(document))
        }
    }

    #[test]
    fn test_snapshot() {
        let mut tp = TestFetcher::new();
        let doc = tp.fetch("default").unwrap();

        let snapshot = Snapshot::build("default", doc.unwrap());

        assert_eq!(1, snapshot.version);

        let flag_variant = snapshot
            .get_flag("default", "flag1")
            .expect("flag1 should exist");

        assert_eq!(flag_variant.key, "flag1");
        assert!(flag_variant.enabled);
        assert_eq!(flag_variant.r#type, flipt::FlagType::Variant);

        let flag_boolean = snapshot
            .get_flag("default", "flag_boolean")
            .expect("flag_boolean should exist");

        assert_eq!(flag_boolean.key, "flag_boolean");
        assert!(flag_boolean.enabled);
        assert_eq!(flag_boolean.r#type, flipt::FlagType::Boolean);

        let evaluation_rules = snapshot
            .get_evaluation_rules("default", "flag1")
            .expect("evaluation rules should exist for flag1");

        assert_eq!(evaluation_rules.len(), 1);
        assert_eq!(evaluation_rules[0].flag_key, "flag1");
        assert_eq!(
            evaluation_rules[0].segment_operator,
            flipt::SegmentOperator::Or
        );
        assert_eq!(evaluation_rules[0].rank, 1);
        assert_eq!(evaluation_rules[0].segments.len(), 1);
        assert_eq!(
            *evaluation_rules[0]
                .segments
                .get("segment1")
                .expect("segment1 should exist"),
            flipt::EvaluationSegment {
                segment_key: "segment1".into(),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::String,
                    property: "fizz".into(),
                    operator: "eq".into(),
                    value: "buzz".into(),
                }],
            }
        );

        let evaluation_distributions = snapshot
            .get_evaluation_distributions("default", &evaluation_rules[0].id)
            .expect("evaluation distributions should exists for the rule");
        assert_eq!(evaluation_distributions.len(), 1);
        assert_eq!(evaluation_distributions[0].rollout, 100.0);
        assert_eq!(evaluation_distributions[0].rule_id, evaluation_rules[0].id);
        assert_eq!(evaluation_distributions[0].variant_key, "variant1");

        let evaluation_rollouts = snapshot
            .get_evaluation_rollouts("default", "flag_boolean")
            .expect("evaluation rollouts should exist for flag_boolean");

        assert_eq!(evaluation_rollouts.len(), 2);
        assert_eq!(evaluation_rollouts[0].rank, 1);
        assert_eq!(
            evaluation_rollouts[0].rollout_type,
            flipt::RolloutType::Segment
        );

        let segment_rollout = evaluation_rollouts[0]
            .segment
            .as_ref()
            .expect("first rollout should be segment");

        assert!(segment_rollout.value);
        assert_eq!(segment_rollout.segment_operator, flipt::SegmentOperator::Or);
        assert_eq!(
            *segment_rollout
                .segments
                .get("segment1")
                .expect("segment1 should exist"),
            flipt::EvaluationSegment {
                segment_key: "segment1".into(),
                match_type: flipt::SegmentMatchType::Any,
                constraints: vec![flipt::EvaluationConstraint {
                    r#type: flipt::ConstraintComparisonType::String,
                    property: "fizz".into(),
                    operator: "eq".into(),
                    value: "buzz".into(),
                }],
            }
        );

        let flags = snapshot.list_flags("default").expect("flags should exist");
        assert_eq!(flags.len(), 2);
        let mut found = 0;
        for flag in flags {
            if flag.key == "flag1" {
                assert_eq!(flag.r#type, flipt::FlagType::Variant);
                found += 1;
            } else if flag.key == "flag_boolean" {
                assert_eq!(flag.r#type, flipt::FlagType::Boolean);
                found += 1;
            }
        }
        assert_eq!(found, 2);
    }

    #[test]
    fn test_empty_snapshot() {
        let snapshot = Snapshot::empty("staging");
        assert_eq!(1, snapshot.version);
        let namespace = snapshot.namespace;
        assert_eq!("staging", namespace.key);
        assert_eq!(0, namespace.flags.len());
        assert_eq!(0, namespace.eval_rules.len());
        assert_eq!(0, namespace.eval_distributions.len());
        assert_eq!(0, namespace.eval_rollouts.len());
    }
}
