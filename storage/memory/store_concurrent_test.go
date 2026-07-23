package memory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryStore_ConcurrentSaveAndLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	const goroutines = 10
	const eventsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := range goroutines {
		aggID := id.NewAggregateID()

		go func() {
			defer wg.Done()

			for idx := range eventsPerGoroutine {
				evt := eventtest.QuickEvent(
					"UserCreated",
					aggID,
					"User",
					event.Version(idx+1),
					nil,
				)
				_ = store.Save(
					ctx,
					id.NewAggregateRef(id.StreamType("User"), aggID),
					[]event.Event{evt},
					event.Version(idx),
				)
			}
		}()

		go func() {
			defer wg.Done()

			for range eventsPerGoroutine {
				_, _ = store.Load(ctx, id.NewAggregateRef(id.StreamType("User"), aggID))
			}
		}()

		_ = i
	}

	wg.Wait()
}

func TestMemoryStore_ReadFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt := eventtest.QuickEvent("Created", aggID, "User", 1, nil)

	_ = store.AppendBatch(
		ctx,
		id.NewAggregateRef(id.StreamType("User"), aggID),
		[]event.Event{evt},
	)

	events, err := store.ReadFrom(ctx, id.EventID{}, 10)
	eventtest.AssertNoError(t, err, "ReadFrom with zero ID")
	eventtest.AssertLen(t, "events", events, 1)
}

func TestMemoryStore_ReadFrom_WithLimit(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	seedTestEvents(t, store, ctx, aggID, 5)

	events, err := store.ReadFrom(ctx, id.EventID{}, 3)
	eventtest.AssertNoError(t, err, "ReadFrom with limit")
	eventtest.AssertLen(t, "events", events, 3)
}
