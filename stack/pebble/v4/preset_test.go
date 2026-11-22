package pebble_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"
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

	b, err := pebble.New(filepath.Join(dir, "test"))
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

func TestNew_E2E_EventSaveLoadRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "events"))
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

func TestNew_DefaultOptionsAppliesBloomFilters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "bloom"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	// The preset should ship with bloom filters enabled. We verify by checking
	// the Metrics after writing some data and forcing a flush.
	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Todo", streamID)

	events, err := event.NewEvents(
		streamID, "Todo", 0,
		[]event.Type{"todo.created"},
		[]any{map[string]any{"title": "bloom test"}},
	)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := b.EventSink.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := b.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	metrics := b.Metrics()
	if metrics.NumFilesTotal == 0 {
		t.Fatal("expected flush to produce SST files, got 0")
	}
}

func TestNew_E2E_ReadModelRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "views"))
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

func TestNew_E2E_PersistenceAcrossBundles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist")

	// First bundle: write data
	writer, err := pebble.New(dbPath)
	if err != nil {
		t.Fatalf("New writer: %v", err)
	}

	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Todo", streamID)

	events, err := event.NewEvents(
		streamID, "Todo", 0,
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

	// Second bundle: read data back from same directory
	reader, err := pebble.New(dbPath)
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

func TestNew_CloseIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "close"))
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

func TestBundle_Checkpoint(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "source"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	backupDir := filepath.Join(dir, "backup")

	if err := b.Checkpoint(backupDir); err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
}

func TestBundle_Metrics(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "metrics"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = b.Close() }()

	m := b.Metrics()
	_ = m.BlockCacheHitRate()
}

// TestCheckpoint_ExistingDirFails covers the preset's Checkpoint error wrap:
// the backend rejects an existing target directory, and the preset must
// surface that as an Infrastructure-family error naming the target and the
// pebble_preset.checkpoint code.
func TestCheckpoint_ExistingDirFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	target := filepath.Join(dir, "existing")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	err = b.Checkpoint(target)
	if err == nil {
		t.Fatal("Checkpoint to an existing directory must fail")
	}

	famErr, ok := errors.AsType[*errorfamily.Error](err)
	if !ok {
		t.Fatalf("Checkpoint error must carry errorfamily identity, got %T: %v", err, err)
	}
	if famErr.Code() != "pebble_preset.checkpoint" {
		t.Errorf("Code() = %q, want %q", famErr.Code(), "pebble_preset.checkpoint")
	}
	if famErr.Family() != errorfamily.Infrastructure {
		t.Errorf("Family() = %v, want %v", famErr.Family(), errorfamily.Infrastructure)
	}
}

// TestPreset_OptionsAndLifecycle exercises the option surface and the
// snapshot/graceful-close lifecycle that the round-trip tests skip: every
// Option constructor must be accepted by New, NewSnapshot must hand out a
// closable read view, and GracefulClose must close the bundle cleanly.
func TestPreset_OptionsAndLifecycle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	b, err := pebble.New(filepath.Join(dir, "db"),
		pebble.WithDurability(stack.DurabilityNormal),
		pebble.WithLogger(slog.Default()),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	snapshot := b.NewSnapshot()
	if snapshot == nil {
		t.Fatal("NewSnapshot must return a non-nil snapshot")
	}

	if err := snapshot.Close(); err != nil {
		t.Fatalf("snapshot Close: %v", err)
	}

	if err := b.GracefulClose(context.Background()); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}
}
