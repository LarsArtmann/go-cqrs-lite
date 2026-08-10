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
// When the DSN is a remote URL, the engine declares a same-datacenter RTT prior
// (Turso_NetworkRTT) via calibration so the planner routes correctly. Live
// probing (ProbeEngine) requires the engine itself to implement Prober; because
// turso delegates to sqliteEngine (unexported), the standard ProbeEngine path is
// a no-op — the prior stands and is labelled "stale" by GetEngineStats until a
// future sqliteengine API allows injecting a probe function.
//
// Sync configuration is handled at the connection level via the DSN. The
// metaengine Engine interface does not expose sync operations — they are
// managed by the operator through the Turso CLI or API.
package tursoengine

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "turso.tech/database/tursogo" // registers "turso" driver with database/sql

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Turso_NetworkRTT is the declared PRIOR for round-trip time to a remote Turso
// (libSQL) server. Turso edge databases typically run in the same region, so we
// model ~2ms (slightly higher than same-datacenter PG to account for the libSQL
// sync layer). Only applied when the DSN is a remote URL.
const Turso_NetworkRTT = 2 * time.Millisecond

// isRemoteDSN returns true when the DSN points to a remote Turso server rather
// than a local file or in-memory database.
func isRemoteDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "libsql://") ||
		strings.HasPrefix(dsn, "https://") ||
		strings.HasPrefix(dsn, "http://")
}

// New creates a Turso-backed metaengine Engine from a DSN. The DSN must be a
// valid Turso/libSQL connection string (file path, ":memory:", or remote URL).
// Empty DSN defaults to ":memory:".
//
// When the DSN is a remote URL (libsql://, https://), the engine declares a
// same-datacenter RTT prior via calibration so the cost-based planner accounts
// for network latency.
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

	if isRemoteDSN(dsn) {
		if cal, ok := eng.(metaengine.Calibratable); ok {
			cal.SetCalibration(metaengine.CalibrationCosts{
				NetworkRTT: Turso_NetworkRTT,
			})
		}
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
