package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func TestMemoryStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK154ME034FVHK95R554AKSE")
	evt1 := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	_ = store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1, evt2, evt3},
		0,
	)

	events, err := store.LoadFromVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		1,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events from version", events, 2)
}

func TestMemoryStore_LoadFromVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK154KER4E8AJ20Q4JD5TJ1E")

	_, err := store.LoadFromVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		0,
	)
	if err == nil {
		t.Error("expected aggregate not found error")
	}
}

func TestMemoryStore_LoadFromVersion_AtEnd(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK154PCGXJ80RFXRASTMSSK0")
	evt := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	_ = store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt},
		0,
	)

	events, err := store.LoadFromVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		1,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events past end", events, 0)
}

func TestMemoryStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt1 := eventtest.QuickEvent("Created", aggID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("Updated", aggID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("Deleted", aggID, "User", 2, nil)

	_ = store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1, evt2, evt3},
	)

	events, err := store.LoadToVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		2,
	)
	eventtest.AssertNoError(t, err, "LoadToVersion")
	eventtest.AssertLen(t, "events", events, 2)
}

func TestMemoryStore_LoadToVersion_ExceedsStreamLength(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt := eventtest.QuickEvent("Created", aggID, "User", 1, nil)

	_ = store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt},
	)

	events, err := store.LoadToVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		100,
	)
	eventtest.AssertNoError(t, err, "LoadToVersion")
	eventtest.AssertLen(t, "events", events, 1)
}

func TestMemoryStore_LoadToVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")

	_, err := store.LoadToVersion(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID), 5)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestMemoryStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")

	now, aggID := eventtest.MakeLoadToTimestampFixtures(
		t,
		store,
		ctx,
		"User",
		aggID,
		[3]event.Version{1, 1, 2},
	)

	loaded, err := store.LoadToTimestamp(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		now.Add(-30*time.Minute),
	)
	eventtest.AssertNoError(t, err, "LoadToTimestamp")
	eventtest.AssertLen(t, "loaded", loaded, 2)
}

func TestMemoryStore_LoadToTimestamp_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()

	_, err := store.LoadToTimestamp(
		context.Background(),
		event.NewAggregateRef("User", parseAggID("01HK1540X0841Y0A6BSX1VKR95")),
		time.Now(),
	)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestMemoryStore_LoadAllFromPosition(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1 := eventtest.MakeTimelineEvents(t, "User", aggID1, []eventtest.TimelineEvent{
		{Type: "Created", Version: 1, Offset: -2 * time.Hour},
	})
	evt2 := eventtest.MakeTimelineEvents(t, "Order", aggID2, []eventtest.TimelineEvent{
		{Type: "Created", Version: 1, Offset: -1 * time.Hour},
	})
	evt3 := eventtest.MakeTimelineEvents(t, "User", aggID1, []eventtest.TimelineEvent{
		{Type: "Updated", Version: 1, Offset: 0},
	})

	_ = store.AppendBatch(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID1), evt1)
	_ = store.AppendBatch(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID2), evt2)
	_ = store.AppendBatch(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID1), evt3)

	all, err := store.ReadAll(ctx)
	eventtest.AssertNoError(t, err, "ReadAll")
	eventtest.AssertLen(t, "all events", all, 3)

	fromPos, err := store.ReadFrom(ctx, evt1[0].ID(), 1)
	eventtest.AssertNoError(t, err, "ReadFrom")
	eventtest.AssertLen(t, "from position", fromPos, 1)
}

func TestMemoryStore_LoadBackwards(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evt1 := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("UserUpdated", aggID, "User", 2, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", aggID, "User", 3, nil)

	err := store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backwardsLoader := event.BackwardsSource(store)
	events, err := backwardsLoader.LoadBackwards(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	eventtest.AssertEventType(t, events, 0, "UserDeleted")
	eventtest.AssertEventType(t, events, 1, "UserUpdated")
	eventtest.AssertEventType(t, events, 2, "UserCreated")
}

func TestMemoryStore_LoadBackwards_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")

	backwardsLoader := event.BackwardsSource(store)
	_, err := backwardsLoader.LoadBackwards(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
	)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}
