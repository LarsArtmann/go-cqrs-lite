package systemtest_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestEventAdapter_LoadByEventID_EndToEnd proves the production shape of the
// EventByIDBackend capability (#32): events saved through a serialized
// EventAdapter over the SQLite engine round-trip back by their globally
// unique ID — the O(1) path dashboards and trace players need, instead of a
// sequential journal scan.
func TestEventAdapter_LoadByEventID_EndToEnd(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	defer eng.Close()

	ctx := context.Background()
	adapter := system.NewEventAdapter(
		eng.(metaengine.StreamLogBackend), "events", system.WithSerialization(),
	)

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Order", streamID)

	events, err := event.NewEvents(streamID, "Order", 0,
		[]event.Type{"order.created", "order.shipped"},
		[]any{map[string]any{"n": 1}, map[string]any{"n": 2}})
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := adapter.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	want := events[1]
	got, err := adapter.LoadByEventID(ctx, want.ID())
	if err != nil {
		t.Fatalf("LoadByEventID: %v", err)
	}
	if got.ID() != want.ID() || got.Type() != want.Type() {
		t.Fatalf("resolved %s/%s, want %s/%s", got.ID(), got.Type(), want.ID(), want.Type())
	}
	if got.Version() != want.Version() {
		t.Fatalf("version %d, want %d", got.Version(), want.Version())
	}

	_, err = adapter.LoadByEventID(ctx, id.NewEventID())
	if !errors.Is(err, event.ErrEventNotFound) {
		t.Fatalf("miss must return event.ErrEventNotFound, got %v", err)
	}
}

// TestEventAdapter_LoadByEventID_UnsupportedBackend pins the degrade
// contract: a backend without the capability returns
// system.ErrLoadByEventIDUnsupported — callers detect it and fall back to
// their own sequential reads; the adapter never hides a journal scan behind
// the O(1)-looking method.
func TestEventAdapter_LoadByEventID_UnsupportedBackend(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := system.NewEventAdapter(
		metaengine.NewMemoryEngine().(metaengine.StreamLogBackend), "events",
	)

	_, err := adapter.LoadByEventID(ctx, id.NewEventID())
	if !errors.Is(err, system.ErrLoadByEventIDUnsupported) {
		t.Fatalf("memory backend must refuse with ErrLoadByEventIDUnsupported, got %v", err)
	}
}
