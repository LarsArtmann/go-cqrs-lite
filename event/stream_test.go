package event_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

func TestSliceIterator_Next(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	clock := func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	events := make([]event.Event, 3)
	for i, typ := range []string{"user.created", "user.updated", "user.deleted"} {
		events[i] = eventtest.NewEventOpts(t, event.Type(typ), aggID, "Test",
			event.Version(i+1), nil, event.WithClock(clock))
	}

	iter := event.NewSliceIterator(events)
	defer iter.Close() //nolint:errcheck // test helper

	var got []string

	for {
		evt, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got = append(got, string(evt.Type()))
	}

	want := []string{"user.created", "user.updated", "user.deleted"}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d", len(got), len(want))
	}

	for i, w := range want {
		if got[i] != w {
			t.Errorf("event[%d] type = %q, want %q", i, got[i], w)
		}
	}
}

func TestSliceIterator_Empty(t *testing.T) {
	t.Parallel()

	iter := event.NewSliceIterator(nil)
	defer iter.Close() //nolint:errcheck // test helper

	_, err := iter.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF for empty iterator, got %v", err)
	}
}

func TestStreamingSource_MemoryStore_LoadStream(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close() //nolint:errcheck // test helper

	aggID := id.NewAggregateID()
	clock := func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	ctx := context.Background()

	wantEvents := []event.Event{
		eventtest.NewEventOpts(
			t,
			"order.placed",
			aggID,
			"Test",
			1,
			[]byte(`{}`),
			event.WithClock(clock),
		),
		eventtest.NewEventOpts(
			t,
			"order.paid",
			aggID,
			"Test",
			2,
			[]byte(`{}`),
			event.WithClock(clock),
		),
	}

	err := store.AppendBatch(
		ctx,
		id.NewAggregateRef(id.AggregateType("Order"), aggID),
		wantEvents,
	)
	if err != nil {
		t.Fatalf("append batch: %v", err)
	}

	var ss event.StreamingSource = store

	iter, err := ss.LoadStream(ctx, id.NewAggregateRef(id.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("LoadStream: %v", err)
	}

	defer iter.Close() //nolint:errcheck // test helper

	var got []event.Version

	for {
		evt, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got = append(got, evt.Version())
	}

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}

	if got[0].Int() != 1 || got[1].Int() != 2 {
		t.Errorf("versions = %v, want [1, 2]", got)
	}
}
