package stream_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

func TestStatusMiddleware_Tombstone(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	bus.UsePublish(stream.StatusMiddleware(
		[]event.Type{"user.deleted"},
		[]event.Type{"user.reactivated"},
	))

	deletedEvt, err := event.NewEvent(
		"user.deleted", id.NewAggregateID(), "User",
		event.Version(1), []byte(`{}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	published := make([]event.Event, 0, 1)

	err = bus.Subscribe("user.deleted", func(_ context.Context, evt event.Event) error {
		published = append(published, evt)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = bus.Publish(context.Background(), deletedEvt)
	if err != nil {
		t.Fatal(err)
	}

	if len(published) != 1 {
		t.Fatalf("got %d events, want 1", len(published))
	}

	md := published[0].Metadata()
	if md.Custom[event.MetadataKeyTombstone] != "true" {
		t.Error("tombstone metadata not set by middleware")
	}
}

func TestStatusMiddleware_Rebirth(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	bus.UsePublish(stream.StatusMiddleware(
		[]event.Type{"user.deleted"},
		[]event.Type{"user.reactivated"},
	))

	reactivatedEvt, err := event.NewEvent(
		"user.reactivated", id.NewAggregateID(), "User",
		event.Version(2), []byte(`{}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	published := make([]event.Event, 0, 1)

	err = bus.Subscribe("user.reactivated", func(_ context.Context, evt event.Event) error {
		published = append(published, evt)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = bus.Publish(context.Background(), reactivatedEvt)
	if err != nil {
		t.Fatal(err)
	}

	if len(published) != 1 {
		t.Fatalf("got %d events, want 1", len(published))
	}

	md := published[0].Metadata()
	if md.Custom[event.MetadataKeyRebirth] != "true" {
		t.Error("rebirth metadata not set by middleware")
	}
}

func TestStatusMiddleware_UnmatchedPassthrough(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	bus.UsePublish(stream.StatusMiddleware(
		[]event.Type{"user.deleted"},
		nil,
	))

	createdEvt, err := event.NewEvent(
		"user.created", id.NewAggregateID(), "User",
		event.Version(1), []byte(`{}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	published := make([]event.Event, 0, 1)

	err = bus.Subscribe("user.created", func(_ context.Context, evt event.Event) error {
		published = append(published, evt)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = bus.Publish(context.Background(), createdEvt)
	if err != nil {
		t.Fatal(err)
	}

	if len(published) != 1 {
		t.Fatalf("got %d events, want 1", len(published))
	}

	md := published[0].Metadata()
	if md.Custom[event.MetadataKeyTombstone] != "" {
		t.Error("unmatched event should not have tombstone metadata")
	}
}
