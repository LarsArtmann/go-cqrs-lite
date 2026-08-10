package listing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestCacheInvalidationMiddleware_InvalidatesAfterPublish(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	reader := listing.NewInMemoryStreamReader(store)
	bus := eventtest.NewFakeBus()
	_ = bus.UsePublish(listing.CacheInvalidationMiddleware(reader))

	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("User", streamID)

	evt, err := event.NewEvent("user.created", streamID, "User", event.Version(1), []byte(`{}`))
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

	evt2, err := event.NewEvent("user.updated", streamID, "User", event.Version(2), []byte(`{}`))
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
