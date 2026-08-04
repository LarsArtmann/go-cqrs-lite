package metaengine

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite" // SQLite driver
)

func TestSnapshotBackend_Memory(t *testing.T) {
	t.Parallel()

	backend := NewMemorySnapshotBackend()
	ctx := context.Background()

	// Save a snapshot.
	if err := backend.SnapshotSave(
		ctx,
		"snapshots",
		"stream-1",
		5,
		[]byte("state-v5"),
	); err != nil {
		t.Fatalf("SnapshotSave: %v", err)
	}

	// Load latest.
	data, ver, err := backend.SnapshotLoad(ctx, "snapshots", "stream-1")
	if err != nil {
		t.Fatalf("SnapshotLoad: %v", err)
	}

	if ver != 5 {
		t.Fatalf("expected version 5, got %d", ver)
	}

	if string(data) != "state-v5" {
		t.Fatalf("expected 'state-v5', got %q", string(data))
	}

	// Overwrite with newer version.
	if err := backend.SnapshotSave(
		ctx,
		"snapshots",
		"stream-1",
		10,
		[]byte("state-v10"),
	); err != nil {
		t.Fatalf("SnapshotSave overwrite: %v", err)
	}

	// LoadAtVersion 15 should return v10 (10 <= 15).
	data, ver, err = backend.SnapshotLoadAtVersion(ctx, "snapshots", "stream-1", 15)
	if err != nil {
		t.Fatalf("SnapshotLoadAtVersion(15): %v", err)
	}

	if ver != 10 || string(data) != "state-v10" {
		t.Fatalf("expected v10/state-v10, got v%d/%q", ver, string(data))
	}

	// LoadAtVersion 7 should fail (only v10 exists, 10 > 7).
	_, _, err = backend.SnapshotLoadAtVersion(ctx, "snapshots", "stream-1", 7)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for v10 at maxVersion 7, got %v", err)
	}

	// Missing snapshot.
	_, _, err = backend.SnapshotLoad(ctx, "snapshots", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Delete.
	if err := backend.SnapshotDelete(ctx, "snapshots", "stream-1"); err != nil {
		t.Fatalf("SnapshotDelete: %v", err)
	}

	_, _, err = backend.SnapshotLoad(ctx, "snapshots", "stream-1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSnapshotBackend_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	defer db.Close()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	// The engine implements SnapshotBackend.
	backend, ok := eng.(SnapshotBackend)
	if !ok {
		t.Fatal("sqliteEngine does not implement SnapshotBackend")
	}

	// Save a snapshot.
	if err := backend.SnapshotSave(
		ctx,
		"snapshots",
		"stream-1",
		5,
		[]byte("state-v5"),
	); err != nil {
		t.Fatalf("SnapshotSave: %v", err)
	}

	// Load latest.
	data, ver, err := backend.SnapshotLoad(ctx, "snapshots", "stream-1")
	if err != nil {
		t.Fatalf("SnapshotLoad: %v", err)
	}

	if ver != 5 || string(data) != "state-v5" {
		t.Fatalf("expected v5/state-v5, got v%d/%q", ver, string(data))
	}

	// Overwrite with newer version.
	if err := backend.SnapshotSave(
		ctx,
		"snapshots",
		"stream-1",
		10,
		[]byte("state-v10"),
	); err != nil {
		t.Fatalf("SnapshotSave overwrite: %v", err)
	}

	// LoadAtVersion 15 should return v10 (10 <= 15).
	data, ver, err = backend.SnapshotLoadAtVersion(ctx, "snapshots", "stream-1", 15)
	if err != nil {
		t.Fatalf("SnapshotLoadAtVersion(15): %v", err)
	}

	if ver != 10 || string(data) != "state-v10" {
		t.Fatalf("expected v10/state-v10, got v%d/%q", ver, string(data))
	}

	// LoadAtVersion 7 should fail (only v10 exists, 10 > 7).
	_, _, err = backend.SnapshotLoadAtVersion(ctx, "snapshots", "stream-1", 7)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for v10 at maxVersion 7, got %v", err)
	}

	// Missing snapshot.
	_, _, err = backend.SnapshotLoad(ctx, "snapshots", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Delete.
	if err := backend.SnapshotDelete(ctx, "snapshots", "stream-1"); err != nil {
		t.Fatalf("SnapshotDelete: %v", err)
	}

	_, _, err = backend.SnapshotLoad(ctx, "snapshots", "stream-1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
