package memory_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryStore_LoadStream(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	defer store.Close()

	streamID := id.NewStreamID()
	clock := func() time.Time { return time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC) }

	ctx := context.Background()

	wantEvents := make([]event.Event, 0, 3)
	for i, typ := range []string{"order.placed", "order.paid", "order.shipped"} {
		wantEvents = append(wantEvents,
			eventtest.NewEventOpts(t, event.Type(typ), streamID, "Test",
				event.Version(i+1), nil, event.WithClock(clock)))
	}

	err := store.AppendBatch(
		ctx,
		id.NewStreamRef(id.StreamType("Order"), streamID),
		wantEvents,
	)
	if err != nil {
		t.Fatalf("append batch: %v", err)
	}

	stream, err := store.LoadStream(ctx, id.NewStreamRef(id.StreamType("Order"), streamID))
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
		id.NewStreamRef("Order", id.NewStreamID()),
	)
	if err == nil {
		t.Fatal("expected error for non-existent stream")
	}

	if !errors.Is(err, event.ErrStreamNotFound) {
		t.Errorf("expected ErrStreamNotFound, got %v", err)
	}
}
