package stack_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// TestMaterialize_StatusMiddlewareBridgeToOnTombstone pins the deprecated
// v4 bridge chain end-to-end: listing.StatusMiddleware marks delete/rebirth
// typed events with tombstone metadata, and Materialize routes that mark to
// the OnTombstone/OnRebirth handlers (ADR-0114 transition path; the v5 cut
// makes deletion purely event-type-driven and deletes this machinery).
func TestMaterialize_StatusMiddlewareBridgeToOnTombstone(t *testing.T) {
	t.Parallel()

	memStore := kv.NewMemStore()
	defer memStore.Close()

	ts := kv.NewTypedStore[userView, stringKey](memStore)

	mat := stack.Materialize[userView, stringKey]{
		Store: ts,
		KeyFromEvent: func(evt event.Event) (stringKey, error) {
			return stringKey(evt.StreamID().String()), nil
		},
		OnCreate: func(_ context.Context, _ event.Event) (*userView, error) {
			return &userView{Name: "alice"}, nil
		},
		OnTombstone: func(
			_ context.Context,
			_ event.Event,
			existing *userView,
		) (*userView, error) {
			view := existing
			if view == nil {
				view = &userView{}
			}

			view.Deleted = true

			return view, nil
		},
		OnRebirth: func(
			_ context.Context,
			_ event.Event,
			existing *userView,
		) (*userView, error) {
			view := existing
			if view == nil {
				view = &userView{}
			}

			view.Deleted = false

			return view, nil
		},
	}

	streamID := id.NewStreamID()
	ctx := context.Background()

	created, err := event.NewEvent("user.created", streamID, "User", event.Version(1), nil)
	if err != nil {
		t.Fatalf("NewEvent created: %v", err)
	}

	deleted, err := event.NewEvent("user.deleted", streamID, "User", event.Version(2), nil)
	if err != nil {
		t.Fatalf("NewEvent deleted: %v", err)
	}

	restored, err := event.NewEvent("user.reactivated", streamID, "User", event.Version(3), nil)
	if err != nil {
		t.Fatalf("NewEvent reactivated: %v", err)
	}

	var marked []event.Event

	capture := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		marked = append(marked, events...)

		return nil
	})

	publish := listing.StatusMiddleware(
		[]event.Type{"user.deleted"},
		[]event.Type{"user.reactivated"},
	)(capture)

	for _, evt := range []event.Event{created, deleted, restored} {
		if err := publish.Publish(ctx, evt); err != nil {
			t.Fatalf("publish %s: %v", evt.Type(), err)
		}
	}

	if len(marked) != 3 {
		t.Fatalf("captured %d events, want 3", len(marked))
	}

	if marked[1].Metadata().Tombstone == nil {
		t.Fatal("delete-typed event was not tombstone-marked by StatusMiddleware")
	}

	for _, evt := range marked {
		if err := mat.Handle(ctx, evt); err != nil {
			t.Fatalf("Handle %s: %v", evt.Type(), err)
		}
	}

	view, err := mat.View(ctx, stringKey(streamID.String()))
	if err != nil {
		t.Fatalf("View: %v", err)
	}

	if view.Name != "alice" {
		t.Errorf("Name = %q, want %q (OnCreate applied)", view.Name, "alice")
	}

	if view.Deleted {
		t.Error("Deleted = true after rebirth, want false (OnRebirth applied last)")
	}

	// Re-run the delete alone to observe the OnTombstone flip.
	deleteOnly, err := event.NewEvent("user.deleted", streamID, "User", event.Version(4), nil)
	if err != nil {
		t.Fatalf("NewEvent delete-only: %v", err)
	}

	var second []event.Event

	capture2 := event.PublisherFunc(func(_ context.Context, events ...event.Event) error {
		second = append(second, events...)

		return nil
	})

	if err := listing.StatusMiddleware(
		[]event.Type{"user.deleted"}, nil,
	)(capture2).Publish(ctx, deleteOnly); err != nil {
		t.Fatalf("publish delete-only: %v", err)
	}

	if err := mat.Handle(ctx, second[0]); err != nil {
		t.Fatalf("Handle delete-only: %v", err)
	}

	final, err := mat.View(ctx, stringKey(streamID.String()))
	if err != nil {
		t.Fatalf("View final: %v", err)
	}

	if !final.Deleted {
		t.Error("Deleted = false after delete event, want true (OnTombstone applied)")
	}
}
