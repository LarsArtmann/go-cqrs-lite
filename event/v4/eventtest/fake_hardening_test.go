package eventtest

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestFakeStore_LoadToVersion_ReturnsCopy(t *testing.T) {
	t.Parallel()

	store := NewFakeStore()
	ctx := context.Background()
	ref := id.NewStreamRef("Test", id.NewStreamID())
	appendTestEvents(t, store, ctx, ref, 3)

	loaded, err := store.LoadToVersion(ctx, ref, 3)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	slices.Reverse(loaded)

	after, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load after in-place sort: %v", err)
	}

	if after[0].Version() != 1 {
		t.Fatalf(
			"in-place mutation of loaded events corrupted the fake: first version = %d",
			after[0].Version(),
		)
	}
}

func TestFakeStore_ReadFrom_ReturnsCopy(t *testing.T) {
	t.Parallel()

	store := NewFakeStore()
	ctx := context.Background()
	ref := id.NewStreamRef("Test", id.NewStreamID())
	appendTestEvents(t, store, ctx, ref, 3)

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	from, err := store.ReadFrom(ctx, all[0].ID(), 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	slices.Reverse(from)

	after, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll after in-place sort: %v", err)
	}

	if after[0].Version() != 1 {
		t.Fatalf(
			"in-place mutation of ReadFrom result corrupted the fake: first version = %d",
			after[0].Version(),
		)
	}
}

func TestFakeStore_ReadAll_OrdersByOccurredAt(t *testing.T) {
	t.Parallel()

	store := NewFakeStore()
	ctx := context.Background()
	base := time.Now()

	refA := id.NewStreamRef("Test", id.NewStreamID())
	refB := id.NewStreamRef("Test", id.NewStreamID())

	// Interleave two streams with known, strictly increasing timestamps —
	// map iteration order must not leak into ReadAll.
	for i := range 3 {
		ts := base.Add(time.Duration(i) * time.Second)
		for _, ref := range []id.StreamRef{refA, refB} {
			evt, err := event.NewEvent("OrderTest", ref.ID, ref.Type, event.Version(i+1), nil,
				event.WithOccurredAt(ts))
			if err != nil {
				t.Fatalf("NewEvent: %v", err)
			}

			if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
				t.Fatalf("AppendBatch: %v", err)
			}
		}
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 6 {
		t.Fatalf("expected 6 events, got %d", len(all))
	}

	for i := 1; i < len(all); i++ {
		if all[i].OccurredAt().Before(all[i-1].OccurredAt()) {
			t.Fatalf("ReadAll violated OccurredAt order at index %d: %v before %v",
				i, all[i].OccurredAt(), all[i-1].OccurredAt())
		}
	}
}

func TestFakeBus_PublishDuringUsePublish_NoRace(t *testing.T) {
	t.Parallel()

	bus := NewFakeBus()
	ctx := context.Background()

	evt, err := NewTestEvent()
	if err != nil {
		t.Fatalf("NewTestEvent: %v", err)
	}

	var wg sync.WaitGroup

	for range 4 {
		wg.Add(2)

		go func() {
			defer wg.Done()
			for range 50 {
				_ = bus.UsePublish(func(next event.Publisher) event.Publisher {
					return event.PublisherFunc(func(c context.Context, es ...event.Event) error {
						return next.Publish(c, es...)
					})
				})
			}
		}()

		go func() {
			defer wg.Done()
			for range 50 {
				_ = bus.Publish(ctx, evt)
			}
		}()
	}

	wg.Wait()
}
