package sqliteengine_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestSQLitePlannedOpsMatrix runs the D3 planned-ops parity matrix on the
// SQLite engine (in-memory; no server required).
func TestSQLitePlannedOpsMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunPlannedOpsMatrix(t, []adttest.Factory{
		{
			Name: "sqlite",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				db, err := sql.Open("sqlite", ":memory:")
				if err != nil {
					t.Fatalf("open sqlite: %v", err)
				}
				db.SetMaxOpenConns(1)

				eng, err := sqliteengine.NewSQLiteEngine(db)
				if err != nil {
					t.Fatalf("NewSQLiteEngine: %v", err)
				}

				return eng
			},
		},
	})
}
