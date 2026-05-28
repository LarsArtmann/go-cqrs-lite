package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestCommandCircuitBreaker_OpensAfterFailures(t *testing.T) {
	t.Parallel()

	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		IsFailure:        func(err error) bool { return true },
	}

	handler := CommandCircuitBreaker(config)(testhelpers.FailingCommandHandler("fail"))

	for i := range 3 {
		if err := handler(t.Context(), &testCommand{}); err == nil {
			t.Fatalf("expected error on attempt %d", i+1)
		}
	}

	err := handler(t.Context(), &testCommand{})
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestCommandCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	t.Parallel()

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		IsFailure:        func(err error) bool { return true },
	}

	failingHandler := CommandCircuitBreaker(config)(testhelpers.FailingCommandHandler("fail"))

	for range 2 {
		_ = failingHandler(t.Context(), &testCommand{})
	}

	_ = failingHandler(t.Context(), &testCommand{})
	time.Sleep(60 * time.Millisecond)

	var called bool
	successHandler := CommandCircuitBreaker(
		config,
	)(
		func(_ context.Context, _ command.Command) error {
			called = true

			return nil
		},
	)

	if err := successHandler(t.Context(), &testCommand{}); err != nil {
		t.Fatalf("expected success after half-open, got %v", err)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestCommandCircuitBreaker_InvalidConfig(t *testing.T) {
	t.Parallel()

	handler := CommandCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 0})(nil)

	err := handler(t.Context(), &testCommand{})
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestEventCircuitBreaker(t *testing.T) {
	t.Parallel()

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		IsFailure:        func(err error) bool { return true },
	}

	handler := EventCircuitBreaker(config)(testhelpers.FailingEventHandler("fail"))
	evt := mustCBTestEvent(t)

	for range 2 {
		_ = handler(t.Context(), evt)
	}

	err := handler(t.Context(), evt)
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestQueryCircuitBreaker(t *testing.T) {
	t.Parallel()

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		IsFailure:        func(err error) bool { return true },
	}

	handler := QueryCircuitBreaker(config)(func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New("fail")
	})

	for range 2 {
		_, _ = handler(t.Context(), &testQuery{})
	}

	_, err := handler(t.Context(), &testQuery{})
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("expected ErrCircuitBreakerOpen, got %v", err)
	}
}

func TestCircuitBreakerConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  CircuitBreakerConfig
		wantErr bool
	}{
		{"valid default", DefaultCircuitBreakerConfig(), false},
		{
			"zero failure threshold",
			CircuitBreakerConfig{FailureThreshold: 0, SuccessThreshold: 1, Timeout: time.Second},
			true,
		},
		{
			"zero success threshold",
			CircuitBreakerConfig{FailureThreshold: 1, SuccessThreshold: 0, Timeout: time.Second},
			true,
		},
		{
			"zero timeout",
			CircuitBreakerConfig{FailureThreshold: 1, SuccessThreshold: 1, Timeout: 0},
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.config.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func mustCBTestEvent(tb testing.TB) event.Event {
	tb.Helper()

	evt, err := event.New(
		event.Type("test.event"),
		id.NewAggregateID(),
		event.AggregateType("Test"),
		event.Version(1),
		map[string]any{"data": "test"},
	)
	if err != nil {
		tb.Fatalf("create test event: %v", err)
	}

	return evt
}
