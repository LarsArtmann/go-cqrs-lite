package dgraphengine

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
)

// RunInTx executes fn within a single Dgraph transaction: every write op the
// engine performs while fn runs is applied atomically — committed together on
// success, discarded together on error. Implements [metaengine.Transactional].
//
// Concurrency: RunInTx calls are serialized (one active transaction per
// engine). Nested RunInTx is rejected — Dgraph has no nested transactions.
// Rejection detects nesting via a marker in the context RunInTx passes to fn:
// propagate fn's ctx into any nested call (standard Go practice). A nested
// call that breaks ctx propagation deadlocks on the serialization mutex
// instead of being rejected — don't do that.
//
// Reads inside fn route through the active transaction too, so read-modify-
// write cycles see their own writes.
func (e *dgraphEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Value(txMarker{}) != nil {
		return errors.New("dgraphengine.RunInTx: nested transactions are not supported")
	}

	e.txMu.Lock()
	defer e.txMu.Unlock()

	txn := e.client.NewTxn()
	e.activeTxn.Store(txn)

	fnErr := fn(context.WithValue(ctx, txMarker{}, txActive{}))

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

// txMarker keys the context value that marks a RunInTx-managed context.
type txMarker struct{}

// txActive is the marker value stored under txMarker.
type txActive struct{}

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
	var resp *api.Response

	err := e.retryOnContention(ctx, true, func() error {
		var err error

		if tx := e.activeTxn.Load(); tx != nil {
			req.CommitNow = false

			resp, err = tx.Do(ctx, req)

			return err
		}

		req.CommitNow = true

		resp, err = e.client.NewTxn().Do(ctx, req)

		return err
	})

	return resp, err
}

// doMutate executes one standalone-shaped mutation under the same rules as
// doWrite. Callers never consume the mutation response, so only the error
// is returned.
func (e *dgraphEngine) doMutate(ctx context.Context, mut *api.Mutation) error {
	return e.retryOnContention(ctx, true, func() error {
		var err error

		if tx := e.activeTxn.Load(); tx != nil {
			mut.CommitNow = false

			_, err = tx.Mutate(ctx, mut)

			return err
		}

		mut.CommitNow = true

		_, err = e.client.NewTxn().Mutate(ctx, mut)

		return err
	})
}

// Dgraph contention retry schedule. Bulk writers (corpus builds, projection
// catch-up) sustain contention for seconds, so the schedule is 6 attempts
// with exponential backoff plus jitter (15ms base doubling, capped at 240ms).
const (
	contentionAttempts = 6
	contentionBase     = 15 * time.Millisecond
	contentionCap      = 240 * time.Millisecond
)

// retryOnContention runs fn, retrying while Dgraph reports a transient
// contention error: a transaction abort ("Transaction has been aborted.
// Please retry") from a concurrent committer, or an Alter rejected while
// transactions are pending ("Pending transactions found"). Retrying the
// whole operation is Dgraph's documented resolution — aborted work never
// committed, and schema applies are idempotent.
//
// txnScoped marks transaction operations: inside RunInTx an aborted txn
// cannot be retried in place — the whole transaction must roll back and
// the CALLER retries — so the first error surfaces immediately. Alter
// callers pass false (schema applies are always retriable).
func (e *dgraphEngine) retryOnContention(
	ctx context.Context,
	txnScoped bool,
	fn func() error,
) error {
	var lastErr error

	for attempt := range contentionAttempts {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if txnScoped && e.inTx() {
			return err
		}

		if !isContentionError(err) {
			return err
		}

		delay := min(contentionBase<<attempt, contentionCap) + rand.N(contentionBase)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// isContentionError reports whether err is Dgraph's transient contention
// class: an aborted read-write transaction or an Alter rejected because
// transactions are still pending.
func isContentionError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "aborted") ||
		strings.Contains(msg, "Pending transactions found")
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
