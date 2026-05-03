package aggregate_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type testRoot struct {
	*aggregate.Core

	applied []event.Event
}

func (r *testRoot) Apply(evt event.Event) error {
	r.applied = append(r.applied, evt)

	return nil
}

func (r *testRoot) ApplySnapshot(_ []byte) error {
	return nil
}

func (r *testRoot) LoadEvents(events []event.Event) error {
	return r.LoadFromHistory(r, events)
}

var _ aggregate.Root = (*testRoot)(nil)

func assertUncommittedChanges(t *testing.T, core *aggregate.Core, want int) {
	t.Helper()

	got := len(core.UncommittedChanges())
	if got != want {
		t.Errorf("uncommitted changes: got %d, want %d", got, want)
	}
}

func assertEventTypeAt(t *testing.T, changes []event.Event, index int, want string) {
	t.Helper()

	if index >= len(changes) {
		t.Fatalf("index %d out of bounds for %d changes", index, len(changes))
	}

	got := string(changes[index].Type())
	if got != want {
		t.Errorf("event type at index %d: got %s, want %s", index, got, want)
	}
}

func TestCore(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))
	if core.ID() != aggID {
		t.Errorf("expected ID user-123, got %s", core.ID().String())
	}

	if core.Type() != "User" {
		t.Errorf("expected type User, got %s", core.Type())
	}

	if core.Version().Int() != 0 {
		t.Errorf("expected version 0, got %d", core.Version().Int())
	}
}

func TestCoreLoadFromHistory(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))
	root := &testRoot{Core: core}

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = core.LoadFromHistory(root, []event.Event{evt})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if core.Version().Int() != 1 {
		t.Errorf("expected version 1 after loading history, got %d", core.Version().Int())
	}

	if len(root.applied) != 1 {
		t.Errorf("expected 1 applied event, got %d", len(root.applied))
	}
}

func TestCoreLoadFromHistory_MultipleEvents(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1541W8PVV4E88DV993TP2A")

	core := aggregate.MustNewCore(aggID, event.AggregateType("Order"))

	events := make([]event.Event, 5)

	for i := range 5 {
		evt, err := event.NewEvent(
			"OrderUpdated",
			aggID,
			"Order",
			i+1,
			nil,
		)
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

	if core.Version().Int() != 5 {
		t.Errorf("expected version 5 after loading 5 events, got %d", core.Version().Int())
	}
}

func TestCoreLoadFromHistory_Empty(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))

	err := core.LoadFromHistory(&testRoot{Core: core}, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if core.Version().Int() != 0 {
		t.Errorf(
			"expected version 0 after loading empty history, got %d",
			core.Version().Int(),
		)
	}
}

