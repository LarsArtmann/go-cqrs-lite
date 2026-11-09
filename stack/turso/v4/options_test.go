package turso_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

func TestNew_BadPath_CleanError(t *testing.T) {
	t.Parallel()

	// A path under a nonexistent directory should fail cleanly.
	_, err := turso.New("/nonexistent/deep/path/that/does/not/exist/db.sqlite")
	if err == nil {
		t.Fatal("expected error for bad path, got nil")
	}
}

func TestNew_WithoutAutoMigrate_NoTables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	bundle, err := turso.New(
		filepath.Join(dir, "nomigrate.db"),
		turso.WithDSN(sqlopt.WithoutAutoMigrate()),
	)
	if err != nil {
		t.Fatalf("New with WithoutAutoMigrate: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// Saving an event should fail because the events table was never created.
	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Test", streamID)

	evts, evErr := event.NewEvents(
		streamID, "Test", 0,
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

func TestNew_WithoutWAL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	bundle, err := turso.New(
		filepath.Join(dir, "nowal.db"),
		turso.WithPragmas(sqlopt.WithoutWAL()),
	)
	if err != nil {
		t.Fatalf("New with WithoutWAL: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// Verify the database accepts writes (WAL disabled didn't break anything).
	ctx := context.Background()
	streamID := id.NewStreamID()
	ref := id.NewStreamRef("NoWal", streamID)

	evts, evErr := event.NewEvents(
		streamID, "NoWal", 0,
		[]event.Type{"nowal.created"},
		[]any{map[string]any{"ok": true}},
	)
	if evErr != nil {
		t.Fatalf("NewEvents: %v", evErr)
	}

	if err := bundle.EventSink.Save(ctx, ref, evts, 0); err != nil {
		t.Fatalf("Save without WAL: %v", err)
	}
}
