package sqliteengine_test

import (
	"context"
	"errors"
	"testing"

	gomega "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Behavioral pins for RunInTx. SQLite runs in-memory, so these execute in
// every unit-test run (unlike the live-server dgraph counterparts, which
// they mirror).
func newTxTestEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := sqliteengine.NewSQLiteEngineFromDSN(":memory:")
	if err != nil {
		tb.Fatalf("NewSQLiteEngineFromDSN: %v", err)
	}

	tb.Cleanup(func() { _ = eng.Close() })

	return eng
}

func TestRunInTx_SQLite_CommitPersists(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := newTxTestEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)
	mb := eng.(metaengine.MapBackend)

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		return mb.MapSet(ctx, "tx_sqlite_commit", "k", map[string]any{"v": 1})
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	_, found, err := mb.MapGet(ctx, "tx_sqlite_commit", "k")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
}

func TestRunInTx_SQLite_RollbackDiscards(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := newTxTestEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)
	mb := eng.(metaengine.MapBackend)

	boom := errors.New("boom: rollback")

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if err := mb.MapSet(ctx, "tx_sqlite_rb", "k", map[string]any{"v": 1}); err != nil {
			return err
		}

		return boom
	})
	g.Expect(err).To(gomega.MatchError(boom))

	_, found, err := mb.MapGet(ctx, "tx_sqlite_rb", "k")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeFalse())
}

func TestRunInTx_SQLite_NestedRejected(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := newTxTestEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		return tx.RunInTx(ctx, func(ctx context.Context) error {
			return nil
		})
	})
	g.Expect(err).To(gomega.HaveOccurred(), "nested RunInTx must be rejected, not deadlock")
}

func TestRunInTx_SQLite_ReadYourWritesInsideTx(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := newTxTestEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)
	mb := eng.(metaengine.MapBackend)

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if err := mb.MapSet(ctx, "tx_sqlite_ryw", "k", map[string]any{"v": 1}); err != nil {
			return err
		}

		_, found, err := mb.MapGet(ctx, "tx_sqlite_ryw", "k")
		if err != nil {
			return err
		}

		if !found {
			t.Error("read inside transaction must see the transaction's own write")
		}

		return nil
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())
}

func TestRunInTx_SQLite_RetryAfterAbort(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := newTxTestEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)
	mb := eng.(metaengine.MapBackend)

	boom := errors.New("boom: abort, then retry")

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if err := mb.MapSet(ctx, "tx_sqlite_retry", "dead", map[string]any{"v": 0}); err != nil {
			return err
		}

		return boom
	})
	g.Expect(err).To(gomega.MatchError(boom))

	err = tx.RunInTx(ctx, func(ctx context.Context) error {
		return mb.MapSet(ctx, "tx_sqlite_retry", "alive", map[string]any{"v": 1})
	})
	g.Expect(err).NotTo(gomega.HaveOccurred(), "RunInTx after an abort must succeed")

	_, foundAlive, err := mb.MapGet(ctx, "tx_sqlite_retry", "alive")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundAlive).To(gomega.BeTrue())

	_, foundDead, err := mb.MapGet(ctx, "tx_sqlite_retry", "dead")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundDead).To(gomega.BeFalse())
}
