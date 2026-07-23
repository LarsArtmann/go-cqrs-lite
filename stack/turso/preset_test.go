package turso_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
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
	dbPath := filepath.Join(dir, "test.db")

	b, err := turso.New(dbPath)
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

	if b.Sync() != nil {
		t.Fatal("Sync() should be nil for local mode")
	}
}

// E2E: save events through the Turso preset, load them back, verify ordering.
func TestNew_E2E_EventSaveLoadRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "events.db")

	b, err := turso.New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	ctx := context.Background()
	aggID := id.NewStreamID()
	ref := id.NewStreamRef("Todo", aggID)

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

// E2E: read-model roundtrip through the Turso preset.
func TestNew_E2E_ReadModelRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "views.db")

	b, err := turso.New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	store, err := stack.ReadModel[todoView, todoKey](
		b.Bundle, codec.JSONCodec{},
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

// E2E: read models PERSIST across a process restart.
func TestNew_E2E_ReadModelPersistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")

	writer, err := turso.New(dbPath)
	if err != nil {
		t.Fatalf("New writer: %v", err)
	}

	wStore, err := stack.ReadModel[todoView, todoKey](
		writer.Bundle, codec.JSONCodec{},
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

	reader, err := turso.New(dbPath)
	if err != nil {
		t.Fatalf("New reader: %v", err)
	}

	defer func() { _ = reader.Close() }()

	rStore, err := stack.ReadModel[todoView, todoKey](
		reader.Bundle, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[todoView, todoKey]("todos:"),
	)
	if err != nil {
		t.Fatalf("ReadModel reader: %v", err)
	}

	got, err := rStore.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}

	if got.Title != "survives restart" {
		t.Fatalf("read model did not persist: %+v", got)
	}
}

// E2E: event persistence across bundle instances.
func TestNew_E2E_EventPersistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist.db")

	writer, err := turso.New(dbPath)
	if err != nil {
		t.Fatalf("New writer: %v", err)
	}

	ctx := context.Background()
	aggID := id.NewStreamID()
	ref := id.NewStreamRef("Todo", aggID)

	events, err := event.NewEvents(
		aggID, "Todo", 0,
		[]event.Type{"todo.created"},
		[]any{map[string]any{"title": "persistent"}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := writer.EventSink.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	reader, err := turso.New(dbPath)
	if err != nil {
		t.Fatalf("New reader: %v", err)
	}

	defer func() { _ = reader.Close() }()

	loaded, err := reader.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event after reopen, got %d", len(loaded))
	}
}
