// Package sqlstore provides a SQL-backed [idempotency.Store] for multi-process
// deduplication. It works with any database/sql driver (SQLite, PostgreSQL, MySQL).
//
// Unlike [idempotency.MemoryStore] (single-process), a SQL store enables
// idempotency across multiple processes that share the same database. This is
// essential for horizontally-scaled command handlers behind a load balancer.
//
// The caller owns the *sql.DB — [Store.Close] is a no-op. Create the table
// via [NewSQLiteStore] or [NewPostgresStore], which call CREATE TABLE IF NOT
// EXISTS automatically.
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
)

// Dialect selects SQL syntax for table creation and placeholders.
type Dialect int

const (
	// DialectSQLite uses ? placeholders and SQLite-specific DDL.
	DialectSQLite Dialect = iota
	// DialectPostgres uses $N placeholders and PostgreSQL-specific DDL.
	DialectPostgres
	// DialectMySQL uses ? placeholders and MySQL-specific DDL.
	// MySQL's ON DUPLICATE KEY UPDATE replaces ON CONFLICT; the conditional
	// update uses IF() since MySQL does not support a WHERE clause in
	// ON DUPLICATE KEY UPDATE.
	DialectMySQL
)

// queries holds pre-built SQL strings for each operation, avoiding runtime
// placeholder interpolation on the hot path.
type queries struct {
	ddl            string
	seen           string
	deleteKey      string
	record         string
	checkAndRecord string
	sweep          string
}

func sqliteQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS idempotency_keys (
		key        TEXT PRIMARY KEY,
		expires_at INTEGER NOT NULL
	);`,
		seen:           `SELECT expires_at FROM idempotency_keys WHERE key = ? LIMIT 1`,
		deleteKey:      `DELETE FROM idempotency_keys WHERE key = ?`,
		record:         `INSERT INTO idempotency_keys (key, expires_at) VALUES (?, ?) ON CONFLICT(key) DO NOTHING`,
		checkAndRecord: `INSERT INTO idempotency_keys (key, expires_at) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET expires_at = excluded.expires_at WHERE idempotency_keys.expires_at < ?`,
		sweep:          `DELETE FROM idempotency_keys WHERE expires_at < ?`,
	}
}

func postgresQueries() queries {
	return queries{
		ddl: `CREATE TABLE IF NOT EXISTS idempotency_keys (
		key        TEXT PRIMARY KEY,
		expires_at BIGINT NOT NULL
	);`,
		seen:           `SELECT expires_at FROM idempotency_keys WHERE key = $1 LIMIT 1`,
		deleteKey:      `DELETE FROM idempotency_keys WHERE key = $1`,
		record:         `INSERT INTO idempotency_keys (key, expires_at) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`,
		checkAndRecord: `INSERT INTO idempotency_keys (key, expires_at) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET expires_at = EXCLUDED.expires_at WHERE idempotency_keys.expires_at < $3`,
		sweep:          `DELETE FROM idempotency_keys WHERE expires_at < $1`,
	}
}

// Store is a SQL-backed [idempotency.Store]. It stores expiry timestamps as
// UnixNano integers. Expired entries are cleaned lazily on read and via the
// optional [Store.Sweep] method.
//
// The caller owns the *sql.DB; [Store.Close] is a no-op.
type Store struct {
	db      *sql.DB
	dialect Dialect
	q       queries
}

// NewSQLiteStore creates a SQLite-backed idempotency store and creates the
// schema if it does not exist. The caller retains ownership of db.
func NewSQLiteStore(ctx context.Context, database *sql.DB) (*Store, error) {
	q := sqliteQueries()
	if _, err := database.ExecContext(ctx, q.ddl); err != nil {
		return nil, fmt.Errorf("sqlstore: create table: %w", err)
	}

	return &Store{db: database, dialect: DialectSQLite, q: q}, nil
}

// NewPostgresStore creates a PostgreSQL-backed idempotency store and creates
// the schema if it does not exist. The caller retains ownership of db.
func NewPostgresStore(ctx context.Context, database *sql.DB) (*Store, error) {
	q := postgresQueries()
	if _, err := database.ExecContext(ctx, q.ddl); err != nil {
		return nil, fmt.Errorf("sqlstore: create table: %w", err)
	}

	return &Store{db: database, dialect: DialectPostgres, q: q}, nil
}

// Seen reports whether the key is currently recorded and not expired.
// Expired entries are lazily deleted.
func (s *Store) Seen(ctx context.Context, key string) (bool, error) {
	var expiresAt int64

	err := s.db.QueryRowContext(ctx, s.q.seen, key).Scan(&expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.sql.seen", "key %q", key,
		)
	}

	if time.Now().UnixNano() >= expiresAt {
		_, _ = s.db.ExecContext(ctx, s.q.deleteKey, key)

		return false, nil
	}

	return true, nil
}

// Record marks the key as seen with the given TTL. If the key is already
// recorded and not expired, it is a no-op (the existing expiry is not extended).
// An expired key is reclaimed lazily by a subsequent Seen or Sweep call; until
// then Record on an expired-but-present row is also a no-op (INSERT ... ON
// CONFLICT DO NOTHING), so the stale expiry is NOT refreshed.
func (s *Store) Record(ctx context.Context, key string, ttl time.Duration) error {
	expiry := time.Now().Add(ttl).UnixNano()

	_, err := s.db.ExecContext(ctx, s.q.record, key, expiry)
	if err != nil {
		return errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.sql.record", "key %q", key,
		)
	}

	return nil
}

// CheckAndRecord atomically claims a key. Returns [idempotency.ErrDuplicate]
// if the key was already recorded and not expired.
//
// Atomicity is guaranteed by a single INSERT ... ON CONFLICT DO UPDATE WHERE
// statement: if the key exists but the row's expires_at is in the past, the
// UPDATE overwrites it. Both SQLite and PostgreSQL evaluate the WHERE clause
// within the same statement, so concurrent callers are serialized at the row
// level by the database engine.
func (s *Store) CheckAndRecord(ctx context.Context, key string, ttl time.Duration) error {
	newExpiry := time.Now().Add(ttl).UnixNano()
	now := time.Now().UnixNano()

	result, err := s.db.ExecContext(ctx, s.q.checkAndRecord, key, newExpiry, now)
	if err != nil {
		return errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.sql.check_and_record", "key %q", key,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.sql.rows_affected", "key %q", key,
		)
	}

	if affected > 0 {
		return nil
	}

	return idempotency.ErrDuplicate
}

// Sweep deletes all expired entries. Call periodically to bound table growth,
// or rely on lazy deletion in [Store.Seen] and [Store.CheckAndRecord].
func (s *Store) Sweep(ctx context.Context) (int64, error) {
	now := time.Now().UnixNano()

	result, err := s.db.ExecContext(ctx, s.q.sweep, now)
	if err != nil {
		return 0, errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.sql.sweep", "",
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errorfamily.Wrapf(
			err, errorfamily.Transient, "idempotency.sql.sweep_rows", "",
		)
	}

	return affected, nil
}

// Close is a no-op. The caller owns the *sql.DB.
func (s *Store) Close() {}
