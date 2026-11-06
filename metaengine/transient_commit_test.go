package metaengine

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// markedAbort is a minimal TransientCommitError for tests.
type markedAbort struct{ err error }

func (e markedAbort) Error() string    { return e.err.Error() }
func (e markedAbort) Unwrap() error    { return e.err }
func (e markedAbort) TransientCommit() {}

func TestIsTransientCommit(t *testing.T) {
	t.Parallel()

	plain := errors.New("connection refused")
	abort := markedAbort{err: errors.New("Transaction has been aborted. Please retry")}
	wrapped := fmt.Errorf("batch apply on dgraph: %w", abort)

	if IsTransientCommit(nil) {
		t.Error("nil error must not be a transient commit")
	}

	if IsTransientCommit(plain) {
		t.Error("plain error must not be a transient commit")
	}

	if !IsTransientCommit(abort) {
		t.Error("marked abort must be a transient commit")
	}

	if !IsTransientCommit(wrapped) {
		t.Error("wrapped marked abort must be a transient commit (errors.As traversal)")
	}
}

// fakeTransactional counts RunInTx calls and answers with a scripted error
// sequence.
type fakeTransactional struct {
	errs []error // one per call; nil = success; past the end: keep last
}

func (f *fakeTransactional) RunInTx(_ context.Context, _ func(context.Context) error) error {
	if len(f.errs) == 0 {
		return nil
	}

	err := f.errs[0]
	f.errs = f.errs[1:]

	return err
}

func TestRunInTxWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("retries transient aborts until success", func(t *testing.T) {
		t.Parallel()

		tx := &fakeTransactional{errs: []error{
			markedAbort{err: errors.New("aborted 1")},
			markedAbort{err: errors.New("aborted 2")},
		}}

		if err := runInTxWithRetry(context.Background(), tx, func(context.Context) error { return nil }); err != nil {
			t.Fatalf("want success after retries, got %v", err)
		}

		if len(tx.errs) != 0 {
			t.Fatalf("want all scripted errors consumed, %d left", len(tx.errs))
		}
	})

	t.Run("non-transient error surfaces after one attempt", func(t *testing.T) {
		t.Parallel()

		plain := errors.New("schema mismatch")

		tx := &fakeTransactional{errs: []error{plain}}

		err := runInTxWithRetry(context.Background(), tx, func(context.Context) error { return nil })
		if !errors.Is(err, plain) {
			t.Fatalf("want the plain error unchanged, got %v", err)
		}

		if len(tx.errs) != 0 {
			t.Fatal("non-transient errors must not be retried")
		}
	})

	t.Run("persistent aborts exhaust the budget and surface", func(t *testing.T) {
		t.Parallel()

		always := markedAbort{err: errors.New("Transaction has been aborted. Please retry")}

		tx := &fakeTransactional{errs: []error{always, always, always, always, always, always}}

		err := runInTxWithRetry(context.Background(), tx, func(context.Context) error { return nil })
		if !IsTransientCommit(err) {
			t.Fatalf("want the abort to surface after exhaustion, got %v", err)
		}

		if len(tx.errs) != 2 {
			t.Fatalf("want exactly %d attempts, %d calls left over", txCommitAttempts, len(tx.errs)-2)
		}
	})
}
