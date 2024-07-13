package flipt_engine_go

type EvalNamespace struct {
	key      string
	flags    map[string]*EvalFlag
	rules    map[string][]*EvalRule
	rollouts map[string][]*EvalRollout
}

type EvalFlag struct {
	key     string
	typ     flagType
	enabled bool
}

type EvalRule struct {
	id string
	// flagkey ?
	rank            int
	segmentOperator segmentOperator
	segments        []*EvalSegment
	distributions   []*EvalDistribution
}

type EvalSegment struct {
	key         string
	matchType   segmentMatchType
	constraints []*EvalConstraint
}

type EvalConstraint struct {
	id       string
	typ      constraintComparisonType
	property string
	operator string
	value    string
}

type EvalRollout struct {
	typ       rolloutType
	rank      int
	segment   *EvalSegmentRollout
	threshold *EvalThresholdRollout
}

type EvalSegmentRollout struct {
	value           bool
	segmentOperator segmentOperator
	segments        []*EvalSegment
}

type EvalThresholdRollout struct {
	percentage float32
	value      bool
}

type EvalDistribution struct {
	ruleID            string
	rollout           float32
	variantKey        string
	variantAttachment string
}
