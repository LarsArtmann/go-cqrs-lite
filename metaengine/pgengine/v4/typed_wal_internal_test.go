package pgengine

import (
	"errors"
	"fmt"
	"testing"
)

type fakeSQLStateError struct{ code string }

func (e *fakeSQLStateError) Error() string    { return "pg: " + e.code }
func (e *fakeSQLStateError) SQLState() string { return e.code }

func TestIsUniqueViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"driver SQLSTATE 23505", &fakeSQLStateError{code: "23505"}, true},
		{"driver other SQLSTATE", &fakeSQLStateError{code: "42P01"}, false},
		{
			"wrapped driver SQLSTATE",
			fmt.Errorf("save: %w", &fakeSQLStateError{code: "23505"}),
			true,
		},
		{
			"message fallback",
			errors.New(`duplicate key value violates unique constraint "events_pkey"`),
			true,
		},
		{"message SQLSTATE text", errors.New("conflicting key (SQLSTATE 23505)"), true},
		{"unrelated error", errors.New("connection refused"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isUniqueViolation(tt.err); got != tt.want {
				t.Errorf("isUniqueViolation(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
