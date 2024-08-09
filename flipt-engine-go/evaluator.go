package flipt_engine_go

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	evaluator   *Evaluator
	evaluatorMu sync.RWMutex
)

type Evaluator struct {
	snapshot *Snapshot
	logger   *zap.Logger
}

func SetEvaluator(snapshot *Snapshot, logger *zap.Logger) {
	evaluatorMu.Lock()
	evaluator = &Evaluator{
		snapshot: snapshot,
		logger:   logger,
	}
	evaluatorMu.Unlock()
}

func GetEvaluator() *Evaluator {
	evaluatorMu.RLock()
	eval := evaluator
	evaluatorMu.RUnlock()

	return eval
}

const (
	opEQ         = "eq"
	opNEQ        = "neq"
	opLT         = "lt"
	opLTE        = "lte"
	opGT         = "gt"
	opGTE        = "gte"
	opEmpty      = "empty"
	opNotEmpty   = "notempty"
	opTrue       = "true"
	opFalse      = "false"
	opPresent    = "present"
	opNotPresent = "notpresent"
	opPrefix     = "prefix"
	opSuffix     = "suffix"
	opIsOneOf    = "isoneof"
	opIsNotOneOf = "isnotoneof"
)

const (
	// totalBucketNum represents how many buckets we can use to determine the consistent hashing
	// distribution and rollout
	totalBucketNum uint = 1000

	// percentMultiplier implies that the multiplier between percentage (100) and totalBucketNum
	percentMultiplier = float32(totalBucketNum) / 100
)

func crc32Num(entityID string, salt string) uint {
	return uint(crc32.ChecksumIEEE([]byte(salt+entityID))) % totalBucketNum
}

func (r *Evaluator) Batch(request *RequestBatchEvaluation) (*ResponseBatchEvaluation, error) {
	if r.snapshot == nil {
		return nil, errors.New("snapshot is nil")
	}

	result := &ResponseBatchEvaluation{
		Responses: make([]*ResponseEvaluation, 0, len(request.Requests)),
	}

	for _, req := range request.Requests {
		flag, err := r.snapshot.getFlag(req.NamespaceKey, req.FlagKey)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				result.Responses = append(result.Responses, &ResponseEvaluation{
					Type: typeError,
					ErrorResponse: &ResponseError{
						FlagKey:      req.FlagKey,
						NamespaceKey: req.NamespaceKey,
						Reason:       errorReasonNotFound,
					},
				})

				continue
			}
			return nil, fmt.Errorf("get flag: %w", err)
		}

		switch flag.typ {
		case variantFlagType:
			variant, err := r.Variant(req, flag)
			if err != nil {
				return nil, fmt.Errorf("variant: %w", err)
			}
			result.Responses = append(result.Responses, &ResponseEvaluation{
				Type:            typeVariant,
				VariantResponse: variant,
			})
		case booleanFlagType:
			boolean, err := r.Boolean(req, flag)
			if err != nil {
				return nil, fmt.Errorf("boolean: %w", err)
			}
			result.Responses = append(result.Responses, &ResponseEvaluation{
				Type:            typeBoolean,
				BooleanResponse: boolean,
			})
		default:
			return nil, fmt.Errorf("unknown flag type: %s", flag.typ.String())
		}
	}

	return result, nil
}

func (r *Evaluator) ListFlags(namespaceKey string) (*ResponseListFlags, error) {
	if r.snapshot == nil {
		return nil, errors.New("snapshot is nil")
	}

	result := &ResponseListFlags{Flags: make([]*ResponseFlag, 0)}

	evalFlags, err := r.snapshot.listFlags(namespaceKey)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return result, nil
		}
	}

	for _, evalFlag := range evalFlags {
		result.Flags = append(result.Flags, &ResponseFlag{
			Key:          evalFlag.key,
			Enabled:      evalFlag.enabled,
			NamespaceKey: namespaceKey,
			Type:         evalFlag.typ,
		})
	}

	return result, nil
}

func (r *Evaluator) Boolean(req *RequestEvaluation, flag *EvalFlag) (*ResponseBoolean, error) {
	eval, err := r.evalBoolean(req, flag)
	if err != nil {
		return nil, err
	}

	return eval, nil
}

