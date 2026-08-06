package metaengine

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

)

// TestConcurrentAppend_Memory verifies that AtomicAppender prevents
// concurrent writers from silently interleaving events. Only one writer
// should succeed per version; the other gets ErrVersionConflict.
func TestConcurrentAppend_Memory(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()
	defer func() { _ = eng.Close() }()

	ap, ok := eng.(AtomicAppender)
	if !ok {
		t.Fatal("memoryEngine does not implement AtomicAppender")
	}

	ctx := context.Background()

	// 10 goroutines race to append to the same stream at expectedVersion=0.
	// Exactly 1 should succeed; 9 should get ErrVersionConflict.
	const goroutines = 10

	var successCount, conflictCount atomic.Int32

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			err := ap.StreamAppendExpected(ctx, "events", "stream-1", 0, []any{"e1"})
			if err == nil {
				successCount.Add(1)
			} else if errors.Is(err, ErrVersionConflict) {
				conflictCount.Add(1)
			}
		}()
	}

	wg.Wait()

	if successCount.Load() != 1 {
		t.Fatalf("expected exactly 1 successful append, got %d", successCount.Load())
	}

	if conflictCount.Load() != goroutines-1 {
		t.Fatalf("expected %d conflicts, got %d", goroutines-1, conflictCount.Load())
	}

	// Verify the stream has exactly 1 entry.
	values, err := eng.(StreamLogBackend).StreamRead(ctx, "events", "stream-1")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}

	if len(values) != 1 {
		t.Fatalf("expected 1 value in stream, got %d", len(values))
	}
}

// TestConcurrentAppend_SQLite verifies optimistic concurrency on SQLite.
func TestConcurrentAppend_SQLite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer db.Close()

	eng, err := newMemoryEngineForTest()
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	defer func() { _ = eng.Close() }()

	ap, ok := eng.(AtomicAppender)
	if !ok {
		t.Fatal("sqliteEngine does not implement AtomicAppender")
	}

	ctx := context.Background()

	// Sequential appends to build version, then a concurrent race.
	if err := ap.StreamAppendExpected(ctx, "events", "s1", 0, []any{"a", "b"}); err != nil {
		t.Fatalf("append v0: %v", err)
	}

	const goroutines = 5

	var successCount, conflictCount atomic.Int32

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			err := ap.StreamAppendExpected(ctx, "events", "s1", 2, []any{"c"})
			if err == nil {
				successCount.Add(1)
			} else if errors.Is(err, ErrVersionConflict) {
				conflictCount.Add(1)
			}
		}()
	}

	wg.Wait()

	if successCount.Load() != 1 {
		t.Fatalf("expected exactly 1 successful append, got %d", successCount.Load())
	}

	if conflictCount.Load() != goroutines-1 {
		t.Fatalf("expected %d conflicts, got %d", goroutines-1, conflictCount.Load())
	}
}
