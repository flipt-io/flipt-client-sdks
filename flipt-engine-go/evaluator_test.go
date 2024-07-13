package flipt_engine_go

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func Test_matchesDateTime(t *testing.T) {
	tests := []struct {
		name       string
		constraint *EvalConstraint
		value      string
		wantMatch  bool
		wantErr    bool
	}{
		{
			name: "present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "present",
			},
			value:     "2006-01-02T15:04:05Z",
			wantMatch: true,
		},
		{
			name: "negative present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "present",
			},
		},
		{
			name: "not present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notpresent",
			},
			wantMatch: true,
		},
		{
			name: "negative notpresent",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notpresent",
			},
			value: "2006-01-02T15:04:05Z",
		},
		{
			name: "not a datetime constraint value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "bar",
			},
			value:   "2006-01-02T15:04:05Z",
			wantErr: true,
		},
		{
			name: "not a datetime context value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "2006-01-02T15:04:05Z",
			},
			value:   "foo",
			wantErr: true,
		},
		{
			name: "eq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "2006-01-02T15:04:05Z",
			},
			value:     "2006-01-02T15:04:05Z",
			wantMatch: true,
		},
		{
			name: "eq date only",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "2006-01-02",
			},
			value:     "2006-01-02",
			wantMatch: true,
		},
		{
			name: "negative eq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "2006-01-02T15:04:05Z",
			},
			value: "2007-01-02T15:04:05Z",
		},
		{
			name: "neq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "neq",
				value:    "2006-01-02T15:04:05Z",
			},
			value:     "2007-01-02T15:04:05Z",
			wantMatch: true,
		},
		{
			name: "negative neq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "neq",
				value:    "2006-01-02T15:04:05Z",
			},
			value: "2006-01-02T15:04:05Z",
		},
		{
			name: "negative neq date only",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "neq",
				value:    "2006-01-02",
			},
			value: "2006-01-02",
		},
		{
			name: "lt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lt",
				value:    "2006-01-02T15:04:05Z",
			},
			value:     "2005-01-02T15:04:05Z",
			wantMatch: true,
		},
		{
			name: "negative lt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lt",
				value:    "2005-01-02T15:04:05Z",
			},
			value: "2006-01-02T15:04:05Z",
		},
		{
			name: "lte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lte",
				value:    "2006-01-02T15:04:05Z",
			},
			value:     "2006-01-02T15:04:05Z",
			wantMatch: true,
		},
		{
			name: "negative lte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lte",
				value:    "2006-01-02T15:04:05Z",
			},
			value: "2007-01-02T15:04:05Z",
		},
		{
			name: "gt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gt",
				value:    "2006-01-02T15:04:05Z",
			},
			value:     "2007-01-02T15:04:05Z",
			wantMatch: true,
		},
		{
			name: "negative gt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gt",
				value:    "2007-01-02T15:04:05Z",
			},
			value: "2006-01-02T15:04:05Z",
		},
		{
			name: "gte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gte",
				value:    "2006-01-02T15:04:05Z",
			},
			value:     "2006-01-02T15:04:05Z",
			wantMatch: true,
		},
		{
			name: "negative gte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gte",
				value:    "2006-01-02T15:04:05Z",
			},
			value: "2005-01-02T15:04:05Z",
		},
		{
			name: "empty value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "2006-01-02T15:04:05Z",
			},
		},
		{
			name: "unknown operator",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "foo",
				value:    "2006-01-02T15:04:05Z",
			},
			value: "2006-01-02T15:04:05Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				constraint = tt.constraint
				value      = tt.value
				wantMatch  = tt.wantMatch
				wantErr    = tt.wantErr
			)

			match, err := matchesDateTime(constraint, value)

			if wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalid)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, wantMatch, match)
		})

	}
}

