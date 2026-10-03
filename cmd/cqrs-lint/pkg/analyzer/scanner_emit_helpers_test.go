package analyzer

import "testing"

// The nsfw-classifier false-positive class (2026-10-03): event emissions
// routed through a constructor helper whose parameter receives the event
// type constant at the CALL site.
//
//	func newRoomEvent(t event.Type) (event.Event, error) { return event.New(t, ...) }
//	newRoomEvent(evtRoomItemDeleted, ...)
//
// Raw detection left soft-delete false and A012 fired on folds that
// demonstrably handle deletion via const case clauses.

func TestEmitHelperIndirectionResolvesSoftDelete(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"events.go": `package rooms

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const evtRoomItemDeleted event.Type = "room.item_deleted"
const evtRoomItemAdded event.Type = "room.item_added"

func newRoomEvent(t event.Type, actor id.ActorID) (event.Event, error) {
	return event.New(t, "rooms", id.NewStreamID("x"), 1, nil, event.WithActor(actor))
}

func deleteRoomItem(actor id.ActorID) (event.Event, error) {
	return newRoomEvent(evtRoomItemDeleted, actor)
}

func addRoomItem(actor id.ActorID) (event.Event, error) {
	return newRoomEvent(evtRoomItemAdded, actor)
}
`,
	})

	if _, ok := ctx.Registry.EventTypesEmitted["room.item_deleted"]; !ok {
		t.Errorf("helper-indirect emission missing room.item_deleted; emitted=%v",
			ctx.Registry.EventTypesEmitted)
	}
	if _, ok := ctx.Registry.EventTypesEmitted["room.item_added"]; !ok {
		t.Errorf("helper-indirect emission missing room.item_added; emitted=%v",
			ctx.Registry.EventTypesEmitted)
	}

	// Re-detect AFTER resolution (the real loader resolves before per-module
	// detection; BuildContextFromSource detects earlier for legacy reasons).
	if !DetectFeatures(ctx).HasSoftDelete {
		t.Error("soft-delete detection must see helper-indirect tombstone emissions")
	}
}

func TestResolveFoldTombstoneCases_ConstIdentifierCase(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"fold.go": `package rooms

import "github.com/larsartmann/go-cqrs-lite/event/v4"

const evtRoomItemDeleted event.Type = "room.item_deleted"

func foldRoom(prev int, evt event.Event) (int, error) {
	switch evt.Type() {
	case evtRoomItemDeleted:
		return -1, nil
	default:
		return prev, nil
	}
}
`,
	})

	if len(ctx.Registry.Folds) != 1 {
		t.Fatalf("got %d folds, want 1", len(ctx.Registry.Folds))
	}

	fold := ctx.Registry.Folds[0]
	if !fold.HasSwitch {
		t.Fatal("fold must have a switch")
	}
	if !fold.HandlesTombstoneEvent {
		t.Errorf("const case evtRoomItemDeleted must resolve to a tombstone case; values=%v",
			fold.SwitchCaseValues)
	}
}

func TestResolveFoldTombstoneCases_LiteralCase(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"fold.go": `package rooms

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func foldRoom(prev int, evt event.Event) (int, error) {
	switch evt.Type() {
	case "room.item_deleted":
		return -1, nil
	default:
		return prev, nil
	}
}
`,
	})

	if len(ctx.Registry.Folds) != 1 {
		t.Fatalf("got %d folds, want 1", len(ctx.Registry.Folds))
	}
	if !ctx.Registry.Folds[0].HandlesTombstoneEvent {
		t.Error("literal tombstone case must mark the fold as handling deletion")
	}
}

func TestResolveFoldTombstoneCases_NonTombstoneStaysUnmarked(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"fold.go": `package rooms

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func foldRoom(prev int, evt event.Event) (int, error) {
	switch evt.Type() {
	case "room.item_added":
		return prev + 1, nil
	default:
		return prev, nil
	}
}
`,
	})

	if len(ctx.Registry.Folds) != 1 {
		t.Fatalf("got %d folds, want 1", len(ctx.Registry.Folds))
	}
	if ctx.Registry.Folds[0].HandlesTombstoneEvent {
		t.Error("non-tombstone cases must not mark the fold")
	}
}
