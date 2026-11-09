package sqliteengine_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSQLite_PushdownStandardTies pins compound-cursor pagination on the
// STANDARD pushdown path: collections without a layout plan route through
// buildStandardScanQuery (meta_map + json_extract), the variant the
// enginetest keyset harness never reaches because metaengine.Plan
// auto-applies a layout for declared sorts. The walk lives in the shared
// enginetest harness.
func TestSQLite_PushdownStandardTies(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	defer db.Close()

	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	defer eng.Close()

	enginetest.RunPushdownStandardTiesTest(t, eng, "pd_ties")
}
