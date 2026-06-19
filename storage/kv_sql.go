package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

// kvTableName is the single table backing every SQLKVStore instance.
const kvTableName = "cqrs_kv"

// SQLKVStore is a [kv.Store] backed by a SQL table (cqrs_kv).
//
// It lets SQL-backed Bundle presets (SQLite, Postgres) persist read models
// without an ephemeral in-memory backend, so a read model survives a process
// restart. The store does NOT own the *sql.DB: the connection lifecycle is
// managed by the [SQLBackend] or the [stack.Bundle] that wraps it, and Close
// is a no-op (matching the borrowed-handle convention of the other SQL stores).
//
// Construct via [NewSQLiteKVStore], [NewSQLKVStore], or
// [NewSQLKVStoreWithDialect]. The cqrs_kv table must exist; the preset's
// auto-migration (SQLiteInitSchema / PostgresInitSchema) creates it.
type SQLKVStore struct {
	sqlpkg.DBHandle
}

// NewSQLKVStore creates a SQLKVStore for PostgreSQL.
func NewSQLKVStore(db *sql.DB) (*SQLKVStore, error) {
	return newSQLKVStoreWithDialect(db, sqlpkg.PostgresDialect{})
}

// NewSQLiteKVStore creates a SQLKVStore for SQLite.
func NewSQLiteKVStore(db *sql.DB) (*SQLKVStore, error) {
	return newSQLKVStoreWithDialect(db, sqlpkg.SQLiteDialect{})
}

// NewSQLKVStoreWithDialect creates a SQLKVStore with an explicit dialect.
func NewSQLKVStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLKVStore, error) {
	return newSQLKVStoreWithDialect(db, d)
}

func newSQLKVStoreWithDialect(db *sql.DB, d sqlpkg.Dialect) (*SQLKVStore, error) {
	handle, err := sqlpkg.NewDBHandle(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLKVStore{DBHandle: handle}, nil
}

func (s *SQLKVStore) upsertSQL() string {
	return fmt.Sprintf(
		"INSERT INTO %s (key, value) VALUES (%s, %s) "+
			"ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		kvTableName,
		s.Dialect.Placeholder(1),
		s.Dialect.Placeholder(2),
	)
}

// Get returns the value for key, or [kv.ErrNotFound] if no row exists.
func (s *SQLKVStore) Get(key []byte) ([]byte, error) {
	q := fmt.Sprintf("SELECT value FROM %s WHERE key = %s", kvTableName, s.Dialect.Placeholder(1))

	var value []byte

	err := s.DB.QueryRowContext(context.Background(), q, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, kv.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("kv-sql: get: %w", err)
	}

	return value, nil
}

// Has reports whether a row exists for key.
func (s *SQLKVStore) Has(key []byte) (bool, error) {
	q := fmt.Sprintf("SELECT 1 FROM %s WHERE key = %s", kvTableName, s.Dialect.Placeholder(1))

	var one int

	err := s.DB.QueryRowContext(context.Background(), q, key).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("kv-sql: has: %w", err)
	}

	return true, nil
}

// Set upserts key/value atomically.
func (s *SQLKVStore) Set(key, value []byte) error {
	_, err := s.DB.ExecContext(context.Background(), s.upsertSQL(), key, value)
	if err != nil {
		return fmt.Errorf("kv-sql: set: %w", err)
	}

	return nil
}

// Delete removes key. Deleting a missing key is a no-op.
func (s *SQLKVStore) Delete(key []byte) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE key = %s", kvTableName, s.Dialect.Placeholder(1))

	_, err := s.DB.ExecContext(context.Background(), q, key)
	if err != nil {
		return fmt.Errorf("kv-sql: delete: %w", err)
	}

	return nil
}

// NewIterator returns an iterator over keys matching prefix in lexicographic
// order. A nil/empty prefix iterates over every key.
func (s *SQLKVStore) NewIterator(prefix []byte) (kv.Iterator, error) {
	query, args := s.iterQuery(prefix)

	rows, err := s.DB.QueryContext(context.Background(), query, args...) //nolint:rowserrcheck
	if err != nil {
		return nil, fmt.Errorf("kv-sql: iterator: %w", err)
	}

	return &sqlKVIterator{rows: rows}, nil
}