func Test_matchesBool(t *testing.T) {
	tests := []struct {
		name       string
		constraint *EvalConstraint
		value      string
		wantMatch  bool
		wantErr    bool
	}{
		{
			name: "present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "present",
			},
			value:     "true",
			wantMatch: true,
		},
		{
			name: "negative present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "present",
			},
		},
		{
			name: "not present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notpresent",
			},
			wantMatch: true,
		},
		{
			name: "negative notpresent",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notpresent",
			},
			value: "true",
		},
		{
			name: "not a bool",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "true",
			},
			value:   "foo",
			wantErr: true,
		},
		{
			name: "is true",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "true",
			},
			value:     "true",
			wantMatch: true,
		},
		{
			name: "negative is true",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "true",
			},
			value: "false",
		},
		{
			name: "is false",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "false",
			},
			value:     "false",
			wantMatch: true,
		},
		{
			name: "negative is false",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "false",
			},
			value: "true",
		},
		{
			name: "empty value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "false",
			},
		},
		{
			name: "unknown operator",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "foo",
			},
			value: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				constraint = tt.constraint
				value      = tt.value
				wantMatch  = tt.wantMatch
				wantErr    = tt.wantErr
			)

			match, err := matchesBool(constraint, value)

			if wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalid)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, wantMatch, match)
		})
	}
}

func Test_matchesNumber(t *testing.T) {
	tests := []struct {
		name       string
		constraint *EvalConstraint
		value      string
		wantMatch  bool
		wantErr    bool
	}{
		{
			name: "present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "present",
			},
			value:     "1",
			wantMatch: true,
		},
		{
			name: "negative present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "present",
			},
		},
		{
			name: "not present",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notpresent",
			},
			wantMatch: true,
		},
		{
			name: "negative notpresent",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notpresent",
			},
			value: "1",
		},
		{
			name: "NAN constraint value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "bar",
			},
			value:   "5",
			wantErr: true,
		},
		{
			name: "NAN context value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "5",
			},
			value:   "foo",
			wantErr: true,
		},
		{
			name: "eq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "42.0",
			},
			value:     "42.0",
			wantMatch: true,
		},
		{
			name: "negative eq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "42.0",
			},
			value: "50",
		},
		{
			name: "neq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "neq",
				value:    "42.0",
			},
			value:     "50",
			wantMatch: true,
		},
		{
			name: "negative neq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "neq",
				value:    "42.0",
			},
			value: "42.0",
		},
		{
			name: "lt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lt",
				value:    "42.0",
			},
			value:     "8",
			wantMatch: true,
		},
		{
			name: "negative lt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lt",
				value:    "42.0",
			},
			value: "50",
		},
		{
			name: "lte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lte",
				value:    "42.0",
			},
			value:     "42.0",
			wantMatch: true,
		},
		{
			name: "negative lte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "lte",
				value:    "42.0",
			},
			value: "102.0",
		},
		{
			name: "gt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gt",
				value:    "10.11",
			},
			value:     "10.12",
			wantMatch: true,
		},
		{
			name: "negative gt",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gt",
				value:    "10.11",
			},
			value: "1",
		},
		{
			name: "gte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gte",
				value:    "10.11",
			},
			value:     "10.11",
			wantMatch: true,
		},
		{
			name: "negative gte",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "gte",
				value:    "10.11",
			},
			value: "0.11",
		},
		{
			name: "empty value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "0.11",
			},
		},
		{
			name: "unknown operator",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "foo",
				value:    "0.11",
			},
			value: "0.11",
		},
		{
			name: "negative suffix empty value",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "suffix",
				value:    "bar",
			},
		},
		{
			name: "is one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isoneof",
				value:    "[3, 3.14159, 4]",
			},
			value:     "3.14159",
			wantMatch: true,
		},
		{
			name: "negative is one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isoneof",
				value:    "[5, 3.14159, 4]",
			},
			value: "9",
		},
		{
			name: "negative is one of (non-number values)",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isoneof",
				value:    "[5, \"str\"]",
			},
			value:   "5",
			wantErr: true,
		},
		{
			name: "is not one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isnotoneof",
				value:    "[5, 3.14159, 4]",
			},
			value:     "3",
			wantMatch: true,
		},
		{
			name: "negative is not one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isnotoneof",
				value:    "[5, 3.14159, 4]",
			},
			value:     "3.14159",
			wantMatch: false,
		},
		{
			name: "negative is not one of (invalid json)",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isnotoneof",
				value:    "[5, 6",
			},
			value:   "5",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				constraint = tt.constraint
				value      = tt.value
				wantMatch  = tt.wantMatch
				wantErr    = tt.wantErr
			)

			match, err := matchesNumber(constraint, value)

			if wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalid)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, wantMatch, match)
		})
	}
}

