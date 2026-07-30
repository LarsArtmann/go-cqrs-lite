package readmodel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
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
		return nil, errorfamily.WrapInfrastructure(err, "kv_sql.create_handle",
			"create DB handle for KV store")
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
func (s *SQLKVStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	q := fmt.Sprintf("SELECT value FROM %s WHERE key = %s", kvTableName, s.Dialect.Placeholder(1))

	var value []byte

	err := s.DB.QueryRowContext(ctx, q, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, kv.ErrNotFound
	}

	if err != nil {
		return nil, errorfamily.WrapTransient(err, "kv_sql.get",
			"get value from KV store")
	}

	return value, nil
}

// Has reports whether a row exists for key.
func (s *SQLKVStore) Has(ctx context.Context, key []byte) (bool, error) {
	q := fmt.Sprintf("SELECT 1 FROM %s WHERE key = %s", kvTableName, s.Dialect.Placeholder(1))

	var one int

	err := s.DB.QueryRowContext(ctx, q, key).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, errorfamily.WrapTransient(err, "kv_sql.has",
			"check existence in KV store")
	}

	return true, nil
}

// Set upserts key/value atomically.
func (s *SQLKVStore) Set(ctx context.Context, key, value []byte) error {
	_, err := s.DB.ExecContext(ctx, s.upsertSQL(), key, value)
	return wrapTransientOrOK(err, "kv_sql.set", "set value in KV store")
}

// Delete removes key. Deleting a missing key is a no-op.
func (s *SQLKVStore) Delete(ctx context.Context, key []byte) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE key = %s", kvTableName, s.Dialect.Placeholder(1))

	_, err := s.DB.ExecContext(ctx, q, key)
	return wrapTransientOrOK(err, "kv_sql.delete", "delete key from KV store")
}

// NewIterator returns an iterator over keys matching prefix in lexicographic
// order. A nil/empty prefix iterates over every key.
func (s *SQLKVStore) NewIterator(ctx context.Context, prefix []byte) (kv.Iterator, error) {
	query, args := s.iterQuery(prefix)

	rows, err := s.DB.QueryContext(ctx, query, args...) //nolint:rowserrcheck
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "kv_sql.iterator",
			"create KV iterator")
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
//
//cqrs-lint:ignore(C001) tx is returned via kv.Batch interface; Commit happens in sqlKVBatch.Commit
func (s *SQLKVStore) Batch(ctx context.Context) (kv.Batch, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "kv_sql.begin_batch",
			"begin KV batch transaction")
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
		it.err = errorfamily.WrapCorruption(err, "kv_sql.scan",
			"scan KV iterator row")

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

	return wrapInfraOrOK(err, "kv_sql.close_iterator", "close KV iterator")
}

type sqlKVBatch struct {
	store  *SQLKVStore
	tx     *sql.Tx
	closed bool
}

func (b *sqlKVBatch) Set(ctx context.Context, key, value []byte) error {
	_, err := b.tx.ExecContext(ctx, b.store.upsertSQL(), key, value)
	return wrapTransientOrOK(err, "kv_sql.batch_set", "batch set in KV store")
}

func (b *sqlKVBatch) Delete(ctx context.Context, key []byte) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE key = %s", kvTableName, b.store.Dialect.Placeholder(1))

	_, err := b.tx.ExecContext(ctx, q, key)
	return wrapTransientOrOK(err, "kv_sql.batch_delete", "batch delete in KV store")
}

func (b *sqlKVBatch) Commit(ctx context.Context) error {
	//cqrs-lint:ignore(C022) library code or intentional pattern
	_ = ctx // retained for kv.Batch interface; tx.Commit is non-context-aware

	if b.closed {
		return nil
	}

	err := b.tx.Commit()
	b.closed = true

	return wrapInfraOrOK(err, "kv_sql.batch_commit", "commit KV batch")
}

func (b *sqlKVBatch) Close() error {
	if b.closed {
		return nil
	}

	b.closed = true

	err := b.tx.Rollback()
	return wrapInfraOrOK(err, "kv_sql.batch_rollback", "rollback KV batch on close")
}

// wrapTransientOrOK returns nil when err is nil, otherwise wraps err as a
// transient error. Collapses the repeated "if err != nil { return WrapTransient(...) }; return nil"
// boilerplate — the unique code stays a parameter.
func wrapTransientOrOK(err error, code, msg string) error {
	if err == nil {
		return nil
	}

	return errorfamily.WrapTransient(err, code, msg)
}

// wrapInfraOrOK returns nil when err is nil, otherwise wraps err as an
// infrastructure error.
func wrapInfraOrOK(err error, code, msg string) error {
	if err == nil {
		return nil
	}

	return errorfamily.WrapInfrastructure(err, code, msg)
}

var (
	_ kv.Store    = (*SQLKVStore)(nil)
	_ kv.Iterator = (*sqlKVIterator)(nil)
	_ kv.Batch    = (*sqlKVBatch)(nil)
)