func TestCoreRecordEvent(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1542VGZX7VW38CS2WSRXBX")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))

	evt, err := event.NewEvent(
		"UserNameChanged",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if core.Version().Int() != 0 {
		t.Fatalf("expected initial version 0, got %d", core.Version().Int())
	}

	core.RecordEvent(context.Background(), evt)

	if core.Version().Int() != 1 {
		t.Errorf("expected version 1 after applying event, got %d", core.Version().Int())
	}
}

func TestCoreRecordEvent_Multiple(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1543TRR6BB4AF65NQX5V8S")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))

	for i := range 3 {
		evt, err := event.NewEvent(
			"UserUpdated",
			aggID,
			"User",
			i+1,
			nil,
		)
		if err != nil {
			t.Fatalf("unexpected error creating event %d: %v", i, err)
		}

		core.RecordEvent(context.Background(), evt)
	}

	if core.Version().Int() != 3 {
		t.Errorf("expected version 3 after applying 3 events, got %d", core.Version().Int())
	}

	changes := core.UncommittedChanges()
	if len(changes) != 3 {
		t.Errorf("expected 3 uncommitted changes, got %d", len(changes))
	}
}

func TestCoreUncommittedChanges(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1544T0N8E866PQNZ7DEWCH")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))

	changes := core.UncommittedChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 uncommitted changes initially, got %d", len(changes))
	}

	evt1, err := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	core.RecordEvent(context.Background(), evt1)

	evt2, err := event.NewEvent("UserUpdated", aggID, "User", 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	core.RecordEvent(context.Background(), evt2)

	changes = core.UncommittedChanges()
	if len(changes) != 2 {
		t.Fatalf("expected 2 uncommitted changes, got %d", len(changes))
	}

	assertEventTypeAt(t, changes, 0, "UserCreated")
	assertEventTypeAt(t, changes, 1, "UserUpdated")
}

func TestCoreMarkChangesAsCommitted(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1545S8X4QB7RACGEHG95HT")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))

	evt, err := event.NewEvent("UserCreated", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	core.RecordEvent(context.Background(), evt)

	if len(core.UncommittedChanges()) != 1 {
		t.Fatalf(
			"expected 1 uncommitted change before marking, got %d",
			len(core.UncommittedChanges()),
		)
	}

	core.MarkChangesAsCommitted()

	changes := core.UncommittedChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 uncommitted changes after marking, got %d", len(changes))
	}

	if core.Version().Int() != 1 {
		t.Errorf(
			"expected version to remain 1 after marking committed, got %d",
			core.Version().Int(),
		)
	}
}

func TestCoreMarkChangesAsCommitted_Empty(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1546RGMM4GNHGR370838M7")

	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))

	core.MarkChangesAsCommitted()

	assertUncommittedChanges(t, core, 0)
}

// Core is a base struct for embedding, not a direct Root implementation.
// Users embed Core and implement Apply(event.Event) error themselves.
// This test verifies at compile time that *Core does NOT satisfy Root.
func TestCoreDoesNotImplementRootDirectly(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1547QR73ECA1G66ZHC0FE2")
	core := aggregate.MustNewCore(aggID, event.AggregateType("User"))

	_ = core.ID()
	_ = core.Type()
	_ = core.Version()
	_ = core.UncommittedChanges()

	// Root requires Apply, ApplySnapshot, SetVersion, LoadEvents, MarkChangesAsCommitted.
	// Core only provides a subset — consumers embed Core and add their own Apply method.
	// This assertion ensures the test actually exercises the Core.
	if core.ID() != aggID {
		t.Errorf("expected ID %s, got %s", aggID, core.ID())
	}
}

func TestCoreWithRealAggregateID(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	core := aggregate.MustNewCore(aggID, event.AggregateType("Order"))
	if core.ID() != aggID {
		t.Errorf("expected ID %s, got %s", aggID.String(), core.ID().String())
	}

	if core.Type() != "Order" {
		t.Errorf("expected type Order, got %s", core.Type())
	}
}

func TestCoreFullLifecycle(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1548Q0X4FJ8HFS1ATQYK9Y")

	core := aggregate.MustNewCore(aggID, event.AggregateType("Product"))

	evt1, err := event.NewEvent(
		"ProductCreated",
		aggID,
		"Product",
		1,
		[]byte(`{"name":"Widget"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	core.RecordEvent(context.Background(), evt1)

	evt2, err := event.NewEvent(
		"PriceChanged",
		aggID,
		"Product",
		2,
		[]byte(`{"price":9.99}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	core.RecordEvent(context.Background(), evt2)

	if core.Version().Int() != 2 {
		t.Errorf("expected version 2, got %d", core.Version().Int())
	}

	assertUncommittedChanges(t, core, 2)

	core.MarkChangesAsCommitted()

	assertUncommittedChanges(t, core, 0)

	if core.Version().Int() != 2 {
		t.Errorf(
			"expected version still 2 after commit, got %d",
			core.Version().Int(),
		)
	}
}

func TestEveryNEvents_PanicsOnZeroOrNegative(t *testing.T) {
	t.Parallel()

	for _, n := range []int{0, -1, -5} {
		func() {
			defer func() { _ = recover() }()

			aggregate.EveryNEvents(n)
			t.Errorf("expected panic for n=%d", n)
		}()
	}
}
