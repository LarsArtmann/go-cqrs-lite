package decider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestWaitForVersion_ImmediateReturn(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	// Write 3 events.
	executeCounter(t, repo, streamID, 0, 0, "CounterCreated", 1)
	executeCounter(t, repo, streamID, 1, 1, "CounterIncremented", 2)
	executeCounter(t, repo, streamID, 2, 2, "CounterIncremented", 3)

	// WaitForVersion(3) should return immediately — version is already visible.
	events, err := repo.WaitForVersion(
		context.Background(), streamID, "Counter", 3,
		decider.WithWaitTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("WaitForVersion: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event (from version 3), got %d", len(events))
	}
}

func TestWaitForVersion_DelayedWrite(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	// Write initial event.
	executeCounter(t, repo, streamID, 0, 0, "CounterCreated", 1)

	// Start waiting for version 2 (not yet written).
	type result struct {
		events []event.Event
		err    error
	}
	done := make(chan result, 1)

	go func() {
		events, err := repo.WaitForVersion(
			context.Background(), streamID, "Counter", 2,
			decider.WithWaitTimeout(2*time.Second),
		)
		done <- result{events: events, err: err}
	}()

	// Give the waiter time to start polling.
	time.Sleep(50 * time.Millisecond)

	// Now write the second event.
	executeCounter(t, repo, streamID, 1, 1, "CounterIncremented", 2)

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("WaitForVersion: %v", res.err)
		}

		if len(res.events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(res.events))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForVersion did not return after delayed write")
	}
}

func TestWaitForVersion_Timeout(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	// Wait for version 1 on a stream with no events — should timeout.
	_, err := repo.WaitForVersion(
		context.Background(), streamID, "Counter", 1,
		decider.WithWaitTimeout(50*time.Millisecond),
		decider.WithPollInterval(10*time.Millisecond),
	)
	if !errors.Is(err, decider.ErrWaitTimeout) {
		t.Fatalf("expected ErrWaitTimeout, got %v", err)
	}
}

func TestWaitForVersion_ContextCancellation(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		_, err := repo.WaitForVersion(
			ctx, streamID, "Counter", 1,
			decider.WithWaitTimeout(10*time.Second),
			decider.WithPollInterval(10*time.Millisecond),
		)
		done <- err
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error after context cancellation, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForVersion did not return after context cancellation")
	}
}

func TestWaitForVersion_InvalidVersion(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	_, err := repo.WaitForVersion(
		context.Background(), streamID, "Counter", 0,
	)
	if err == nil {
		t.Fatal("expected error for version 0, got nil")
	}
}

func TestWaitForVersion_ReturnsEventsFromTargetVersion(t *testing.T) {
	t.Parallel()

	repo, store, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	// Write 5 events via the store directly.
	ref := id.NewStreamRef("Counter", streamID)

	events := make([]event.Event, 5)

	for i := range 5 {
		evt, _ := event.NewEvent(
			"CounterIncremented", streamID, "Counter",
			event.Version(i+1),
			[]byte(`{}`),
		)
		events[i] = evt
	}

	_ = store.AppendBatch(context.Background(), ref, events)

	// WaitForVersion(3) should return events from version 3 onwards.
	result, err := repo.WaitForVersion(
		context.Background(), streamID, "Counter", 3,
		decider.WithWaitTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("WaitForVersion: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 events (versions 3,4,5), got %d", len(result))
	}
}

func TestWaitForVersion_PreservesCallerDeadline(t *testing.T) {
	t.Parallel()

	repo, _, _ := newTestRepo(t)
	streamID := id.NewStreamID()

	// Caller context has a shorter deadline than the WaitTimeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()

	_, err := repo.WaitForVersion(
		ctx, streamID, "Counter", 1,
		decider.WithWaitTimeout(10*time.Second),
		decider.WithPollInterval(5*time.Millisecond),
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should respect the caller's 30ms deadline, not the 10s WaitTimeout.
	if elapsed > 500*time.Millisecond {
		t.Fatalf("WaitForVersion took %v, expected < 500ms (caller deadline 30ms)", elapsed)
	}
}
