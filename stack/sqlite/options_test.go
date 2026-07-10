package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"
)

func TestNew_BadDSN_CleanError(t *testing.T) {
	t.Parallel()

	// A path under a nonexistent directory should fail when the preset
	// tries to execute PRAGMAs or DDL against it.
	_, err := sqlite.New("/nonexistent/deep/path/that/does/not/exist/db.sqlite")
	if err == nil {
		t.Fatal("expected error for bad DSN, got nil")
	}
}

func TestNew_WithoutAutoMigrate_NoTables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	bundle, err := sqlite.New(
		filepath.Join(dir, "nomigrate.db"),
		sqlite.WithoutAutoMigrate(),
	)
	if err != nil {
		t.Fatalf("New with WithoutAutoMigrate: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// Saving an event should fail because the events table was never created.
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Test", aggID)

	evts, evErr := event.NewEvents(
		aggID, "Test", 0,
		[]event.Type{"test.created"},
		[]any{map[string]any{"ok": true}},
	)
	if evErr != nil {
		t.Fatalf("NewEvents: %v", evErr)
	}

	saveErr := bundle.EventSink.Save(ctx, ref, evts, 0)
	if saveErr == nil {
		t.Fatal("expected Save to fail without schema, but it succeeded")
	}
}

func TestNew_WithOptimizations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	bundle, err := sqlite.New(
		filepath.Join(dir, "optimized.db"),
		sqlite.WithOptimizations(),
	)
	if err != nil {
		t.Fatalf("New with WithOptimizations: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// Verify the database accepts writes (optimizations didn't break schema).
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Opt", aggID)

	evts, evErr := event.NewEvents(
		aggID, "Opt", 0,
		[]event.Type{"opt.created"},
		[]any{map[string]any{"ok": true}},
	)
	if evErr != nil {
		t.Fatalf("NewEvents: %v", evErr)
	}

	if err := bundle.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save with optimizations: %v", err)
	}
}
