package event_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestSliceStream_Next(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	clock := func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	events := make([]event.Event, 3)
	for i, typ := range []string{"user.created", "user.updated", "user.deleted"} {
		events[i] = testhelpers.NewEventOpts(t, event.Type(typ), aggID, "Test",
			event.Version(i+1), nil, event.WithClock(clock))
	}

	stream := event.NewSliceStream(events)
	defer stream.Close() //nolint:errcheck // test helper

	var got []string

	for {
		evt, ok := stream.Next()
		if !ok {
			break
		}

		got = append(got, string(evt.Type()))
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
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

func TestSliceStream_Empty(t *testing.T) {
	t.Parallel()

	stream := event.NewSliceStream(nil)
	defer stream.Close() //nolint:errcheck // test helper

	_, ok := stream.Next()
	if ok {
		t.Error("expected Next to return false for empty stream")
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
}

func TestStoreStreamAdapter_LoadStream(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close() //nolint:errcheck // test helper

	aggID := id.NewAggregateID()
	clock := func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	ctx := context.Background()

	wantEvents := []event.Event{
		testhelpers.NewEventOpts(
			t,
			"order.placed",
			aggID,
			"Test",
			1,
			[]byte(`{}`),
			event.WithClock(clock),
		),
		testhelpers.NewEventOpts(
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
		event.NewAggregateRef(event.AggregateType("Order"), aggID),
		wantEvents,
	)
	if err != nil {
		t.Fatalf("append batch: %v", err)
	}

	adapter := event.NewStoreStreamAdapter(store)

	stream, err := adapter.LoadStream(
		ctx,
		event.NewAggregateRef(event.AggregateType("Order"), aggID),
	)
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}

	defer stream.Close() //nolint:errcheck // test helper

	var got []event.Version

	for {
		evt, ok := stream.Next()
		if !ok {
			break
		}

		got = append(got, evt.Version())
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}

	if got[0].Int() != 1 || got[1].Int() != 2 {
		t.Errorf("versions = %v, want [1, 2]", got)
	}
}

func TestStoreStreamAdapter_LoadStream_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close() //nolint:errcheck // test helper

	adapter := event.NewStoreStreamAdapter(store)

	_, err := adapter.LoadStream(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("Order"), id.NewAggregateID()),
	)
	if err == nil {
		t.Fatal("expected error for non-existent aggregate")
	}

	if !isAggregateNotFound(err) {
		t.Errorf("expected ErrAggregateNotFound, got %v", err)
	}
}

func isAggregateNotFound(err error) bool {
	return errors.Is(err, event.ErrAggregateNotFound)
}