func (r *Evaluator) Variant(req *RequestEvaluation, flag *EvalFlag) (*ResponseVariant, error) {
	eval, err := r.evalVariant(req, flag)
	if err != nil {
		return nil, err
	}

	return eval, nil
}

func (r *Evaluator) evalBoolean(req *RequestEvaluation, flag *EvalFlag) (*ResponseBoolean, error) {
	resp := &ResponseBoolean{
		FlagKey: req.FlagKey,
		Enabled: false,
		Reason:  reasonDefault,
	}

	if flag.typ != booleanFlagType {
		resp.Reason = reasonError
		return nil, errors.New("flag type is not boolean")
	}

	if !flag.enabled {
		resp.Reason = reasonDisabled
		return resp, nil
	}

	rollouts, err := r.snapshot.getRollouts(req.NamespaceKey, req.FlagKey)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.Enabled = flag.enabled
			return resp, nil
		}
		return nil, fmt.Errorf("get rollouts: %w", err)
	}

	var lastRank int
	for _, rollout := range rollouts {
		if rollout.rank < lastRank {
			return nil, fmt.Errorf("rollout type %s rank: %d detected out of order", rollout.typ, rollout.rank)
		}
		lastRank = rollout.rank

		if rollout.threshold != nil {
			hash := crc32.ChecksumIEEE([]byte(req.EntityID + req.FlagKey))

			normalizedValue := float32(int(hash) % 100)
			if normalizedValue < rollout.threshold.percentage {
				resp.Enabled = rollout.threshold.value
				resp.Reason = reasonMatch
				return resp, nil
			}
		} else if rollout.segment != nil {

			var segmentMatches int
			for _, segment := range rollout.segment.segments {
				matched, err := r.matchConstraints(req.Context, segment.constraints, segment.matchType, req.EntityID)
				if err != nil {
					return nil, fmt.Errorf("match constraints, rollout type '%s', segment_key '%s': %w", rollout.typ, segment.key, err)
				}

				if matched {
					segmentMatches++
				}
			}

			switch rollout.segment.segmentOperator {
			case orSegmentOperator:
				if segmentMatches < 1 {
					continue
				}
			case andSegmentOperator:
				if len(rollout.segment.segments) != segmentMatches {
					continue
				}
			}

			resp.Enabled = rollout.segment.value
			resp.Reason = reasonMatch
			return resp, nil
		}
	}

	resp.Enabled = flag.enabled
	return resp, nil
}

