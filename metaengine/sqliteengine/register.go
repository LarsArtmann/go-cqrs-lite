package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // SQLite driver registered with database/sql

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func init() {
	metaengine.RegisterDriver(
		"sqlite",
		func(ctx context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			dsn := cfg.DSN
			if dsn == "" {
				dsn = ":memory:"
			}

			db, err := sql.Open("sqlite", dsn)
			if err != nil {
				return nil, fmt.Errorf("sqliteengine: open %q: %w", dsn, err)
			}

			db.SetMaxOpenConns(1)

			for _, pragma := range cfg.Pragmas {
				if _, err := db.ExecContext(ctx, "PRAGMA "+pragma); err != nil {
					_ = db.Close()

					return nil, fmt.Errorf("sqliteengine: pragma %q: %w", pragma, err)
				}
			}

			return NewSQLiteEngine(db) //nolint:contextcheck,wrapcheck // takes *sql.DB
		},
	)
}
