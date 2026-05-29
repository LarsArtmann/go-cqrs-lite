package memory_test

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

func TestMemoryStore_LoadStream(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close() //nolint:errcheck // test helper

	aggID := id.NewAggregateID()
	clock := func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	ctx := context.Background()

	wantEvents := make([]event.Event, 0, 3)
	for i, typ := range []string{"order.placed", "order.paid", "order.shipped"} {
		wantEvents = append(wantEvents,
			testhelpers.NewEventOpts(t, event.Type(typ), aggID, "Test",
				event.Version(i+1), nil, event.WithClock(clock)))
	}

	err := store.AppendBatch(
		ctx,
		event.NewAggregateRef(event.AggregateType("Order"), aggID),
		wantEvents,
	)
	if err != nil {
		t.Fatalf("append batch: %v", err)
	}

	stream, err := store.LoadStream(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}

	defer stream.Close() //nolint:errcheck // test helper

	var got []event.Type

	for {
		evt, ok := stream.Next()
		if !ok {
			break
		}

		got = append(got, evt.Type())
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("got %d events, want 3", len(got))
	}

	want := []event.Type{"order.placed", "order.paid", "order.shipped"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("event[%d] type = %q, want %q", i, got[i], w)
		}
	}
}

func TestMemoryStore_LoadStream_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close() //nolint:errcheck // test helper

	_, err := store.LoadStream(
		context.Background(),
		event.NewAggregateRef("Order", id.NewAggregateID()),
	)
	if err == nil {
		t.Fatal("expected error for non-existent aggregate")
	}

	if !errors.Is(err, event.ErrAggregateNotFound) {
		t.Errorf("expected ErrAggregateNotFound, got %v", err)
	}
}