func (r *Evaluator) evalVariant(req *RequestEvaluation, flag *EvalFlag) (*ResponseVariant, error) {
	resp := &ResponseVariant{
		FlagKey:     flag.key,
		Match:       false,
		Reason:      reasonUnknown,
		SegmentKeys: []string{},
	}

	if flag.typ != variantFlagType {
		resp.Reason = reasonError
		return nil, errors.New("flag type is not variant")
	}

	if !flag.enabled {
		resp.Reason = reasonDisabled
		return resp, nil
	}

	rules, err := r.snapshot.getRules(req.NamespaceKey, req.FlagKey)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return resp, nil
		}
		return nil, fmt.Errorf("get rules: %w", err)
	}

	var lastRank int
	for _, rule := range rules {
		if rule.rank < lastRank {
			resp.Reason = reasonError
			return nil, fmt.Errorf("rule_id '%s' rank '%d' detected out of order", rule.id, rule.rank)
		}

		lastRank = rule.rank

		segmentKeys := make([]string, 0, len(rule.segments))
		segmentMatches := 0

		// тут пока всегда будет только один сегмент т к мультисегментация в снапшоте работает неправильно
		for _, segment := range rule.segments {
			matched, err := r.matchConstraints(req.Context, segment.constraints, segment.matchType, req.EntityID)
			if err != nil {
				resp.Reason = reasonError
				return nil, fmt.Errorf("match constraints, rule_id '%s', segment_key '%s: %w", rule.id, segment.key, err)
			}

			if matched {
				segmentKeys = append(segmentKeys, segment.key)
				segmentMatches++
			}
		}

		switch rule.segmentOperator {
		case orSegmentOperator:
			if segmentMatches < 1 {
				continue
			}
		case andSegmentOperator:
			if len(rule.segments) != segmentMatches {
				continue
			}
		}

		if len(segmentKeys) > 0 {
			resp.SegmentKeys = segmentKeys
		}

		var (
			validDistributions []*EvalDistribution
			buckets            []int
		)

		for _, distribution := range rule.distributions {
			// don't include 0% rollouts
			if distribution.rollout > 0 {
				validDistributions = append(validDistributions, distribution)

				if buckets == nil {
					bucket := int(distribution.rollout * percentMultiplier)
					buckets = append(buckets, bucket)
				} else {
					bucket := buckets[len(buckets)-1] + int(distribution.rollout*percentMultiplier)
					buckets = append(buckets, bucket)
				}
			}
		}

		// no distributions for rule
		if len(validDistributions) == 0 {
			resp.Match = true
			resp.Reason = reasonMatch
			return resp, nil
		}

		var (
			bucket = crc32Num(req.EntityID, flag.key)
			// sort.SearchInts searches for x in a sorted slice of ints and returns the index
			// as specified by Search. The return value is the index to insert x if x is
			// not present (it could be len(a)).
			index = sort.SearchInts(buckets, int(bucket)+1)
		)

		// if index is outside of our existing buckets then it does not match any distribution
		if index == len(validDistributions) {
			return resp, nil
		}

		d := validDistributions[index]

		resp.Match = true
		resp.VariantKey = d.variantKey
		resp.VariantAttachment = d.variantAttachment
		resp.Reason = reasonMatch
		return resp, nil
	} // end rule loop

	return resp, nil
}

func (r *Evaluator) matchConstraints(
	evalCtx map[string]string,
	constraints []*EvalConstraint,
	segmentMatchType segmentMatchType,
	entityId string,
) (bool, error) {

	constraintMatches := 0
	for _, constraint := range constraints {
		property := evalCtx[constraint.property]

		var (
			match bool
			err   error
		)

		switch constraint.typ {
		case stringConstraintComparisonType:
			match = matchesString(constraint, property)
		case numberConstraintComparisonType:
			match, err = matchesNumber(constraint, property)
		case booleanConstraintComparisonType:
			match, err = matchesBool(constraint, property)
		case datetimeConstraintComparisonType:
			match, err = matchesDateTime(constraint, property)
		case entityIDConstraintComparisonType:
			match = matchesString(constraint, entityId)
		default:
			return false, fmt.Errorf("unknown constraint type: %s", constraint.typ)
		}

		if err != nil {
			r.logger.Error("error matching constraint", zap.String("property", constraint.property), zap.Error(err))
			// don't return here because we want to continue to evaluate the other constraints
		}

		if match {
			// increase the matchCount
			constraintMatches++

			switch segmentMatchType {
			case anySegmentMatchType:
				// can short circuit here since we had at least one match
				break
			default:
				// keep looping as we need to match all constraints
				continue
			}
		} else {
			// no match
			switch segmentMatchType {
			case allSegmentMatchType:
				// we can short circuit because we must match all constraints
				break
			default:
				// keep looping to see if we match the next constraint
				continue
			}
		}
	}

	var matched = true

	switch segmentMatchType {
	case allSegmentMatchType:
		if len(constraints) != constraintMatches {
			matched = false
		}

	case anySegmentMatchType:
		if len(constraints) > 0 && constraintMatches == 0 {
			matched = false
		}
	default:
		matched = false
	}

	return matched, nil
}

func matchesString(constraint *EvalConstraint, property string) bool {
	switch constraint.operator {
	case opEmpty:
		return len(strings.TrimSpace(property)) == 0
	case opNotEmpty:
		return len(strings.TrimSpace(property)) != 0
	}

	if property == "" {
		return false
	}

	value := constraint.value

	switch constraint.operator {
	case opEQ:
		return value == property
	case opNEQ:
		return value != property
	case opPrefix:
		return strings.HasPrefix(strings.TrimSpace(property), value)
	case opSuffix:
		return strings.HasSuffix(strings.TrimSpace(property), value)
	case opIsOneOf:
		values := []string{}
		if err := json.Unmarshal([]byte(value), &values); err != nil {
			return false
		}
		return slices.Contains(values, property)
	case opIsNotOneOf:
		values := []string{}
		if err := json.Unmarshal([]byte(value), &values); err != nil {
			return false
		}
		return !slices.Contains(values, property)
	}

	return false
}

