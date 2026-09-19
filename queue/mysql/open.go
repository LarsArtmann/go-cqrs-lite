package mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // MySQL driver (CGo-free)
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
)

// Store is the networked queue.Store over one MySQL database.
type Store[T any] struct {
	db    *sql.DB
	codec queue.Codec[T]
	// ownsDB is true only for pools the store opened itself (Open). A
	// pool handed in via OpenDB stays caller-owned: Close must not tear
	// down a pool the caller still uses.
	ownsDB bool
}

// Compile-time contract checks; the conformance suite pins the semantics.
// art-dupl:accept engine scaffolding twin of queue/sqlite+postgres open.go; dep-isolated modules, conformance pins semantics
var (
	_ queue.Store[int] = (*Store[int])(nil)
	_ queue.FactTx     = (*Store[int])(nil)
)

// StoreOption configures optional Store behavior.
type StoreOption[T any] func(*storeOptions[T])

type storeOptions[T any] struct {
	codec queue.Codec[T]
}

// WithCodec pins the payload serialization (default JSONCodec).
func WithCodec[T any](c queue.Codec[T]) StoreOption[T] {
	return func(o *storeOptions[T]) { o.codec = c }
}

// Open connects to dsn (e.g.
// "user:pass@tcp(127.0.0.1:3306)/tasks?parseTime=true"), applies the
// schema, and returns a ready store. The pool is store-owned: Close
// closes it.
func Open[T any](dsn string, opts ...StoreOption[T]) (*Store[T], error) {
	options := storeOptions[T]{codec: queue.JSONCodec[T]()}
	for _, opt := range opts {
		opt(&options)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("queue/mysql: open: %w", err)
	}

	db.SetMaxOpenConns(8)

	store := &Store[T]{db: db, codec: options.codec, ownsDB: true}

	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()

		return nil, err
	}

	return store, nil
}

// OpenDB wires an already-open *sql.DB (caller-owned: Close does not
// close it). The DB must be a "mysql" driver connection to the database
// holding the queue tables.
func OpenDB[T any](db *sql.DB, opts ...StoreOption[T]) (*Store[T], error) {
	//art-dupl:accept engine scaffolding twin of queue/sqlite OpenDB; dep-isolated modules, conformance pins semantics
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

// migrate applies the schema (idempotent, one statement at a time).
func (s *Store[T]) migrate(ctx context.Context) error {
	for _, stmt := range schemaStmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("queue/mysql: migrate: %w", err)
		}
	}

	return nil
}

// Close releases the store's resources. A pool the store opened via
// Open is closed; a caller-owned pool passed to OpenDB is left running —
// its lifecycle belongs to the caller.
func (s *Store[T]) Close() error {
	if !s.ownsDB {
		return nil
	}

	//art-dupl:accept engine scaffolding twin of queue/sqlite Close; only the pool handle differs
	return s.db.Close()
}

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
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// encodePayload serializes a payload through the store's codec.
func (s *Store[T]) encodePayload(v T) ([]byte, error) {
	b, err := s.codec.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("queue/mysql: encode payload: %w", err)
	}

	return b, nil
}