func (s *SQLKVStore) iterQuery(prefix []byte) (string, []any) {
	if len(prefix) == 0 {
		return fmt.Sprintf("SELECT key, value FROM %s ORDER BY key", kvTableName), nil
	}

	end, bounded := prefixEnd(prefix)
	p1 := s.Dialect.Placeholder(1)

	if bounded {
		return fmt.Sprintf(
			"SELECT key, value FROM %s WHERE key >= %s AND key < %s ORDER BY key",
			kvTableName, p1, s.Dialect.Placeholder(2),
		), []any{prefix, end}
	}

	return fmt.Sprintf("SELECT key, value FROM %s WHERE key >= %s ORDER BY key", kvTableName, p1),
		[]any{prefix}
}

// Batch returns a batch backed by a single SQL transaction.
func (s *SQLKVStore) Batch() (kv.Batch, error) {
	tx, err := s.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("kv-sql: begin batch: %w", err)
	}

	return &sqlKVBatch{store: s, tx: tx}, nil
}

// prefixEnd returns the smallest key strictly greater than every key that
// starts with prefix (the exclusive upper bound of the prefix range). The bool
// is false when no such bound exists (prefix is all 0xFF bytes → unbounded).
func prefixEnd(prefix []byte) ([]byte, bool) {
	end := append([]byte{}, prefix...)

	for i := range slices.Backward(end) {
		if end[i] < 0xFF {
			end[i]++

			return end[:i+1], true
		}
	}

	return nil, false
}

type sqlKVIterator struct {
	rows  *sql.Rows
	key   []byte
	value []byte
	done  bool
	err   error
}

func (it *sqlKVIterator) Next() bool {
	if it.done {
		return false
	}

	if !it.rows.Next() {
		it.done = true
		it.err = it.rows.Err()

		return false
	}

	var key, value []byte

	err := it.rows.Scan(&key, &value)
	if err != nil {
		it.done = true
		it.err = fmt.Errorf("kv-sql: scan: %w", err)

		return false
	}

	it.key = key
	it.value = value

	return true
}

func (it *sqlKVIterator) Key() []byte   { return it.key }
func (it *sqlKVIterator) Value() []byte { return it.value }
func (it *sqlKVIterator) Error() error  { return it.err }

func (it *sqlKVIterator) Close() error {
	if it.rows == nil {
		return nil
	}

	err := it.rows.Close()
	it.rows = nil

	if err != nil {
		return fmt.Errorf("kv-sql: close iterator: %w", err)
	}

	return nil
}

type sqlKVBatch struct {
	store  *SQLKVStore
	tx     *sql.Tx
	closed bool
}

func (b *sqlKVBatch) Set(key, value []byte) error {
	_, err := b.tx.ExecContext(context.Background(), b.store.upsertSQL(), key, value)
	if err != nil {
		return fmt.Errorf("kv-sql: batch set: %w", err)
	}

	return nil
}

func (b *sqlKVBatch) Delete(key []byte) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE key = %s", kvTableName, b.store.Dialect.Placeholder(1))

	_, err := b.tx.ExecContext(context.Background(), q, key)
	if err != nil {
		return fmt.Errorf("kv-sql: batch delete: %w", err)
	}

	return nil
}

func (b *sqlKVBatch) Commit() error {
	if b.closed {
		return nil
	}

	err := b.tx.Commit()
	b.closed = true

	if err != nil {
		return fmt.Errorf("kv-sql: batch commit: %w", err)
	}

	return nil
}

func (b *sqlKVBatch) Close() error {
	if b.closed {
		return nil
	}

	b.closed = true

	return b.tx.Rollback()
}

var (
	_ kv.Store    = (*SQLKVStore)(nil)
	_ kv.Iterator = (*sqlKVIterator)(nil)
	_ kv.Batch    = (*sqlKVBatch)(nil)
)
