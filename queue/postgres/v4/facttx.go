package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
)

// Compile-time capability check: the engine provides the same-tx fact
// append seam (queue.FactTx).
var _ queue.FactTx = (*Store[int])(nil)

// WithFacts runs fn inside one transaction; every sink append commits
// with it or rolls back with fn's error — the same same-tx guarantee
// the store's own mutations enjoy (the ADR-0001 lineage).
func (s *Store[T]) WithFacts(ctx context.Context, fn func(sink queue.FactSink) error) error {
	//art-dupl:accept dialect twin of queue/sqlite WithFacts; conformance pins the atomicity split
	return s.withTx(ctx, func(tx pgx.Tx) error {
		return fn(factSink[T]{store: s, tx: tx})
	})
}

// factSink appends facts through the store's in-tx append path.
type factSink[T any] struct {
	store *Store[T]
	tx    pgx.Tx
}

// Append records one fact in the sink's transaction.
func (fs factSink[T]) Append(ctx context.Context, f facts.Fact) error {
	return fs.store.appendFact(ctx, fs.tx, f)
}
