package adttest

import (
	"database/sql"
	"testing"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

func mustProbeEngine(t *testing.T, db *sql.DB) (e interface{ Close() error }) {
	t.Helper()
	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = eng.Close() })
	return eng
}
