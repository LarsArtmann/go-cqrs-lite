package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	_ "modernc.org/sqlite" // pure-Go SQLite driver (CGo-free)
)

// Store is the embedded, durable queue.Store over one SQLite file.
type Store[T any] struct {
	db    *sql.DB
	codec queue.Codec[T]
}

// Compile-time contract check; the conformance suite pins the semantics.
var _ queue.Store[int] = (*Store[int])(nil)

// StoreOption configures optional Store behavior.
type StoreOption[T any] func(*storeOptions[T])

type storeOptions[T any] struct {
	codec queue.Codec[T]
}

// WithCodec pins the payload serialization (default JSONCodec).
func WithCodec[T any](c queue.Codec[T]) StoreOption[T] {
	return func(o *storeOptions[T]) { o.codec = c }
}

// Open opens (creating if needed) the queue database at path. One queue
// per database file.
func Open[T any](path string, opts ...StoreOption[T]) (*Store[T], error) {
	options := storeOptions[T]{codec: queue.JSONCodec[T]()}
	for _, opt := range opts {
		opt(&options)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("queue/sqlite: open: %w", err)
	}

	// Serialize writers: one connection makes every SELECT…UPDATE sequence
	// inside a transaction atomic without relying on BEGIN IMMEDIATE tricks.
	db.SetMaxOpenConns(1)

	store := &Store[T]{db: db, codec: options.codec}

	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()

		return nil, err
	}

	return store, nil
}

// OpenDB wires an already-open *sql.DB (caller-owned: Close does not
// close it beyond the pool the database/sql machinery owns). The DB must
// carry the same single-writer discipline for the claim invariants.
func OpenDB[T any](db *sql.DB, opts ...StoreOption[T]) (*Store[T], error) {
	options := storeOptions[T]{codec: queue.JSONCodec[T]()}
	for _, opt := range opts {
		opt(&options)
	}

	store := &Store[T]{db: db, codec: options.codec}
	if err := store.migrate(context.Background()); err != nil {
		return nil, err
	}

	return store, nil
}

// migrate applies the schema (idempotent) and converges legacy
// databases: the dedup column is probed before its partial unique index
// is created, so fresh and legacy databases take the same path.
func (s *Store[T]) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("queue/sqlite: migrate: %w", err)
	}

	var dedupCol int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'dedup_key'`).Scan(&dedupCol); err != nil {
		return fmt.Errorf("queue/sqlite: migrate: check dedup_key: %w", err)
	}

	if dedupCol == 0 {
		if _, err := s.db.ExecContext(ctx,
			`ALTER TABLE tasks ADD COLUMN dedup_key TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("queue/sqlite: migrate: add dedup_key: %w", err)
		}
	}

	// Only tasks that opt into deduplication participate, so arbitrary
	// tasks without a key never collide.
	if _, err := s.db.ExecContext(ctx,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_dedup ON tasks(dedup_key) WHERE dedup_key != ''`); err != nil {
		return fmt.Errorf("queue/sqlite: migrate: dedup index: %w", err)
	}

	return nil
}

// Close releases the database connection.
func (s *Store[T]) Close() error { return s.db.Close() }

// withTx runs fn inside one transaction, rolling back on error.
func (s *Store[T]) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()

		return err
	}

	return tx.Commit()
}

// taskQuerier is the shared query surface of *sql.DB, *sql.Tx and
// *sql.Conn for the load/scan helpers.
type taskQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// encodePayload serializes a payload through the store's codec.
func (s *Store[T]) encodePayload(v T) ([]byte, error) {
	b, err := s.codec.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("queue/sqlite: encode payload: %w", err)
	}

	return b, nil
}
