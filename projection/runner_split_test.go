package projection_test

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
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

// TestRunner_RunReplayThenRunLive_ReadYourWrites is the core scenario from the
// cqrs-htmx feedback: RunReplay returns synchronously with the read model caught
// up (no time.Sleep), then RunLive tails live events in the background.
func TestRunner_RunReplayThenRunLive_ReadYourWrites(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	evt1 := mustNewEvent(t, "UserCreated", id.NewAggregateID())
	evt2 := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	ctx := context.Background()

	if err := store.Save(
		ctx, event.NewAggregateRef("User", evt1.AggregateID()), []event.Event{evt1}, 0,
	); err != nil {
		t.Fatalf("save evt1: %v", err)
	}

	if err := store.Save(
		ctx, event.NewAggregateRef("User", evt2.AggregateID()), []event.Event{evt2}, 0,
	); err != nil {
		t.Fatalf("save evt2: %v", err)
	}

	bus := eventtest.NewFakeBus()
	t.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()
	t.Cleanup(func() { _ = checkpoint.Close() })

	// Wrap the bus so we can observe when RunLive has actually subscribed,
	// avoiding a publish-before-subscribe race (MemoryBus drops un-subscribed events).
	ready := make(chan struct{})
	signalBus := &subscribeSignalBus{Subscriber: bus, ready: ready}

	runner, err := projection.NewRunner(store, signalBus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	var handled atomic.Int32

	if err := runner.Register(event.NewProjection(
		"ryw-proj",
		func(_ context.Context, _ event.Event) error {
			handled.Add(1)
			return nil
		},
		[]event.Type{"UserCreated"},
	)); err != nil {
		t.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Synchronous replay: the read model is caught up the moment this returns.
	if err := runner.RunReplay(runCtx); err != nil {
		t.Fatalf("RunReplay: %v", err)
	}

	if handled.Load() != 2 {
		t.Fatalf("after RunReplay: handled %d events, want 2 (read-your-writes, no sleep)",
			handled.Load())
	}

	// Live phase blocks; run it in the background.
	liveDone := make(chan error, 1)
	go func() { liveDone <- runner.RunLive(runCtx) }()

	<-ready // RunLive has subscribed; safe to publish now.

	liveEvt := mustNewEvent(t, "UserCreated", id.NewAggregateID())

	if err := bus.Publish(runCtx, liveEvt); err != nil {
		t.Fatalf("publish live event: %v", err)
	}

	// The brand-new live event is handled by the running live phase.
	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) && handled.Load() != 3 {
		time.Sleep(time.Millisecond)
	}

	if handled.Load() != 3 {
		t.Fatalf("live event not handled: count %d, want 3", handled.Load())
	}

	cancel()

	if err := <-liveDone; err != nil {
		t.Fatalf("RunLive returned error: %v", err)
	}
}

func TestRunner_RunLiveWithoutRunReplay(t *testing.T) {
	t.Parallel()

	runner, _, _ := newTestRunnerWithReady(t)
	registerNoopProjection(t, runner, "live-first", []event.Type{"UserCreated"})

	err := runner.RunLive(context.Background())
	if !errors.Is(err, projection.ErrReplayRequired) {
		t.Fatalf("RunLive before RunReplay: got %v, want ErrReplayRequired", err)
	}
}

func TestRunner_DoubleRunReplay(t *testing.T) {
	t.Parallel()

	runner, _, _ := newTestRunnerWithReady(t)
	defer func() { _ = runner.Close() }()
	registerNoopProjection(t, runner, "double-replay", []event.Type{"UserCreated"})

	if err := runner.RunReplay(context.Background()); err != nil {
		t.Fatalf("first RunReplay: %v", err)
	}

	err := runner.RunReplay(context.Background())
	if !errors.Is(err, projection.ErrAlreadyRunning) {
		t.Fatalf("second RunReplay: got %v, want ErrAlreadyRunning", err)
	}
}

func TestRunner_DoubleRunLive(t *testing.T) {
	t.Parallel()

	runner, _, _ := newTestRunnerWithReady(t)
	defer func() { _ = runner.Close() }()
	registerNoopProjection(t, runner, "double-live", []event.Type{"UserCreated"})

	if err := runner.RunReplay(context.Background()); err != nil {
		t.Fatalf("RunReplay: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = runner.RunLive(ctx) }()

	// Give the first RunLive time to enter the live state.
	time.Sleep(50 * time.Millisecond)

	err := runner.RunLive(ctx)
	if !errors.Is(err, projection.ErrAlreadyRunning) {
		t.Fatalf("second RunLive: got %v, want ErrAlreadyRunning", err)
	}
}

// TestRunner_RunReplayWithoutRunLive_CloseDoesNotHang verifies that Close
// returns promptly if RunLive was never started, and resets the runner so it
// can be reused.
func TestRunner_RunReplayWithoutRunLive_CloseDoesNotHang(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)
	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	registerNoopProjection(t, runner, "no-live", []event.Type{"UserCreated"})

	if err := runner.RunReplay(context.Background()); err != nil {
		t.Fatalf("RunReplay: %v", err)
	}

	done := make(chan struct{})

	go func() {
		_ = runner.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close hung after RunReplay without RunLive")
	}

	// After Close reset the ready state, the runner is reusable.
	if err := runner.RunReplay(context.Background()); err != nil {
		t.Fatalf("reuse RunReplay after Close: %v", err)
	}

	_ = runner.Close()
}

// TestRunner_ConcurrentRunLive_NoCorruption hammers RunLive from many
// goroutines: exactly one must win, the rest must get ErrAlreadyRunning, and
// Close must not hang (which would indicate a clobbered done/cancel pair).
func TestRunner_ConcurrentRunLive_NoCorruption(t *testing.T) {
	t.Parallel()

	runner, _, _ := newTestRunnerWithReady(t)
	registerNoopProjection(t, runner, "race-proj", []event.Type{"UserCreated"})

	if err := runner.RunReplay(context.Background()); err != nil {
		t.Fatalf("RunReplay: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		wg       sync.WaitGroup
		success  atomic.Int32
		rejected atomic.Int32
	)

	for range 16 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			err := runner.RunLive(ctx)

			switch {
			case err == nil:
				success.Add(1)
			case errors.Is(err, projection.ErrAlreadyRunning):
				rejected.Add(1)
			default:
				t.Errorf("unexpected RunLive error: %v", err)
			}
		}()
	}

	// Wait for the CAS storm to settle: 15 losers should return quickly.
	for range 100 {
		if rejected.Load() == 15 {
			break
		}

		time.Sleep(5 * time.Millisecond)
	}

	if rejected.Load() != 15 {
		t.Fatalf("expected 15 rejected RunLive, got %d", rejected.Load())
	}

	// Exactly one winner should be blocking on subscribeLive right now.
	if !runner.IsRunning() {
		t.Fatal("expected IsRunning() true after concurrent RunLive")
	}

	// Cancel the context so the winner exits; then verify exactly 1 success.
	cancel()
	wg.Wait()

	if success.Load() != 1 {
		t.Fatalf("expected exactly 1 winning RunLive, got %d", success.Load())
	}

	// The winner is still blocking on subscribeLive. Close must read the
	// winner's done/cancel pair (not a clobbered loser's) and stop it.
	closeDone := make(chan struct{})

	go func() {
		_ = runner.Close()
		close(closeDone)
	}()

	select {
	case <-closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Close hung — done/cancel were corrupted by concurrent RunLive")
	}
}

// returns the runner to idle so it is not stuck in the ready state.
func TestRunner_RunReplayError_ResetsToIdle(t *testing.T) {
	t.Parallel()

	bus, checkpoint := newTestBusAndCheckpoint(t)
	loader := &failingJournal{err: errors.New("journal offline")}

	runner, err := projection.NewRunner(loader, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	registerNoopProjection(t, runner, "fail-replay", []event.Type{"UserCreated"})

	if err := runner.RunReplay(context.Background()); err == nil {
		t.Fatal("expected error from failing journal")
	}

	// State was reset to idle: RunLive must report "replay required", not
	// "already running".
	err = runner.RunLive(context.Background())
	if !errors.Is(err, projection.ErrReplayRequired) {
		t.Fatalf("RunLive after failed RunReplay: got %v, want ErrReplayRequired", err)
	}
}
