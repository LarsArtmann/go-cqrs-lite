package claimkit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
)

// sqliteTimeFormat is a fixed-width RFC3339 variant that always emits 9
// fractional digits so lexicographic comparison matches chronological order
// (same encoding scheduling/sqlstore proved in production).
const sqliteTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

// ph renders a dialect placeholder for a 1-based position.
func ph(d claiming.Dialect, n int) string {
	switch d {
	case claiming.DialectPostgres, claiming.DialectDuckDB:
		return "$" + itoa(n)
	case claiming.DialectMySQL:
		return "?"
	default: // SQLite ordinal form
		return "?" + itoa(n)
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}

	return itoa(n/10) + string(rune('0'+n%10))
}

// encodeTime renders a time value as the dialect's storage form: fixed-width
// RFC3339 text on SQLite (lexicographically ordered), native time.Time on
// Postgres/MySQL (TIMESTAMP WITH TIME ZONE / DATETIME(3) columns).
func (c *Claims) encodeTime(t time.Time) any {
	if c.dialect == claiming.DialectSQLite {
		return t.Format(sqliteTimeFormat)
	}

	return t
}

// decodeTime normalizes a scanned timestamp column across dialects: text
// (SQLite), time.Time (Postgres/MySQL drivers), or nil (SQL NULL).
func decodeTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case nil:
		return time.Time{}, nil
	case time.Time:
		return t, nil
	case string:
		return time.Parse(time.RFC3339Nano, t)
	case []byte:
		return time.Parse(time.RFC3339Nano, string(t))
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type %T", v)
	}
}

func ensureClaimsTables(ctx context.Context, db *sql.DB, d claiming.Dialect) error {
	for _, ddl := range claimsDDL(d) {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("ensure table: %w", err)
		}
	}

	return ensureIndexes(ctx, db, d)
}

// ensureIndexes creates the claim/dedup indexes idempotently. Postgres and
// SQLite support IF NOT EXISTS; MySQL-compatible servers do not, so the
// index is probed in information_schema first (the claiming/migrate.go
// pattern).
func ensureIndexes(ctx context.Context, db *sql.DB, d claiming.Dialect) error {
	if d == claiming.DialectMySQL {
		const probe = `
SELECT COUNT(*) FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?`

		for _, idx := range []struct{ table, name, ddl string }{
			{
				"meta_due_claims", "idx_meta_due_claims_due",
				"CREATE INDEX idx_meta_due_claims_due ON meta_due_claims(collection, due_at)",
			},
			{
				"meta_claim_facts", "idx_meta_claim_facts_key",
				"CREATE INDEX idx_meta_claim_facts_key ON meta_claim_facts(collection, key, seq)",
			},
			{
				"meta_dedup", "idx_meta_dedup_expiry",
				"CREATE INDEX idx_meta_dedup_expiry ON meta_dedup(collection, expires_at)",
			},
		} {
			var count int

			err := db.QueryRowContext(ctx, probe, idx.table, idx.name).Scan(&count)
			if err != nil {
				return fmt.Errorf("probe index %s: %w", idx.name, err)
			}

			if count > 0 {
				continue
			}

			if _, err := db.ExecContext(ctx, idx.ddl); err != nil {
				return fmt.Errorf("ensure index %s: %w", idx.name, err)
			}
		}

		return nil
	}

	for _, ddl := range []string{
		`CREATE INDEX IF NOT EXISTS idx_meta_due_claims_due ON meta_due_claims(collection, due_at)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_claim_facts_key ON meta_claim_facts(collection, key, seq)`,
		`CREATE INDEX IF NOT EXISTS idx_meta_dedup_expiry ON meta_dedup(collection, expires_at)`,
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("ensure index: %w", err)
		}
	}

	return nil
}

