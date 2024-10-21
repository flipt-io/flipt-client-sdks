package flipt_engine_go

const (
	booleanFlagType flagType = "BOOLEAN_FLAG_TYPE"
	variantFlagType flagType = "VARIANT_FLAG_TYPE"

	andSegmentOperator segmentOperator = "AND_SEGMENT_OPERATOR"
	orSegmentOperator  segmentOperator = "OR_SEGMENT_OPERATOR"

	allSegmentMatchType segmentMatchType = "ALL_SEGMENT_MATCH_TYPE"
	anySegmentMatchType segmentMatchType = "ANY_SEGMENT_MATCH_TYPE"

	stringConstraintComparisonType    constraintComparisonType = "STRING_CONSTRAINT_COMPARISON_TYPE"
	numberConstraintComparisonType    constraintComparisonType = "NUMBER_CONSTRAINT_COMPARISON_TYPE"
	booleanConstraintComparisonType   constraintComparisonType = "BOOLEAN_CONSTRAINT_COMPARISON_TYPE"
	datetimeConstraintComparisonType  constraintComparisonType = "DATETIME_CONSTRAINT_COMPARISON_TYPE"
	entityIDConstraintComparisonType  constraintComparisonType = "ENTITY_ID_CONSTRAINT_COMPARISON_TYPE"
	unknownIDConstraintComparisonType constraintComparisonType = "UNKNOWN_CONSTRAINT_COMPARISON_TYPE"

	segmentRolloutType   rolloutType = "SEGMENT_ROLLOUT_TYPE"
	thresholdRolloutType rolloutType = "THRESHOLD_ROLLOUT_TYPE"
	unknownRolloutType   rolloutType = "UNKNOWN_ROLLOUT_TYPE"
)

type flagType string

func (r flagType) String() string {
	return string(r)
}

type segmentOperator string

func (r segmentOperator) String() string {
	return string(r)
}

type segmentMatchType string

func (r segmentMatchType) String() string {
	return string(r)
}

type constraintComparisonType string

func (r constraintComparisonType) String() string {
	return string(r)
}

type rolloutType string

func (r rolloutType) String() string {
	return string(r)
}

type Document struct {
	Namespace Namespace `json:"namespace"`
	Flags     []Flag    `json:"flags"`
}

type Namespace struct {
	Key string `json:"key"`
}

type Flag struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Type        flagType  `json:"type"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Rules       []Rule    `json:"rules"`
	Rollouts    []Rollout `json:"rollouts"`
}

type Rule struct {
	ID              string          `json:"id"`
	Segments        []Segment       `json:"segments"`
	Rank            int             `json:"rank"`
	SegmentOperator segmentOperator `json:"segmentOperator"`
	Distributions   []Distribution  `json:"distributions"`
}

type Segment struct {
	Key         string              `json:"key"`
	MatchType   segmentMatchType    `json:"matchType"`
	Constraints []SegmentConstraint `json:"constraints"`
}

type SegmentConstraint struct {
	ID       string                   `json:"id"`
	Type     constraintComparisonType `json:"type"`
	Property string                   `json:"property"`
	Operator string                   `json:"operator"`
	Value    string                   `json:"value"`
}

type Distribution struct {
	ID                string  `json:"id"`
	RuleID            string  `json:"ruleId"`
	VariantID         string  `json:"variantId"`
	VariantKey        string  `json:"variantKey"`
	VariantAttachment string  `json:"variantAttachment"`
	Rollout           float32 `json:"rollout"`
}

type Rollout struct {
	Type      rolloutType    `json:"type"`
	Rank      int            `json:"rank"`
	Segment   SegmentRollout `json:"segment"`
	Threshold Threshold      `json:"threshold"`
}

type SegmentRollout struct {
	Value           bool            `json:"value"`
	SegmentOperator segmentOperator `json:"segmentOperator"`
	Segments        []Segment       `json:"segments"`
}

type Threshold struct {
	Percentage float32 `json:"percentage"`
	Value      bool    `json:"value"`
}
