package dgraphengine_test

import (
	"context"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDgraph_Multimap_EmptyCollection(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MultimapBackend)

	// MultiGet on a collection with no data should return empty, not error.
	got, err := mb.MultiGet(ctx, "empty-mm", "nonexistent-key")
	if err != nil {
		t.Fatalf("MultiGet on empty collection: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("MultiGet on empty collection = %d items, want 0", len(got))
	}
}

func TestDgraph_Multimap_AddAndGetRoundtrip(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MultimapBackend)

	col := uniqueCollection(t, "mm_roundtrip")

	// Add multiple values to the same key.
	for _, v := range []string{"alpha", "beta", "gamma"} {
		if err := mb.MultiAdd(ctx, col, "k1", v); err != nil {
			t.Fatalf("MultiAdd %s: %v", v, err)
		}
	}

	got, err := mb.MultiGet(ctx, col, "k1")
	if err != nil {
		t.Fatalf("MultiGet: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("MultiGet = %d items, want 3", len(got))
	}
}

func TestDgraph_Log_EmptyCollection(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()
	lb := eng.(metaengine.LogBackend)

	// LogTail on empty collection with limit=0 (all entries).
	got, err := lb.LogTail(ctx, "empty-log", 0)
	if err != nil {
		t.Fatalf("LogTail on empty collection (limit=0): %v", err)
	}

	if len(got) != 0 {
		t.Errorf("LogTail on empty = %d items, want 0", len(got))
	}

	// LogTail on empty collection with limit > entries.
	got, err = lb.LogTail(ctx, "empty-log", 100)
	if err != nil {
		t.Fatalf("LogTail on empty collection (limit=100): %v", err)
	}

	if len(got) != 0 {
		t.Errorf("LogTail on empty with large limit = %d items, want 0", len(got))
	}
}

func TestDgraph_Log_AppendAndTailRoundtrip(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()
	lb := eng.(metaengine.LogBackend)

	col := uniqueCollection(t, "log_roundtrip")

	for i, v := range []string{"first", "second", "third"} {
		if err := lb.LogAppend(ctx, col, v); err != nil {
			t.Fatalf("LogAppend %d: %v", i, err)
		}
		// Sleep to ensure unique nanosecond timestamps.
		time.Sleep(1 * time.Millisecond)
	}

	// LogTail with no limit should return all entries.
	got, err := lb.LogTail(ctx, col, 0)
	if err != nil {
		t.Fatalf("LogTail (all): %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("LogTail = %d items, want 3", len(got))
	}

	// LogTail with limit=2 should return the last 2 entries.
	got, err = lb.LogTail(ctx, col, 2)
	if err != nil {
		t.Fatalf("LogTail (limit=2): %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("LogTail limit=2 = %d items, want 2", len(got))
	}

	// LogTail with limit > entries should return all entries.
	got, err = lb.LogTail(ctx, col, 100)
	if err != nil {
		t.Fatalf("LogTail (limit=100): %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("LogTail limit=100 = %d items, want 3", len(got))
	}
}
