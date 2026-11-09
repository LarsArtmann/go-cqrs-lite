package metaengine

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// TransientCommitError is implemented by errors returned from
// [Transactional.RunInTx] when the commit failed transiently with NOTHING
// committed (the conflict-abort class): the transaction rolled back
// completely, so re-running the whole RunInTx call — closure included — is
// safe. Engines whose commits can conflict-abort (Dgraph) mark their commit
// errors with it; callers that cannot tolerate the abort answer with
// IsTransientCommit and retry.
type TransientCommitError interface {
	error

	// TransientCommit marks the error: the transaction was aborted BEFORE
	// any write became visible.
	TransientCommit()
}

// IsTransientCommit reports whether err (or anything it wraps) is a
// transient, nothing-committed transaction abort that the caller may retry
// as a whole.
func IsTransientCommit(err error) bool {
	_, ok := errors.AsType[TransientCommitError](err)

	return ok
}

// txCommitRetry bounds whole-transaction retries on transient commit
// aborts, with jittered backoff. Conflict storms (bulk writers racing a
// projection apply) clear within a few hundred milliseconds.
const (
	txCommitAttempts = 4
	txCommitBase     = 15 * time.Millisecond
)

// runInTxWithRetry runs tx.RunInTx, retrying the WHOLE call — closure
// included — while the commit fails as a transient, nothing-committed
// abort. Any other error (including aborts that persist past the budget)
// surfaces unchanged. The closure must only perform engine writes: that is
// the RunInTx contract, and it is what makes re-running it safe.
func runInTxWithRetry(ctx context.Context, tx Transactional, fn func(context.Context) error) error {
	var err error

	for attempt := range txCommitAttempts {
		err = tx.RunInTx(ctx, fn)
		if err == nil || !IsTransientCommit(err) {
			return err
		}

		delay := txCommitBase<<attempt + rand.N(txCommitBase)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return err
}
