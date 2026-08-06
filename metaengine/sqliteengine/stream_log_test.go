package sqliteengine

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestStreamLogBackend_SQLiteRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	defer eng.Close()

	be, ok := eng.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("sqliteEngine does not implement StreamLogBackend")
	}

	// Append values.
	vals := []any{"event1", "event2", "event3"}
	if err := be.StreamAppend(ctx, "events", "stream-1", vals); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	// Read back.
	read, err := be.StreamRead(ctx, "events", "stream-1")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}
	if len(read) != 3 {
		t.Fatalf("expected 3 values, got %d", len(read))
	}

	// Version.
	ver, err := be.StreamVersion(ctx, "events", "stream-1")
	if err != nil {
		t.Fatalf("StreamVersion: %v", err)
	}
	if ver != 3 {
		t.Fatalf("expected version 3, got %d", ver)
	}

	// Journal read all.
	all, err := be.JournalReadAll(ctx, "events")
	if err != nil {
		t.Fatalf("JournalReadAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 journal entries, got %d", len(all))
	}

	// Journal read from.
	from, err := be.JournalReadFrom(ctx, "events", 1, 0)
	if err != nil {
		t.Fatalf("JournalReadFrom: %v", err)
	}
	if len(from) != 2 {
		t.Fatalf("expected 2 entries after seq 1, got %d", len(from))
	}
}

func TestStreamLogBackend_SQLiteAtomicAppender(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	defer eng.Close()

	ap, ok := eng.(metaengine.AtomicAppender)
	if !ok {
		t.Fatal("sqliteEngine does not implement AtomicAppender")
	}

	// Append with correct expected version (0).
	if err := ap.StreamAppendExpected(ctx, "events", "stream-1", 0, []any{"a", "b"}); err != nil {
		t.Fatalf("StreamAppendExpected v0: %v", err)
	}

	// Append with correct expected version (2).
	if err := ap.StreamAppendExpected(ctx, "events", "stream-1", 2, []any{"c"}); err != nil {
		t.Fatalf("StreamAppendExpected v2: %v", err)
	}

	// Append with wrong expected version (should conflict).
	err = ap.StreamAppendExpected(ctx, "events", "stream-1", 0, []any{"d"})
	if !errors.Is(err, metaengine.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	// Verify final state.
	be := eng.(metaengine.StreamLogBackend)
	read, _ := be.StreamRead(ctx, "events", "stream-1")
	if len(read) != 3 {
		t.Fatalf("expected 3 values, got %d", len(read))
	}
}

func TestStreamLogBackend_SQLiteMultipleStreams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()

	eng, _ := NewSQLiteEngine(db)
	defer eng.Close()

	be := eng.(metaengine.StreamLogBackend)

	// Append to two streams.
	_ = be.StreamAppend(ctx, "events", "stream-A", []any{"a1", "a2"})
	_ = be.StreamAppend(ctx, "events", "stream-B", []any{"b1"})

	// Verify stream isolation.
	aRead, _ := be.StreamRead(ctx, "events", "stream-A")
	if len(aRead) != 2 {
		t.Fatalf("expected 2 in stream-A, got %d", len(aRead))
	}

	bRead, _ := be.StreamRead(ctx, "events", "stream-B")
	if len(bRead) != 1 {
		t.Fatalf("expected 1 in stream-B, got %d", len(bRead))
	}

	// Journal has all 3 entries.
	all, _ := be.JournalReadAll(ctx, "events")
	if len(all) != 3 {
		t.Fatalf("expected 3 journal entries, got %d", len(all))
	}
}
