package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

func TestMultiDB_SeparateViewDB(t *testing.T) {
	t.Parallel()

	// Open with a separate view DB.
	bundle, err := sqlite.New(
		":memory:",
		sqlite.WithDSN(sqlopt.WithViewDB(":memory:")),
	)
	if err != nil {
		t.Fatalf("sqlite.New with ViewDB: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	// Verify the read model store works.
	type userView struct {
		Name string
	}

	store := kv.NewTypedStore[userView, testKey](bundle.ReadModels)

	ctx := context.Background()
	id := testKey("user-1")

	err = store.Set(ctx, id, &userView{Name: "Alice"})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if val.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", val.Name)
	}
}

// TestMultiDB_Routing proves the multi-DB split routes each concern to the
// correct database. It writes an event, a command, and a read-model entry
// through a Bundle wired with WithEventDB + WithQueryDB + WithViewDB, then
// closes the Bundle, reopens each database file independently, and counts
// rows per table. This is the only way to catch a routing bug where stores
// silently land in the wrong database.
func TestMultiDB_Routing(t *testing.T) {
	t.Parallel()

	// Real files (not :memory:) so each database survives Close and can be
	// reopened for row inspection.
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	eventPath := filepath.Join(dir, "events.db")
	queryPath := filepath.Join(dir, "queries.db")
	viewPath := filepath.Join(dir, "views.db")

	bundle, err := sqlite.New(
		primaryPath,
		sqlite.WithDSN(
			sqlopt.WithEventDB(eventPath),
			sqlopt.WithQueryDB(queryPath),
			sqlopt.WithViewDB(viewPath),
		),
	)
	if err != nil {
		t.Fatalf("sqlite.New multi-db: %v", err)
	}

	ctx := context.Background()
	streamID := id.NewStreamID()

	// Event → must land in the event DB.
	ref := id.NewStreamRef("Test", streamID)
	evts, err := event.NewEvents(
		streamID, "Test", 0,
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
	cmdRef := command.NewStreamRef("Test", streamID)
	cmd, err := command.NewPersistedCommand("test.create", cmdRef, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := bundle.CommandSink.Save(ctx, cmdRef, cmd); err != nil {
		t.Fatalf("Save command: %v", err)
	}

	// Read model → must land in the view DB.
	viewStore := kv.NewTypedStore[routingView, testKey](bundle.ReadModels)

	if err := viewStore.Set(ctx, testKey("v1"), &routingView{Name: "hello"}); err != nil {
		t.Fatalf("Set view: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify routing: each concern is in its assigned DB and absent from the
	// others. Before the fix, openSecondaryStores placed all five stores in
	// whichever DB called it last, so events and commands both ended up in
	// the query DB while the event DB sat unused.
	assertRowCount(t, eventPath, "events", 1)
	assertRowCount(t, eventPath, "commands", 0)
	assertRowCount(t, queryPath, "commands", 1)
	assertRowCount(t, queryPath, "events", 0)
	assertRowCount(t, viewPath, "cqrs_kv", 1)
	assertRowCount(t, primaryPath, "events", 0)
	assertRowCount(t, primaryPath, "commands", 0)
}

// assertRowCount reopens dbPath read-only and asserts the row count in table.
func assertRowCount(t *testing.T, dbPath, table string, want int) {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open %s: %v", filepath.Base(dbPath), err)
	}
	defer func() { _ = sqlDB.Close() }()

	var got int

	err = sqlDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&got)
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

type testKey string

func (t testKey) String() string { return string(t) }

var _ stack.Bundle = stack.Bundle{}

// TestMultiDB_PersistenceAcrossReopen verifies that data written through a
// multi-DB split survives closing and reopening the bundle — proving each
// database file is a real persistent store, not an in-memory alias.
func TestMultiDB_PersistenceAcrossReopen(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	eventPath := filepath.Join(dir, "events.db")
	viewPath := filepath.Join(dir, "views.db")

	// Write phase: create bundle with split, write event + read model.
	writer, err := sqlite.New(
		primaryPath,
		sqlite.WithDSN(
			sqlopt.WithEventDB(eventPath),
			sqlopt.WithViewDB(viewPath),
		),
	)
	if err != nil {
		t.Fatalf("New writer: %v", err)
	}

	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Todo", streamID)

	evts, err := event.NewEvents(
		streamID, "Todo", 0,
		[]event.Type{"todo.created"},
		[]any{map[string]any{"title": "multi-db persistence"}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := writer.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	wStore, err := stack.ReadModel[routingView, testKey](
		writer, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[routingView, testKey]("todos:"),
	)
	if err != nil {
		t.Fatalf("ReadModel writer: %v", err)
	}

	if err := wStore.Set(ctx, "1", &routingView{Name: "persisted"}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	// Read phase: reopen with same paths, verify data survived.
	reader, err := sqlite.New(
		primaryPath,
		sqlite.WithDSN(
			sqlopt.WithEventDB(eventPath),
			sqlopt.WithViewDB(viewPath),
		),
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

	rStore, err := stack.ReadModel[routingView, testKey](
		reader, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[routingView, testKey]("todos:"),
	)
	if err != nil {
		t.Fatalf("ReadModel reader: %v", err)
	}

	got, err := rStore.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}

	if got.Name != "persisted" {
		t.Fatalf("read model did not persist: %+v", got)
	}
}

// TestNew_WithForeignKeys verifies that the WithForeignKeys option enables
// PRAGMA foreign_keys=ON without errors.
func TestNew_WithForeignKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := sqlite.New(
		filepath.Join(dir, "fk.db"),
		sqlite.WithPragmas(sqlopt.WithForeignKeys()),
	)
	if err != nil {
		t.Fatalf("New with foreign keys: %v", err)
	}

	defer func() { _ = b.Close() }()

	// Verify the database accepts writes (schema is valid with FK on).
	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("FK", streamID)

	evts, err := event.NewEvents(
		streamID, "FK", 0,
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
