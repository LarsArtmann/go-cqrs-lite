package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
)

// Store is the networked queue.Store over a PostgreSQL pool.
type Store[T any] struct {
	pool  *pgxpool.Pool
	codec queue.Codec[T]
	// ownsPool is true only for pools the store opened itself (Open). A
	// pool handed in via OpenWithPool stays caller-owned: Close must not
	// tear down a pool the caller still uses.
	ownsPool bool
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

// Open connects to dsn (e.g. "postgres://user:pass@host:5432/db"),
// applies the schema, and returns a ready store. maxConns bounds the
// pool (0 = pgx default).
func Open[T any](ctx context.Context, dsn string, maxConns int32, opts ...StoreOption[T]) (*Store[T], error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("queue/postgres: parse dsn: %w", err)
	}

	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("queue/postgres: connect: %w", err)
	}

	store, err := wrapPool(ctx, pool, opts...)
	if err != nil {
		pool.Close()

		return nil, err
	}

	store.ownsPool = true

	return store, nil
}

// OpenWithPool wraps a caller-owned pool into a ready store, applying
// the schema on it. The caller keeps pool ownership: Store.Close does
// NOT close a caller-owned pool — shut it down yourself once the whole
// application is done with it.
func OpenWithPool[T any](ctx context.Context, pool *pgxpool.Pool, opts ...StoreOption[T]) (*Store[T], error) {
	if pool == nil {
		return nil, errors.New("queue/postgres: nil pool")
	}

	return wrapPool(ctx, pool, opts...)
}

// wrapPool applies the schema and options to a pool.
func wrapPool[T any](ctx context.Context, pool *pgxpool.Pool, opts ...StoreOption[T]) (*Store[T], error) {
	options := storeOptions[T]{codec: queue.JSONCodec[T]()}
	for _, opt := range opts {
		opt(&options)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		return nil, fmt.Errorf("queue/postgres: migrate: %w", err)
	}

	return &Store[T]{pool: pool, codec: options.codec}, nil
}

// Close releases the store's resources. A pool the store opened via
// Open is closed; a caller-owned pool passed to OpenWithPool is left
// running — its lifecycle belongs to the caller.
func (s *Store[T]) Close() error {
	if s.ownsPool {
		s.pool.Close()
	}

	return nil
}

// withTx runs fn in one transaction; ANY error rolls back.
func (s *Store[T]) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)

		return err
	}

	return tx.Commit(ctx)
}

// rower is the shared query surface of the pool and a tx.
type rower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// encodePayload serializes a payload through the store's codec.
func (s *Store[T]) encodePayload(v T) (string, error) {
	b, err := s.codec.Encode(v)
	if err != nil {
		return "", fmt.Errorf("queue/postgres: encode payload: %w", err)
	}

	return string(b), nil
}