func matchesNumber(constraint *EvalConstraint, property string) (bool, error) {
	switch constraint.operator {
	case opNotPresent:
		return len(strings.TrimSpace(property)) == 0, nil
	case opPresent:
		return len(strings.TrimSpace(property)) != 0, nil
	}

	// can't parse an empty string
	if property == "" {
		return false, nil
	}

	n, err := strconv.ParseFloat(property, 64)
	if err != nil {
		return false, fmt.Errorf("parsing number from %s: %w", property, ErrInvalid)
	}

	if constraint.operator == opIsOneOf {
		values := []float64{}
		if err := json.Unmarshal([]byte(constraint.value), &values); err != nil {
			return false, fmt.Errorf("invalid value for constraint %s: %w", constraint.value, ErrInvalid)
		}
		return slices.Contains(values, n), nil
	} else if constraint.operator == opIsNotOneOf {
		values := []float64{}
		if err := json.Unmarshal([]byte(constraint.value), &values); err != nil {
			return false, fmt.Errorf("invalid value for constraint %s: %w", constraint.value, ErrInvalid)
		}
		return !slices.Contains(values, n), nil
	}

	// TODO: we should consider parsing this at creation time since it doesn't change and it doesnt make sense to allow invalid constraint values
	value, err := strconv.ParseFloat(constraint.value, 64)
	if err != nil {
		return false, fmt.Errorf("parsing number from %s: %w", constraint.value, ErrInvalid)
	}

	switch constraint.operator {
	case opEQ:
		return value == n, nil
	case opNEQ:
		return value != n, nil
	case opLT:
		return n < value, nil
	case opLTE:
		return n <= value, nil
	case opGT:
		return n > value, nil
	case opGTE:
		return n >= value, nil
	}

	return false, nil
}

func matchesBool(constraint *EvalConstraint, property string) (bool, error) {
	switch constraint.operator {
	case opNotPresent:
		return len(strings.TrimSpace(property)) == 0, nil
	case opPresent:
		return len(strings.TrimSpace(property)) != 0, nil
	}

	// can't parse an empty string
	if property == "" {
		return false, nil
	}

	value, err := strconv.ParseBool(property)
	if err != nil {
		return false, fmt.Errorf("parsing boolean from %s: %w", property, ErrInvalid)
	}

	switch constraint.operator {
	case opTrue:
		return value, nil
	case opFalse:
		return !value, nil
	}

	return false, nil
}

func matchesDateTime(constraint *EvalConstraint, property string) (bool, error) {
	switch constraint.operator {
	case opNotPresent:
		return len(strings.TrimSpace(property)) == 0, nil
	case opPresent:
		return len(strings.TrimSpace(property)) != 0, nil
	}

	// can't parse an empty string
	if property == "" {
		return false, nil
	}

	d, err := tryParseDateTime(property)
	if err != nil {
		return false, err
	}

	value, err := tryParseDateTime(constraint.value)
	if err != nil {
		return false, err
	}

	switch constraint.operator {
	case opEQ:
		return value.Equal(d), nil
	case opNEQ:
		return !value.Equal(d), nil
	case opLT:
		return d.Before(value), nil
	case opLTE:
		return d.Before(value) || value.Equal(d), nil
	case opGT:
		return d.After(value), nil
	case opGTE:
		return d.After(value) || value.Equal(d), nil
	}

	return false, nil
}

func tryParseDateTime(v string) (time.Time, error) {
	if d, err := time.Parse(time.RFC3339, v); err == nil {
		return d.UTC(), nil
	}

	if d, err := time.Parse(time.DateOnly, v); err == nil {
		return d.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("parsing datetime from %s: %w", v, ErrInvalid)
}
