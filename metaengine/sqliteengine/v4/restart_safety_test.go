package sqliteengine_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestSQLiteRestartSafety_StreamAndJournal verifies that reopening a persistent
// SQLite database does NOT reset seq counters to zero — which would cause
// silent key collisions and data loss (see enginetest.RunRestartSafetyTest).
func TestSQLiteRestartSafety_StreamAndJournal(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyTest(t, func(path string) (metaengine.Engine, error) {
		return sqliteengine.NewSQLiteEngineFromDSN(path)
	})
}

// TestSQLiteRestartSafety_FromDB verifies seq seeding when using
// NewSQLiteEngine (caller-owned *sql.DB path).
func TestSQLiteRestartSafety_FromDB(t *testing.T) {
	t.Parallel()

	enginetest.RunRestartSafetyFromDBTest(t,
		func(dir string) (metaengine.Engine, error) {
			return sqliteengine.NewSQLiteEngineFromDSN(filepath.Join(dir, "sqlite.db"))
		},
		func(dir string) (metaengine.Engine, error) {
			db, err := sql.Open("sqlite", filepath.Join(dir, "sqlite.db"))
			if err != nil {
				return nil, fmt.Errorf("raw sqlite open: %w", err)
			}

			return sqliteengine.NewSQLiteEngine(db)
		},
	)
}
