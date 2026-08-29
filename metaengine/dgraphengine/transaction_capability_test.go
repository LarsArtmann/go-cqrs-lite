package dgraphengine_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	gomega "github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- T41: Transactional capability ---

// TestRunInTx_CommitPersistsAllWrites verifies the happy path: every write
// performed inside the transaction becomes visible after it returns.
func TestRunInTx_CommitPersistsAllWrites(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	tx, ok := eng.(metaengine.Transactional)
	g.Expect(ok).To(gomega.BeTrue(), "dgraphEngine must implement metaengine.Transactional")

	mb := eng.(metaengine.MapBackend)
	col := uniqueCollection(t, "tx_commit")

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		for _, k := range []string{"a", "b", "c"} {
			if err := mb.MapSet(ctx, col, k, map[string]any{"v": k}); err != nil {
				return err
			}
		}

		return nil
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	for _, k := range []string{"a", "b", "c"} {
		_, found, err := mb.MapGet(ctx, col, k)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(found).To(gomega.BeTrue(), "key %q must be visible after commit", k)
	}
}

// TestRunInTx_RollbackDiscardsAllWrites verifies atomicity: when fn fails,
// no write from the transaction is visible afterwards.
func TestRunInTx_RollbackDiscardsAllWrites(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)
	mb := eng.(metaengine.MapBackend)
	col := uniqueCollection(t, "tx_rollback")

	boom := errors.New("boom: abort before commit")

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		if err := mb.MapSet(ctx, col, "x", map[string]any{"v": 1}); err != nil {
			return err
		}

		if err := mb.MapSet(ctx, col, "y", map[string]any{"v": 2}); err != nil {
			return err
		}

		return boom
	})
	g.Expect(err).To(gomega.MatchError(boom))

	_, foundX, err := mb.MapGet(ctx, col, "x")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundX).To(gomega.BeFalse(), "write must be rolled back")

	_, foundY, err := mb.MapGet(ctx, col, "y")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(foundY).To(gomega.BeFalse(), "write must be rolled back")
}

// TestRunInTx_SerializesConcurrentCalls verifies the documented concurrency
// contract: concurrent RunInTx calls are serialized, both succeed, and all
// writes from both transactions are visible afterwards.
func TestRunInTx_SerializesConcurrentCalls(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)
	mb := eng.(metaengine.MapBackend)
	col := uniqueCollection(t, "tx_concurrent")

	const goroutines = 4

	var wg sync.WaitGroup

	errs := make(chan error, goroutines)

	for i := range goroutines {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			err := tx.RunInTx(ctx, func(ctx context.Context) error {
				return mb.MapSet(ctx, col, "shared", map[string]any{"writer": i})
			})
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		g.Expect(err).NotTo(gomega.HaveOccurred())
	}

	_, found, err := mb.MapGet(ctx, col, "shared")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(found).To(gomega.BeTrue())
}

// TestRunInTx_NestedRejected verifies nested RunInTx fails loudly instead of
// deadlocking or corrupting the outer transaction.
func TestRunInTx_NestedRejected(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	tx := eng.(metaengine.Transactional)

	err := tx.RunInTx(ctx, func(ctx context.Context) error {
		return tx.RunInTx(ctx, func(ctx context.Context) error {
			return nil
		})
	})
	g.Expect(err).To(gomega.HaveOccurred(), "nested RunInTx must be rejected")
}

// --- T41: harness conformance capabilities ---

func TestDgraphEngine_CapabilityConformance(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustNewDgraphEngine(t)

	_, isHC := eng.(metaengine.HealthChecker)
	g.Expect(isHC).To(gomega.BeTrue(), "HealthCheck")

	_, isProber := eng.(metaengine.Prober)
	g.Expect(isProber).To(gomega.BeTrue(), "Prober")

	_, isTransactMeasurer := eng.(metaengine.TransactMeasurer)
	g.Expect(isTransactMeasurer).To(gomega.BeTrue(), "TransactMeasurer")

	_, isCalibratable := eng.(metaengine.Calibratable)
	g.Expect(isCalibratable).To(gomega.BeTrue(), "Calibratable")

	hc := eng.(metaengine.HealthChecker)
	g.Expect(hc.HealthCheck(context.Background())).To(gomega.Succeed())

	prober := eng.(metaengine.Prober)
	rtt, err := prober.Probe(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(rtt).NotTo(gomega.BeZero(), "Probe must return a measured RTT, not zero")
}

// --- T41: empty-collection MapScan ---

func TestDgraphEngine_MapScan_EmptyCollection(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	eng := mustNewDgraphEngine(t)
	ctx := context.Background()

	sb := eng.(metaengine.ScanBackend)
	col := uniqueCollection(t, "scan_empty")

	result, err := sb.MapScan(ctx, col, nil, nil, nil, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.Items).To(gomega.BeEmpty())
	g.Expect(result.HasMore).To(gomega.BeFalse())
}
