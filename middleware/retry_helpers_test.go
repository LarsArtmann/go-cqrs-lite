package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func alwaysRetryable() func(_ error) bool {
	return func(_ error) bool { return true }
}

func retryConfigBasic() RetryConfig {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = alwaysRetryable()

	return config
}

func retryConfigFast() RetryConfig {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = time.Millisecond
	config.IsRetryable = alwaysRetryable()

	return config
}

func retryConfigSlow() RetryConfig {
	config := DefaultRetryConfig()
	config.MaxAttempts = 5
	config.InitialDelay = 50 * time.Millisecond
	config.IsRetryable = alwaysRetryable()

	return config
}

func newEventNonRetryableHandler(callCount *int, errMsg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		*callCount++

		return errors.New(errMsg)
	}
}

func setupEventNonRetryableTest(
	t testing.TB,
	mw event.Middleware,
	callCount *int,
	errMsg string,
) (event.Handler, event.Event) {
	t.Helper()

	handler := mw(newEventNonRetryableHandler(callCount, errMsg))

	evt, err := testhelpers.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return handler, evt
}

func newCommandNonRetryableHandler(callCount *int, errMsg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*callCount++

		return errors.New(errMsg)
	}
}

func setupCommandNonRetryableTest(
	t testing.TB,
	mw command.Middleware,
	callCount *int,
	errMsg string,
) (command.Handler, command.Command) {
	t.Helper()

	handler := mw(newCommandNonRetryableHandler(callCount, errMsg))
	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	return handler, cmd
}
