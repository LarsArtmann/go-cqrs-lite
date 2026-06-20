package projection

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

func TestRunner_DeadLetterHandler(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	store := memory.NewMemoryStore()
	checkpointStore := memory.NewMemoryCheckpointStore()

	var (
		dlqMu      sync.Mutex
		dlqEntries []struct {
			projection string
			evt        event.Event
			err        error
		}
	)

	dlqHandler := func(ctx context.Context, projectionName string, evt event.Event, err error) {
		dlqMu.Lock()
		dlqEntries = append(dlqEntries, struct {
			projection string
			evt        event.Event
			err        error
		}{projectionName, evt, err})
		dlqMu.Unlock()
	}

	failingProj := event.NewProjection(
		"failing-projection",
		func(ctx context.Context, evt event.Event) error {
			return errors.New("always fails")
		},
		[]event.Type{"user.created"},
	)

	runner, err := NewRunner(
		store, bus, checkpointStore,
		WithDeadLetterHandler(dlqHandler),
	)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	if regErr := runner.Register(failingProj); regErr != nil {
		t.Fatalf("Register: %v", regErr)
	}

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		_ = runner.Run(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond) // let runner subscribe before publishing

	evt := mustCreateTestEvent(t, "user.created", id.NewAggregateID(), 1)
	if pubErr := bus.Publish(t.Context(), evt); pubErr != nil {
		t.Fatalf("Publish: %v", pubErr)
	}

	// Wait for DLQ entry (deterministic poll)
	dlqMu.Lock()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(dlqEntries) == 0 {
		dlqMu.Unlock()
		time.Sleep(5 * time.Millisecond)
		dlqMu.Lock()
	}
	if len(dlqEntries) == 0 {
		dlqMu.Unlock()
		t.Fatal("expected dead letter entries, got none")
	}
	dlqMu.Unlock()
	cancel()
	<-done

	dlqMu.Lock()
	defer dlqMu.Unlock()

	if len(dlqEntries) == 0 {
		t.Fatal("expected dead letter entries, got none")
	}

	if dlqEntries[0].projection != "failing-projection" {
		t.Fatalf("expected projection 'failing-projection', got %q", dlqEntries[0].projection)
	}

	if dlqEntries[0].evt.ID() != evt.ID() {
		t.Fatalf("expected event ID %s, got %s", evt.ID(), dlqEntries[0].evt.ID())
	}
}

func TestRunner_Reset(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	store := memory.NewMemoryStore()
	checkpointStore := memory.NewMemoryCheckpointStore()

	evtID := id.NewEventID()
	if saveErr := checkpointStore.Save(
		t.Context(),
		"test-projection",
		event.Checkpoint{EventID: evtID, ProcessedAt: time.Now()},
	); saveErr != nil {
		t.Fatalf("Save checkpoint: %v", saveErr)
	}

	runner, err := NewRunner(store, bus, checkpointStore)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	loaded, loadErr := checkpointStore.Load(t.Context(), "test-projection")
	if loadErr != nil {
		t.Fatalf("Load before reset: %v", loadErr)
	}

	if loaded.EventID != evtID {
		t.Fatalf("expected checkpoint %s before reset, got %s", evtID, loaded)
	}

	if resetErr := runner.Reset(t.Context(), "test-projection"); resetErr != nil {
		t.Fatalf("Reset: %v", resetErr)
	}

	afterReset, loadErr := checkpointStore.Load(t.Context(), "test-projection")
	if loadErr != nil {
		t.Fatalf("Load after reset: %v", loadErr)
	}

	if !afterReset.IsZero() {
		t.Fatalf("expected zero checkpoint after reset, got %s", afterReset)
	}
}

func TestRunner_DeadLetterHandler_WithRetry(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	store := memory.NewMemoryStore()
	checkpointStore := memory.NewMemoryCheckpointStore()

	var (
		dlqMu    sync.Mutex
		dlqCount int
	)

	dlqHandler := func(ctx context.Context, projectionName string, evt event.Event, err error) {
		dlqMu.Lock()
		dlqCount++
		dlqMu.Unlock()
	}

	var callCount atomic.Int32
	failingProj := event.NewProjection(
		"retry-projection",
		func(ctx context.Context, evt event.Event) error {
			callCount.Add(1)

			return errors.New("always fails")
		},
		[]event.Type{"user.created"},
	)

	runner, err := NewRunner(
		store, bus, checkpointStore,
		WithDeadLetterHandler(dlqHandler),
		WithRetry(2, 10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	if regErr := runner.Register(failingProj); regErr != nil {
		t.Fatalf("Register: %v", regErr)
	}

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		_ = runner.Run(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond) // let runner subscribe before publishing

	evt := mustCreateTestEvent(t, "user.created", id.NewAggregateID(), 1)
	if pubErr := bus.Publish(t.Context(), evt); pubErr != nil {
		t.Fatalf("Publish: %v", pubErr)
	}

	// Wait for DLQ processing (deterministic poll)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && dlqCount == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done

	dlqMu.Lock()
	defer dlqMu.Unlock()

	if dlqCount != 1 {
		t.Fatalf("expected 1 dead letter entry, got %d", dlqCount)
	}

	expectedCalls := int32(1 + 2)
	if callCount.Load() != expectedCalls {
		t.Fatalf(
			"expected %d handler calls (1 initial + 2 retries), got %d",
			expectedCalls,
			callCount.Load(),
		)
	}
}

func mustCreateTestEvent(
	tb testing.TB,
	eventType string,
	aggID id.AggregateID,
	version int,
) event.Event {
	tb.Helper()

	evt, err := event.New(
		event.Type(eventType),
		aggID,
		event.AggregateType("TestAggregate"),
		event.Version(version),
		map[string]any{"data": "test"},
	)
	if err != nil {
		tb.Fatalf("create test event: %v", err)
	}

	return evt
}
