package testhelpers_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestFakeStore_Save_DefaultPath(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt := testhelpers.QuickEvent("Created", aggID, "User", 1, nil)

	err := store.Save(ctx, "User", aggID, []event.Event{evt}, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

func TestFakeStore_Load_DefaultPath_DefensiveCopy(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt := testhelpers.QuickEvent("Created", aggID, "User", 1, nil)

	_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt})

	loaded, err := store.Load(ctx, "User", aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	loaded[0] = nil

	again, _ := store.Load(ctx, "User", aggID)
	if again[0] == nil {
		t.Fatal("Load returned a reference, not a defensive copy")
	}
}

func TestFakeStore_LoadFromVersion_DefaultPath(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt1 := testhelpers.QuickEvent("Created", aggID, "User", 1, nil)
	evt2 := testhelpers.QuickEvent("Updated", aggID, "User", 2, nil)
	evt3 := testhelpers.QuickEvent("Deleted", aggID, "User", 3, nil)

	_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt1, evt2, evt3})

	events, err := store.LoadFromVersion(ctx, "User", aggID, 1)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	testhelpers.AssertEventVersion(t, events, 0, 2)
}

func TestFakeStore_LoadFromVersion_EmptyStream(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()

	events, err := store.LoadFromVersion(ctx, "User", aggID, 0)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}

	if events != nil {
		t.Fatalf("expected nil events for empty stream, got %d", len(events))
	}
}

func TestFakeStore_SaveFn(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	called := false

	store.SaveFn(func(
		_ context.Context,
		_ event.AggregateType,
		_ id.AggregateID,
		_ []event.Event,
		_ event.Version,
	) error {
		called = true

		return nil
	})

	err := store.Save(context.Background(), "User", id.NewAggregateID(), nil, 0)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !called {
		t.Fatal("expected SaveFn to be called")
	}
}

func TestFakeStore_Close_DefaultPath(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()

	err := store.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFakeStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt1 := testhelpers.QuickEvent("Created", aggID, "User", 1, nil)
	evt2 := testhelpers.QuickEvent("Updated", aggID, "User", 2, nil)
	evt3 := testhelpers.QuickEvent("Deleted", aggID, "User", 3, nil)

	_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt1, evt2, evt3})

	events, err := store.LoadToVersion(ctx, "User", aggID, 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestFakeStore_LoadToVersion_EmptyStream(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()

	events, err := store.LoadToVersion(ctx, "User", aggID, 5)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestFakeStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	now := time.Now()

	evt1, _ := event.NewEvent(
		"Created",
		aggID,
		"User",
		1,
		nil,
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2, _ := event.NewEvent(
		"Updated",
		aggID,
		"User",
		2,
		nil,
		event.WithOccurredAt(now.Add(-1*time.Hour)),
	)
	evt3, _ := event.NewEvent("Deleted", aggID, "User", 3, nil, event.WithOccurredAt(now))

	_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt1, evt2, evt3})

	events, err := store.LoadToTimestamp(ctx, "User", aggID, now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestFakeStore_LoadToTimestamp_EmptyStream(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()

	events, err := store.LoadToTimestamp(ctx, "User", aggID, time.Now())
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestFakeStore_LoadFn(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	called := false

	store.LoadFn(func(_ event.AggregateType, _ id.AggregateID) ([]event.Event, error) {
		called = true

		return nil, nil
	})

	_, _ = store.Load(context.Background(), "User", id.NewAggregateID())
	if !called {
		t.Fatal("expected LoadFn to be called")
	}
}

func TestFakeStore_LoadFromVersionFn(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	called := false

	store.LoadFromVersionFn(func(
		_ event.AggregateType, _ id.AggregateID, _ event.Version,
	) ([]event.Event, error) {
		called = true

		return nil, nil
	})

	_, _ = store.LoadFromVersion(context.Background(), "User", id.NewAggregateID(), 1)
	if !called {
		t.Fatal("expected LoadFromVersionFn to be called")
	}
}

func TestFakeStore_CloseFn(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	called := false

	store.CloseFn(func() error {
		called = true

		return nil
	})

	_ = store.Close()
	if !called {
		t.Fatal("expected CloseFn to be called")
	}
}

func TestFakeStore_ReadAll(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	agg1 := id.NewAggregateID()
	agg2 := id.NewAggregateID()
	evt1 := testhelpers.QuickEvent("Created", agg1, "User", 1, nil)
	evt2 := testhelpers.QuickEvent("Created", agg2, "Order", 1, nil)

	_ = store.AppendBatch(ctx, "User", agg1, []event.Event{evt1})
	_ = store.AppendBatch(ctx, "Order", agg2, []event.Event{evt2})

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
}

func TestFakeStore_ReadFrom(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	agg1 := id.NewAggregateID()
	evt1 := testhelpers.QuickEvent("Created", agg1, "User", 1, nil)
	evt2 := testhelpers.QuickEvent("Updated", agg1, "User", 2, nil)

	_ = store.AppendBatch(ctx, "User", agg1, []event.Event{evt1, evt2})

	from, err := store.ReadFrom(ctx, evt1.ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(from) != 1 {
		t.Fatalf("expected 1 event after first, got %d", len(from))
	}

	if from[0].Version() != 2 {
		t.Errorf("event version = %d, want 2", from[0].Version())
	}
}

func TestFakeStore_ReadFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	agg1 := id.NewAggregateID()
	evt1 := testhelpers.QuickEvent("Created", agg1, "User", 1, nil)

	_ = store.AppendBatch(ctx, "User", agg1, []event.Event{evt1})

	from, err := store.ReadFrom(ctx, id.EventID{}, 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(from) != 1 {
		t.Fatalf("expected 1 event from start, got %d", len(from))
	}
}

func TestFakeStore_ReadFrom_WithLimit(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	agg1 := id.NewAggregateID()
	evt1 := testhelpers.QuickEvent("Created", agg1, "User", 1, nil)
	evt2 := testhelpers.QuickEvent("Updated", agg1, "User", 2, nil)
	evt3 := testhelpers.QuickEvent("Deleted", agg1, "User", 3, nil)

	_ = store.AppendBatch(ctx, "User", agg1, []event.Event{evt1, evt2, evt3})

	from, err := store.ReadFrom(ctx, id.EventID{}, 2)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(from) != 2 {
		t.Fatalf("expected 2 events with limit, got %d", len(from))
	}
}
