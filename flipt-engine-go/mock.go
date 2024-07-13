package flipt_engine_go

import (
	"go.uber.org/zap"
)

func SetSnapshotMocksFromBatch(logger *zap.Logger) {
	snapshot := NewSnapshot()
	store := map[string]*EvalNamespace{
		"default": {
			key: "default",
			flags: map[string]*EvalFlag{
				"toggle1": {
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			rules: map[string][]*EvalRule{
				"toggle1": {
					{
						id:              "1",
						rank:            1,
						segmentOperator: orSegmentOperator,
						segments: []*EvalSegment{
							{
								key:       "segment1",
								matchType: anySegmentMatchType,
								constraints: []*EvalConstraint{
									{
										id:       "constraint_id1",
										typ:      stringConstraintComparisonType,
										property: "city_id",
										operator: "eq",
										value:    "1",
									},
								},
							},
						},
					},
				},
			},
		},
		"namespaceWithoutFlags": {
			key:   "namespaceWithoutFlags",
			flags: map[string]*EvalFlag{},
		},
		"mobile": {
			key: "mobile",
			flags: map[string]*EvalFlag{
				"toggle1": {
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
				"toggle2": {
					key:     "toggle2",
					typ:     variantFlagType,
					enabled: true,
				},
				"toggle3": {
					key:     "toggle3",
					typ:     booleanFlagType,
					enabled: true,
				},
				"toggle4": {
					key:     "toggle4",
					typ:     booleanFlagType,
					enabled: false,
				},
				"toggle5": {
					key:     "toggle5",
					typ:     variantFlagType,
					enabled: false,
				},
				"toggle6": {
					key:     "toggle6",
					typ:     variantFlagType,
					enabled: true,
				},
				"toggle7": {
					key:     "toggle7",
					typ:     booleanFlagType,
					enabled: true,
				},
				"toggle8": {
					key:     "toggle8",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			rules: map[string][]*EvalRule{
				"toggle1": {
					{
						id:              "rule_id1",
						rank:            1,
						segmentOperator: orSegmentOperator,
						segments: []*EvalSegment{
							{
								key:       "segment1",
								matchType: anySegmentMatchType,
								constraints: []*EvalConstraint{
									{
										id:       "1",
										typ:      stringConstraintComparisonType,
										property: "city_id",
										operator: "eq",
										value:    "1",
									},
								},
							},
						},
						distributions: []*EvalDistribution{
							{
								ruleID:            "rule_id1",
								rollout:           100,
								variantKey:        "var1",
								variantAttachment: "attach1",
							},
						},
					},
				},
				"toggle2": {
					{
						id:              "rule_id2",
						rank:            1,
						segmentOperator: orSegmentOperator,
						segments: []*EvalSegment{
							{
								key:       "segment2",
								matchType: anySegmentMatchType,
								constraints: []*EvalConstraint{
									{
										id:       "1",
										typ:      numberConstraintComparisonType,
										property: "city_id",
										operator: "eq",
										value:    "1",
									},
								},
							},
						},
						distributions: []*EvalDistribution{
							{
								ruleID:            "rule_id2",
								rollout:           100,
								variantKey:        "var2",
								variantAttachment: "attach2",
							},
						},
					},
				},
				"toggle6": {
					{
						id:              "rule_id3",
						rank:            1,
						segmentOperator: orSegmentOperator,
						segments: []*EvalSegment{
							{
								key:       "segment1",
								matchType: anySegmentMatchType,
								constraints: []*EvalConstraint{
									{
										id:       "constraint_id1",
										typ:      unknownIDConstraintComparisonType,
										property: "city_id",
										operator: "eq",
										value:    "1",
									},
								},
							},
						},
					},
				},
			},
			rollouts: map[string][]*EvalRollout{
				"toggle3": {
					{
						typ:  thresholdRolloutType,
						rank: 1,
						threshold: &EvalThresholdRollout{
							percentage: 50,
							value:      false,
						},
					},
				},
				"toggle7": {
					{
						typ:  segmentRolloutType,
						rank: 1,
						segment: &EvalSegmentRollout{
							value:           false,
							segmentOperator: orSegmentOperator,
							segments: []*EvalSegment{
								{
									key:       "segment1",
									matchType: anySegmentMatchType,
									constraints: []*EvalConstraint{
										{
											id:       "sadasd",
											typ:      stringConstraintComparisonType,
											property: "city_id",
											operator: "eq",
											value:    "1",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	snapshot.replaceStore(store)
	SetEvaluator(snapshot, logger)
}

func SetSnapshotMocksFromListFlags(logger *zap.Logger) {
	snapshot := NewSnapshot()
	store := map[string]*EvalNamespace{
		"default": {
			key: "default",
			flags: map[string]*EvalFlag{
				"toggle1": {
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
				"toggle2": {
					key:     "toggle2",
					typ:     booleanFlagType,
					enabled: false,
				},
			},
		},
		"namespaceWithoutFlags": {
			key:   "namespaceWithoutFlags",
			flags: map[string]*EvalFlag{},
		},
		"mobile": {
			key: "mobile",
			flags: map[string]*EvalFlag{
				"toggle1": {
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
				"toggle2": {
					key:     "toggle2",
					typ:     variantFlagType,
					enabled: false,
				},
				"toggle3": {
					key:     "toggle3",
					typ:     booleanFlagType,
					enabled: true,
				},
				"toggle4": {
					key:     "toggle4",
					typ:     booleanFlagType,
					enabled: false,
				},
			},
		},
	}

	snapshot.replaceStore(store)
	SetEvaluator(snapshot, logger)
}
