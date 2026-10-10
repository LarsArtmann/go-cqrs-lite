package watermill_test

// ErrReentrantPublish guard tests (ADR-0154): a nested synchronous publish
// from a delivery handler previously deadlocked forever under
// BlockPublishUntilSubscriberAck (live stack evidence:
// docs/evidence/2026-10-09_deriver-bus-deadlock.md). It must now fail fast
// with the named sentinel, everything else must be unaffected.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func reentrancyEvent(t *testing.T, eventType event.Type) event.Event {
	t.Helper()

	ref := id.NewStreamRef("Task", id.NewStreamID())
	evt, err := event.New(eventType, ref.ID, "Task", event.Version(1), map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}

	return evt
}

// TestEventBus_ReentrantPublishFailsFast pins the loud-fail contract: a
// handler publishing back to the same bus synchronously gets
// ErrReentrantPublish on the nested Publish (surfaced through the handler
// error path) instead of a permanent hang. The 5s watchdog makes a
// regression (hang) fail the test rather than wedge it forever.
func TestEventBus_ReentrantPublishFailsFast(t *testing.T) {
	t.Parallel()

	bus := watermill.NewEventBus()
	t.Cleanup(func() { _ = bus.Close() })

	nested := make(chan error, 1)

	err := bus.Subscribe("task.updated", func(ctx context.Context, evt event.Event) error {
		nested <- bus.Publish(ctx, reentrancyEvent(t, "task.updated"))

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- bus.Publish(context.Background(), reentrancyEvent(t, "task.updated"))
	}()

	select {
	case pubErr := <-done:
		if pubErr != nil {
			t.Fatalf("outer publish must succeed (handler acked; nested error is the handler's), got: %v", pubErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("outer publish hung — the reentrancy guard did not fire (deadlock regression)")
	}

	select {
	case err := <-nested:
		if !errors.Is(err, watermill.ErrReentrantPublish) {
			t.Fatalf("nested publish must fail with ErrReentrantPublish, got: %v", err)
		}
	default:
		t.Fatal("nested publish result missing — handler did not run")
	}
}

// TestEventBus_AsyncEscapePublishAllowed pins that a handler publishing on
// its OWN goroutine with the mark cleared (event.WithoutDeliveryMark — the
// deriver.WithAsyncDispatch pattern) publishes cleanly: the guard targets
// the synchronous same-goroutine cycle only.
func TestEventBus_AsyncEscapePublishAllowed(t *testing.T) {
	t.Parallel()

	bus := watermill.NewEventBus()
	t.Cleanup(func() { _ = bus.Close() })

	published := make(chan error, 1)

	err := bus.Subscribe("task.created", func(ctx context.Context, evt event.Event) error {
		go func() {
			published <- bus.Publish(event.WithoutDeliveryMark(ctx), reentrancyEvent(t, "task.updated"))
		}()

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = bus.Subscribe("task.updated", func(_ context.Context, _ event.Event) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := bus.Publish(context.Background(), reentrancyEvent(t, "task.updated")); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-published:
		if err != nil {
			t.Fatalf("async escaped publish must succeed, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("async escaped publish never completed")
	}
}

// TestEventBus_NonNestedPublishUnaffected pins the guard's scope: ordinary
// publishes on ordinary contexts are untouched.
func TestEventBus_NonNestedPublishUnaffected(t *testing.T) {
	t.Parallel()

	bus := watermill.NewEventBus()
	t.Cleanup(func() { _ = bus.Close() })

	if err := bus.Publish(context.Background(), reentrancyEvent(t, "task.updated")); err != nil {
		t.Fatalf("plain publish must be unaffected by the guard, got: %v", err)
	}

	if event.ContextInDelivery(context.Background()) {
		t.Fatal("plain contexts must never report in-delivery")
	}
}
