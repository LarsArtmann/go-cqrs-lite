package memory_test

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

func TestMemoryStore_LoadStream(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	aggID := id.NewAggregateID()
	clock := func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	ctx := context.Background()

	wantEvents := make([]event.Event, 0, 3)
	for i, typ := range []string{"order.placed", "order.paid", "order.shipped"} {
		wantEvents = append(wantEvents,
			eventtest.NewEventOpts(t, event.Type(typ), aggID, "Test",
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

	defer stream.Close()

	var got []event.Type

	for {
		evt, err := stream.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("stream next: %v", err)
		}

		got = append(got, evt.Type())
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
	defer store.Close()

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
