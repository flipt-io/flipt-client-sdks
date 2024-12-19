package flipt_engine_go

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    *Document
		wantErr error
	}{
		{
			name:    "payload is nil",
			payload: nil,
			want:    nil,
			wantErr: errors.New("payload is nil"),
		},
		{
			name: "successful parsing",
			payload: []byte(`{
								"namespace": {
									"key": "mobile"
								},
								"flags": [
									{
										"key": "mobile_toggle1",
										"name": "mobile_toggle1",
										"description": "",
										"enabled": true,
										"type": "VARIANT_FLAG_TYPE",
										"createdAt": "2024-04-10T06:52:25.301105Z",
										"updatedAt": "2024-04-10T06:52:25.301105Z",
										"rules": [],
										"rollouts": []
									},
									{
										"key": "mobile_toggle2",
										"name": "mobile_toggle2",
										"description": "",
										"enabled": false,
										"type": "VARIANT_FLAG_TYPE",
										"createdAt": "2024-04-10T06:52:34.481445Z",
										"updatedAt": "2024-04-10T07:01:11.573840Z",
										"rules": [
											{
												"id": "6238b8ee-08ee-4dbd-b058-e994402e3f5c",
												"segments": [
													{
														"key": "segment1",
														"name": "",
														"description": "",
														"matchType": "ANY_SEGMENT_MATCH_TYPE",
														"createdAt": null,
														"updatedAt": null,
														"constraints": []
													}
												],
												"rank": 1,
												"segmentOperator": "OR_SEGMENT_OPERATOR",
												"distributions": []
											},
											{
												"id": "4ea8f45f-5855-452b-87a5-df332548e540",
												"segments": [
													{
														"key": "segment2",
														"name": "",
														"description": "",
														"matchType": "ALL_SEGMENT_MATCH_TYPE",
														"createdAt": null,
														"updatedAt": null,
														"constraints": [
															{
																"id": "27bc6bc1-8023-4e0b-9a7a-d73aebf0f201",
																"type": "NUMBER_CONSTRAINT_COMPARISON_TYPE",
																"property": "country_id",
																"operator": "eq",
																"value": "2"
															},
															{
																"id": "7baca777-274a-4934-8f3f-63cfe8dd531e",
																"type": "STRING_CONSTRAINT_COMPARISON_TYPE",
																"property": "city_id",
																"operator": "eq",
																"value": "2"
															}
														]
													}
												],
												"rank": 2,
												"segmentOperator": "OR_SEGMENT_OPERATOR",
												"distributions": []
											},
											{
												"id": "ba65a954-b3b1-4c03-b7c6-a759433cc045",
												"segments": [
													{
														"key": "segment3",
														"name": "",
														"description": "",
														"matchType": "ANY_SEGMENT_MATCH_TYPE",
														"createdAt": null,
														"updatedAt": null,
														"constraints": [
															{
																"id": "9e3b552e-7ac3-4e71-aa78-b39535fddff5",
																"type": "STRING_CONSTRAINT_COMPARISON_TYPE",
																"property": "city_id",
																"operator": "eq",
																"value": "2"
															},
															{
																"id": "c04291f2-53b1-4bfd-bb5b-701a18f02053",
																"type": "NUMBER_CONSTRAINT_COMPARISON_TYPE",
																"property": "country_id",
																"operator": "eq",
																"value": "2"
															}
														]
													}
												],
												"rank": 3,
												"segmentOperator": "OR_SEGMENT_OPERATOR",
												"distributions": [
													{
														"id": "",
														"ruleId": "",
														"variantId": "0790c9bd-ca24-4f37-95e4-b7a0c8c63ec4",
														"variantKey": "variant1",
														"variantAttachment": "21312",
														"rollout": 100
													}
												]
											},
											{
												"id": "70814943-65e1-4729-b750-88641d21a0b6",
												"segments": [
													{
														"key": "segment4",
														"name": "",
														"description": "",
														"matchType": "ALL_SEGMENT_MATCH_TYPE",
														"createdAt": null,
														"updatedAt": null,
														"constraints": [
															{
																"id": "2842abce-2504-4d99-ab35-b32fccc90b37",
																"type": "STRING_CONSTRAINT_COMPARISON_TYPE",
																"property": "country_id",
																"operator": "eq",
																"value": "1"
															},
															{
																"id": "b039cc47-46b3-4bee-98e1-b4f4325b3315",
																"type": "NUMBER_CONSTRAINT_COMPARISON_TYPE",
																"property": "city_id",
																"operator": "eq",
																"value": "1"
															}
														]
													}
												],
												"rank": 4,
												"segmentOperator": "OR_SEGMENT_OPERATOR",
												"distributions": [
													{
														"id": "",
														"ruleId": "",
														"variantId": "0790c9bd-ca24-4f37-95e4-b7a0c8c63ec4",
														"variantKey": "variant1",
														"variantAttachment": "21312",
														"rollout": 49.97
													},
													{
														"id": "",
														"ruleId": "",
														"variantId": "f13da667-57d1-40b1-a033-41e9a82255d1",
														"variantKey": "variant2",
														"variantAttachment": "24123",
														"rollout": 50.03
													}
												]
											}
										],
										"rollouts": []
									},
									{
										"key": "mobile_toggle3",
										"name": "mobile_toggle3",
										"description": "",
										"enabled": true,
										"type": "BOOLEAN_FLAG_TYPE",
										"createdAt": "2024-04-10T07:01:05.596365Z",
										"updatedAt": "2024-04-10T07:01:05.596365Z",
										"rules": [],
										"rollouts": [
											{
												"type": "SEGMENT_ROLLOUT_TYPE",
												"rank": 1,
												"segment": {
													"value": false,
													"segmentOperator": "OR_SEGMENT_OPERATOR",
													"segments": [
														{
															"key": "segment1",
															"name": "",
															"description": "",
															"matchType": "ANY_SEGMENT_MATCH_TYPE",
															"createdAt": null,
															"updatedAt": null,
															"constraints": []
														}
													]
												}
											},
											{
												"type": "SEGMENT_ROLLOUT_TYPE",
												"rank": 2,
												"segment": {
													"value": true,
													"segmentOperator": "OR_SEGMENT_OPERATOR",
													"segments": [
														{
															"key": "segment2",
															"name": "",
															"description": "",
															"matchType": "ALL_SEGMENT_MATCH_TYPE",
															"createdAt": null,
															"updatedAt": null,
															"constraints": [
																{
																	"id": "27bc6bc1-8023-4e0b-9a7a-d73aebf0f201",
																	"type": "NUMBER_CONSTRAINT_COMPARISON_TYPE",
																	"property": "country_id",
																	"operator": "eq",
																	"value": "2"
																},
																{
																	"id": "7baca777-274a-4934-8f3f-63cfe8dd531e",
																	"type": "STRING_CONSTRAINT_COMPARISON_TYPE",
																	"property": "city_id",
																	"operator": "eq",
																	"value": "2"
																}
															]
														}
													]
												}
											},
											{
												"type": "SEGMENT_ROLLOUT_TYPE",
												"rank": 3,
												"segment": {
													"value": false,
													"segmentOperator": "OR_SEGMENT_OPERATOR",
													"segments": [
														{
															"key": "segment3",
															"name": "",
															"description": "",
															"matchType": "ANY_SEGMENT_MATCH_TYPE",
															"createdAt": null,
															"updatedAt": null,
															"constraints": [
																{
																	"id": "9e3b552e-7ac3-4e71-aa78-b39535fddff5",
																	"type": "STRING_CONSTRAINT_COMPARISON_TYPE",
																	"property": "city_id",
																	"operator": "eq",
																	"value": "2"
																},
																{
																	"id": "c04291f2-53b1-4bfd-bb5b-701a18f02053",
																	"type": "NUMBER_CONSTRAINT_COMPARISON_TYPE",
																	"property": "country_id",
																	"operator": "eq",
																		"value": "2"
																	}
																]
															}
														]
													}
												},
												{
													"type": "THRESHOLD_ROLLOUT_TYPE",
													"rank": 4,
													"threshold": {
														"percentage": 42,
														"value": true
													}
												}
											]
										}
									]
								}`),
			want: &Document{
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
								Distributions:   []Distribution{},
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
								Distributions:   []Distribution{},
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
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &HTTPParser{}
			got, err := r.Parse(tt.payload)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
