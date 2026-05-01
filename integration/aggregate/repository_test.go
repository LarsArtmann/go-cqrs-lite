package aggregate_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

type order struct {
	*aggregate.Core

	status string
}

const orderAggregateType event.AggregateType = "Order"

var _ aggregate.Root = (*order)(nil)

func newOrder(orderID id.AggregateID) *order {
	return &order{Core: aggregate.MustNewCore(orderID, orderAggregateType)}
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

func (o *order) ApplySnapshot(state []byte) error {
	var s struct {
		Status string `json:"status"`
	}

	err := json.Unmarshal(state, &s)
	if err != nil {
		return err
	}

	o.status = s.Status

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
		o.ID(),
		orderAggregateType,
		o.Version().Int()+1,
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
		o.ID(),
		orderAggregateType,
		o.Version().Int()+1,
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

	_ = bus.SubscribeAll(testhelpers.AppendEventsHandler(&received))

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

type failingApplyRoot struct {
	*aggregate.Core
}

func (r *failingApplyRoot) Apply(_ event.Event) error {
	return errors.New("apply failed")
}

func (r *failingApplyRoot) ApplySnapshot(_ []byte) error {
	return nil
}

var _ aggregate.Root = (*failingApplyRoot)(nil)

func (r *failingApplyRoot) LoadEvents(events []event.Event) error {
	return r.LoadFromHistory(r, events)
}

func TestCore_LoadFromHistory_ApplyError(t *testing.T) {
	t.Parallel()

	orderID := id.NewAggregateID()
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	o := newOrder(orderID)
	ctx := context.Background()

	err := o.Place(ctx)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	repo := aggregate.NewRepository(store, bus)

	err = repo.Save(ctx, o)
	if err != nil {
		t.Fatalf("save order: %v", err)
	}

	failing := &failingApplyRoot{Core: aggregate.MustNewCore(orderID, orderAggregateType)}

	err = repo.Load(ctx, failing)
	if err == nil {
		t.Error("expected error when Apply fails")
	}
}

type failingStore struct{}

func (f *failingStore) Save(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ []event.Event,
	_ event.Version,
) error {
	return errors.New("store save failed")
}

func (f *failingStore) AppendBatch(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ []event.Event,
) error {
	return errors.New("store append batch failed")
}

func (f *failingStore) Load(
	_ context.Context,
	_ event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	return nil, fmt.Errorf("store load failed for %s", aggregateID)
}

func (f *failingStore) LoadFromVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) ([]event.Event, error) {
	return nil, errors.New("store load from version failed")
}

func (f *failingStore) Delete(_ context.Context, _ event.AggregateType, _ id.AggregateID) error {
	return errors.New("store delete failed")
}

func (f *failingStore) Close() error { return nil }

func TestEventSourcedRepository_Save_StoreError(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(&failingStore{}, bus)

	o := newOrder(id.NewAggregateID())

	err := o.Place(context.Background())
	if err != nil {
		t.Fatalf("place order: %v", err)
	}

	err = repo.Save(context.Background(), o)
	if err == nil {
		t.Error("expected error when store.Save fails")
	}
}

func TestEventSourcedRepository_Load_StoreError(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	repo := aggregate.NewRepository(&failingStore{}, bus)

	o := newOrder(id.NewAggregateID())

	err := repo.Load(context.Background(), o)
	if err == nil {
		t.Error("expected error when store.Load fails")
	}
}
