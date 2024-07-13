package flipt_engine_go

import (
	"fmt"
	"sync"
)

type Snapshot struct {
	store sync.Map
}

func NewSnapshot() *Snapshot {
	return &Snapshot{}
}

func (r *Snapshot) replaceStore(namespaces map[string]*EvalNamespace) {
	for key, value := range namespaces {
		r.store.Store(key, value)
	}
}

func (r *Snapshot) cleanStore() {
	r.store.Range(func(key, value any) bool {
		r.store.Delete(key)
		return true
	})
}

func (r *Snapshot) getFlag(namespaceKey, flagKey string) (*EvalFlag, error) {
	namespace, exists := r.store.Load(namespaceKey)
	if !exists {
		return nil, fmt.Errorf("namespaceKey '%s' %w", namespaceKey, ErrNotFound)
	}

	evalNamespace, ok := namespace.(*EvalNamespace)
	if !ok {
		return nil, fmt.Errorf("cast failed, namespaceKey '%s'", namespaceKey)
	}

	flag, exists := evalNamespace.flags[flagKey]
	if !exists {
		return nil, fmt.Errorf("flagKey '%s' %w, namespaceKey '%s'", flagKey, ErrNotFound, namespaceKey)
	}

	return flag, nil
}

func (r *Snapshot) getRules(namespaceKey, flagKey string) ([]*EvalRule, error) {
	namespace, exists := r.store.Load(namespaceKey)
	if !exists {
		return nil, fmt.Errorf("namespaceKey '%s' %w", namespaceKey, ErrNotFound)
	}

	evalNamespace, ok := namespace.(*EvalNamespace)
	if !ok {
		return nil, fmt.Errorf("cast failed, namespaceKey '%s'", namespaceKey)
	}

	rules, exists := evalNamespace.rules[flagKey]
	if !exists {
		return nil, fmt.Errorf("rules %w, namespaceKey '%s', flagKey '%s'", ErrNotFound, namespaceKey, flagKey)
	}

	return rules, nil
}

func (r *Snapshot) listFlags(namespaceKey string) ([]*EvalFlag, error) {
	namespace, exists := r.store.Load(namespaceKey)
	if !exists {
		return nil, fmt.Errorf("namespaceKey '%s' %w", namespaceKey, ErrNotFound)
	}

	evalNamespace, ok := namespace.(*EvalNamespace)
	if !ok {
		return nil, fmt.Errorf("cast failed, namespaceKey '%s'", namespaceKey)
	}

	flags := make([]*EvalFlag, 0, len(evalNamespace.flags))
	for _, flag := range evalNamespace.flags {
		flags = append(flags, flag)
	}
	return flags, nil
}

func (r *Snapshot) getRollouts(namespaceKey, flagKey string) ([]*EvalRollout, error) {
	namespace, exists := r.store.Load(namespaceKey)
	if !exists {
		return nil, fmt.Errorf("namespaceKey '%s' %w", namespaceKey, ErrNotFound)
	}

	evalNamespace, ok := namespace.(*EvalNamespace)
	if !ok {
		return nil, fmt.Errorf("cast failed, namespaceKey '%s'", namespaceKey)
	}

	rollouts, exists := evalNamespace.rollouts[flagKey]
	if !exists {
		return nil, fmt.Errorf("rollouts %w, namespaceKey '%s, flagKey '%s'", ErrNotFound, namespaceKey, flagKey)
	}

	return rollouts, nil
}

