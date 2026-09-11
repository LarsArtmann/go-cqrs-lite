package dgraphengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/dgraph-io/dgo/v240"
	gomega "github.com/onsi/gomega"
)

var errAborted = errors.New("Transaction has been aborted. Please retry")

var errPendingTxns = errors.New("Pending transactions found. Please retry operation")

// TestIsContentionError pins the transient-contention matcher: Dgraph aborts
// and pending-transaction Alter rejections are retriable (including when
// wrapped), everything else — including case variants Dgraph does not emit —
// is not.
func TestIsContentionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"abort verbatim", errAborted, true},
		{"pending verbatim", errPendingTxns, true},
		{"wrapped abort", fmt.Errorf("dgraphengine.map_set: %w", errAborted), true},
		{"wrapped pending", fmt.Errorf("dgraphengine.alter: %w", errPendingTxns), true},
		{"connection refused", errors.New("connection refused"), false},
		{"context canceled", context.Canceled, false},
		{"EOF", io.EOF, false},
		{"case mismatch abort", errors.New("Transaction has been Aborted. Please retry"), false},
		{"case mismatch pending", errors.New("pending transactions found"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isContentionError(tt.err); got != tt.want {
				t.Errorf("isContentionError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// newRetryTestEngine returns an engine whose inTx state matches wantInTx.
func newRetryTestEngine(inTx bool) *dgraphEngine {
	e := &dgraphEngine{}

	if inTx {
		e.activeTxn.Store(&dgo.Txn{})
	}

	return e
}

// TestRetryOnContention_SucceedsAfterTransientAborts verifies the standalone
// path: contention errors are retried until fn succeeds.
func TestRetryOnContention_SucceedsAfterTransientAborts(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	e := newRetryTestEngine(false)

	calls := 0

	err := e.retryOnContention(context.Background(), true, func() error {
		calls++

		if calls <= 2 {
			return errAborted
		}

		return nil
	})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(calls).To(gomega.Equal(3))
}

// TestRetryOnContention_AlterPathRetriesPending verifies txnScoped=false:
// pending-transaction Alter rejections retry even though no RunInTx is
// active — schema applies are always retriable.
func TestRetryOnContention_AlterPathRetriesPending(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	e := newRetryTestEngine(false)

	calls := 0

	err := e.retryOnContention(context.Background(), false, func() error {
		calls++

		if calls == 1 {
			return errPendingTxns
		}

		return nil
	})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(calls).To(gomega.Equal(2))
}

// TestRetryOnContention_NonContentionFailsFast verifies unrelated errors
// surface on the first attempt without retrying.
func TestRetryOnContention_NonContentionFailsFast(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	e := newRetryTestEngine(false)

	boom := errors.New("connection refused")

	calls := 0

	err := e.retryOnContention(context.Background(), true, func() error {
		calls++

		return boom
	})

	g.Expect(err).To(gomega.MatchError(boom))
	g.Expect(calls).To(gomega.Equal(1))
}

// TestRetryOnContention_TxnScopedSurfacesImmediately verifies the RunInTx
// contract: inside an active transaction even a contention error surfaces
// immediately — the CALLER must re-run the whole transaction.
func TestRetryOnContention_TxnScopedSurfacesImmediately(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	e := newRetryTestEngine(true)

	calls := 0

	err := e.retryOnContention(context.Background(), true, func() error {
		calls++

		return errAborted
	})

	g.Expect(err).To(gomega.MatchError(errAborted))
	g.Expect(calls).To(gomega.Equal(1))
}

// TestRetryOnContention_ExhaustsAttempts verifies the attempt budget: after
// contentionAttempts contention errors, the last error is returned.
func TestRetryOnContention_ExhaustsAttempts(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	e := newRetryTestEngine(false)

	calls := 0

	err := e.retryOnContention(context.Background(), true, func() error {
		calls++

		return errAborted
	})

	g.Expect(err).To(gomega.MatchError(errAborted))
	g.Expect(calls).To(gomega.Equal(contentionAttempts))
}

// TestRetryOnContention_ContextCancelDuringBackoff verifies a cancelled
// context aborts the retry wait with the context error.
func TestRetryOnContention_ContextCancelDuringBackoff(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	e := newRetryTestEngine(false)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0

	err := e.retryOnContention(ctx, true, func() error {
		calls++

		return errAborted
	})

	g.Expect(err).To(gomega.MatchError(context.Canceled))
	g.Expect(calls).To(gomega.Equal(1))
}
