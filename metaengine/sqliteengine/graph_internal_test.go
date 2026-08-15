package sqliteengine

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// TestProbeRecursiveCTE_TrueOnSQLite verifies the CTE capability probe
// succeeds on a real SQLite driver (modernc), which means engines built on
// it use the single-query recursive traversal path.
func TestProbeRecursiveCTE_TrueOnSQLite(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if !probeRecursiveCTE(db) {
		t.Fatal("probeRecursiveCTE should succeed on modernc sqlite (WITH RECURSIVE supported)")
	}
}
