package event_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// failingPublisher fails the first n publishes with the given error, then
// succeeds. Records the attempt count.
type failingPublisher struct {
	n         int32
	err       error
	attempts  atomic.Int32
	published atomic.Int32
}

func (f *failingPublisher) Publish(_ context.Context, _ ...event.Event) error {
	if f.attempts.Add(1) <= f.n {
		return f.err
	}
	f.published.Add(1)
	return nil
}

func newTestEvents(t *testing.T, n int) []event.Event {
	t.Helper()

	events := make([]event.Event, 0, n)
	for i := range n {
		evt, err := event.New(
			event.Type("TestPublished"),
			id.NewStreamID(),
			"Test",
			1,
			map[string]string{"n": string(rune('a' + i))},
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		events = append(events, evt)
	}
	return events
}

func TestPublishRetry_SucceedsAfterTransientFailure(t *testing.T) {
	t.Parallel()

	delegate := &failingPublisher{n: 2, err: errors.New("transient bus hiccup")}
	mw := event.PublishRetry(4, time.Millisecond, 5*time.Millisecond, slog.Default())
	pub := mw(delegate)

	if err := pub.Publish(context.Background(), newTestEvents(t, 1)...); err != nil {
		t.Fatalf("publish should succeed after retries: %v", err)
	}
	if got := delegate.attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3 (1 initial + 2 retries)", got)
	}
}

func TestPublishRetry_NonRetryableFailsImmediately(t *testing.T) {
	t.Parallel()

	delegate := &failingPublisher{n: 99, err: errorfamily.NewRejection("test.rejection", "no")}
	mw := event.PublishRetry(4, time.Millisecond, 5*time.Millisecond, slog.Default())
	pub := mw(delegate)

	err := pub.Publish(context.Background(), newTestEvents(t, 1)...)
	if err == nil {
		t.Fatal("expected the rejection to propagate")
	}
	if got := delegate.attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (Rejection must not be retried)", got)
	}
}

func TestPublishRetry_ExhaustsAttempts(t *testing.T) {
	t.Parallel()

	delegate := &failingPublisher{n: 99, err: errors.New("always transient")}
	mw := event.PublishRetry(3, time.Millisecond, 2*time.Millisecond, slog.Default())
	pub := mw(delegate)

	if err := pub.Publish(context.Background(), newTestEvents(t, 1)...); err == nil {
		t.Fatal("expected exhaustion error")
	}
	if got := delegate.attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestPublishRetry_ContextCancelledBetweenAttempts(t *testing.T) {
	t.Parallel()

	delegate := &failingPublisher{n: 99, err: errors.New("transient")}
	mw := event.PublishRetry(5, 50*time.Millisecond, time.Second, slog.Default())
	pub := mw(delegate)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := pub.Publish(ctx, newTestEvents(t, 1)...); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
