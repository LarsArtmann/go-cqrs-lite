//go:build cgo

package duckdbengine_test

import (
	"os"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_DuckDB runs the AutoCRUDByConvention soak against the
// DuckDB engine (columnar OLAP backend) to verify no memory leaks and CRUD
// lifecycle correctness under sustained write load. DuckDB's columnar storage
// and memory management differ from both Memory and Pebble — this test
// catches OLAP-specific issues (e.g. unbounded append, MVCC bloat).
//
// Skips in -short mode. Skips when SOAK_SKIP_DUCKDB=1 (for CI that cannot
// afford the ~80-100s runtime). Runtime: ~80s without -race, ~100s with -race.
//
// NOT parallel: RunAutoCRUDSoak asserts on the process-global heap.
func TestSoak_AutoCRUD_DuckDB(t *testing.T) {
	if testing.Short() {
		t.Skip("DuckDB soak: skipped in -short mode (own budget; run via #test/#test-all-backends)")
	}

	if os.Getenv("SOAK_SKIP_DUCKDB") == "1" {
		t.Skip("DuckDB soak: skipped by SOAK_SKIP_DUCKDB=1")
	}

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	// store.Close() inside RunAutoCRUDSoak closes the engine.
	enginetest.RunAutoCRUDSoak(t, eng)
}
