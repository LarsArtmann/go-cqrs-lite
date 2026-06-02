package listing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/listing/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func collectPublished(t *testing.T, bus *memory.MemoryBus, eventType event.Type) *[]event.Event {
	t.Helper()

	published := make([]event.Event, 0, 1)

	if err := bus.Subscribe(eventType, func(_ context.Context, evt event.Event) error {
		published = append(published, evt)

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	return &published
}

func setupStatusMiddlewareBus(
	t *testing.T,
	tombstoneTypes, rebirthTypes []event.Type,
) *memory.MemoryBus {
	t.Helper()

	bus := memory.NewMemoryBus()
	_ = bus.UsePublish(listing.StatusMiddleware(tombstoneTypes, rebirthTypes))

	return bus
}

func assertPublishSingleEvent(
	t *testing.T,
	bus *memory.MemoryBus,
	evt event.Event,
	eventType event.Type,
	wantMetadataKey event.MetadataKey, wantMetadataVal string,
) {
	t.Helper()

	published := collectPublished(t, bus, eventType)

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatal(err)
	}

	if len(*published) != 1 {
		t.Fatalf("got %d events, want 1", len(*published))
	}

	md := (*published)[0].Metadata()
	if md.Custom[wantMetadataKey] != wantMetadataVal {
		t.Errorf(
			"metadata[%s] = %q, want %q",
			wantMetadataKey,
			md.Custom[wantMetadataKey],
			wantMetadataVal,
		)
	}
}

func TestStatusMiddleware_Tombstone(t *testing.T) {
	t.Parallel()

	bus := setupStatusMiddlewareBus(
		t,
		[]event.Type{"user.deleted"},
		[]event.Type{"user.reactivated"},
	)

	deletedEvt, err := event.NewEvent(
		"user.deleted", id.NewAggregateID(), "User",
		event.Version(1), []byte(`{}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	assertPublishSingleEvent(t, bus, deletedEvt, "user.deleted",
		event.MetadataKeyTombstone, "true")
}

func TestStatusMiddleware_Rebirth(t *testing.T) {
	t.Parallel()

	bus := setupStatusMiddlewareBus(
		t,
		[]event.Type{"user.deleted"},
		[]event.Type{"user.reactivated"},
	)

	reactivatedEvt, err := event.NewEvent(
		"user.reactivated", id.NewAggregateID(), "User",
		event.Version(2), []byte(`{}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	assertPublishSingleEvent(t, bus, reactivatedEvt, "user.reactivated",
		event.MetadataKeyRebirth, "true")
}

func TestStatusMiddleware_UnmatchedPassthrough(t *testing.T) {
	t.Parallel()

	bus := setupStatusMiddlewareBus(
		t,
		[]event.Type{"user.deleted"},
		nil,
	)

	createdEvt, err := event.NewEvent(
		"user.created", id.NewAggregateID(), "User",
		event.Version(1), []byte(`{}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	assertPublishSingleEvent(t, bus, createdEvt, "user.created",
		event.MetadataKeyTombstone, "")
}
