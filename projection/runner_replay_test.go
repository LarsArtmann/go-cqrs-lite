package projection_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func TestRunner_ReplayFromStore(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	evt1 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt2 := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	ctx := context.Background()

	err := store.Save(ctx, "User", evt1.AggregateID(), []event.Event{evt1}, 0)
	if err != nil {
		t.Fatalf("Save evt1: %v", err)
	}

	err = store.Save(ctx, "User", evt2.AggregateID(), []event.Event{evt2}, 0)
	if err != nil {
		t.Fatalf("Save evt2: %v", err)
	}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	replayDone := make(chan struct{})

	var replayed []string

	var replayMu sync.Mutex

	err = runner.Register(event.NewProjection(
		"replay-proj",
		func(_ context.Context, evt event.Event) error {
			replayMu.Lock()

			replayed = append(replayed, string(evt.Type()))

			if len(replayed) == 2 {
				close(replayDone)
			}

			replayMu.Unlock()

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)

		close(done)
	}()

	select {
	case <-replayDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replay to complete")
	}

	cancel()

	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 2 {
		t.Errorf("replayed %d events, want 2", len(replayed))
	}

	savedCP, err := checkpoint.Load(ctx, "replay-proj")
	if err != nil {
		t.Fatalf("checkpoint load: %v", err)
	}

	if savedCP.IsZero() {
		t.Error("checkpoint should be saved after replay")
	}
}

func TestRunner_ReplayWithCheckpoint(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	evt1 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt2 := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	ctx := context.Background()

	err := store.Save(ctx, "User", evt1.AggregateID(), []event.Event{evt1}, 0)
	if err != nil {
		t.Fatalf("Save evt1: %v", err)
	}

	err = store.Save(ctx, "User", evt2.AggregateID(), []event.Event{evt2}, 0)
	if err != nil {
		t.Fatalf("Save evt2: %v", err)
	}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	err = checkpoint.Save(ctx, "replay-proj", evt1.ID())
	if err != nil {
		t.Fatalf("Save checkpoint: %v", err)
	}

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	replayDone := make(chan struct{})

	var replayed []id.EventID

	var replayMu sync.Mutex

	registerReplayProjection(t, runner, "replay-proj", replayDone, &replayed, &replayMu)

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)

		close(done)
	}()

	select {
	case <-replayDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replay to complete")
	}

	cancel()

	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 1 {
		t.Errorf("replayed %d events, want 1 (skipped past checkpoint)", len(replayed))
	}

	if len(replayed) > 0 && replayed[0] != evt2.ID() {
		t.Errorf("replayed event = %v, want %v (evt after checkpoint)", replayed[0], evt2.ID())
	}
}

func TestRunner_ReplayFiltersUnmatchedTypes(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	t.Cleanup(func() { _ = store.Close() })

	userEvt := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	orderEvt := mustNewEvent(t, "OrderPlaced", id.NewAggregateID())

	ctx := context.Background()

	err := store.Save(ctx, "User", userEvt.AggregateID(), []event.Event{userEvt}, 0)
	if err != nil {
		t.Fatalf("Save userEvt: %v", err)
	}

	err = store.Save(ctx, "Order", orderEvt.AggregateID(), []event.Event{orderEvt}, 0)
	if err != nil {
		t.Fatalf("Save orderEvt: %v", err)
	}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	var replayed []string

	var replayMu sync.Mutex

	err = runner.Register(event.NewProjection(
		"user-only-proj",
		func(_ context.Context, evt event.Event) error {
			replayMu.Lock()
			replayed = append(replayed, string(evt.Type()))
			replayMu.Unlock()

			return nil
		},
		[]event.Type{"UserCreated"},
	))
	if err != nil {
		t.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 1 {
		t.Errorf("replayed %d events, want 1 (OrderPlaced should be filtered)", len(replayed))
	}

	if len(replayed) > 0 && replayed[0] != "UserCreated" {
		t.Errorf("replayed event = %q, want UserCreated", replayed[0])
	}
}

func TestRunner_ReplayWithSeekableJournal(t *testing.T) {
	t.Parallel()

	evt1 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt2 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt3 := mustNewEvent(t, "OrderPlaced", id.NewAggregateID())

	store := &seekableJournalStore{events: []event.Event{evt1, evt2, evt3}}

	bus := memory.NewMemoryBus()

	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()

	t.Cleanup(func() { _ = checkpoint.Close() })

	ctx := context.Background()

	err := checkpoint.Save(ctx, "user-proj", evt1.ID())
	if err != nil {
		t.Fatalf("Save checkpoint: %v", err)
	}

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	replayDone := make(chan struct{})

	var replayed []id.EventID

	var replayMu sync.Mutex

	registerReplayProjection(t, runner, "user-proj", replayDone, &replayed, &replayMu)

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)
		close(done)
	}()

	select {
	case <-replayDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for seekable replay")
	}

	cancel()

	<-done

	replayMu.Lock()
	defer replayMu.Unlock()

	if len(replayed) != 1 {
		t.Errorf("replayed %d events, want 1 (evt2 via SeekableJournal)", len(replayed))
	}

	if len(replayed) > 0 && replayed[0] != evt2.ID() {
		t.Errorf("replayed event = %v, want %v", replayed[0], evt2.ID())
	}
}

func TestRunner_ReplayEmptyStore(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)

	loader := &emptyJournal{}

	runner, err := projection.NewRunner(loader, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	registerNoopProjection(t, runner, "test-proj", []event.Type{"UserCreated"})

	runCtx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		_ = runner.Run(runCtx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done
}
