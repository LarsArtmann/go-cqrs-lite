package storage_test

import (
	"context"
	"errors"
	"io"
	"testing"

	_ "modernc.org/sqlite" // pure-Go SQLite driver

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func newSQLiteStreamStore(t *testing.T) *storage.SQLEventStore {
	t.Helper()

	ctx := context.Background()
	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := storage.SQLiteInitSchema(ctx, db); err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	store, err := storage.NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	return store
}

func mustStreamEvent(t *testing.T, typ string, aggID id.AggregateID, ver int) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type(typ),
		aggID,
		"Order",
		event.Version(ver),
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func seedStreamEvents(
	t *testing.T,
	store *storage.SQLEventStore,
	count int,
) (id.AggregateRef, []event.Event) {
	t.Helper()

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Order", aggID)
	events := make([]event.Event, count)
	for i := range count {
		events[i] = mustStreamEvent(t, "order.updated", aggID, i+1)
	}

	if err := store.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	return ref, events
}

func drainIter(iter event.EventIterator) ([]event.Event, error) {
	defer func() { _ = iter.Close() }()

	var out []event.Event
	for {
		evt, err := iter.Next()
		if errors.Is(err, io.EOF) {
			return out, nil
		}

		if err != nil {
			return out, err
		}

		out = append(out, evt)
	}
}

func TestSQLStream_LoadStream(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	ref, want := seedStreamEvents(t, store, 5)

	iter, err := store.LoadStream(context.Background(), ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}

	got, err := drainIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d", len(got), len(want))
	}
}

func TestSQLStream_LoadStream_Empty(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	ref := id.NewAggregateRef("Order", id.NewAggregateID())

	iter, err := store.LoadStream(context.Background(), ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}

	got, err := drainIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected 0 events, got %d", len(got))
	}
}

func TestSQLStream_LoadStreamFromVersion(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	ref, _ := seedStreamEvents(t, store, 5)

	iter, err := store.LoadStreamFromVersion(context.Background(), ref, event.Version(3))
	if err != nil {
		t.Fatalf("LoadStreamFromVersion: %v", err)
	}

	got, err := drainIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 events after v3, got %d", len(got))
	}

	if got[0].Version().Int() != 4 {
		t.Errorf("first version = %d, want 4", got[0].Version().Int())
	}
}

func TestSQLStream_ReadStream(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	seedStreamEvents(t, store, 3)
	seedStreamEvents(t, store, 2)

	iter, err := store.ReadStream(context.Background())
	if err != nil {
		t.Fatalf("ReadStream: %v", err)
	}

	got, err := drainIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 5 {
		t.Fatalf("expected 5 total events, got %d", len(got))
	}
}

func TestSQLStream_ReadStreamFrom_AfterID(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	seedStreamEvents(t, store, 5)

	all, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	iter, err := store.ReadStreamFrom(context.Background(), all[1].ID(), 2)
	if err != nil {
		t.Fatalf("ReadStreamFrom: %v", err)
	}

	got, err := drainIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 events after position, got %d", len(got))
	}
}

func TestSQLStream_ReadStreamFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	seedStreamEvents(t, store, 5)

	iter, err := store.ReadStreamFrom(context.Background(), id.EventID{}, 3)
	if err != nil {
		t.Fatalf("ReadStreamFrom zero ID: %v", err)
	}

	got, err := drainIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 events with limit, got %d", len(got))
	}
}

func TestSQLStream_CloseIdempotent(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	ref, _ := seedStreamEvents(t, store, 2)

	iter, err := store.LoadStream(context.Background(), ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}

	if err := iter.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := iter.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	_, err = iter.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("Next after Close = %v, want io.EOF", err)
	}
}

func TestSQLStream_StreamMatchesSlice(t *testing.T) {
	t.Parallel()

	store := newSQLiteStreamStore(t)
	ref, _ := seedStreamEvents(t, store, 10)

	sliceEvents, err := store.Load(context.Background(), ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	iter, err := store.LoadStream(context.Background(), ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}

	streamEvents, err := drainIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(sliceEvents) != len(streamEvents) {
		t.Fatalf("slice %d != stream %d", len(sliceEvents), len(streamEvents))
	}

	for i := range sliceEvents {
		if sliceEvents[i].ID() != streamEvents[i].ID() {
			t.Errorf("event %d ID mismatch", i)
		}
	}
}
