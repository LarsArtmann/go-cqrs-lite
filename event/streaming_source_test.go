package event_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestSliceIterator_YieldsAllEvents(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evts := []event.Event{
		eventtest.NewEvent(t, "test.created", aggID, "Test", 1, nil),
		eventtest.NewEvent(t, "test.updated", aggID, "Test", 2, nil),
		eventtest.NewEvent(t, "test.deleted", aggID, "Test", 3, nil),
	}

	iter := event.NewSliceIterator(evts)
	defer iter.Close()

	count := 0

	for {
		_, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count++
	}

	if count != 3 {
		t.Errorf("expected 3 events, got %d", count)
	}
}

func TestSliceIterator_EmptySlice(t *testing.T) {
	t.Parallel()

	iter := event.NewSliceIterator(nil)
	defer iter.Close()

	_, err := iter.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF on empty iterator, got %v", err)
	}
}

func TestSliceIterator_CloseStopsIteration(t *testing.T) {
	t.Parallel()

	evts := []event.Event{
		eventtest.NewEvent(t, "test.created", id.NewAggregateID(), "Test", 1, nil),
	}

	iter := event.NewSliceIterator(evts)

	if err := iter.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := iter.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF after Close, got %v", err)
	}
}

func TestSliceIterator_CloseIdempotent(t *testing.T) {
	t.Parallel()

	iter := event.NewSliceIterator(nil)

	for i := 0; i < 3; i++ {
		if err := iter.Close(); err != nil {
			t.Fatalf("Close call %d: %v", i, err)
		}
	}
}

func TestSliceIterator_AdaptsMemoryStore(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("User", aggID)

	evts := []event.Event{
		eventtest.NewEvent(t, "user.created", aggID, "User", 1, []byte(`{}`)),
		eventtest.NewEvent(t, "user.updated", aggID, "User", 2, []byte(`{}`)),
	}

	if err := store.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	iter := event.NewSliceIterator(loaded)
	defer iter.Close()

	var collected []event.Event

	for {
		evt, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("Next: %v", err)
		}

		collected = append(collected, evt)
	}

	if len(collected) != 2 {
		t.Fatalf("expected 2 events, got %d", len(collected))
	}
}
