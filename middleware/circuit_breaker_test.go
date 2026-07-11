package middleware

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func testCBConfig(failureThreshold, successThreshold int) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: failureThreshold,
		SuccessThreshold: successThreshold,
		Timeout:          50 * time.Millisecond,
		IsFailure:        func(err error) bool { return true },
	}
}

func TestCommandCircuitBreaker_OpensAfterFailures(t *testing.T) {
	t.Parallel()

	config := testCBConfig(3, 1)

	handler := CommandCircuitBreaker(config)(failingCommandHandler("fail"))

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

	config := testCBConfig(2, 1)

	failingHandler := CommandCircuitBreaker(config)(failingCommandHandler("fail"))

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

	config := testCBConfig(2, 1)

	handler := EventCircuitBreaker(config)(eventtest.FailingEventHandler("fail"))
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

	config := testCBConfig(2, 1)

	handler := QueryCircuitBreaker(config)(failingQueryHandler("fail"))

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

func TestCommandCircuitBreaker_HalfOpenToClosedViaSuccesses(t *testing.T) {
	t.Parallel()

	config := testCBConfig(2, 3)

	circuitBreakerMW := CommandCircuitBreaker(config)
	failingHandler := circuitBreakerMW(failingCommandHandler("fail"))

	for range 2 {
		_ = failingHandler(t.Context(), &testCommand{})
	}

	_ = failingHandler(t.Context(), &testCommand{})
	time.Sleep(60 * time.Millisecond)

	successHandler := circuitBreakerMW(NoopCommandHandler())

	for i := range 3 {
		if err := successHandler(t.Context(), &testCommand{}); err != nil {
			t.Fatalf("expected success in half-open attempt %d, got %v", i+1, err)
		}
	}

	failAgain := circuitBreakerMW(failingCommandHandler("fail"))
	for range 2 {
		_ = failAgain(t.Context(), &testCommand{})
	}

	err := failAgain(t.Context(), &testCommand{})
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("expected circuit to be closed (reset) and reopen, got %v", err)
	}
}

func TestCommandCircuitBreaker_HalfOpenReopensOnFailure(t *testing.T) {
	t.Parallel()

	config := testCBConfig(1, 2)

	circuitBreakerMW := CommandCircuitBreaker(config)
	handler := circuitBreakerMW(failingCommandHandler("fail"))

	_ = handler(t.Context(), &testCommand{})
	_ = handler(t.Context(), &testCommand{})
	time.Sleep(60 * time.Millisecond)

	err := handler(t.Context(), &testCommand{})
	if err == nil {
		t.Fatal("expected error from failing handler in half-open")
	}

	err = handler(t.Context(), &testCommand{})
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("expected circuit to reopen after half-open failure, got %v", err)
	}
}

func TestCommandCircuitBreaker_NonFailureErrorRecordsSuccess(t *testing.T) {
	t.Parallel()

	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		IsFailure:        func(err error) bool { return false },
	}

	handler := CommandCircuitBreaker(config)(failingCommandHandler("fail"))

	for i := range 10 {
		err := handler(t.Context(), &testCommand{})
		if err == nil {
			t.Fatalf("expected error from handler on attempt %d", i+1)
		}
	}

	err := handler(t.Context(), &testCommand{})
	if errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatal("circuit should not open when errors are not classified as failures")
	}
}

func TestCommandCircuitBreaker_WithLogger(t *testing.T) {
	t.Parallel()

	config := testCBConfig(1, 1)

	handler := CommandCircuitBreaker(
		config,
		WithLogger(slog.Default()),
	)(
		failingCommandHandler("fail"),
	)

	_ = handler(t.Context(), &testCommand{})
	_ = handler(t.Context(), &testCommand{})
}

func mustCBTestEvent(tb testing.TB) event.Event {
	tb.Helper()

	evt, err := event.New(
		event.Type("test.event"),
		id.NewAggregateID(),
		id.AggregateType("Test"),
		event.Version(1),
		map[string]any{"data": "test"},
	)
	if err != nil {
		tb.Fatalf("create test event: %v", err)
	}

	return evt
}
