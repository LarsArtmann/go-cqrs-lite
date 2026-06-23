package turso_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/turso/v3"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
)

// TestMultiDB_Routing proves the multi-DB split routes each concern to the
// correct database. It writes an event, a command, and a read-model entry
// through a Bundle wired with WithEventDB + WithQueryDB + WithViewDB, then
// closes the Bundle, reopens each database file independently, and counts
// rows per table.
//
// Before the fix, openSecondaryStores placed all five stores in whichever DB
// called it last, so events and commands both ended up in the query DB while
// the event DB sat unused.
func TestMultiDB_Routing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	eventPath := filepath.Join(dir, "events.db")
	queryPath := filepath.Join(dir, "queries.db")
	viewPath := filepath.Join(dir, "views.db")

	bundle, err := turso.New(
		primaryPath,
		turso.WithEventDB(eventPath),
		turso.WithQueryDB(queryPath),
		turso.WithViewDB(viewPath),
	)
	if err != nil {
		t.Fatalf("turso.New multi-db: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()

	// Event → must land in the event DB.
	ref := event.NewAggregateRef("Test", aggID)
	evts, err := event.NewEvents(
		aggID, "Test", 0,
		[]event.Type{"test.created"},
		[]any{map[string]any{"name": "alpha"}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := bundle.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	// Command → must land in the query DB.
	cmdRef := command.NewAggregateRef("Test", aggID)
	cmd, err := command.NewPersistedCommand("test.create", cmdRef, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := bundle.CommandSink.Save(ctx, cmdRef, cmd); err != nil {
		t.Fatalf("Save command: %v", err)
	}

	// Read model → must land in the view DB.
	viewStore := kv.NewTypedStore[routingView, todoKey](bundle.ReadModels)

	if err := viewStore.Set(ctx, todoKey("v1"), &routingView{Name: "hello"}); err != nil {
		t.Fatalf("Set view: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify routing: each concern is in its assigned DB and absent from the
	// others.
	assertTursoRowCount(t, eventPath, "events", 1)
	assertTursoRowCount(t, eventPath, "commands", 0)
	assertTursoRowCount(t, queryPath, "commands", 1)
	assertTursoRowCount(t, queryPath, "events", 0)
	assertTursoRowCount(t, viewPath, "cqrs_kv", 1)
	assertTursoRowCount(t, primaryPath, "events", 0)
	assertTursoRowCount(t, primaryPath, "commands", 0)
}

// assertTursoRowCount reopens dbPath read-only and asserts the row count in
// table.
func assertTursoRowCount(t *testing.T, dbPath, table string, want int) {
	t.Helper()

	db, err := cqrsturso.Open(cqrsturso.DbPath(dbPath))
	if err != nil {
		t.Fatalf("open %s: %v", filepath.Base(dbPath), err)
	}
	defer func() { _ = db.Close() }()

	var got int

	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&got)
	if err != nil {
		t.Fatalf("count %s.%s: %v", filepath.Base(dbPath), table, err)
	}

	if got != want {
		t.Errorf("%s: table %q has %d rows, want %d", filepath.Base(dbPath), table, got, want)
	}
}

type routingView struct {
	Name string `json:"name"`
}
