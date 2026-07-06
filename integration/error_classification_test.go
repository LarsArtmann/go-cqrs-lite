package integration_test

import (
	"errors"
	"fmt"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/graph/v3"
	"github.com/larsartmann/go-cqrs-lite/middleware/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// TestErrorClassification verifies that sentinels from modules that were
// converted from plain errors.New to errorfamily.NewRejection (etc.) are now
// classified correctly by the taxonomy. Before the fix, these fell through to
// the Transient default and were retried by the retry middleware — even
// though they represent non-retryable Rejections.
func TestErrorClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantFamily errorfamily.Family
		wantRetry  bool
	}{
		// graph: schema and sink violations are Rejections
		{"graph path not found", graph.ErrPathNotFound, errorfamily.Rejection, false},
		{
			"graph wrapped schema error",
			fmt.Errorf("validate: %w", errors.New("graph schema: duplicate node label")),
			errorfamily.Transient, // plain errors.New still defaults to Transient
			true,
		},

		// stack: bundle misconfiguration is Rejection
		{"stack empty bundle", stack.ErrEmpty, errorfamily.Rejection, false},
		{"stack missing event store", stack.ErrMissingEventStore, errorfamily.Rejection, false},
		{"stack missing read models", stack.ErrMissingReadModels, errorfamily.Rejection, false},
		{"stack missing journal", stack.ErrMissingJournal, errorfamily.Rejection, false},

		// middleware: meter required is Rejection (programmer error)
		{"middleware meter required", middleware.ErrMeterRequired, errorfamily.Rejection, false},

		// middleware: already-classified sentinels stay correct
		{
			"middleware retry exhausted",
			middleware.ErrRetryExhausted,
			errorfamily.Infrastructure,
			false,
		},
		{"middleware panic recovered", middleware.ErrPanicRecovered, errorfamily.Corruption, false},
		{
			"middleware validation failed",
			middleware.ErrValidationFailed,
			errorfamily.Rejection,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			family := errorfamily.Classify(tc.err)
			if family != tc.wantFamily {
				t.Errorf("Classify(%v) = %v, want %v", tc.err, family, tc.wantFamily)
			}

			retry := errorfamily.IsRetryable(tc.err)
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

	if errorfamily.Classify(wrapped) != errorfamily.Rejection {
		t.Errorf("wrapped stack.ErrMissingEventStore classified as %v, want Rejection",
			errorfamily.Classify(wrapped))
	}

	if errorfamily.IsRetryable(wrapped) {
		t.Error("wrapped stack.ErrMissingEventStore should not be retryable")
	}
}
