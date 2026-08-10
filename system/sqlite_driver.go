package system

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // SQLite driver registered with database/sql

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// createSQLiteEngine creates a SQLite metaengine from a DriverConfig.
// This lives in system/ (not sqliteengine/) because it needs database/sql
// and modernc.org/sqlite, which are system/ deps.
// In v5 Phase 4, this will move to sqliteengine/register.go.
func createSQLiteEngine(ctx context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("system: open sqlite %q: %w", dsn, err)
	}

	db.SetMaxOpenConns(1)

	for _, pragma := range cfg.Pragmas {
		if _, err := db.ExecContext(ctx, "PRAGMA "+pragma); err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("system: sqlite pragma %q: %w", pragma, err)
		}
	}

	return sqliteengine.NewSQLiteEngine(db) //nolint:wrapcheck,contextcheck // takes *sql.DB
}
