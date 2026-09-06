package system_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	_ "modernc.org/sqlite"
)

// TestEventAdapter_ReadFrom_InterleavedCollections proves the end-to-end
// positional-resumption contract: EventAdapter.ReadFrom must resume after a
// specific event WITHOUT re-delivering earlier entries, even when another
// collection's appends share the engine's global seq counter. SQL engines with
// a global AUTOINCREMENT used to filter JournalReadFrom on raw seq values,
// re-delivering entries whose position was already consumed — duplicate
// projection processing for consumers running two collections on one engine.
func TestEventAdapter_ReadFrom_InterleavedCollections(t *testing.T) {
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
	slb := eng.(metaengine.StreamLogBackend)

	// Write to a SECOND collection first so the "events" collection's raw seqs
	// start above 1 (the precondition for the historical re-delivery bug).
	if err := slb.StreamAppend(ctx, "commands", "cmd-1", []any{"noop"}); err != nil {
		t.Fatalf("noise append: %v", err)
	}

	adapter := system.NewEventAdapter(slb, "events", system.WithSerialization())

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Order", streamID)

	first, err := event.NewEvents(streamID, "Order", 0,
		[]event.Type{"order.created"}, []any{map[string]any{"n": 1}})
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := adapter.Save(ctx, ref, first, 0); err != nil {
		t.Fatalf("save first: %v", err)
	}

	second, err := event.NewEvents(streamID, "Order", 1,
		[]event.Type{"order.updated"}, []any{map[string]any{"n": 2}})
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := adapter.Save(ctx, ref, second, 1); err != nil {
		t.Fatalf("save second: %v", err)
	}

	// Interleave another noise row between event reads.
	if err := slb.StreamAppend(ctx, "commands", "cmd-2", []any{"noop"}); err != nil {
		t.Fatalf("noise append: %v", err)
	}

	// Cold-cache resume after the FIRST event (no prior ReadFrom call) must
	// return exactly [order.updated] — never re-deliver order.created.
	got, err := adapter.ReadFrom(ctx, first[0].ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom(first): %v", err)
	}

	if len(got) != 1 || got[0].Type() != "order.updated" {
		t.Fatalf("ReadFrom(first) = %d events (types %v), want exactly [order.updated]",
			len(got), eventTypes(got))
	}

	// Warm-cache path: resume after the LAST event returns nothing.
	got, err = adapter.ReadFrom(ctx, got[0].ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom(last): %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("ReadFrom(last) = %d events, want 0 (end of journal)", len(got))
	}
}

func eventTypes(events []event.Event) []event.Type {
	types := make([]event.Type, 0, len(events))
	for _, evt := range events {
		types = append(types, evt.Type())
	}

	return types
}
