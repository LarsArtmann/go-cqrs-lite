// Package tursoengine provides a Turso (libSQL)-backed metaengine Engine.
//
// Turso is a fork of SQLite (libSQL) that supports local embedded databases,
// remote replication, and edge deployment. The SQL dialect is identical to
// SQLite, so this engine delegates to [sqliteengine.NewSQLiteEngine] for all
// storage operations. The only difference is the database driver: tursogo
// registers as "turso" with database/sql.
//
// This module exists OUTSIDE the zero-dependency metaengine core (ADR-0062)
// because it requires the turso.tech/database/tursogo dependency.
//
// DSN mapping:
//   - Empty DSN → ":memory:" (local in-memory libSQL database)
//   - File path → local embedded libSQL database (e.g. "/data/app.db")
//   - URL → remote Turso server (e.g. "libsql://my-db.turso.io")
//
// Sync configuration is handled at the connection level via the DSN. The
// metaengine Engine interface does not expose sync operations — they are
// managed by the operator through the Turso CLI or API.
package tursoengine

import (
	"context"
	"database/sql"
	"fmt"

	_ "turso.tech/database/tursogo" // registers "turso" driver with database/sql

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// New creates a Turso-backed metaengine Engine from a DSN. The DSN must be a
// valid Turso/libSQL connection string (file path, ":memory:", or remote URL).
// Empty DSN defaults to ":memory:".
func New(dsn string) (metaengine.Engine, error) {
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("turso", dsn)
	if err != nil {
		return nil, fmt.Errorf("tursoengine: open %q: %w", dsn, err)
	}

	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db) //nolint:contextcheck,wrapcheck // takes *sql.DB
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("tursoengine: init: %w", err)
	}

	return eng, nil
}

func init() {
	metaengine.RegisterDriver(
		"turso",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			return New(cfg.DSN) //nolint:contextcheck // constructor doesn't take ctx
		},
	)
}
