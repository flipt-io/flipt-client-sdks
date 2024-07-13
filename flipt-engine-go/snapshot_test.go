package flipt_engine_go

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeSnapshot(t *testing.T) {
	tests := []struct {
		name string
		doc  *Document
		want map[string]*EvalNamespace
	}{
		{
			name: "successfull parsing internal struct",
			doc: &Document{
				Namespace: Namespace{
					Key: "mobile",
				},
				Flags: []Flag{
					{
						Key:         "mobile_toggle1",
						Name:        "mobile_toggle1",
						Type:        variantFlagType,
						Description: "",
						Enabled:     true,
						Rules:       []Rule{},
						Rollouts:    []Rollout{},
					},
					{
						Key:         "mobile_toggle2",
						Name:        "mobile_toggle2",
						Type:        variantFlagType,
						Description: "",
						Enabled:     false,
						Rules: []Rule{
							{
								ID: "6238b8ee-08ee-4dbd-b058-e994402e3f5c",
								Segments: []Segment{
									{
										Key:         "segment1",
										MatchType:   anySegmentMatchType,
										Constraints: []SegmentConstraint{},
									},
								},
								Rank:            1,
								SegmentOperator: orSegmentOperator,
								Distributions:   nil,
							},
							{
								ID: "4ea8f45f-5855-452b-87a5-df332548e540",
								Segments: []Segment{
									{
										Key:       "segment2",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "27bc6bc1-8023-4e0b-9a7a-d73aebf0f201",
												Type:     numberConstraintComparisonType,
												Property: "country_id",
												Operator: "eq",
												Value:    "2",
											},
											{
												ID:       "7baca777-274a-4934-8f3f-63cfe8dd531e",
												Type:     stringConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "2",
											},
										},
									},
								},
								Rank:            2,
								SegmentOperator: orSegmentOperator,
								Distributions:   nil,
							},
							{
								ID: "ba65a954-b3b1-4c03-b7c6-a759433cc045",
								Segments: []Segment{
									{
										Key:       "segment3",
										MatchType: anySegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "9e3b552e-7ac3-4e71-aa78-b39535fddff5",
												Type:     stringConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "2",
											},
											{
												ID:       "c04291f2-53b1-4bfd-bb5b-701a18f02053",
												Type:     numberConstraintComparisonType,
												Property: "country_id",
												Operator: "eq",
												Value:    "2",
											},
										},
									},
								},
								Rank:            3,
								SegmentOperator: orSegmentOperator,
								Distributions: []Distribution{
									{
										ID:                "",
										RuleID:            "",
										VariantID:         "0790c9bd-ca24-4f37-95e4-b7a0c8c63ec4",
										VariantKey:        "variant1",
										VariantAttachment: "21312",
										Rollout:           100,
									},
								},
							},
							{
								ID: "70814943-65e1-4729-b750-88641d21a0b6",
								Segments: []Segment{
									{
										Key:       "segment4",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "2842abce-2504-4d99-ab35-b32fccc90b37",
												Type:     stringConstraintComparisonType,
												Property: "country_id",
												Operator: "eq",
												Value:    "1",
											},
											{
												ID:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
												Type:     numberConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "1",
											},
										},
									},
								},
								Rank:            4,
								SegmentOperator: orSegmentOperator,
								Distributions: []Distribution{
									{
										ID:                "",
										RuleID:            "",
										VariantID:         "0790c9bd-ca24-4f37-95e4-b7a0c8c63ec4",
										VariantKey:        "variant1",
										VariantAttachment: "21312",
										Rollout:           49.97,
									},
									{
										ID:                "",
										RuleID:            "",
										VariantID:         "f13da667-57d1-40b1-a033-41e9a82255d1",
										VariantKey:        "variant2",
										VariantAttachment: "24123",
										Rollout:           50.03,
									},
								},
							},
						},
						Rollouts: []Rollout{},
					},
					{
						Key:         "mobile_toggle3",
						Name:        "mobile_toggle3",
						Type:        booleanFlagType,
						Description: "",
						Enabled:     true,
						Rules:       []Rule{},
						Rollouts: []Rollout{
							{
								Type: segmentRolloutType,
								Rank: 1,
								Segment: SegmentRollout{
									Value:           false,
									SegmentOperator: orSegmentOperator,
									Segments: []Segment{
										{
											Key:         "segment1",
											MatchType:   anySegmentMatchType,
											Constraints: []SegmentConstraint{},
										},
									},
								},
								Threshold: Threshold{},
							},
							{
								Type: segmentRolloutType,
								Rank: 2,
								Segment: SegmentRollout{
									Value:           true,
									SegmentOperator: orSegmentOperator,
									Segments: []Segment{
										{
											Key:       "segment2",
											MatchType: allSegmentMatchType,
											Constraints: []SegmentConstraint{
												{
													ID:       "27bc6bc1-8023-4e0b-9a7a-d73aebf0f201",
													Type:     numberConstraintComparisonType,
													Property: "country_id",
													Operator: "eq",
													Value:    "2",
												},
												{
													ID:       "7baca777-274a-4934-8f3f-63cfe8dd531e",
													Type:     stringConstraintComparisonType,
													Property: "city_id",
													Operator: "eq",
													Value:    "2",
												},
											},
										},
									},
								},
								Threshold: Threshold{},
							},
							{
								Type: segmentRolloutType,
								Rank: 3,
								Segment: SegmentRollout{
									Value:           false,
									SegmentOperator: orSegmentOperator,
									Segments: []Segment{
										{
											Key:       "segment3",
											MatchType: anySegmentMatchType,
											Constraints: []SegmentConstraint{
												{
													ID:       "9e3b552e-7ac3-4e71-aa78-b39535fddff5",
													Type:     stringConstraintComparisonType,
													Property: "city_id",
													Operator: "eq",
													Value:    "2",
												},
												{
													ID:       "c04291f2-53b1-4bfd-bb5b-701a18f02053",
													Type:     numberConstraintComparisonType,
													Property: "country_id",
													Operator: "eq",
													Value:    "2",
												},
											},
										},
									},
								},
								Threshold: Threshold{},
							},
							{
								Type:    thresholdRolloutType,
								Rank:    4,
								Segment: SegmentRollout{},
								Threshold: Threshold{
									Percentage: 42,
									Value:      true,
								},
							},
						},
					},
				},
			},
			want: map[string]*EvalNamespace{
				"mobile": {
					key: "mobile",
					flags: map[string]*EvalFlag{
						"mobile_toggle1": {
							key:     "mobile_toggle1",
							typ:     variantFlagType,
							enabled: true,
						},
						"mobile_toggle2": {
							key:     "mobile_toggle2",
							typ:     variantFlagType,
							enabled: false,
						},
						"mobile_toggle3": {
							key:     "mobile_toggle3",
							typ:     booleanFlagType,
							enabled: true,
						},
					},
					rules: map[string][]*EvalRule{
						"mobile_toggle2": {
							{
								id:              "6238b8ee-08ee-4dbd-b058-e994402e3f5c",
								rank:            1,
								segmentOperator: orSegmentOperator,
								segments: []*EvalSegment{
									{
										key:         "segment1",
										matchType:   anySegmentMatchType,
										constraints: []*EvalConstraint{},
									},
								},
								distributions: []*EvalDistribution{},
							},
							{
								id:              "4ea8f45f-5855-452b-87a5-df332548e540",
								rank:            2,
								segmentOperator: orSegmentOperator,
								segments: []*EvalSegment{
									{
										key:       "segment2",
										matchType: allSegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "27bc6bc1-8023-4e0b-9a7a-d73aebf0f201",
												typ:      numberConstraintComparisonType,
												property: "country_id",
												operator: "eq",
												value:    "2",
											},
											{
												id:       "7baca777-274a-4934-8f3f-63cfe8dd531e",
												typ:      stringConstraintComparisonType,
												property: "city_id",
												operator: "eq",
												value:    "2",
											},
										},
									},
								},
								distributions: []*EvalDistribution{},
							},
							{
								id:              "ba65a954-b3b1-4c03-b7c6-a759433cc045",
								rank:            3,
								segmentOperator: orSegmentOperator,
								segments: []*EvalSegment{
									{
										key:       "segment3",
										matchType: anySegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "9e3b552e-7ac3-4e71-aa78-b39535fddff5",
												typ:      stringConstraintComparisonType,
												property: "city_id",
												operator: "eq",
												value:    "2",
											},
											{
												id:       "c04291f2-53b1-4bfd-bb5b-701a18f02053",
												typ:      numberConstraintComparisonType,
												property: "country_id",
												operator: "eq",
												value:    "2",
											},
										},
									},
								},
								distributions: []*EvalDistribution{
									{
										ruleID:            "ba65a954-b3b1-4c03-b7c6-a759433cc045",
										rollout:           100,
										variantKey:        "variant1",
										variantAttachment: "21312",
									},
								},
							},
							{
								id:              "70814943-65e1-4729-b750-88641d21a0b6",
								rank:            4,
								segmentOperator: orSegmentOperator,
								segments: []*EvalSegment{
									{
										key:       "segment4",
										matchType: allSegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
												typ:      stringConstraintComparisonType,
												property: "country_id",
												operator: "eq",
												value:    "1",
											},
											{
												id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
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
										ruleID:            "70814943-65e1-4729-b750-88641d21a0b6",
										rollout:           49.97,
										variantKey:        "variant1",
										variantAttachment: "21312",
									},
									{
										ruleID:            "70814943-65e1-4729-b750-88641d21a0b6",
										rollout:           50.03,
										variantKey:        "variant2",
										variantAttachment: "24123",
									},
								},
							},
						},
					},
					rollouts: map[string][]*EvalRollout{
						"mobile_toggle3": {
							{
								typ:  segmentRolloutType,
								rank: 1,
								segment: &EvalSegmentRollout{
									value:           false,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:         "segment1",
											matchType:   anySegmentMatchType,
											constraints: []*EvalConstraint{},
										},
									},
								},
								threshold: nil,
							},
							{
								typ:  segmentRolloutType,
								rank: 2,
								segment: &EvalSegmentRollout{
									value:           true,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:       "segment2",
											matchType: allSegmentMatchType,
											constraints: []*EvalConstraint{
												{
													id:       "27bc6bc1-8023-4e0b-9a7a-d73aebf0f201",
													typ:      numberConstraintComparisonType,
													property: "country_id",
													operator: "eq",
													value:    "2",
												},
												{
													id:       "7baca777-274a-4934-8f3f-63cfe8dd531e",
													typ:      stringConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "2",
												},
											},
										},
									},
								},
								threshold: nil,
							},
							{
								typ:  segmentRolloutType,
								rank: 3,
								segment: &EvalSegmentRollout{
									value:           false,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:       "segment3",
											matchType: anySegmentMatchType,
											constraints: []*EvalConstraint{
												{
													id:       "9e3b552e-7ac3-4e71-aa78-b39535fddff5",
													typ:      stringConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "2",
												},
												{
													id:       "c04291f2-53b1-4bfd-bb5b-701a18f02053",
													typ:      numberConstraintComparisonType,
													property: "country_id",
													operator: "eq",
													value:    "2",
												},
											},
										},
									},
								},
								threshold: nil,
							},
							{
								typ:     thresholdRolloutType,
								rank:    4,
								segment: nil,
								threshold: &EvalThresholdRollout{
									percentage: 42,
									value:      true,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "check duplicate",
			doc: &Document{
				Namespace: Namespace{
					Key: "mobile",
				},
				Flags: []Flag{
					{
						Key:         "mobile_toggle2",
						Name:        "mobile_toggle2",
						Type:        variantFlagType,
						Description: "",
						Enabled:     true,
						Rules: []Rule{
							{
								ID: "6238b8ee-08ee-4dbd-b058-e994402e3f5c",
								Segments: []Segment{
									{
										Key:       "ios_platform",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id1",
												Type:     stringConstraintComparisonType,
												Property: "platform",
												Operator: "eq",
												Value:    "ios",
											},
										},
									},
									{
										Key:       "vdk",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id2",
												Type:     numberConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "1",
											},
										},
									},
								},
								Rank:            1,
								SegmentOperator: andSegmentOperator,
								Distributions:   nil,
							},
							{
								ID: "6238b8ee-08ee-4dbd-b058-e994402e3f5c",
								Segments: []Segment{
									{
										Key:       "ios_platform",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id1",
												Type:     stringConstraintComparisonType,
												Property: "platform",
												Operator: "eq",
												Value:    "ios",
											},
										},
									},
									{
										Key:       "vdk",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id2",
												Type:     numberConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "1",
											},
										},
									},
								},
								Rank:            1,
								SegmentOperator: andSegmentOperator,
								Distributions:   nil,
							},
							{
								ID: "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
								Segments: []Segment{
									{
										Key:       "moscow",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id3",
												Type:     numberConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "680",
											},
										},
									},
									{
										Key:       "russia",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id4",
												Type:     numberConstraintComparisonType,
												Property: "country_id",
												Operator: "eq",
												Value:    "1",
											},
										},
									},
									{
										Key:       "ios_platform",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id5",
												Type:     stringConstraintComparisonType,
												Property: "platform",
												Operator: "eq",
												Value:    "ios",
											},
										},
									},
								},
								Rank:            2,
								SegmentOperator: andSegmentOperator,
								Distributions: []Distribution{
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
								},
							},
							{
								ID: "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
								Segments: []Segment{
									{
										Key:       "moscow",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id3",
												Type:     numberConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "680",
											},
										},
									},
									{
										Key:       "russia",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id4",
												Type:     numberConstraintComparisonType,
												Property: "country_id",
												Operator: "eq",
												Value:    "1",
											},
										},
									},
									{
										Key:       "ios_platform",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id5",
												Type:     stringConstraintComparisonType,
												Property: "platform",
												Operator: "eq",
												Value:    "ios",
											},
										},
									},
								},
								Rank:            2,
								SegmentOperator: andSegmentOperator,
								Distributions: []Distribution{
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
								},
							},
							{
								ID: "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
								Segments: []Segment{
									{
										Key:       "moscow",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id3",
												Type:     numberConstraintComparisonType,
												Property: "city_id",
												Operator: "eq",
												Value:    "680",
											},
										},
									},
									{
										Key:       "russia",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id4",
												Type:     numberConstraintComparisonType,
												Property: "country_id",
												Operator: "eq",
												Value:    "1",
											},
										},
									},
									{
										Key:       "ios_platform",
										MatchType: allSegmentMatchType,
										Constraints: []SegmentConstraint{
											{
												ID:       "constraint_id5",
												Type:     stringConstraintComparisonType,
												Property: "platform",
												Operator: "eq",
												Value:    "ios",
											},
										},
									},
								},
								Rank:            2,
								SegmentOperator: andSegmentOperator,
								Distributions: []Distribution{
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id1",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id1",
										VariantKey:        "varkey1",
										VariantAttachment: "varattach1",
										Rollout:           33.34,
									},
									{
										ID:                "distribuiton_id2",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id2",
										VariantKey:        "varkey2",
										VariantAttachment: "varattach2",
										Rollout:           33.33,
									},
									{
										ID:                "distribuiton_id3",
										RuleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										VariantID:         "variant_id3",
										VariantKey:        "varkey3",
										VariantAttachment: "varattach3",
										Rollout:           33.33,
									},
								},
							},
						},
						Rollouts: []Rollout{},
					},
				},
			},
			want: map[string]*EvalNamespace{
				"mobile": {
					key: "mobile",
					flags: map[string]*EvalFlag{
						"mobile_toggle2": {
							key:     "mobile_toggle2",
							typ:     variantFlagType,
							enabled: true,
						},
					},
					rules: map[string][]*EvalRule{
						"mobile_toggle2": {
							{
								id:              "6238b8ee-08ee-4dbd-b058-e994402e3f5c",
								rank:            1,
								segmentOperator: andSegmentOperator,
								segments: []*EvalSegment{
									{
										key:       "ios_platform",
										matchType: allSegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "constraint_id1",
												typ:      stringConstraintComparisonType,
												property: "platform",
												operator: "eq",
												value:    "ios",
											},
										},
									},
									{
										key:       "vdk",
										matchType: allSegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "constraint_id2",
												typ:      numberConstraintComparisonType,
												property: "city_id",
												operator: "eq",
												value:    "1",
											},
										},
									},
								},
								distributions: []*EvalDistribution{},
							},
							{
								id:              "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
								rank:            2,
								segmentOperator: andSegmentOperator,
								segments: []*EvalSegment{
									{
										key:       "moscow",
										matchType: allSegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "constraint_id3",
												typ:      numberConstraintComparisonType,
												property: "city_id",
												operator: "eq",
												value:    "680",
											},
										},
									},
									{
										key:       "russia",
										matchType: allSegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "constraint_id4",
												typ:      numberConstraintComparisonType,
												property: "country_id",
												operator: "eq",
												value:    "1",
											},
										},
									},
									{
										key:       "ios_platform",
										matchType: allSegmentMatchType,
										constraints: []*EvalConstraint{
											{
												id:       "constraint_id5",
												typ:      stringConstraintComparisonType,
												property: "platform",
												operator: "eq",
												value:    "ios",
											},
										},
									},
								},
								distributions: []*EvalDistribution{
									{
										ruleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										rollout:           33.34,
										variantKey:        "varkey1",
										variantAttachment: "varattach1",
									},
									{
										ruleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										rollout:           33.33,
										variantKey:        "varkey2",
										variantAttachment: "varattach2",
									},
									{
										ruleID:            "591dd9ab-9c3e-4bda-a5ef-77315d2d2eea",
										rollout:           33.33,
										variantKey:        "varkey3",
										variantAttachment: "varattach3",
									},
								},
							},
						},
					},
					rollouts: map[string][]*EvalRollout{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := NewSnapshot()
			got := snapshot.makeSnapshot(tt.doc)
			assert.Equal(t, tt.want, got)
		})
	}
}