func Test_matchesString(t *testing.T) {
	tests := []struct {
		name       string
		constraint *EvalConstraint
		value      string
		wantMatch  bool
	}{
		{
			name: "eq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "bar",
			},
			value:     "bar",
			wantMatch: true,
		},
		{
			name: "negative eq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "eq",
				value:    "bar",
			},
			value: "baz",
		},
		{
			name: "neq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "neq",
				value:    "bar",
			},
			value:     "baz",
			wantMatch: true,
		},
		{
			name: "negative neq",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "neq",
				value:    "bar",
			},
			value: "bar",
		},
		{
			name: "empty",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "empty",
			},
			value:     " ",
			wantMatch: true,
		},
		{
			name: "negative empty",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "empty",
			},
			value: "bar",
		},
		{
			name: "not empty",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notempty",
			},
			value:     "bar",
			wantMatch: true,
		},
		{
			name: "negative not empty",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "notempty",
			},
			value: "",
		},
		{
			name: "unknown operator",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "foo",
				value:    "bar",
			},
			value: "bar",
		},
		{
			name: "prefix",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "prefix",
				value:    "ba",
			},
			value:     "bar",
			wantMatch: true,
		},
		{
			name: "negative prefix",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "prefix",
				value:    "bar",
			},
			value: "nope",
		},
		{
			name: "suffix",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "suffix",
				value:    "ar",
			},
			value:     "bar",
			wantMatch: true,
		},
		{
			name: "negative suffix",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "suffix",
				value:    "bar",
			},
			value: "nope",
		},
		{
			name: "is one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isoneof",
				value:    "[\"bar\", \"baz\"]",
			},
			value:     "baz",
			wantMatch: true,
		},
		{
			name: "negative is one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isoneof",
				value:    "[\"bar\", \"baz\"]",
			},
			value: "nope",
		},
		{
			name: "negative is one of (invalid json)",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isoneof",
				value:    "[\"bar\", \"baz\"",
			},
			value: "bar",
		},
		{
			name: "negative is one of (non-string values)",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isoneof",
				value:    "[\"bar\", 5]",
			},
			value: "bar",
		},
		{
			name: "is not one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isnotoneof",
				value:    "[\"bar\", \"baz\"]",
			},
			value: "baz",
		},
		{
			name: "negative is not one of",
			constraint: &EvalConstraint{
				property: "foo",
				operator: "isnotoneof",
				value:    "[\"bar\", \"baz\"]",
			},
			value:     "nope",
			wantMatch: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				constraint = tt.constraint
				value      = tt.value
				wantMatch  = tt.wantMatch
			)

			match := matchesString(constraint, value)
			assert.Equal(t, wantMatch, match)
		})
	}
}

