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

func TestCacheInvalidationMiddleware_InvalidatesAfterPublish(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	reader := listing.NewInMemoryAggregateReader(store)
	bus := memory.NewMemoryBus()
	_ = bus.UsePublish(listing.CacheInvalidationMiddleware(reader))

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)

	evt, err := event.NewEvent("user.created", aggID, "User", event.Version(1), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatal(err)
	}

	page, err := reader.List(ctx, listing.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("before publish: got %d items, want 1", len(page.Items))
	}

	evt2, err := event.NewEvent("user.updated", aggID, "User", event.Version(2), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := bus.Publish(ctx, evt2); err != nil {
		t.Fatal(err)
	}

	if err := store.Save(ctx, ref, []event.Event{evt2}, event.Version(1)); err != nil {
		t.Fatal(err)
	}

	page2, err := reader.ListWithStatus(ctx, listing.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(page2.Items) != 1 {
		t.Fatalf("after publish+save: got %d items, want 1", len(page2.Items))
	}

	if page2.Items[0].Ref.Version != event.Version(2) {
		t.Errorf("version = %d, want 2", page2.Items[0].Ref.Version)
	}
}

func TestCacheInvalidationMiddleware_NoInvalidationOnPublishError(t *testing.T) {
	t.Parallel()

	invalidated := false
	invalidator := &stubInvalidator{onInvalidate: func() { invalidated = true }}

	mw := listing.CacheInvalidationMiddleware(invalidator)
	next := event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		return event.ErrBusClosed
	})

	wrapped := mw(next)
	err := wrapped.Publish(context.Background())
	if err == nil {
		t.Fatal("expected error from wrapped publisher")
	}

	if invalidated {
		t.Error("cache should not be invalidated when publish fails")
	}
}

type stubInvalidator struct {
	onInvalidate func()
}

func (s *stubInvalidator) InvalidateCache() { s.onInvalidate() }
