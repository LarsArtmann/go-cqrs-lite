package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

type todoKey string

func (k todoKey) String() string { return string(k) }

type todoView struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func TestNew_ProducesWorkingBundle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db")

	b, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.EventSink == nil {
		t.Fatal("EventSink not set")
	}

	if b.EventSource == nil {
		t.Fatal("EventSource not set")
	}

	if b.CommandSink == nil {
		t.Fatal("CommandSink not set")
	}

	if b.SnapshotStore == nil {
		t.Fatal("SnapshotStore not set")
	}

	if b.Publisher == nil {
		t.Fatal("Publisher not set (bus)")
	}

	if b.ReadModels == nil {
		t.Fatal("ReadModels not set")
	}
}

// E2E: save events through the SQLite preset, load them back, verify
// ordering. This proves the Bundle actually persists to SQLite.
func TestNew_E2E_EventSaveLoadRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "events.db")

	b, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Todo", streamID)

	types := []event.Type{"todo.created", "todo.renamed", "todo.completed"}
	payloads := []any{
		map[string]any{"title": "buy milk"},
		map[string]any{"title": "buy oat milk"},
		map[string]any{"at": "now"},
	}

	events, err := event.NewEvents(streamID, "Todo", 0, types, payloads)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := b.EventSink.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("EventSink.Save: %v", err)
	}

	loaded, err := b.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("EventSource.Load: %v", err)
	}

	if len(loaded) != len(events) {
		t.Fatalf("loaded %d events, want %d", len(loaded), len(events))
	}

	for i, typ := range types {
		if loaded[i].Type() != typ {
			t.Errorf("event[%d] type = %s, want %s", i, loaded[i].Type(), typ)
		}
	}
}

// E2E: read-model roundtrip through the SQLite preset.
// Read models are persisted to the cqrs_kv table (SQL kv.Store).
func TestNew_E2E_ReadModelRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "views.db")

	b, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	store, err := stack.ReadModel[todoView, todoKey](
		b, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[todoView, todoKey]("todos:"),
	)
	if err != nil {
		t.Fatalf("ReadModel: %v", err)
	}

	ctx := context.Background()

	if err := store.Set(ctx, "1", &todoView{Title: "persisted", Done: true}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Title != "persisted" || !got.Done {
		t.Fatalf("read model mismatch: %+v", got)
	}
}

// E2E: read models now PERSIST across a process restart, because the SQLite
// preset backs them with a SQL kv.Store (cqrs_kv table) instead of kv.MemStore.
// This is the defining behaviour of the persistent read-model feature.
func TestNew_E2E_ReadModelPersistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "persist.db")

	// First instance: write a read model, then close.
	writer, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("New writer: %v", err)
	}

	wStore, err := stack.ReadModel[todoView, todoKey](
		writer, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[todoView, todoKey]("todos:"),
	)
	if err != nil {
		t.Fatalf("ReadModel writer: %v", err)
	}

	ctx := context.Background()

	if err := wStore.Set(ctx, "1", &todoView{Title: "survives restart", Done: true}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	// Second instance on the same DSN: the read model must still be there.
	reader, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("New reader: %v", err)
	}

	defer func() { _ = reader.Close() }()

	rStore, err := stack.ReadModel[todoView, todoKey](
		reader, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[todoView, todoKey]("todos:"),
	)
	if err != nil {
		t.Fatalf("ReadModel reader: %v", err)
	}

	got, err := rStore.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}

	if got.Title != "survives restart" || !got.Done {
		t.Fatalf("read model did not persist across reopen: %+v", got)
	}
}

func TestNew_InMemoryDSN(t *testing.T) {
	t.Parallel()

	// ":memory:" should work for ephemeral test databases.
	b, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("New(:memory:): %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.EventSink == nil {
		t.Fatal("EventSink not set for :memory: database")
	}
}

func TestNew_WithoutWAL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := sqlite.New(filepath.Join(dir, "nowal.db"), sqlite.WithPragmas(sqlopt.WithoutWAL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.EventSink == nil {
		t.Fatal("EventSink not set")
	}
}

func TestNew_CloseIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := sqlite.New(filepath.Join(dir, "close.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