func (r *Snapshot) makeSnapshot(doc *Document) map[string]*EvalNamespace {
	flags := make(map[string]*EvalFlag)
	rules := make(map[string][]*EvalRule)
	rollouts := make(map[string][]*EvalRollout)

	for _, flag := range doc.Flags {
		flags[flag.Key] = &EvalFlag{
			key:     flag.Key,
			typ:     flag.Type,
			enabled: flag.Enabled,
		}

		evalRulesCollection := make([]*EvalRule, 0, len(flag.Rules))
		checkDuplicateInRules := make(map[string]struct{}) // the snapshot request receives duplicates of the rules, so we cut them off
		for _, rule := range flag.Rules {
			_, exists := checkDuplicateInRules[rule.ID]
			if exists {
				continue
			}
			checkDuplicateInRules[rule.ID] = struct{}{}

			evalRule := &EvalRule{
				id:              rule.ID,
				rank:            rule.Rank,
				segmentOperator: rule.SegmentOperator,
				segments:        make([]*EvalSegment, 0, len(rule.Segments)),
				distributions:   make([]*EvalDistribution, 0, len(rule.Distributions)),
			}

			for _, segment := range rule.Segments {
				evalConstraints := make([]*EvalConstraint, 0, len(segment.Constraints))
				for _, constraint := range segment.Constraints {
					evalConstraints = append(evalConstraints, &EvalConstraint{
						id:       constraint.ID,
						typ:      constraint.Type,
						property: constraint.Property,
						operator: constraint.Operator,
						value:    constraint.Value,
					})
				}

				evalRule.segments = append(evalRule.segments, &EvalSegment{
					key:         segment.Key,
					matchType:   segment.MatchType,
					constraints: evalConstraints,
				})
			}

			checkDuplicateInDistribution := make(map[string]struct{}) // also here
			for _, distribution := range rule.Distributions {
				_, exists = checkDuplicateInDistribution[distribution.VariantID]
				if exists {
					continue
				}
				checkDuplicateInDistribution[distribution.VariantID] = struct{}{}

				evalRule.distributions = append(evalRule.distributions, &EvalDistribution{
					ruleID:            rule.ID,
					rollout:           distribution.Rollout,
					variantKey:        distribution.VariantKey,
					variantAttachment: distribution.VariantAttachment,
				})
			}

			evalRulesCollection = append(evalRulesCollection, evalRule)
		}

		if len(evalRulesCollection) > 0 {
			rules[flag.Key] = evalRulesCollection
		}

		evalRolloutCollection := make([]*EvalRollout, 0, len(flag.Rollouts))
		for _, rollout := range flag.Rollouts {
			evalRollout := &EvalRollout{
				typ:  rollout.Type,
				rank: rollout.Rank,
			}

			if rollout.Type == thresholdRolloutType {
				evalRollout.threshold = &EvalThresholdRollout{
					percentage: rollout.Threshold.Percentage,
					value:      rollout.Threshold.Value,
				}
			}

			if rollout.Type == segmentRolloutType {
				evalRolloutSegments := make([]*EvalSegment, 0, len(rollout.Segment.Segments))
				for _, segment := range rollout.Segment.Segments {
					evalRolloutSegmentConstraints := make([]*EvalConstraint, 0, len(segment.Constraints))
					for _, constraint := range segment.Constraints {
						evalRolloutSegmentConstraints = append(evalRolloutSegmentConstraints, &EvalConstraint{
							id:       constraint.ID,
							typ:      constraint.Type,
							property: constraint.Property,
							operator: constraint.Operator,
							value:    constraint.Value,
						})
					}

					evalRolloutSegments = append(evalRolloutSegments, &EvalSegment{
						key:         segment.Key,
						matchType:   segment.MatchType,
						constraints: evalRolloutSegmentConstraints,
					})
				}

				evalRollout.segment = &EvalSegmentRollout{
					value:           rollout.Segment.Value,
					segmentOperator: rollout.Segment.SegmentOperator,
					segments:        evalRolloutSegments,
				}
			}

			evalRolloutCollection = append(evalRolloutCollection, evalRollout)
		}

		if len(evalRolloutCollection) > 0 {
			rollouts[flag.Key] = evalRolloutCollection
		}
	}

	return map[string]*EvalNamespace{
		doc.Namespace.Key: {
			key:      doc.Namespace.Key,
			flags:    flags,
			rules:    rules,
			rollouts: rollouts,
		},
	}
}