func claimsDDL(d claiming.Dialect) []string {
	switch d {
	case claiming.DialectPostgres:
		return []string{
			`CREATE TABLE IF NOT EXISTS meta_due_claims (
	collection   TEXT NOT NULL,
	key          TEXT NOT NULL,
	due_at       TIMESTAMP WITH TIME ZONE NOT NULL,
	lease_until  TIMESTAMP WITH TIME ZONE,
	owner        TEXT NOT NULL DEFAULT '',
	payload      BYTEA NOT NULL,
	created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
	PRIMARY KEY (collection, key)
)`,
			`CREATE TABLE IF NOT EXISTS meta_claim_facts (
	seq        BIGSERIAL PRIMARY KEY,
	collection TEXT NOT NULL,
	key        TEXT NOT NULL,
	type       TEXT NOT NULL,
	payload    BYTEA,
	at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
)`,
			`CREATE TABLE IF NOT EXISTS meta_dedup (
	collection  TEXT NOT NULL,
	key         TEXT NOT NULL,
	expires_at  TIMESTAMP WITH TIME ZONE NOT NULL,
	PRIMARY KEY (collection, key)
)`,
		}
	case claiming.DialectMySQL:
		return []string{
			"CREATE TABLE IF NOT EXISTS meta_due_claims (\n" +
				"collection   VARCHAR(255) NOT NULL,\n" +
				"key          VARCHAR(255) NOT NULL,\n" +
				"due_at       DATETIME(3) NOT NULL,\n" +
				"lease_until  DATETIME(3) NULL,\n" +
				"owner        VARCHAR(255) NOT NULL DEFAULT '',\n" +
				"payload      BLOB NOT NULL,\n" +
				"created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),\n" +
				"PRIMARY KEY (collection, key)\n" +
				")",
			"CREATE TABLE IF NOT EXISTS meta_claim_facts (\n" +
				"seq        BIGINT AUTO_INCREMENT PRIMARY KEY,\n" +
				"collection VARCHAR(255) NOT NULL,\n" +
				"key        VARCHAR(255) NOT NULL,\n" +
				"type       VARCHAR(255) NOT NULL,\n" +
				"payload    BLOB,\n" +
				"at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)\n" +
				")",
			"CREATE TABLE IF NOT EXISTS meta_dedup (\n" +
				"collection  VARCHAR(255) NOT NULL,\n" +
				"key         VARCHAR(255) NOT NULL,\n" +
				"expires_at  DATETIME(3) NOT NULL,\n" +
				"PRIMARY KEY (collection, key)\n" +
				")",
		}
	case claiming.DialectDuckDB:
		// DuckDB: native TIMESTAMP (microsecond) columns, dollar placeholders,
		// and a sequence for fact positions (no AUTOINCREMENT). Claims ride
		// the numbered IN-subquery UPDATE..RETURNING shape; DuckDB's
		// single-process single-writer serialization replaces row locks.
		return []string{
			`CREATE TABLE IF NOT EXISTS meta_due_claims (
	collection   VARCHAR NOT NULL,
	key          VARCHAR NOT NULL,
	due_at       TIMESTAMP NOT NULL,
	lease_until  TIMESTAMP,
	owner        VARCHAR NOT NULL DEFAULT '',
	payload      BLOB NOT NULL,
	created_at   TIMESTAMP NOT NULL DEFAULT now(),
	PRIMARY KEY (collection, key)
)`,
	`CREATE SEQUENCE IF NOT EXISTS meta_claim_facts_seq`,
			`CREATE TABLE IF NOT EXISTS meta_claim_facts (
	seq        BIGINT PRIMARY KEY DEFAULT nextval('meta_claim_facts_seq'),
	collection VARCHAR NOT NULL,
	key        VARCHAR NOT NULL,
	type       VARCHAR NOT NULL,
	payload    BLOB,
	recorded_at TIMESTAMP NOT NULL DEFAULT now()
)`,
			`CREATE TABLE IF NOT EXISTS meta_dedup (
	collection  VARCHAR NOT NULL,
	key         VARCHAR NOT NULL,
	expires_at  TIMESTAMP NOT NULL,
	PRIMARY KEY (collection, key)
)`,
		}
	default: // SQLite
		return []string{
			`CREATE TABLE IF NOT EXISTS meta_due_claims (
	collection   TEXT NOT NULL,
	key          TEXT NOT NULL,
	due_at       TEXT NOT NULL,
	lease_until  TEXT,
	owner        TEXT NOT NULL DEFAULT '',
	payload      BLOB NOT NULL,
	created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now')),
	PRIMARY KEY (collection, key)
)`,
			`CREATE TABLE IF NOT EXISTS meta_claim_facts (
	seq        INTEGER PRIMARY KEY AUTOINCREMENT,
	collection TEXT NOT NULL,
	key        TEXT NOT NULL,
	type       TEXT NOT NULL,
	payload    BLOB,
	at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
)`,
			`CREATE TABLE IF NOT EXISTS meta_dedup (
	collection  TEXT NOT NULL,
	key         TEXT NOT NULL,
	expires_at  TEXT NOT NULL,
	PRIMARY KEY (collection, key)
)`,
		}
	}
}

// insertClaimStmt builds the idempotent claim insert per dialect:
// ON CONFLICT DO NOTHING (SQLite/Postgres) or ON DUPLICATE KEY UPDATE
// no-op (MySQL-compatible servers).
func insertClaimStmt(
	d claiming.Dialect,
	collection, key string,
	dueAt any,
	payload []byte,
) (string, []any) {
	if payload == nil {
		payload = []byte{} // NOT NULL columns keep a zero-length blob
	}

	switch d {
	case claiming.DialectPostgres, claiming.DialectDuckDB:
		return `INSERT INTO meta_due_claims (collection, key, due_at, payload)
VALUES ($1, $2, $3, $4) ON CONFLICT (collection, key) DO NOTHING`,
			[]any{collection, key, dueAt, payload}
	case claiming.DialectMySQL:
		return "INSERT INTO meta_due_claims (collection, key, due_at, payload)\n" +
				"VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE key = key",
			[]any{collection, key, dueAt, payload}
	default: // SQLite ordinal form
		return `INSERT INTO meta_due_claims (collection, key, due_at, payload)
VALUES (?1, ?2, ?3, ?4) ON CONFLICT(collection, key) DO NOTHING`,
			[]any{collection, key, dueAt, payload}
	}
}
