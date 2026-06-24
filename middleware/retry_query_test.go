package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func TestQueryRetry_Success(t *testing.T) {
	t.Parallel()

	config := retryConfigBasic()

	mw := QueryRetry(config)

	callCount := 0
	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		callCount++
		if callCount < 2 {
			return nil, errors.New("transient")
		}

		return "ok", nil
	})

	result, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "ok" {
		t.Errorf("expected ok, got %v", result)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestQueryRetry_AllAttemptsFail(t *testing.T) {
	t.Parallel()

	config := retryConfigFast()

	mw := QueryRetry(config)

	handler := mw(failingQueryHandler("always fail"))

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func errorQueryHandler(
	errMsg string,
	callCount *int,
) query.Handler { //nolint:staticcheck // SA1019: testing deprecated Handler backward compat
	return func(_ context.Context, _ query.Query) (any, error) {
		*callCount++

		return nil, errors.New(errMsg)
	}
}

func setupQueryRetryHandler(
	t *testing.T,
	config RetryConfig,
	errMsg string,
) (query.Handler, *int) { //nolint:staticcheck // SA1019: testing deprecated Handler backward compat
	t.Helper()
	mw := QueryRetry(config)
	callCount := 0
	handler := mw(errorQueryHandler(errMsg, &callCount))

	return handler, &callCount
}

func TestQueryRetry_NonRetryable(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = func(_ error) bool { return false }

	handler, callCount := setupQueryRetryHandler(t, config, "non-retryable")

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	if *callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", *callCount)
	}
}

func TestQueryRetry_ContextCancellation(t *testing.T) {
	t.Parallel()

	config := retryConfigSlow()

	handler, _ := setupQueryRetryHandler(t, config, "transient")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := handler(ctx, &testQuery{})
	if err == nil {
		t.Fatal("expected error from canceled context")
	}

	assertRetryCanceled(t, err)
}
