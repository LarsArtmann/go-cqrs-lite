package pebble_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/pebble/v2"
)

func newStreamTestStore(t *testing.T) *cqrspebble.EventStore {
	t.Helper()

	dir := t.TempDir()
	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("pebble.Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	return cqrspebble.NewStore(database, slog.Default())
}

func seedPebbleStreamEvents(
	t *testing.T,
	store *cqrspebble.EventStore,
	count int,
) (event.AggregateRef, []event.Event) {
	t.Helper()

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Order", aggID)
	events := make([]event.Event, count)
	for i := range count {
		evt, err := event.NewEvent(
			event.Type("order.updated"),
			aggID,
			"Order",
			event.Version(i+1),
			[]byte(`{}`),
		)
		if err != nil {
			t.Fatalf("NewEvent: %v", err)
		}

		events[i] = evt
	}

	if err := store.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	return ref, events
}

func drainPebbleIter(iter event.EventIterator) ([]event.Event, error) {
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

func TestPebbleStream_LoadStream(t *testing.T) {
	t.Parallel()

	store := newStreamTestStore(t)
	ref, want := seedPebbleStreamEvents(t, store, 5)

	iter, err := store.LoadStream(context.Background(), ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}

	got, err := drainPebbleIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d", len(got), len(want))
	}
}

func TestPebbleStream_LoadStreamFromVersion(t *testing.T) {
	t.Parallel()

	store := newStreamTestStore(t)
	ref, _ := seedPebbleStreamEvents(t, store, 5)

	iter, err := store.LoadStreamFromVersion(context.Background(), ref, event.Version(3))
	if err != nil {
		t.Fatalf("LoadStreamFromVersion: %v", err)
	}

	got, err := drainPebbleIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 events after v3, got %d", len(got))
	}
}

func TestPebbleStream_ReadStream(t *testing.T) {
	t.Parallel()

	store := newStreamTestStore(t)
	seedPebbleStreamEvents(t, store, 3)
	seedPebbleStreamEvents(t, store, 2)

	iter, err := store.ReadStream(context.Background())
	if err != nil {
		t.Fatalf("ReadStream: %v", err)
	}

	got, err := drainPebbleIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 5 {
		t.Fatalf("expected 5 events, got %d", len(got))
	}
}

func TestPebbleStream_ReadStreamFrom_AfterID(t *testing.T) {
	t.Parallel()

	store := newStreamTestStore(t)
	seedPebbleStreamEvents(t, store, 5)

	all, err := store.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	iter, err := store.ReadStreamFrom(context.Background(), all[1].ID(), 2)
	if err != nil {
		t.Fatalf("ReadStreamFrom: %v", err)
	}

	got, err := drainPebbleIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
}

func TestPebbleStream_ReadStreamFrom_ZeroID(t *testing.T) {
	t.Parallel()

	store := newStreamTestStore(t)
	seedPebbleStreamEvents(t, store, 5)

	iter, err := store.ReadStreamFrom(context.Background(), id.EventID{}, 3)
	if err != nil {
		t.Fatalf("ReadStreamFrom: %v", err)
	}

	got, err := drainPebbleIter(iter)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}
}

func TestPebbleStream_CloseIdempotent(t *testing.T) {
	t.Parallel()

	store := newStreamTestStore(t)
	ref, _ := seedPebbleStreamEvents(t, store, 2)

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
