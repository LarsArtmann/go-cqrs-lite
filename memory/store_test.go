package memory_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt1 := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("UserUpdated", aggID, "User", 1, nil)

	err := store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1},
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt2},
		1,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events", events, 2)
}

func TestMemoryStore_VersionConflict(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)

	err := store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt},
		5,
	)
	if err == nil {
		t.Error("expected version conflict error")
	}
}

func TestMemoryStore_AggregateNotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	_, err := store.Load(
		ctx,
		event.NewAggregateRef(
			event.AggregateType("User"),
			id.MustParseAggregateID("01HK154KER4E8AJ20Q4JD5TJ1E"),
		),
	)
	if err == nil {
		t.Error("expected aggregate not found error")
	}
}

func TestMemoryStore_LoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154ME034FVHK95R554AKSE")
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

func TestMemoryStore_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	_ = store.Close()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)

	err := store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt},
		0,
	)
	if err == nil {
		t.Error("expected store closed error")
	}
}

func TestMemoryStore_ClosedLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	_, err := store.Load(
		ctx,
		event.NewAggregateRef(
			event.AggregateType("User"),
			id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"),
		),
	)
	if err == nil {
		t.Error("expected store closed error on Load")
	}
}

func TestMemoryStore_ClosedLoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	_, err := store.LoadFromVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		0,
	)
	if err == nil {
		t.Error("expected store closed error on LoadFromVersion")
	}
}

func TestMemoryStore_LoadFromVersion_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154KER4E8AJ20Q4JD5TJ1E")

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

	aggID := id.MustParseAggregateID("01HK154PCGXJ80RFXRASTMSSK0")
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

func TestMemoryStore_AppendBatch(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154QBR6CK7JX737HQB4V58")
	evt1 := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1, evt2, evt3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events", events, 3)
}

func TestMemoryStore_AppendBatch_AppendsToExisting(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK154RB0WD5V767Z27XMXRX0")
	evt1 := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)
	_ = store.Save(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt1},
		0,
	)

	evt2 := eventtest.QuickEvent("UserUpdated", aggID, "User", 1, nil)
	evt3 := eventtest.QuickEvent("UserDeleted", aggID, "User", 2, nil)

	err := store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
		[]event.Event{evt2, evt3},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events total", events, 3)
}

func TestMemoryStore_AppendBatch_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	aggID := id.MustParseAggregateID("01HK154SA8Y7AMZCYV919GE46K")
	evt := eventtest.QuickEvent("UserCreated", aggID, "User", 1, nil)

	err := store.AppendBatch(
		context.Background(),
		event.NewAggregateRef("User", aggID),
		[]event.Event{evt},
	)
	if err == nil {
		t.Error("expected store closed error")
	}
}

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
					event.NewAggregateRef(event.AggregateType("User"), aggID),
					[]event.Event{evt},
					event.Version(idx),
				)
			}
		}()

		go func() {
			defer wg.Done()

			for range eventsPerGoroutine {
				_, _ = store.Load(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID))
			}
		}()

		_ = i
	}

	wg.Wait()
}

func TestMemoryStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
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

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
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

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	_, err := store.LoadToVersion(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID), 5)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestMemoryStore_LoadToVersion_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	_, err := store.LoadToVersion(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), id.NewAggregateID()),
		1,
	)
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}

func TestMemoryStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

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
		event.NewAggregateRef("User", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")),
		time.Now(),
	)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got: %v", err)
	}
}

func TestMemoryStore_LoadToTimestamp_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	_, err := store.LoadToTimestamp(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("User"), id.NewAggregateID()),
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error for closed store")
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

func TestMemoryStore_ReadFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt := eventtest.QuickEvent("Created", aggID, "User", 1, nil)

	_ = store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
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

func TestMemoryStore_ReadFrom_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	_, err := store.ReadFrom(context.Background(), id.EventID{}, 10)
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}

func TestMemoryStore_LoadBackwards(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
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

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	backwardsLoader := event.BackwardsSource(store)
	_, err := backwardsLoader.LoadBackwards(
		ctx,
		event.NewAggregateRef(event.AggregateType("User"), aggID),
	)
	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Fatalf("expected ErrAggregateNotFound, got %v", err)
	}
}

func TestMemoryStore_LoadBackwards_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	backwardsLoader := event.BackwardsSource(store)
	_, err := backwardsLoader.LoadBackwards(context.Background(), event.AggregateRef{})
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}

func seedTestEvents(
	t *testing.T,
	store *memory.MemoryStore,
	ctx context.Context,
	aggID id.AggregateID,
	n int,
) {
	t.Helper()

	for i := range n {
		evt := eventtest.QuickEvent("Created", aggID, "User", event.Version(i+1), nil)
		_ = store.AppendBatch(
			ctx,
			event.NewAggregateRef(event.AggregateType("User"), aggID),
			[]event.Event{evt},
		)
	}
}
