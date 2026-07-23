package contracttest

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// MultiDBFactory creates a [stack.Bundle] configured with a multi-DB split
// (separate databases for events, commands/queries, and read models) and
// returns the metadata needed to verify routing correctness.
//
// The factory MUST use real files (not ":memory:") so databases survive Close
// and can be reopened for row inspection.
//
// Presets that support multi-DB (sqlite, turso, postgres) should wire this
// into their test suites. Presets without multi-DB (memory, pebble) skip it.
type MultiDBFactory func(t *testing.T) (*MultiDBTest, error)

// MultiDBTest bundles a multi-DB [stack.Bundle] with the DSN paths and a
// row-counting helper so [RunMultiDBSuite] can verify that each concern lands
// in the correct database after Close.
type MultiDBTest struct {
	*stack.Bundle

	// EventDSN is the DSN/path where events, snapshots, and checkpoints
	// should be persisted.
	EventDSN string
	// QueryDSN is the DSN/path where commands and queries should be
	// persisted.
	QueryDSN string
	// ViewDSN is the DSN/path where read models should be persisted.
	ViewDSN string
	// CountRows returns the number of rows in table within the database at
	// dsn. Called AFTER the Bundle is closed.
	CountRows func(t *testing.T, dsn, table string) int
}

// RunMultiDBSuite verifies that a multi-DB Bundle routes each concern to the
// correct database. It writes an event, a command, and a read-model entry
// through the Bundle, closes it, then reopens each database independently and
// counts rows per table. This catches routing bugs where stores silently land
// in the wrong database — the exact class of bug that was present in the
// initial multi-DB implementation.
//
// Usage:
//
//	func TestMultiDBContract(t *testing.T) {
//	    contracttest.RunMultiDBSuite(t, func(t *testing.T) (*contracttest.MultiDBTest, error) {
//	        dir := t.TempDir()
//	        eventDSN := filepath.Join(dir, "events.db")
//	        queryDSN := filepath.Join(dir, "queries.db")
//	        viewDSN := filepath.Join(dir, "views.db")
//	        b, err := sqlite.New(filepath.Join(dir, "primary.db"),
//	            sqlite.WithDSN(
//	                sqlopt.WithEventDB(eventDSN),
//	                sqlopt.WithQueryDB(queryDSN),
//	                sqlopt.WithViewDB(viewDSN),
//	            ),
//	        )
//	        if err != nil {
//	            return nil, err
//	        }
//	        return &contracttest.MultiDBTest{
//	            Bundle:    b,
//	            EventDSN:  eventDSN,
//	            QueryDSN:  queryDSN,
//	            ViewDSN:   viewDSN,
//	            CountRows: countSQLiteRows,
//	        }, nil
//	    })
//	}
func RunMultiDBSuite(t *testing.T, factory MultiDBFactory) {
	t.Helper()

	t.Run("FullRouting", func(t *testing.T) { testFullRouting(t, factory) })
}

// testFullRouting is the comprehensive routing proof. It writes all three
// concern types through one Bundle, closes it, and verifies mutual exclusion:
// events are ONLY in the event DB, commands ONLY in the query DB, views ONLY
// in the view DB, and nothing in any other database.
func testFullRouting(t *testing.T, factory MultiDBFactory) {
	t.Helper()

	mt, err := factory(t)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewStreamID()

	// Event → must land in the event DB.
	ref := id.NewStreamRef("Routing", aggID)

	evts, err := event.NewEvents(
		aggID, "Routing", 0,
		[]event.Type{"routing.created"},
		[]any{map[string]any{"name": "alpha"}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err = mt.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	// Command → must land in the query DB.
	cmdRef := command.NewStreamRef("Routing", aggID)

	cmd, err := command.NewPersistedCommand("routing.create", cmdRef, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err = mt.CommandSink.Save(ctx, cmdRef, cmd); err != nil {
		t.Fatalf("Save command: %v", err)
	}

	// Read model → must land in the view DB.
	store, err := stack.ReadModel[contractView, contractKey](
		mt.Bundle, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[contractView, contractKey]("routing:"),
	)
	if err != nil {
		t.Fatalf("ReadModel: %v", err)
	}

	if err = store.Set(ctx, "1", &contractView{Title: "routed", Done: false}); err != nil {
		t.Fatalf("Set view: %v", err)
	}

	// Close so all writes are flushed to disk.
	if err = mt.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// ── Event DB: events yes, commands no, views no ──
	assertRows(t, mt, mt.EventDSN, "events", 1, "event DB should have 1 event")
	assertRows(t, mt, mt.EventDSN, "commands", 0, "event DB should have 0 commands")
	assertRows(t, mt, mt.EventDSN, "cqrs_kv", 0, "event DB should have 0 views")

	// ── Query DB: commands yes, events no, views no ──
	assertRows(t, mt, mt.QueryDSN, "commands", 1, "query DB should have 1 command")
	assertRows(t, mt, mt.QueryDSN, "events", 0, "query DB should have 0 events")
	assertRows(t, mt, mt.QueryDSN, "cqrs_kv", 0, "query DB should have 0 views")

	// ── View DB: views yes, events no, commands no ──
	assertRows(t, mt, mt.ViewDSN, "cqrs_kv", 1, "view DB should have 1 view")
	assertRows(t, mt, mt.ViewDSN, "events", 0, "view DB should have 0 events")
	assertRows(t, mt, mt.ViewDSN, "commands", 0, "view DB should have 0 commands")
}

func assertRows(t *testing.T, mt *MultiDBTest, dsn, table string, want int, msg string) {
	t.Helper()

	got := mt.CountRows(t, dsn, table)
	if got != want {
		t.Errorf("%s: got %d rows in %s, want %d", msg, got, table, want)
	}
}
