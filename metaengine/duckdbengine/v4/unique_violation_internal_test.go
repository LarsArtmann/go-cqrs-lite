package duckdbengine

import (
	"errors"
	"testing"
)

// Both DuckDB wordings of a unique rejection map to ErrVersionConflict;
// without the second one a duplicate inside one batch is reported as a
// generic (retryable) error instead of the optimistic-concurrency conflict.
func TestIsUniqueViolation_BothWordings(t *testing.T) {
	t.Parallel()

	committed := errors.New(`Constraint Error: Duplicate key "a, b" violates unique constraint.`)
	inBatch := errors.New(`Constraint Error: PRIMARY KEY or UNIQUE constraint violation: duplicate key "a, b"`)
	other := errors.New(`Constraint Error: NOT NULL constraint violation: col`)

	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"committed duplicate":    {committed, true},
		"duplicate in one batch": {inBatch, true},
		"other constraint":       {other, false},
		"nil":                    {nil, false},
	} {
		if got := isUniqueViolation(tc.err); got != tc.want {
			t.Errorf("%s: isUniqueViolation = %v, want %v", name, got, tc.want)
		}
	}
}
