package projections_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4"
)

func TestDeclarations_ConstructWithoutPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func() any
	}{
		{"DeadLetterQueue", func() any { return projections.DeadLetterQueue() }},
		{"RetryCount", func() any { return projections.RetryCount() }},
		{"FailureLog", func() any { return projections.FailureLog() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.fn(); got == nil {
				t.Fatalf("%s returned nil", tt.name)
			}
		})
	}
}

func TestAll_ReturnsThreeDeclarations(t *testing.T) {
	t.Parallel()

	all := projections.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(all))
	}
}
