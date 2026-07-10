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
	ref := id.NewAggregateRef("Test", aggID)
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

// TestMultiDB_SeparateViewDB verifies that a separate view DB is used for
// read models when WithViewDB is set, matching the SQLite preset test.
func TestMultiDB_SeparateViewDB(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	bundle, err := turso.New(
		filepath.Join(dir, "primary.db"),
		turso.WithViewDB(filepath.Join(dir, "views.db")),
	)
	if err != nil {
		t.Fatalf("turso.New with ViewDB: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	store := kv.NewTypedStore[routingView, todoKey](bundle.ReadModels)

	ctx := context.Background()

	if err := store.Set(ctx, todoKey("v1"), &routingView{Name: "Alice"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := store.Get(ctx, todoKey("v1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if val.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", val.Name)
	}
}

// TestNew_WithForeignKeys verifies that the WithForeignKeys option works
// without errors and the database is usable.
func TestNew_WithForeignKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := turso.New(
		filepath.Join(dir, "fk.db"),
		turso.WithForeignKeys(),
	)
	if err != nil {
		t.Fatalf("New with foreign keys: %v", err)
	}

	defer func() { _ = b.Close() }()

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("FK", aggID)

	evts, err := event.NewEvents(
		aggID, "FK", 0,
		[]event.Type{"fk.created"},
		[]any{map[string]any{"ok": true}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := b.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save with FK: %v", err)
	}
}

// TestNew_WithOptimizations verifies that the WithOptimizations option
// applies CQRS indexes and PRAGMA optimizations without errors.
func TestNew_WithOptimizations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := turso.New(
		filepath.Join(dir, "opt.db"),
		turso.WithOptimizations(),
	)
	if err != nil {
		t.Fatalf("New with optimizations: %v", err)
	}

	defer func() { _ = b.Close() }()

	// Verify the database is fully functional with optimizations applied.
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Opt", aggID)

	evts, err := event.NewEvents(
		aggID, "Opt", 0,
		[]event.Type{"opt.created"},
		[]any{map[string]any{"optimized": true}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := b.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save with optimizations: %v", err)
	}

	loaded, err := b.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}
}

// TestMultiDB_PersistenceAcrossReopen verifies that data written through a
// multi-DB split survives closing and reopening the bundle.
func TestMultiDB_PersistenceAcrossReopen(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	eventPath := filepath.Join(dir, "events.db")
	viewPath := filepath.Join(dir, "views.db")

	writer, err := turso.New(
		primaryPath,
		turso.WithEventDB(eventPath),
		turso.WithViewDB(viewPath),
	)
	if err != nil {
		t.Fatalf("New writer: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Todo", aggID)

	evts, err := event.NewEvents(
		aggID, "Todo", 0,
		[]event.Type{"todo.created"},
		[]any{map[string]any{"title": "multi-db persistence"}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := writer.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	reader, err := turso.New(
		primaryPath,
		turso.WithEventDB(eventPath),
		turso.WithViewDB(viewPath),
	)
	if err != nil {
		t.Fatalf("New reader: %v", err)
	}

	defer func() { _ = reader.Close() }()

	loaded, err := reader.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load after reopen: %v", err)
	}

	if len(loaded) != 1 || loaded[0].Type() != "todo.created" {
		t.Fatalf("expected 1 event 'todo.created', got %+v", loaded)
	}
}

// TestNewSync_RejectsMultiDBOptions verifies that NewSync returns an explicit
// error when multi-DB options are passed. Multi-DB split is incompatible with
// sync because all stores must share one syncing database for consistent
// Push/Pull replication.
func TestNewSync_RejectsMultiDBOptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()

	tests := []struct {
		name string
		opt  turso.Option
	}{
		{"WithEventDB", turso.WithEventDB(filepath.Join(dir, "events.db"))},
		{"WithQueryDB", turso.WithQueryDB(filepath.Join(dir, "queries.db"))},
		{"WithViewDB", turso.WithViewDB(filepath.Join(dir, "views.db"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := turso.NewSync(
				ctx,
				filepath.Join(dir, tt.name+".db"),
				"libsql://fake.turso.io",
				"fake-token",
				tt.opt,
			)
			if err == nil {
				t.Fatalf("NewSync with %s should return error", tt.name)
			}
		})
	}
}
