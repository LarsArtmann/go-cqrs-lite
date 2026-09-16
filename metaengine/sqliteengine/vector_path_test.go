package sqliteengine_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestVectorSearchPath_ModerncScansInGo pins the reported execution path on
// the pure-Go driver (go-scan) and exercises a search through the lazily
// probed path — the probe must fire on first use and cache, not at
// construction (which stays query-free).
func TestVectorSearchPath_ModerncScansInGo(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	vp, ok := eng.(metaengine.VectorPathReporter)
	if !ok {
		t.Fatal("sqlite engine must implement metaengine.VectorPathReporter")
	}

	if got := vp.VectorSearchPath(); got != metaengine.VectorPathScan {
		t.Fatalf("modernc vector path = %q, want %q", got, metaengine.VectorPathScan)
	}

	ctx := context.Background()

	if err := eng.(metaengine.VectorBackend).VectorInsert(ctx, "docs", metaengine.Embedding{
		ID: "a", Values: []float32{1, 0},
	}); err != nil {
		t.Fatalf("VectorInsert: %v", err)
	}

	results, err := eng.(metaengine.VectorBackend).VectorSearch(
		ctx,
		"docs",
		[]float32{1, 0},
		1,
		"cosine",
	)
	if err != nil {
		t.Fatalf("VectorSearch: %v", err)
	}

	if len(results) != 1 || results[0].ID != "a" {
		t.Fatalf("VectorSearch = %+v, want exactly a", results)
	}

	if got := vp.VectorSearchPath(); got != metaengine.VectorPathScan {
		t.Fatalf("vector path after cached probe = %q, want %q", got, metaengine.VectorPathScan)
	}
}
