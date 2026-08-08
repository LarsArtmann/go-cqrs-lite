package sqliteengine_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSoak_AutoCRUD_SQLite runs the AutoCRUDByConvention soak against the
// SQLite engine (SQL backend) to verify no memory leaks and CRUD lifecycle
// correctness under sustained write load. SQLite's B-tree page management and
// WAL behavior differ from Memory and Pebble — this test catches SQL-specific
// issues (e.g. unbounded statement cache, connection pool leaks).
func TestSoak_AutoCRUD_SQLite(t *testing.T) {
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

	// store.Close() inside RunAutoCRUDSoak closes the engine.
	enginetest.RunAutoCRUDSoak(t, eng)
}
