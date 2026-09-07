//go:build cgo

package duckdbengine_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestDuckDBRestartSafety_StreamAndJournal verifies that reopening a persistent
// DuckDB database does NOT reset seq counters — which would cause silent key
// collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestDuckDBRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	probe, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	if err := probe.Close(); err != nil {
		t.Fatalf("probe close: %v", err)
	}

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return duckdbengine.New(path)
	})
}

// TestDuckDBRestartSafety_FromDB verifies seq seeding when using
// NewFromDB (caller-owned *sql.DB path).
func TestDuckDBRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "test.duckdb")

	// Phase 1: Open via New, write, close.
	eng1, err := duckdbengine.New(dir)
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	slb1, ok := eng1.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("engine must implement StreamLogBackend")
	}

	if err := slb1.StreamAppend(ctx, "events", "s1", []any{"a", "b"}); err != nil {
		t.Fatalf("first StreamAppend: %v", err)
	}

	if err := eng1.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	// Phase 2: Open a raw *sql.DB on the same file, wrap via NewFromDB,
	// append more.
	db, err := sql.Open("duckdb", dir)
	if err != nil {
		t.Fatalf("raw duckdb open: %v", err)
	}

	eng2, err := duckdbengine.NewFromDB(db)
	if err != nil {
		t.Fatalf("FromDB open: %v", err)
	}

	defer func() { _ = eng2.Close() }()

	slb2, ok := eng2.(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("reopened engine must implement StreamLogBackend")
	}

	if err := slb2.StreamAppend(ctx, "events", "s1", []any{"c"}); err != nil {
		t.Fatalf("post-restart StreamAppend: %v", err)
	}

	ver, err := slb2.StreamVersion(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamVersion after restart: %v", err)
	}

	if ver != 3 {
		t.Fatalf("FromDB restart: stream version = %d, want 3", ver)
	}

	values, err := slb2.StreamRead(ctx, "events", "s1")
	if err != nil {
		t.Fatalf("StreamRead after restart: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("FromDB restart: stream should retain all 3 events, got %d", len(values))
	}
}
