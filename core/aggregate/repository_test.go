package aggregate_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type order struct {
	*aggregate.Core

	status string
}

const orderAggregateType event.AggregateType = "Order"

var (
	_ aggregate.Root          = (*order)(nil)
	_ aggregate.HistoryLoader = (*order)(nil)
)

func newOrder(orderID id.AggregateID) *order {
	return &order{Core: aggregate.NewCore(orderID, orderAggregateType)}
}

func (o *order) Status() string { return o.status }

func (o *order) Apply(evt event.Event) error {
	switch evt.Type() {
	case "OrderPlaced":
		var p struct {
			Status string `json:"status"`
		}

		err := json.Unmarshal(evt.Payload(), &p)
		if err != nil {
			return err
		}

		o.status = p.Status
	case "OrderShipped":
		o.status = "shipped"
	}

	return nil
}

func (o *order) LoadEvents(events []event.Event) error {
	return o.LoadFromHistory(o, events)
}

func (o *order) Place(ctx context.Context) error {
	payload, err := json.Marshal(struct {
		Status string `json:"status"`
	}{Status: "placed"})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	evt, err := event.NewEvent(
		"OrderPlaced",
		id.MustParseAggregateID(o.ID()),
		orderAggregateType,
		o.Version()+1,
		payload,
	)
	if err != nil {
		return err
	}

	o.status = "placed"
	o.RecordEvent(ctx, evt)

	return nil
}

func (o *order) Ship(ctx context.Context) error {
	evt, err := event.NewEvent(
		"OrderShipped",
		id.MustParseAggregateID(o.ID()),
		orderAggregateType,
		o.Version()+1,
		nil,
	)
	if err != nil {
		return err
	}

	o.status = "shipped"
	o.RecordEvent(ctx, evt)

	return nil
}

func TestEventSourcedRepository_Save(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(store, bus)

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	err := o.Place(context.Background())
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(context.Background(), o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	if len(o.UncommittedChanges()) != 0 {
		t.Error("expected no uncommitted changes after save")
	}

	if o.Version() != 1 {
		t.Errorf("expected version 1 after save, got %d", o.Version())
	}
}

func TestEventSourcedRepository_Save_NoChanges(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(store, bus)

	o := newOrder(id.NewAggregateID())

	err := repo.Save(context.Background(), o)
	if err != nil {
		t.Fatalf("save with no changes should not error: %v", err)
	}
}

func TestEventSourcedRepository_Load(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(store, bus)

	orderID := id.NewAggregateID()
	ctx := context.Background()

	o := newOrder(orderID)

	err := o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = o.Ship(ctx)
	if err != nil {
		t.Fatalf("ship order: %v", err)
	}

	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	loaded := newOrder(orderID)

	err = repo.Load(ctx, loaded)
	if err != nil {
		t.Fatalf("load order: %v", err)
	}

	if loaded.Version() != 2 {
		t.Errorf("expected version 2, got %d", loaded.Version())
	}

	if loaded.Status() != "shipped" {
		t.Errorf("expected status shipped, got %s", loaded.Status())
	}
}

func TestEventSourcedRepository_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(store, bus)

	o := newOrder(id.NewAggregateID())

	err := repo.Load(context.Background(), o)
	if err == nil {
		t.Error("expected error when loading non-existent aggregate")
	}
}

func TestEventSourcedRepository_Roundtrip(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(store, bus)

	orderID := id.NewAggregateID()
	ctx := context.Background()

	o := newOrder(orderID)

	err := o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	loaded := newOrder(orderID)

	err = repo.Load(ctx, loaded)
	if err != nil {
		t.Fatalf("load order: %v", err)
	}

	err = loaded.Ship(ctx)
	if err != nil {
		t.Fatalf("ship order: %v", err)
	}

	err = repo.Save(ctx, loaded)
	if err != nil {
		t.Fatalf("save shipped order: %v", err)
	}

	final := newOrder(orderID)

	err = repo.Load(ctx, final)
	if err != nil {
		t.Fatalf("load final order: %v", err)
	}

	if final.Version() != 2 {
		t.Errorf("expected version 2, got %d", final.Version())
	}

	if final.Status() != "shipped" {
		t.Errorf("expected status shipped, got %s", final.Status())
	}
}

func TestEventSourcedRepository_EventsPublished(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(store, bus)

	var received []event.Event

	_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		received = append(received, evt)

		return nil
	})

	o := newOrder(id.NewAggregateID())

	err := o.Place(context.Background())
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(context.Background(), o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	if len(received) != 1 {
		t.Errorf("expected 1 event published, got %d", len(received))
	}

	if received[0].Type() != "OrderPlaced" {
		t.Errorf("expected event type OrderPlaced, got %s", received[0].Type())
	}
}
