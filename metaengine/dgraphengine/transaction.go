package dgraphengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
)

// RunInTx executes fn within a single Dgraph transaction: every write op the
// engine performs while fn runs is applied atomically — committed together on
// success, discarded together on error. Implements [metaengine.Transactional].
//
// Concurrency: RunInTx calls are serialized (one active transaction per
// engine). Nested RunInTx is rejected — Dgraph has no nested transactions.
//
// Reads inside fn route through the active transaction too, so read-modify-
// write cycles see their own writes.
func (e *dgraphEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	e.txMu.Lock()
	defer e.txMu.Unlock()

	if e.activeTxn.Load() != nil {
		return errors.New("dgraphengine.RunInTx: nested transactions are not supported")
	}

	txn := e.client.NewTxn()
	e.activeTxn.Store(txn)

	fnErr := fn(ctx)

	e.activeTxn.Store(nil)

	if fnErr != nil {
		_ = txn.Discard(ctx)

		return fnErr
	}

	if err := txn.Commit(ctx); err != nil {
		// A conflicting commit leaves the txn aborted; Discard is a no-op
		// then, but keeps the client-side state clean on other errors.
		_ = txn.Discard(ctx)

		return fmt.Errorf("dgraphengine.RunInTx: commit: %w", err)
	}

	return nil
}

// writeTx returns the transaction a write op must use: the active RunInTx
// transaction, or nil meaning "own standalone txn" (the op commits itself
// via CommitNow).
func (e *dgraphEngine) writeTx() *dgo.Txn {
	return e.activeTxn.Load()
}

// inTx reports whether an outer RunInTx transaction is active.
func (e *dgraphEngine) inTx() bool {
	return e.activeTxn.Load() != nil
}

// doWrite executes one write request. Standalone: a fresh txn with CommitNow
// (the historical single-op behavior). In-tx: the request joins the shared
// transaction — CommitNow must be cleared (it would end the whole txn) and
// the commit is deferred to RunInTx.
func (e *dgraphEngine) doWrite(ctx context.Context, req *api.Request) (*api.Response, error) {
	if tx := e.activeTxn.Load(); tx != nil {
		req.CommitNow = false

		return tx.Do(ctx, req)
	}

	req.CommitNow = true

	return e.client.NewTxn().Do(ctx, req)
}

// doMutate executes one standalone-shaped mutation under the same rules as
// doWrite.
func (e *dgraphEngine) doMutate(ctx context.Context, mut *api.Mutation) (*api.Response, error) {
	if tx := e.activeTxn.Load(); tx != nil {
		mut.CommitNow = false

		return tx.Mutate(ctx, mut)
	}

	mut.CommitNow = true

	return e.client.NewTxn().Mutate(ctx, mut)
}

// readTx returns the transaction a read op must use: the active RunInTx
// transaction (read-your-writes), or a fresh read-only txn (cheaper —
// bypasses RAFT for reads outside transactions).
func (e *dgraphEngine) readTx() *dgo.Txn {
	if tx := e.activeTxn.Load(); tx != nil {
		return tx
	}

	return e.client.NewReadOnlyTxn()
}
