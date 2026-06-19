package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
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
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Todo", aggID)

	types := []event.Type{"todo.created", "todo.renamed", "todo.completed"}
	payloads := []any{
		map[string]any{"title": "buy milk"},
		map[string]any{"title": "buy oat milk"},
		map[string]any{"at": "now"},
	}

	events, err := event.NewEvents(aggID, "Todo", 0, types, payloads)
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
// Read models are currently in-memory (kv.MemStore); this test verifies the
// wiring works end-to-end even though read models aren't SQL-persisted yet.
func TestNew_E2E_ReadModelRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "views.db")

	b, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	store, err := stack.ReadModel[todoView, todoKey](b, codec.JSONCodec{},
		readmodel.WithKeyPrefix[todoView, todoKey]("todos:"),
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

	b, err := sqlite.New(filepath.Join(dir, "nowal.db"), sqlite.WithoutWAL())
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
