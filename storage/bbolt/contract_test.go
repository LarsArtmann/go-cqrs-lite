package bbolt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestContract_SaveAndLoad(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreSaveAndLoad(t, backend.EventStore(), cfg)
}

func TestContract_ConcurrencyConflict(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreConcurrencyConflict(t, backend.EventStore(), cfg)
}

func TestContract_AppendBatch(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreAppendBatch(t, backend.EventStore(), cfg)
}

func TestContract_LoadFromVersion(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreLoadFromVersion(t, backend.EventStore(), cfg)
}

func TestContract_MetadataRoundtrip(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	cfg := eventtest.IssueStoreConfig()
	eventtest.TestStoreMetadataRoundtrip(t, backend.EventStore(), cfg, "")
}

func TestContract_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)

	var _ event.Store = backend.EventStore()
	var _ event.EventSink = backend.EventStore()
	var _ event.EventSource = backend.EventStore()
}

func TestContract_LoadToVersion(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	aggID, evts := makeNEvents(t, cfg, 3)
	ref := id.NewStreamRef(cfg.AggType, aggID)

	if err := store.AppendBatch(ctx, ref, evts); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.LoadToVersion(ctx, ref, 2)
	if err != nil {
		t.Fatalf("LoadToVersion(2): %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events up to version 2, got %d", len(loaded))
	}

	eventtest.AssertEventVersion(t, loaded, 0, 1)
	eventtest.AssertEventVersion(t, loaded, 1, 2)

	all, err := store.LoadToVersion(ctx, ref, 3)
	if err != nil {
		t.Fatalf("LoadToVersion(3): %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 events up to version 3, got %d", len(all))
	}
}

func TestContract_LoadToVersion_EmptyStream(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	ref := id.NewStreamRef(cfg.AggType, id.NewStreamID())

	_, err := store.LoadToVersion(ctx, ref, 5)
	if !errors.Is(err, event.ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound for empty stream, got %v", err)
	}
}

func TestContract_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	aggID := id.NewStreamID()
	ref := id.NewStreamRef(cfg.AggType, aggID)

	baseTime := time.Now().Add(-3 * time.Hour)

	evts := make([]event.Event, 3)
	for i := 0; i < 3; i++ {
		evt := cfg.NewTestEvent(t, aggID, event.Version(i+1),
			event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Hour)),
		)
		evts[i] = evt
	}

	if err := store.AppendBatch(ctx, ref, evts); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	cutoff := baseTime.Add(90 * time.Minute)
	loaded, err := store.LoadToTimestamp(ctx, ref, cutoff)
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events before cutoff, got %d", len(loaded))
	}

	eventtest.AssertEventVersion(t, loaded, 0, 1)
	eventtest.AssertEventVersion(t, loaded, 1, 2)
}

func TestContract_ReadAllAcrossStreams(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	for i := 0; i < 3; i++ {
		aggID, evt := newStreamEvent(t, cfg)
		ref := id.NewStreamRef(cfg.AggType, aggID)

		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	events, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events in journal, got %d", len(events))
	}

	seen := make(map[string]bool)
	for _, evt := range events {
		seen[evt.ID().String()] = true
	}

	if len(seen) != 3 {
		t.Fatalf("expected 3 unique events, got %d unique", len(seen))
	}
}

func TestContract_ReadFromWithLimit(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	for i := 0; i < 5; i++ {
		aggID, evt := newStreamEvent(t, cfg)
		ref := id.NewStreamRef(cfg.AggType, aggID)

		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 5 {
		t.Fatalf("expected 5 events, got %d", len(all))
	}

	limited, err := store.ReadFrom(ctx, all[2].ID(), 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(limited) != 2 {
		t.Fatalf("expected 2 events after position 2, got %d", len(limited))
	}

	limited2, err := store.ReadFrom(ctx, all[1].ID(), 2)
	if err != nil {
		t.Fatalf("ReadFrom with limit: %v", err)
	}

	if len(limited2) != 2 {
		t.Fatalf("expected 2 events with limit=2, got %d", len(limited2))
	}
}

func TestContract_ReadFromZeroID(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	for i := 0; i < 3; i++ {
		aggID, evt := newStreamEvent(t, cfg)
		ref := id.NewStreamRef(cfg.AggType, aggID)

		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	fromStart, err := store.ReadFrom(ctx, id.EventID{}, 0)
	if err != nil {
		t.Fatalf("ReadFrom zero ID: %v", err)
	}

	if len(fromStart) != 3 {
		t.Fatalf("expected 3 events from start, got %d", len(fromStart))
	}
}

func TestContract_AppendBatchMultiEvent(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	aggID, evts := makeNEvents(t, cfg, 3)
	ref := id.NewStreamRef(cfg.AggType, aggID)

	if err := store.AppendBatch(ctx, ref, evts); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 events, got %d", len(loaded))
	}

	eventtest.AssertEventVersion(t, loaded, 0, 1)
	eventtest.AssertEventVersion(t, loaded, 1, 2)
	eventtest.AssertEventVersion(t, loaded, 2, 3)
}

func TestContract_ConcurrentSavesDifferentStreams(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	done := make(chan error, 10)

	for i := range 10 {
		go func(idx int) {
			aggID, evt := newStreamEvent(t, cfg)
			ref := id.NewStreamRef(cfg.AggType, aggID)

			done <- store.Save(ctx, ref, []event.Event{evt}, 0)
		}(i)
	}

	for range 10 {
		if err := <-done; err != nil {
			t.Fatalf("concurrent save failed: %v", err)
		}
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 10 {
		t.Fatalf("expected 10 events from concurrent saves, got %d", len(all))
	}
}

func TestContract_LoadEmptyStream(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	ref := id.NewStreamRef(cfg.AggType, id.NewStreamID())

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load empty stream: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("expected 0 events for empty stream, got %d", len(loaded))
	}
}

func makeNEvents(
	t *testing.T,
	cfg eventtest.StoreTestConfig,
	n int,
) (id.StreamID, []event.Event) {
	t.Helper()

	aggID := id.NewStreamID()
	evts := make([]event.Event, n)
	for i := range n {
		evts[i] = cfg.NewTestEvent(t, aggID, event.Version(i+1))
	}

	return aggID, evts
}

func newStreamEvent(
	t *testing.T,
	cfg eventtest.StoreTestConfig,
) (id.StreamID, event.Event) {
	t.Helper()

	aggID := id.NewStreamID()
	evt := cfg.NewTestEvent(t, aggID, 1)
	return aggID, evt
}

