package decider_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// countLoadStore wraps event.Store to count Load calls with a delay
// that ensures concurrent callers overlap, testing singleflight coalescing.
type countLoadStore struct {
	event.Store

	count atomic.Int32
}

func (c *countLoadStore) Load(ctx context.Context, ref id.AggregateRef) ([]event.Event, error) {
	c.count.Add(1)
	time.Sleep(50 * time.Millisecond)

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

	aggID := id.NewAggregateID()

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	const numGoroutines = 5

	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			state, version, loadErr := repo.Load(context.Background(), aggID, "Counter")
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
		}()
	}
	wg.Wait()

	if got := store.count.Load(); got != 1 {
		t.Errorf("store.Load called %d times, want 1 (coalesced by singleflight)", got)
	}
}

func TestLoad_DifferentAggregatesNotCoalesced(t *testing.T) {
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

	agg1 := id.NewAggregateID()
	agg2 := id.NewAggregateID()

	mustAppendBatch(t, store, "Counter", agg1, []event.Event{
		makeEvent(t, "CounterCreated", agg1, 1),
	})
	mustAppendBatch(t, store, "Counter", agg2, []event.Event{
		makeEvent(t, "CounterCreated", agg2, 1),
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _, _ = repo.Load(context.Background(), agg1, "Counter")
	}()
	go func() {
		defer wg.Done()
		_, _, _ = repo.Load(context.Background(), agg2, "Counter")
	}()
	wg.Wait()

	if got := store.count.Load(); got != 2 {
		t.Errorf("store.Load called %d times, want 2 (different aggregates not coalesced)", got)
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

	aggID := id.NewAggregateID()

	mustAppendBatch(t, store, "Counter", aggID, []event.Event{
		makeEvent(t, "CounterCreated", aggID, 1),
	})

	const numGoroutines = 5

	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, _, loadErr := repo.Load(context.Background(), aggID, "Counter")
			if loadErr != nil {
				t.Errorf("Load error: %v", loadErr)

				return
			}
		}()
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
