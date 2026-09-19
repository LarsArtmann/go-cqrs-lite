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

func newSQLiteEventAdapter(t *testing.T) (metaengine.StreamLogBackend, *system.EventAdapter) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	slb := eng.(metaengine.StreamLogBackend)

	return slb, system.NewEventAdapter(slb, "events", system.WithSerialization())
}

func appendEvents(
	t *testing.T,
	ctx context.Context,
	adapter *system.EventAdapter,
	ref id.StreamRef,
	version int,
	types []event.Type,
) []event.Event {
	t.Helper()

	events, err := event.NewEvents(ref.ID, ref.Type, event.Version(version),
		types, []any{map[string]any{"v": version}})
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := adapter.Save(ctx, ref, events, event.Version(version)); err != nil {
		t.Fatalf("save at version %d: %v", version, err)
	}

	return events
}

// TestEventAdapter_ReadFrom_ZeroCursorPagedDrain simulates a catch-up
// subscriber (the projectionhost worker pattern): start at the zero cursor,
// then repeatedly resume after the last delivered event. On a
// SeqSeekableStreamLog backend every page is an O(log n) token seek; the
// drain must deliver each event exactly once — no duplicates, no skips —
// across interleaved appends to another collection sharing the seq counter.
func TestEventAdapter_ReadFrom_ZeroCursorPagedDrain(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	slb, adapter := newSQLiteEventAdapter(t)

	ref := id.NewStreamRef("Cart", id.NewStreamID())

	var all []event.Event

	for i := range 4 {
		evts := appendEvents(t, ctx, adapter, ref, i, []event.Type{"cart.item_added"})

		all = append(all, evts...)

		// Interleave noise into another collection between event batches:
		// the shared AUTOINCREMENT gives the events collection gapped tokens.
		if err := slb.StreamAppend(ctx, "commands", "cmd", []any{"noop"}); err != nil {
			t.Fatalf("noise append: %v", err)
		}
	}

	var (
		drain  []event.Event
		cursor id.EventID
	)

	for {
		page, err := adapter.ReadFrom(ctx, cursor, 2)
		if err != nil {
			t.Fatalf("ReadFrom page: %v", err)
		}

		if len(page) == 0 {
			break
		}

		drain = append(drain, page...)
		cursor = page[len(page)-1].ID()
	}

	if len(drain) != len(all) {
		t.Fatalf("paged drain delivered %d events, want %d", len(drain), len(all))
	}

	for i := range all {
		if drain[i].ID() != all[i].ID() {
			t.Fatalf("drain[%d] = %v, want %v (exactly-once order)", i, drain[i].ID(), all[i].ID())
		}
	}
}

// TestEventAdapter_ReadFrom_UnknownIDReadsFromStart pins the
// unknown-cursor-semantics of the token path: an event ID that is not in the
// journal resolves to "read from the start" (cursor 0), matching the
// position-based path.
func TestEventAdapter_ReadFrom_UnknownIDReadsFromStart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, adapter := newSQLiteEventAdapter(t)

	ref := id.NewStreamRef("Order", id.NewStreamID())
	all := appendEvents(t, ctx, adapter, ref, 0, []event.Type{"order.created"})

	got, err := adapter.ReadFrom(ctx, id.NewEventID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom(unknown): %v", err)
	}

	if len(got) != 1 || got[0].ID() != all[0].ID() {
		t.Fatalf("ReadFrom(unknown) = %d events, want the full journal (1)", len(got))
	}
}
