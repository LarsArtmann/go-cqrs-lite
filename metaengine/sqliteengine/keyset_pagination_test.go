package sqliteengine_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSQLite_KeysetPagination pins the compound-cursor contract through the
// SQLite engine's raw, pushdown, and planned scan surfaces.
func TestSQLite_KeysetPagination(t *testing.T) {
	t.Parallel()

	newEngine := func(t *testing.T) metaengine.Engine {
		t.Helper()

		db, err := sql.Open("sqlite", "file::memory:?cache=shared")
		if err != nil {
			t.Fatalf("sql.Open: %v", err)
		}

		t.Cleanup(func() { _ = db.Close() })

		db.SetMaxOpenConns(1)

		eng, err := sqliteengine.NewSQLiteEngine(db)
		if err != nil {
			t.Fatalf("NewSQLiteEngine: %v", err)
		}

		return eng
	}

	// Each Run call gets its own engine: the harness's store.Close closes
	// the engine it wrapped, so sharing one instance double-closes.
	enginetest.RunKeysetPaginationTest(t, newEngine(t))
	enginetest.RunKeysetExactEndTest(t, newEngine(t))
}
