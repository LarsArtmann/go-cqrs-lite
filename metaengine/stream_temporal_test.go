package metaengine

import (
	"context"
	"database/sql"
	"testing"

)

func TestStreamReadAsOfVersion_Memory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := NewMemoryEngine()
	backend := eng.(StreamLogBackend)

	values := []any{"a", "b", "c", "d", "e"}
	if err := backend.StreamAppend(ctx, "events", "s1", values); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	temporal := eng.(StreamTemporalReader)

	// Read first 3.
	result, err := temporal.StreamReadAsOfVersion(ctx, "events", "s1", 3)
	if err != nil {
		t.Fatalf("StreamReadAsOfVersion(3): %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 values, got %d", len(result))
	}

	// Read all (exceeds stream length).
	result, err = temporal.StreamReadAsOfVersion(ctx, "events", "s1", 100)
	if err != nil {
		t.Fatalf("StreamReadAsOfVersion(100): %v", err)
	}

	if len(result) != 5 {
		t.Fatalf("expected 5 values, got %d", len(result))
	}

	// Read 0 returns empty.
	result, err = temporal.StreamReadAsOfVersion(ctx, "events", "s1", 0)
	if err != nil {
		t.Fatalf("StreamReadAsOfVersion(0): %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 values, got %d", len(result))
	}
}

func TestStreamReadAsOfVersion_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	defer db.Close()

	eng, err := newMemoryEngineForTest()
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	backend := eng.(StreamLogBackend)

	values := []any{"a", "b", "c", "d", "e"}
	if err := backend.StreamAppend(ctx, "events", "s1", values); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	temporal := eng.(StreamTemporalReader)

	// Read first 3.
	result, err := temporal.StreamReadAsOfVersion(ctx, "events", "s1", 3)
	if err != nil {
		t.Fatalf("StreamReadAsOfVersion(3): %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 values, got %d", len(result))
	}

	// Read all (exceeds stream length).
	result, err = temporal.StreamReadAsOfVersion(ctx, "events", "s1", 100)
	if err != nil {
		t.Fatalf("StreamReadAsOfVersion(100): %v", err)
	}

	if len(result) != 5 {
		t.Fatalf("expected 5 values, got %d", len(result))
	}

	// Read 0 returns empty.
	result, err = temporal.StreamReadAsOfVersion(ctx, "events", "s1", 0)
	if err != nil {
		t.Fatalf("StreamReadAsOfVersion(0): %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 values, got %d", len(result))
	}
}
