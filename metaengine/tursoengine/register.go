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
// probing is wired through sqliteengine.SetProber: the probe function times a
// db.PingContext round-trip to the remote libSQL server, and ProbeEngine's
// IsRemote guard ensures it only runs for remote configurations.
//
// Sync configuration is handled at the connection level via the DSN. The
// metaengine Engine interface does not expose sync operations — they are
// managed by the operator through the Turso CLI or API.
package tursoengine

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
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

// redactDSN strips credentials from a DSN so connection errors never leak
// secrets into logs. It removes URL userinfo (libsql://token@host) and the
// authToken query parameter; non-URL DSNs (file paths, :memory:) pass through
// unchanged. Redaction is best-effort: an unparseable URL is replaced with a
// fixed placeholder rather than risked in an error message.
func redactDSN(dsn string) string {
	if !isRemoteDSN(dsn) {
		return dsn
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "libsql://[redacted]"
	}

	if u.User != nil {
		u.User = url.User("redacted")
	}

	if u.RawQuery != "" {
		q := u.Query()
		for key := range q {
			if strings.EqualFold(key, "authtoken") ||
				strings.EqualFold(key, "token") ||
				strings.EqualFold(key, "apikey") {
				q.Set(key, "[redacted]")
			}
		}
		u.RawQuery = q.Encode()
	}

	return u.String()
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
		return nil, fmt.Errorf("tursoengine: open %q: %w", redactDSN(dsn), err)
	}

	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db) //nolint:contextcheck,wrapcheck // takes *sql.DB
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("tursoengine: init: %w", err)
	}

	// Nobody outside this function holds the *sql.DB handle, so the engine
	// must own (and on Close, close) the connection.
	sqliteengine.OwnDB(eng)

	if isRemoteDSN(dsn) {
		if cal, ok := eng.(metaengine.Calibratable); ok {
			cal.SetCalibration(metaengine.CalibrationCosts{
				NetworkRTT: Turso_NetworkRTT,
			})
		}

		// Inject a live probe function so ProbeEngine can measure real RTT.
		// The probe times a PingContext round-trip to the remote libSQL server.
		if prober, ok := eng.(sqliteengine.ProberSetter); ok {
			prober.SetProber(func(ctx context.Context) (time.Duration, error) {
				start := time.Now()
				if err := db.PingContext(ctx); err != nil {
					return 0, err
				}

				return time.Since(start), nil
			})
		}
	}

	return eng, nil
}

func init() {
	metaengine.RegisterDriver(
		"turso",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			if err := metaengine.RejectDurabilityTier("turso", cfg); err != nil {
				return nil, err
			}

			return New(cfg.DSN) //nolint:contextcheck // constructor doesn't take ctx
		},
	)
}
