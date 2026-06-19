package pebble_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrspebble "github.com/larsartmann/go-cqrs-lite/pebble/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func TestBackend_AllStores(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	if backend.EventStore() == nil {
		t.Fatal("EventStore() returned nil")
	}

	if backend.CommandStore() == nil {
		t.Fatal("CommandStore() returned nil")
	}

	if backend.QueryStore() == nil {
		t.Fatal("QueryStore() returned nil")
	}

	if backend.SnapshotStore() == nil {
		t.Fatal("SnapshotStore() returned nil")
	}

	if backend.CheckpointStore() == nil {
		t.Fatal("CheckpointStore() returned nil")
	}

	if backend.ReadModels() == nil {
		t.Fatal("ReadModels() returned nil")
	}
}

func TestBackend_CommandStoreE2E(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	cmdStore := backend.CommandStore()

	ref := command.NewAggregateRef("User", id.NewAggregateID())
	cmd, err := command.NewPersistedCommand("user.create", ref, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := cmdStore.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := cmdStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}
}

func TestBackend_QueryStoreE2E(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	qStore := backend.QueryStore()

	q, err := query.NewPersistedQuery("user.list", []byte(`{}`))
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	if err := qStore.SaveQuery(ctx, q); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	all, err := qStore.ReadAllQueries(ctx)
	if err != nil {
		t.Fatalf("ReadAllQueries: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 query, got %d", len(all))
	}
}

func TestBackend_ReadModelsE2E(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backend, err := cqrspebble.Open(dir, &pebble.Options{}, slog.Default())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	t.Cleanup(func() { _ = backend.Close() })

	kvStore := backend.ReadModels()

	if err := kvStore.Set([]byte("rm:user:1"), []byte(`{"name":"alice"}`)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := kvStore.Get([]byte("rm:user:1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(val) != `{"name":"alice"}` {
		t.Errorf("unexpected value: %s", string(val))
	}

	has, err := kvStore.Has([]byte("rm:user:1"))
	if err != nil {
		t.Fatalf("Has: %v", err)
	}

	if !has {
		t.Error("expected Has to return true")
	}
}