func TestEvaluator_matchConstraints(t *testing.T) {
	type args struct {
		evalCtx          map[string]string
		constraints      []*EvalConstraint
		segmentMatchType segmentMatchType
		entityId         string
	}
	tests := []struct {
		name    string
		args    args
		match   bool
		wantErr error
	}{
		{
			name: "isMatch: constraints is empty",
			args: args{
				evalCtx: map[string]string{
					"city_id": "1",
				},
				constraints:      nil,
				segmentMatchType: anySegmentMatchType,
				entityId:         "77777",
			},
			match:   true,
			wantErr: nil,
		},
		{
			name: "isMatch: one constraint ANY segment",
			args: args{
				evalCtx: map[string]string{
					"city_id": "1",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: anySegmentMatchType,
				entityId:         "77777",
			},
			match:   true,
			wantErr: nil,
		},
		{
			name: "notMatch: one constraint ANY segment",
			args: args{
				evalCtx: map[string]string{
					"city_id": "2",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: anySegmentMatchType,
				entityId:         "77777",
			},
			match:   false,
			wantErr: nil,
		},
		{
			name: "isMatch: one constraint ALL segment",
			args: args{
				evalCtx: map[string]string{
					"city_id": "1",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   true,
			wantErr: nil,
		},
		{
			name: "notMatch: one constraint ALL segment",
			args: args{
				evalCtx: map[string]string{
					"city_id": "2",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   false,
			wantErr: nil,
		},
		{
			name: "isMatch: two constraint ANY segment, city_id is eq",
			args: args{
				evalCtx: map[string]string{
					"city_id":    "1",
					"country_id": "2",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
					{
						id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
						typ:      stringConstraintComparisonType,
						property: "country_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: anySegmentMatchType,
				entityId:         "77777",
			},
			match:   true,
			wantErr: nil,
		},
		{
			name: "isMatch: two constraint ANY segment, country_id is eq",
			args: args{
				evalCtx: map[string]string{
					"city_id":    "2",
					"country_id": "1",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
					{
						id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
						typ:      stringConstraintComparisonType,
						property: "country_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: anySegmentMatchType,
				entityId:         "77777",
			},
			match:   true,
			wantErr: nil,
		},
		{
			name: "notMatch: two constraint ANY segment",
			args: args{
				evalCtx: map[string]string{
					"city_id":    "2",
					"country_id": "2",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
					{
						id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
						typ:      stringConstraintComparisonType,
						property: "country_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: anySegmentMatchType,
				entityId:         "77777",
			},
			match:   false,
			wantErr: nil,
		},
		{
			name: "notMatch: two constraint ALL segment, country_id and city_id is not eq",
			args: args{
				evalCtx: map[string]string{
					"city_id":    "2",
					"country_id": "2",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
					{
						id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
						typ:      stringConstraintComparisonType,
						property: "country_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   false,
			wantErr: nil,
		},
		{
			name: "notMatch: two constraint ALL segment, country_id is not eq",
			args: args{
				evalCtx: map[string]string{
					"city_id":    "2",
					"country_id": "1",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
					{
						id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
						typ:      stringConstraintComparisonType,
						property: "country_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   false,
			wantErr: nil,
		},
		{
			name: "notMatch: two constraint ALL segment, city_id is not eq",
			args: args{
				evalCtx: map[string]string{
					"city_id":    "1",
					"country_id": "2",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
					{
						id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
						typ:      stringConstraintComparisonType,
						property: "country_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   false,
			wantErr: nil,
		},
		{
			name: "isMatch: two constraint ALL segment",
			args: args{
				evalCtx: map[string]string{
					"city_id":    "1",
					"country_id": "1",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      numberConstraintComparisonType,
						property: "city_id",
						operator: "eq",
						value:    "1",
					},
					{
						id:       "b039cc47-46b3-4bee-98e1-b4f4325b3315",
						typ:      stringConstraintComparisonType,
						property: "country_id",
						operator: "eq",
						value:    "1",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   true,
			wantErr: nil,
		},
		{
			name: "isMatch: property is bool",
			args: args{
				evalCtx: map[string]string{
					"mode": "true",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      booleanConstraintComparisonType,
						property: "mode",
						operator: "true",
						value:    "true",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   true,
			wantErr: nil,
		},
		{
			name: "notMatch: property is bool",
			args: args{
				evalCtx: map[string]string{
					"mode": "false",
				},
				constraints: []*EvalConstraint{
					{
						id:       "2842abce-2504-4d99-ab35-b32fccc90b37",
						typ:      booleanConstraintComparisonType,
						property: "mode",
						operator: "true",
						value:    "true",
					},
				},
				segmentMatchType: allSegmentMatchType,
				entityId:         "77777",
			},
			match:   false,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Evaluator{
				snapshot: NewSnapshot(),
				logger:   zaptest.NewLogger(t),
			}

			match, err := r.matchConstraints(tt.args.evalCtx, tt.args.constraints, tt.args.segmentMatchType, tt.args.entityId)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equalf(t, tt.match, match, "matchConstraints(%v, %v, %v, %v)", tt.args.evalCtx, tt.args.constraints, tt.args.segmentMatchType, tt.args.entityId)
			}
		})
	}
}

func TestEvaluator_variant(t *testing.T) {
	type args struct {
		req  *RequestEvaluation
		flag *EvalFlag
	}
	type mock struct {
		namespaces map[string]*EvalNamespace
	}
	type want struct {
		want    *ResponseVariant
		wantErr error
	}
	tests := []struct {
		name string
		args
		mock
		want
	}{
		{
			name: "flag type is not variant",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     booleanFlagType,
					enabled: false,
				},
			},
			want: want{
				want:    nil,
				wantErr: errors.New("flag type is not variant"),
			},
		},
		{
			name: "flag is disabled",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: false,
				},
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             false,
					SegmentKeys:       []string{},
					Reason:            reasonDisabled,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "empty rules",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key:   "mobile",
						rules: nil,
					},
				},
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             false,
					SegmentKeys:       []string{},
					Reason:            reasonUnknown,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "rules out of order",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"toggle1": {
								{
									id:              "1",
									rank:            1,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:       "segment1",
											matchType: allSegmentMatchType,
											constraints: []*EvalConstraint{
												{
													id:       "1",
													typ:      stringConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "2",
												},
											},
										},
									},
								},
								{
									id:              "2",
									rank:            0,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:         "segment2",
											matchType:   allSegmentMatchType,
											constraints: nil,
										},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want:    nil,
				wantErr: errors.New("rule_id '2' rank '0' detected out of order"),
			},
		},
		{
			name: "match = true (1 segment, 0 constraints, 0 distributions",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"toggle1": {
								{
									id:              "1",
									rank:            1,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:         "segment1",
											matchType:   anySegmentMatchType,
											constraints: nil,
										},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = true (1 segment, 1 constraint, 0 distributions",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
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
													id:       "1",
													typ:      numberConstraintComparisonType,
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = true (1 segment, 2 constraint ALL segment operator, 0 distributions",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"toggle1": {
								{
									id:              "1",
									rank:            1,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:       "segment1",
											matchType: allSegmentMatchType,
											constraints: []*EvalConstraint{
												{
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = false, country_id is not eq (1 segment, 2 constraint ALL segment operator, 0 distributions)",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"toggle1": {
								{
									id:              "1",
									rank:            1,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:       "segment1",
											matchType: allSegmentMatchType,
											constraints: []*EvalConstraint{
												{
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             false,
					SegmentKeys:       []string{},
					Reason:            reasonUnknown,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = false, city_id is not eq (1 segment, 2 constraint ALL segment operator, 0 distributions)",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "2",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"toggle1": {
								{
									id:              "1",
									rank:            1,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:       "segment1",
											matchType: allSegmentMatchType,
											constraints: []*EvalConstraint{
												{
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             false,
					SegmentKeys:       []string{},
					Reason:            reasonUnknown,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = false, all context is not eq (1 segment, 2 constraint ALL segment operator, 0 distributions)",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "2",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"toggle1": {
								{
									id:              "1",
									rank:            1,
									segmentOperator: orSegmentOperator,
									segments: []*EvalSegment{
										{
											key:       "segment1",
											matchType: allSegmentMatchType,
											constraints: []*EvalConstraint{
												{
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             false,
					SegmentKeys:       []string{},
					Reason:            reasonUnknown,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = true, city_id is eq (1 segment, 2 constraint ANY segment operator, 0 distributions",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
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
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = true, country_id is eq (1 segment, 2 constraint ANY segment operator, 0 distributions",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "2",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
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
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = false, all context is not eq (1 segment, 2 constraint ANY segment operator, 0 distributions",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "2",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
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
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             false,
					SegmentKeys:       []string{},
					Reason:            reasonUnknown,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = false, all context is eq (1 segment, 2 constraint ANY segment operator, 0 distributions",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
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
													id:       "1",
													typ:      numberConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "",
					VariantAttachment: "",
				},
				wantErr: nil,
			},
		},
		{
			name: "error in match constraints",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
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
													id:       "1",
													typ:      unknownIDConstraintComparisonType,
													property: "city_id",
													operator: "eq",
													value:    "1",
												},
												{
													id:       "2",
													typ:      stringConstraintComparisonType,
													property: "country_id",
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
			},
			want: want{
				want:    nil,
				wantErr: errors.New("match constraints, rule_id '1', segment_key 'segment1: unknown constraint type: UNKNOWN_CONSTRAINT_COMPARISON_TYPE"),
			},
		},
		{
			name: "match = true, single distribution",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "777323",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
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
											ruleID:            "1",
											rollout:           100,
											variantKey:        "var1",
											variantAttachment: "attach1",
										},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "var1",
					VariantAttachment: "attach1",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = true, multi distribution (variant 1)",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "mobile_toggle1",
					EntityID:     "4",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "mobile_toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"mobile_toggle1": {
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
											ruleID:            "1",
											rollout:           50,
											variantKey:        "var1",
											variantAttachment: "attach1",
										},
										{
											ruleID:            "1",
											rollout:           50,
											variantKey:        "var2",
											variantAttachment: "attach2",
										},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "mobile_toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "var1",
					VariantAttachment: "attach1",
				},
				wantErr: nil,
			},
		},
		{
			name: "match = true, multi distribution (variant 2)",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "mobile_toggle1",
					EntityID:     "1",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "mobile_toggle1",
					typ:     variantFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "mobile",
						rules: map[string][]*EvalRule{
							"mobile_toggle1": {
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
											ruleID:            "1",
											rollout:           50,
											variantKey:        "var1",
											variantAttachment: "attach1",
										},
										{
											ruleID:            "1",
											rollout:           50,
											variantKey:        "var2",
											variantAttachment: "attach2",
										},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseVariant{
					FlagKey:           "mobile_toggle1",
					Match:             true,
					SegmentKeys:       []string{"segment1"},
					Reason:            reasonMatch,
					VariantKey:        "var2",
					VariantAttachment: "attach2",
				},
				wantErr: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Evaluator{
				snapshot: NewSnapshot(),
				logger:   zaptest.NewLogger(t),
			}
			r.snapshot.replaceStore(tt.mock.namespaces)

			response, err := r.evalVariant(tt.args.req, tt.args.flag)
			if tt.want.wantErr != nil {
				assert.EqualError(t, err, tt.want.wantErr.Error())
				assert.Equal(t, tt.want.want, response)
			} else {
				assert.NoError(t, err)
				assert.Equalf(t, tt.want.want, response, "variant(%v, %v)", tt.args.req, tt.args.flag)
			}
		})
	}
}

func TestEvaluator_boolean(t *testing.T) {
	type args struct {
		req  *RequestEvaluation
		flag *EvalFlag
	}
	type mock struct {
		namespaces map[string]*EvalNamespace
	}
	type want struct {
		want    *ResponseBoolean
		wantErr error
	}
	tests := []struct {
		name string
		args
		mock
		want
	}{
		{
			name: "flag type is not boolean",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     variantFlagType,
					enabled: false,
				},
			},
			want: want{
				want:    nil,
				wantErr: errors.New("flag type is not boolean"),
			},
		},
		{
			name: "flag is disabled",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     booleanFlagType,
					enabled: false,
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "toggle1",
					Enabled: false,
					Reason:  reasonDisabled,
				},
				wantErr: nil,
			},
		},
		{
			name: "rollouts not found",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "toggle1",
					Enabled: true,
					Reason:  reasonDefault,
				},
				wantErr: nil,
			},
		},
		{
			name: "rollouts len is 0",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "mobile",
					FlagKey:      "toggle1",
					EntityID:     "77777",
					Context: map[string]string{
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "toggle1",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"mobile": {
						key: "toggle1",
						rollouts: map[string][]*EvalRollout{
							"toggle1": {},
						},
					},
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "toggle1",
					Enabled: true,
					Reason:  reasonDefault,
				},
				wantErr: nil,
			},
		},
		{
			name: "default rule fallthrough with percentage rollout",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello": "world",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  thresholdRolloutType,
									rank: 1,
									threshold: &EvalThresholdRollout{
										percentage: 5,
										value:      false,
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: true,
					Reason:  reasonDefault,
				},
				wantErr: nil,
			},
		},
		{
			name: "percentage rule match",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello": "world",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  thresholdRolloutType,
									rank: 1,
									threshold: &EvalThresholdRollout{
										percentage: 70,
										value:      false,
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "the first threshold rollout is match",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello": "world",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  thresholdRolloutType,
									rank: 1,
									threshold: &EvalThresholdRollout{
										percentage: 70,
										value:      false,
									},
								},
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "the second segment rollout is match",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello": "world",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  thresholdRolloutType,
									rank: 1,
									threshold: &EvalThresholdRollout{
										percentage: 65,
										value:      false,
									},
								},
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      stringConstraintComparisonType,
														property: "hello",
														operator: "eq",
														value:    "world",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment match, multiple constraints any operator",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello":   "world",
						"city_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: anySegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      stringConstraintComparisonType,
														property: "hello",
														operator: "eq",
														value:    "world",
													},
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      numberConstraintComparisonType,
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment not match, multiple constraints any operator",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello":   "worl2d",
						"city_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: anySegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      stringConstraintComparisonType,
														property: "hello",
														operator: "eq",
														value:    "world",
													},
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      numberConstraintComparisonType,
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: true,
					Reason:  reasonDefault,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment match, multiple constraints with all operator",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello":   "world",
						"city_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      stringConstraintComparisonType,
														property: "hello",
														operator: "eq",
														value:    "world",
													},
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      numberConstraintComparisonType,
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment not match, multiple constraints with all operator",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"hello":   "world",
						"city_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      stringConstraintComparisonType,
														property: "hello",
														operator: "eq",
														value:    "world",
													},
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      numberConstraintComparisonType,
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: true,
					Reason:  reasonDefault,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment is match, entity_id constraint",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context:      map[string]string{},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      entityIDConstraintComparisonType,
														property: "entity",
														operator: "eq",
														value:    "test-entity",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment is match, multiple segment operator AND",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: andSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      numberConstraintComparisonType,
														property: "city_id",
														operator: "eq",
														value:    "1",
													},
												},
											},
											{
												key:       "segment2",
												matchType: anySegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      stringConstraintComparisonType,
														property: "country_id",
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment is not match, multiple segment operator AND",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: andSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      numberConstraintComparisonType,
														property: "city_id",
														operator: "eq",
														value:    "1",
													},
												},
											},
											{
												key:       "segment2",
												matchType: anySegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      stringConstraintComparisonType,
														property: "country_id",
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: true,
					Reason:  reasonDefault,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment is match, multiple segment operator OR",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      numberConstraintComparisonType,
														property: "city_id",
														operator: "eq",
														value:    "1",
													},
												},
											},
											{
												key:       "segment2",
												matchType: anySegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      stringConstraintComparisonType,
														property: "country_id",
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment is not match, multiple segment operator OR",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "2",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      numberConstraintComparisonType,
														property: "city_id",
														operator: "eq",
														value:    "1",
													},
												},
											},
											{
												key:       "segment2",
												matchType: anySegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      stringConstraintComparisonType,
														property: "country_id",
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: true,
					Reason:  reasonDefault,
				},
				wantErr: nil,
			},
		},
		{
			name: "rules out of order",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "2",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  thresholdRolloutType,
									rank: 1,
									threshold: &EvalThresholdRollout{
										percentage: 5,
										value:      false,
									},
								},
								{
									typ:  segmentRolloutType,
									rank: 0,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      numberConstraintComparisonType,
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
				},
			},
			want: want{
				want:    nil,
				wantErr: errors.New("rollout type SEGMENT_ROLLOUT_TYPE rank: 0 detected out of order"),
			},
		},
		{
			name: "unknown match constraint type",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "2",
						"country_id": "2",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 0,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
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
						},
					},
				},
			},
			want: want{
				want:    nil,
				wantErr: errors.New("match constraints, rollout type 'SEGMENT_ROLLOUT_TYPE', segment_key 'segment1': unknown constraint type: UNKNOWN_CONSTRAINT_COMPARISON_TYPE"),
			},
		},
		{
			name: "segment is match, duplicate constraint operator OR",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: orSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      numberConstraintComparisonType,
														property: "city_id",
														operator: "eq",
														value:    "1",
													},
												},
											},
											{
												key:       "segment2",
												matchType: anySegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      stringConstraintComparisonType,
														property: "country_id",
														operator: "eq",
														value:    "1",
													},
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      numberConstraintComparisonType,
														property: "country_id",
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
		{
			name: "segment is match, duplicate constraint operator ALL",
			args: args{
				req: &RequestEvaluation{
					NamespaceKey: "test-namespace",
					FlagKey:      "test-flag",
					EntityID:     "test-entity",
					Context: map[string]string{
						"city_id":    "1",
						"country_id": "1",
					},
				},
				flag: &EvalFlag{
					key:     "test-flag",
					typ:     booleanFlagType,
					enabled: true,
				},
			},
			mock: mock{
				namespaces: map[string]*EvalNamespace{
					"test-namespace": {
						key: "test-flag",
						rollouts: map[string][]*EvalRollout{
							"test-flag": {
								{
									typ:  segmentRolloutType,
									rank: 2,
									segment: &EvalSegmentRollout{
										value:           false,
										segmentOperator: andSegmentOperator,
										segments: []*EvalSegment{
											{
												key:       "segment1",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "25af15c6-e6d9-4b7c-a046-d58ee36b1a35",
														typ:      numberConstraintComparisonType,
														property: "city_id",
														operator: "eq",
														value:    "1",
													},
												},
											},
											{
												key:       "segment2",
												matchType: allSegmentMatchType,
												constraints: []*EvalConstraint{
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      stringConstraintComparisonType,
														property: "country_id",
														operator: "eq",
														value:    "1",
													},
													{
														id:       "a2cab611-8289-4273-9cd8-53d6a418acd8",
														typ:      numberConstraintComparisonType,
														property: "country_id",
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
				},
			},
			want: want{
				want: &ResponseBoolean{
					FlagKey: "test-flag",
					Enabled: false,
					Reason:  reasonMatch,
				},
				wantErr: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Evaluator{
				snapshot: NewSnapshot(),
				logger:   zaptest.NewLogger(t),
			}
			r.snapshot.replaceStore(tt.mock.namespaces)

			response, err := r.Boolean(tt.args.req, tt.args.flag)
			if tt.want.wantErr != nil {
				assert.EqualError(t, err, tt.want.wantErr.Error())
				assert.Equal(t, tt.want.want, response)
			} else {
				assert.NoError(t, err)
				assert.Equalf(t, tt.want.want, response, "variant(%v, %v)", tt.args.req, tt.args.flag)
			}
		})
	}
}
