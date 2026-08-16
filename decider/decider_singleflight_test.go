package decider_test

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// countLoadStore wraps event.Store to count Load calls with a delay
// that ensures concurrent callers overlap, testing singleflight coalescing.
type countLoadStore struct {
	event.Store

	count atomic.Int32
}

// delay holds the Load call open long enough for all concurrent goroutines
// released by the start barrier to arrive at singleflight.Do. 200ms is
// sufficient even under -race (which inflates scheduling latency 5-10x).
func (c *countLoadStore) delay() {
	time.Sleep(200 * time.Millisecond)
}

func (c *countLoadStore) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	c.count.Add(1)
	c.delay()

	return c.Store.Load(ctx, ref)
}

func TestLoad_ConcurrentLoadsCoalescedBySingleflight(t *testing.T) {
	t.Parallel()

	inner := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	store := &countLoadStore{Store: inner}

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
	})

	const numGoroutines = 5

	// Start barrier: all goroutines wait on this channel so they reach
	// singleflight's Do at nearly the same time. Without this, the goroutine
	// scheduler may launch them sequentially, causing some to miss the
	// in-flight singleflight call (the 200ms delay in countLoadStore is the
	// coalescing window; Gosched after close yields so goroutines enter Do ASAP).
	start := make(chan struct{})

	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Go(func() {
			<-start

			state, version, loadErr := repo.Load(context.Background(), streamID, "Counter")
			if loadErr != nil {
				t.Errorf("Load error: %v", loadErr)

				return
			}
			if version != 1 {
				t.Errorf("version = %d, want 1", version)
			}
			if state.Value != 1 {
				t.Errorf("state.Value = %d, want 1", state.Value)
			}
		})
	}

	close(start)
	runtime.Gosched() // yield so released goroutines can enter singleflight.Do promptly
	wg.Wait()

	if got := store.count.Load(); got != 1 {
		t.Errorf("store.Load called %d times, want 1 (coalesced by singleflight)", got)
	}
}

func TestLoad_DifferentStreamsNotCoalesced(t *testing.T) {
	t.Parallel()

	inner := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	store := &countLoadStore{Store: inner}

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	stream1 := id.NewStreamID()
	stream2 := id.NewStreamID()

	mustAppendBatch(t, store, "Counter", stream1, []event.Event{
		makeEvent(t, "CounterCreated", stream1, 1),
	})
	mustAppendBatch(t, store, "Counter", stream2, []event.Event{
		makeEvent(t, "CounterCreated", stream2, 1),
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _, _ = repo.Load(context.Background(), stream1, "Counter")
	}()
	go func() {
		defer wg.Done()
		_, _, _ = repo.Load(context.Background(), stream2, "Counter")
	}()
	wg.Wait()

	if got := store.count.Load(); got != 2 {
		t.Errorf("store.Load called %d times, want 2 (different streams not coalesced)", got)
	}
}

func TestLoad_WithLoadCoalescingDisabled(t *testing.T) {
	t.Parallel()

	inner := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	store := &countLoadStore{Store: inner}

	d := decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
	}

	repo, err := decider.NewRepository(
		store,
		bus,
		d,
		decider.WithLoadCoalescing[counterState](false),
	)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
	})

	const numGoroutines = 5

	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Go(func() {
			_, _, loadErr := repo.Load(context.Background(), streamID, "Counter")
			if loadErr != nil {
				t.Errorf("Load error: %v", loadErr)

				return
			}
		})
	}
	wg.Wait()

	if got := store.count.Load(); got != numGoroutines {
		t.Errorf(
			"store.Load called %d times, want %d (coalescing disabled)",
			got,
			numGoroutines,
		)
	}
}

// gateLoadStore holds Load open until released, letting a test sequence a
// leader, a coalesced follower, and a leader-context cancellation
// deterministically.
type gateLoadStore struct {
	event.Store

	entered chan struct{}
	release chan struct{}
	count   atomic.Int32
}

func (g *gateLoadStore) Load(ctx context.Context, ref id.StreamRef) ([]event.Event, error) {
	g.count.Add(1)
	close(g.entered)
	<-g.release

	return g.Store.Load(ctx, ref)
}

// TestLoad_LeaderCancelDoesNotAbortCoalescedLoad pins the WithoutCancel fix:
// the leader's cancelled context must not abort the shared in-flight load
// for coalesced followers.
func TestLoad_LeaderCancelDoesNotAbortCoalescedLoad(t *testing.T) {
	t.Parallel()

	inner := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	store := &gateLoadStore{
		Store:   inner,
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}

	d := decider.Decider[counterState]{Initial: counterState{}, Apply: applyCounter}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}

	streamID := id.NewStreamID()

	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
	})

	leaderCtx, cancelLeader := context.WithCancel(context.Background())

	type loadResult struct {
		version event.Version
		err     error
	}

	leaderRes := make(chan loadResult, 1)
	go func() {
		_, version, loadErr := repo.Load(leaderCtx, streamID, "Counter")
		leaderRes <- loadResult{version: version, err: loadErr}
	}()

	<-store.entered // leader is inside store.Load, inside singleflight.Do

	followerRes := make(chan loadResult, 1)
	go func() {
		_, version, loadErr := repo.Load(context.Background(), streamID, "Counter")
		followerRes <- loadResult{version: version, err: loadErr}
	}()

	// Give the follower a moment to join the in-flight singleflight group.
	time.Sleep(50 * time.Millisecond)

	cancelLeader() // leader gives up — the shared load must keep running
	close(store.release)

	select {
	case res := <-followerRes:
		if res.err != nil {
			t.Fatalf("follower Load aborted by leader cancellation: %v", res.err)
		}
		if res.version != 1 {
			t.Errorf("follower version = %d, want 1", res.version)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("follower Load did not finish")
	}

	select {
	case <-leaderRes:
		// The leader may observe success (WithoutCancel) or its own
		// cancellation; only the follower's success is the contract.
	case <-time.After(5 * time.Second):
		t.Fatal("leader Load did not finish")
	}

	if got := store.count.Load(); got != 1 {
		t.Errorf("store.Load called %d times, want 1 (coalesced)", got)
	}
}
