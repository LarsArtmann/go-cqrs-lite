package aggregate_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/aggregate"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type testRoot struct {
	*aggregate.Core
	applied []event.Event
}

func (r *testRoot) Apply(evt event.Event) error {
	r.applied = append(r.applied, evt)
	return nil
}

var _ aggregate.Root = (*testRoot)(nil)

func TestCore(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-123", event.AggregateType("User"))
	if core.ID() != "user-123" {
		t.Errorf("expected ID user-123, got %s", core.ID())
	}
	if core.Type() != "User" {
		t.Errorf("expected type User, got %s", core.Type())
	}
	if core.Version() != 0 {
		t.Errorf("expected version 0, got %d", core.Version())
	}
}

func TestCoreLoadFromHistory(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-123", event.AggregateType("User"))
	root := &testRoot{Core: core}

	evt, err := event.NewEvent("UserCreated", "user-123", "User", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = core.LoadFromHistory(root, []event.Event{evt})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if core.Version() != 1 {
		t.Errorf("expected version 1 after loading history, got %d", core.Version())
	}
	if len(root.applied) != 1 {
		t.Errorf("expected 1 applied event, got %d", len(root.applied))
	}
}

func TestCoreLoadFromHistory_MultipleEvents(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("order-1", event.AggregateType("Order"))

	events := make([]event.Event, 5)
	for i := range 5 {
		evt, err := event.NewEvent("OrderUpdated", "order-1", "Order", i+1, nil)
		if err != nil {
			t.Fatalf("unexpected error creating event %d: %v", i, err)
		}
		events[i] = evt
	}

	root := &testRoot{Core: core}

	err := core.LoadFromHistory(root, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if core.Version() != 5 {
		t.Errorf("expected version 5 after loading 5 events, got %d", core.Version())
	}
}

func TestCoreLoadFromHistory_Empty(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-123", event.AggregateType("User"))

	err := core.LoadFromHistory(&testRoot{Core: core}, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if core.Version() != 0 {
		t.Errorf("expected version 0 after loading empty history, got %d", core.Version())
	}
}

func TestCoreRecordEvent(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-456", event.AggregateType("User"))

	evt, err := event.NewEvent("UserNameChanged", "user-456", "User", 1, []byte(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if core.Version() != 0 {
		t.Fatalf("expected initial version 0, got %d", core.Version())
	}

	core.RecordEvent(context.Background(), evt)

	if core.Version() != 1 {
		t.Errorf("expected version 1 after applying event, got %d", core.Version())
	}
}

func TestCoreRecordEvent_Multiple(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-789", event.AggregateType("User"))

	for i := range 3 {
		evt, err := event.NewEvent("UserUpdated", "user-789", "User", i+1, nil)
		if err != nil {
			t.Fatalf("unexpected error creating event %d: %v", i, err)
		}
		core.RecordEvent(context.Background(), evt)
	}

	if core.Version() != 3 {
		t.Errorf("expected version 3 after applying 3 events, got %d", core.Version())
	}

	changes := core.UncommittedChanges()
	if len(changes) != 3 {
		t.Errorf("expected 3 uncommitted changes, got %d", len(changes))
	}
}

func TestCoreUncommittedChanges(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-abc", event.AggregateType("User"))

	changes := core.UncommittedChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 uncommitted changes initially, got %d", len(changes))
	}

	evt1, err := event.NewEvent("UserCreated", "user-abc", "User", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	core.RecordEvent(context.Background(), evt1)

	evt2, err := event.NewEvent("UserUpdated", "user-abc", "User", 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	core.RecordEvent(context.Background(), evt2)

	changes = core.UncommittedChanges()
	if len(changes) != 2 {
		t.Fatalf("expected 2 uncommitted changes, got %d", len(changes))
	}

	if changes[0].Type() != "UserCreated" {
		t.Errorf("expected first event type UserCreated, got %s", changes[0].Type())
	}
	if changes[1].Type() != "UserUpdated" {
		t.Errorf("expected second event type UserUpdated, got %s", changes[1].Type())
	}
}

func TestCoreMarkChangesAsCommitted(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-def", event.AggregateType("User"))

	evt, err := event.NewEvent("UserCreated", "user-def", "User", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	core.RecordEvent(context.Background(), evt)

	if len(core.UncommittedChanges()) != 1 {
		t.Fatalf("expected 1 uncommitted change before marking, got %d", len(core.UncommittedChanges()))
	}

	core.MarkChangesAsCommitted()

	changes := core.UncommittedChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 uncommitted changes after marking, got %d", len(changes))
	}

	if core.Version() != 1 {
		t.Errorf("expected version to remain 1 after marking committed, got %d", core.Version())
	}
}

func TestCoreMarkChangesAsCommitted_Empty(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("user-ghi", event.AggregateType("User"))

	core.MarkChangesAsCommitted()

	if len(core.UncommittedChanges()) != 0 {
		t.Errorf("expected 0 uncommitted changes on empty aggregate, got %d", len(core.UncommittedChanges()))
	}
}

// Core is a base struct for embedding, not a direct Root implementation.
// Users embed Core and implement Apply(event.Event) error themselves.
func TestCoreDoesNotImplementRootDirectly(t *testing.T) {
	t.Parallel()

	// Core provides ID, Type, Version, UncommittedChanges, MarkChangesAsCommitted,
	// but Root requires Apply(event.Event) error — Core has RecordEvent(ctx, evt) instead.
	// This is intentional: domain aggregates embed Core and add their own Apply method.
	core := aggregate.NewCore("user-embed", event.AggregateType("User"))
	_ = core
}

func TestCoreWithRealAggregateID(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	core := aggregate.NewCore(aggID, event.AggregateType("Order"))

	if core.ID() != aggID.String() {
		t.Errorf("expected ID %s, got %s", aggID.String(), core.ID())
	}
	if core.Type() != "Order" {
		t.Errorf("expected type Order, got %s", core.Type())
	}
}

func TestCoreFullLifecycle(t *testing.T) {
	t.Parallel()

	core := aggregate.NewCore("product-1", event.AggregateType("Product"))

	evt1, err := event.NewEvent("ProductCreated", "product-1", "Product", 1, []byte(`{"name":"Widget"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	core.RecordEvent(context.Background(), evt1)

	evt2, err := event.NewEvent("PriceChanged", "product-1", "Product", 2, []byte(`{"price":9.99}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	core.RecordEvent(context.Background(), evt2)

	if core.Version() != 2 {
		t.Errorf("expected version 2, got %d", core.Version())
	}
	if len(core.UncommittedChanges()) != 2 {
		t.Errorf("expected 2 changes, got %d", len(core.UncommittedChanges()))
	}

	core.MarkChangesAsCommitted()

	if len(core.UncommittedChanges()) != 0 {
		t.Errorf("expected 0 changes after commit, got %d", len(core.UncommittedChanges()))
	}
	if core.Version() != 2 {
		t.Errorf("expected version still 2 after commit, got %d", core.Version())
	}
}
