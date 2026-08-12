package bbolt

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestStreaming_LoadStream(t *testing.T) {
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

	iter, err := store.LoadStream(ctx, ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}
	defer iter.Close()

	var loaded []event.Event
	for {
		evt, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		loaded = append(loaded, evt)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 events, got %d", len(loaded))
	}

	eventtest.AssertEventVersion(t, loaded, 0, 1)
	eventtest.AssertEventVersion(t, loaded, 1, 2)
	eventtest.AssertEventVersion(t, loaded, 2, 3)
}

func TestStreaming_LoadStreamFromVersion(t *testing.T) {
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

	iter, err := store.LoadStreamFromVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadStreamFromVersion: %v", err)
	}
	defer iter.Close()

	var loaded []event.Event
	for {
		evt, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		loaded = append(loaded, evt)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 events after version 1, got %d", len(loaded))
	}

	eventtest.AssertEventVersion(t, loaded, 0, 2)
	eventtest.AssertEventVersion(t, loaded, 1, 3)
}

func TestStreaming_LoadStreamEmpty(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	ref := id.NewStreamRef(cfg.AggType, id.NewStreamID())

	iter, err := store.LoadStream(ctx, ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}
	defer iter.Close()

	_, err = iter.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF on empty stream, got %v", err)
	}
}

func TestStreaming_ReadStream(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	for i := range 5 {
		aggID, evt := newStreamEvent(t, cfg)
		ref := id.NewStreamRef(cfg.AggType, aggID)

		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	iter, err := store.ReadStream(ctx)
	if err != nil {
		t.Fatalf("ReadStream: %v", err)
	}
	defer iter.Close()

	count := 0
	for {
		_, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		count++
	}

	if count != 5 {
		t.Fatalf("expected 5 events, got %d", count)
	}
}

func TestStreaming_ReadStreamFromWithSkip(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	for i := range 5 {
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

	iter, err := store.ReadStreamFrom(ctx, all[2].ID(), 0)
	if err != nil {
		t.Fatalf("ReadStreamFrom: %v", err)
	}
	defer iter.Close()

	count := 0
	for {
		_, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		count++
	}

	if count != 2 {
		t.Fatalf("expected 2 events after position 2, got %d", count)
	}
}

func TestStreaming_ReadStreamFromWithLimit(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	for i := range 5 {
		aggID, evt := newStreamEvent(t, cfg)
		ref := id.NewStreamRef(cfg.AggType, aggID)

		if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	iter, err := store.ReadStreamFrom(ctx, id.EventID{}, 3)
	if err != nil {
		t.Fatalf("ReadStreamFrom: %v", err)
	}
	defer iter.Close()

	count := 0
	for {
		_, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		count++
	}

	if count != 3 {
		t.Fatalf("expected 3 events with limit=3, got %d", count)
	}
}

func TestStreaming_CloseThenNext(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	store := backend.EventStore()
	ctx := context.Background()
	cfg := eventtest.IssueStoreConfig()

	aggID, evt := newStreamEvent(t, cfg)
	ref := id.NewStreamRef(cfg.AggType, aggID)

	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	iter, err := store.LoadStream(ctx, ref)
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}

	if err := iter.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err = iter.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF after Close, got %v", err)
	}
}

func TestStreaming_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)

	var _ event.StreamingSource = backend.EventStore()
	var _ event.StreamingJournal = backend.EventStore()
}
