package watermill_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// TestRouterIntegration_CorrelationIDAndRetry exercises both
// CorrelationIDMiddleware and NewRetryMiddleware through a real Watermill
// Router with an in-memory GoChannel pub/sub. This proves the middleware
// wrappers compose correctly in a real handler pipeline — not just in
// isolated unit tests.
func TestRouterIntegration_CorrelationIDAndRetry(t *testing.T) {
	t.Parallel()

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{Persistent: true}, //nolint:exhaustruct // test config
		watermill.NopLogger{},
	)

	router, err := message.NewRouter(
		message.RouterConfig{}, //nolint:exhaustruct // default config for test
		watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	// Add middleware: correlation ID first, then retry
	router.AddMiddleware(cqrswatermill.CorrelationIDMiddleware())
	router.AddMiddleware(cqrswatermill.NewRetryMiddleware(cqrswatermill.RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}))

	var (
		callCount  atomic.Int32
		gotCorrID  atomic.Value
		handlerRan atomic.Bool
	)

	router.AddHandler(
		"test-handler",
		"input",
		pubSub,
		"output",
		pubSub,
		func(msg *message.Message) ([]*message.Message, error) {
			count := callCount.Add(1)

			// Fail first 2 attempts to exercise retry middleware
			if count < 3 { //nolint:mnd // 2 failures
				return nil, errTransient
			}

			corrID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
			gotCorrID.Store(corrID)
			handlerRan.Store(true)

			return []*message.Message{msg}, nil
		},
	)

	// Run router
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	routerReady := make(chan struct{})

	go func() {
		close(routerReady)
		_ = router.Run(ctx) //nolint:errcheck // test cleanup
	}()

	defer func() { _ = router.Close() }()

	// Wait for router to start
	select {
	case <-router.Running():
	case <-time.After(2 * time.Second):
		t.Fatal("router did not start within 2s")
	}

	// Publish message with correlation ID
	msg := message.NewMessage("test-msg-id", []byte("payload"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "trace-abc-123")

	_ = routerReady // router is running

	err = pubSub.Publish("input", msg)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Wait for handler to succeed after retries
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if handlerRan.Load() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !handlerRan.Load() {
		t.Fatal("handler did not complete within 5s")
	}

	// Verify retry middleware retried the expected number of times
	if got := callCount.Load(); got != 3 { //nolint:mnd // 2 failures + 1 success
		t.Errorf("handler called %d times, want 3", got)
	}

	// Verify correlation ID was propagated through the router pipeline
	corrID, _ := gotCorrID.Load().(string)
	if corrID != "trace-abc-123" {
		t.Errorf("correlation ID = %q, want %q", corrID, "trace-abc-123")
	}
}

var errTransient = &transientError{}

type transientError struct{}

func (e *transientError) Error() string { return "transient failure" }
