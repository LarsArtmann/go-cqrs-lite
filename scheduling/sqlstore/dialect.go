package sqlstore

import "errors"

// ErrUnknownDialect is returned when an unsupported [Dialect] is passed to a
// constructor.
var ErrUnknownDialect = errors.New("sqlstore: unknown dialect")

// Dialect selects SQL syntax for table creation and placeholders.
// Intentional duplicate: see idempotency/sqlstore/store.go. Values MUST match.
// art-dupl:accept intentional cross-module duplicate — separate go.mod, values MUST match
type Dialect int

const (
	// DialectSQLite uses ? placeholders and stores timestamps as RFC3339 text.
	DialectSQLite Dialect = iota
	// DialectPostgres uses $N placeholders and native TIMESTAMP WITH TIME ZONE.
	DialectPostgres
	// DialectMySQL uses ? placeholders and native DATETIME(3).
	DialectMySQL
)

// sqliteTimeFormat is a fixed-width RFC3339 variant that always emits 9
// fractional digits so lexicographic comparison matches chronological order.
const sqliteTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

type queries struct {
	ddl        string
	schedule   string
	due        string
	deleteByID string
}

func sqliteQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS timers (
	id         TEXT PRIMARY KEY,
	fire_at    TEXT NOT NULL,
	payload    BLOB NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`,
		schedule:   `INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		due:        `SELECT id, fire_at, payload FROM timers WHERE fire_at <= ? ORDER BY fire_at ASC`,
		deleteByID: `DELETE FROM timers WHERE id = ?`,
	}
}

func postgresQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS timers (
	id         TEXT PRIMARY KEY,
	fire_at    TIMESTAMP WITH TIME ZONE NOT NULL,
	payload    BYTEA NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_timers_fire_at ON timers(fire_at);`,
		schedule:   `INSERT INTO timers (id, fire_at, payload) VALUES ($1, $2, $3) ON CONFLICT(id) DO NOTHING`,
		due:        `SELECT id, fire_at, payload FROM timers WHERE fire_at <= $1 ORDER BY fire_at ASC`,
		deleteByID: `DELETE FROM timers WHERE id = $1`,
	}
}

func mysqlQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS timers (
	id         VARCHAR(255) PRIMARY KEY,
	fire_at    DATETIME(3) NOT NULL,
	payload    BLOB NOT NULL,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);`,
		schedule: "INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?) " +
			"ON DUPLICATE KEY UPDATE id = id",
		due:        `SELECT id, fire_at, payload FROM timers WHERE fire_at <= ? ORDER BY fire_at ASC`,
		deleteByID: `DELETE FROM timers WHERE id = ?`,
	}
}
