package integration_test

import (
	"errors"
	"fmt"
	"testing"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/graph/v3"
	"github.com/larsartmann/go-cqrs-lite/middleware/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// TestErrorClassification verifies that sentinels from modules that were
// converted from plain errors.New to event.NewRejection (etc.) are now
// classified correctly by the taxonomy. Before the fix, these fell through to
// the Transient default and were retried by the retry middleware — even
// though they represent non-retryable Rejections.
func TestErrorClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantFamily cqrsevent.Family
		wantRetry  bool
	}{
		// graph: schema and sink violations are Rejections
		{"graph path not found", graph.ErrPathNotFound, cqrsevent.Rejection, false},
		{
			"graph wrapped schema error",
			fmt.Errorf("validate: %w", errors.New("graph schema: duplicate node label")),
			cqrsevent.Transient, // plain errors.New still defaults to Transient
			true,
		},

		// stack: bundle misconfiguration is Rejection
		{"stack empty bundle", stack.ErrEmpty, cqrsevent.Rejection, false},
		{"stack missing event store", stack.ErrMissingEventStore, cqrsevent.Rejection, false},
		{"stack missing read models", stack.ErrMissingReadModels, cqrsevent.Rejection, false},
		{"stack missing journal", stack.ErrMissingJournal, cqrsevent.Rejection, false},

		// middleware: meter required is Rejection (programmer error)
		{"middleware meter required", middleware.ErrMeterRequired, cqrsevent.Rejection, false},

		// middleware: already-classified sentinels stay correct
		{
			"middleware retry exhausted",
			middleware.ErrRetryExhausted,
			cqrsevent.Infrastructure,
			false,
		},
		{"middleware panic recovered", middleware.ErrPanicRecovered, cqrsevent.Corruption, false},
		{
			"middleware validation failed",
			middleware.ErrValidationFailed,
			cqrsevent.Rejection,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			family := cqrsevent.Classify(tc.err)
			if family != tc.wantFamily {
				t.Errorf("Classify(%v) = %v, want %v", tc.err, family, tc.wantFamily)
			}

			retry := cqrsevent.IsRetryable(tc.err)
			if retry != tc.wantRetry {
				t.Errorf("IsRetryable(%v) = %v, want %v", tc.err, retry, tc.wantRetry)
			}
		})
	}
}

// TestErrorClassificationWrappedChain verifies that classification survives
// %w wrapping — a core taxonomy contract. A wrapped Rejection must still
// classify as Rejection.
func TestErrorClassificationWrappedChain(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("dispatch failed: %w", stack.ErrMissingEventStore)

	if cqrsevent.Classify(wrapped) != cqrsevent.Rejection {
		t.Errorf("wrapped stack.ErrMissingEventStore classified as %v, want Rejection",
			cqrsevent.Classify(wrapped))
	}

	if cqrsevent.IsRetryable(wrapped) {
		t.Error("wrapped stack.ErrMissingEventStore should not be retryable")
	}
}
