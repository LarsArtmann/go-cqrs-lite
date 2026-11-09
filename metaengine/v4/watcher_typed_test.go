package metaengine

import (
	"context"
	"testing"
	"time"
)

// TestWatchTyped_MemoryEngine verifies the WatchTyped convenience function
// returns a typed channel that delivers values without runtime type assertions
// for the Memory engine (fast path: value is already the correct Go type).
func TestWatchTyped_MemoryEngine(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, w := WatchTyped[testTask](store, ctx, "tasks", nil)
	defer w.Close()

	err := store.Apply(ctx, "task_created", testTask{ID: "t1", Title: "Hello"})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case val := <-ch:
		if val.ID != testTaskID("t1") {
			t.Errorf("expected ID t1, got %s", val.ID)
		}
		if val.Title != "Hello" {
			t.Errorf("expected Title 'Hello', got %s", val.Title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

// TestWatchTyped_SQLiteEngine verifies the WatchTyped convenience function
// works with the SQLite engine, where values arrive as map[string]any and are
// reified to V via JSON round-trip (the reify fallback path).
func TestWatchTyped_SQLiteEngine(t *testing.T) {
	t.Skip("SQLite-specific — moved to sqliteengine module after ADR-0115")
	t.Parallel()

	store := newSQLiteTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, w := WatchTyped[testTask](store, ctx, "tasks", nil)
	defer w.Close()

	err := store.Apply(ctx, "task_created", testTask{ID: "t2", Title: "SQLite"})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case val := <-ch:
		if val.ID != testTaskID("t2") {
			t.Errorf("expected ID t2, got %s", val.ID)
		}
		if val.Title != "SQLite" {
			t.Errorf("expected Title 'SQLite', got %s", val.Title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SQLite notification")
	}
}

// TestWatchTypedWithSeq verifies the WatchTypedWithSeq convenience function
// returns SeqValue pairs on the typed channel.
func TestWatchTypedWithSeq(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, w := WatchTypedWithSeq[testTask](store, ctx, "tasks", nil)
	defer w.Close()

	err := store.Apply(ctx, "task_created", testTask{ID: "t3", Title: "Seq"})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case sv := <-ch:
		if sv.Value.ID != testTaskID("t3") {
			t.Errorf("expected ID t3, got %s", sv.Value.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SeqValue notification")
	}
}
