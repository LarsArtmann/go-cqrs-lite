package aggregate_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func TestEventSourcedRepository_Save_WithOutbox(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	outbox := memory.NewMemoryOutboxStore()

	repo := aggregate.NewRepository(store, bus, aggregate.WithOutbox(outbox))

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	err := o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order with outbox: %v", err)
	}

	// Events should be in the outbox, not directly on the bus
	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll outbox: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(entries))
	}

	if len(entries[0].Events) != 1 {
		t.Errorf("expected 1 event in outbox entry, got %d", len(entries[0].Events))
	}
}

func TestEventSourcedRepository_Save_WithOutbox_NoChanges(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	outbox := memory.NewMemoryOutboxStore()

	repo := aggregate.NewRepository(store, bus, aggregate.WithOutbox(outbox))

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	// Save without any changes
	err := repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order with no changes: %v", err)
	}

	entries, err := outbox.PollPending(ctx, 10)
	if err != nil {
		t.Fatalf("poll outbox: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 outbox entries for no changes, got %d", len(entries))
	}
}

func TestEventSourcedRepository_Save_WithoutOutbox_PublishesToBus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	var busEvents []event.Event

	err := bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		busEvents = append(busEvents, evt)

		return nil
	})
	if err != nil {
		t.Fatalf("subscribe to bus: %v", err)
	}

	repo := aggregate.NewRepository(store, bus)

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	err = o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order without outbox: %v", err)
	}

	if len(busEvents) != 1 {
		t.Errorf("expected 1 event on bus, got %d", len(busEvents))
	}
}
