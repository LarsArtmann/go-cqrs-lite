package tursoengine_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// matViewPropertyDBCounter gives every rapid iteration its own database file
// (t.TempDir() returns the SAME path for every call on a test, so the
// filename must provide the isolation).
var matViewPropertyDBCounter atomic.Int64

// matViewPropertySpecs is the full view matrix on the accelerated engine:
// scalar SUM/COUNT/AVG/MIN/MAX plus grouped SUM/AVG over "customer" (same
// matrix the read bench uses).
func matViewPropertySpecs() []metaengine.MaterializedViewSpec {
	return []metaengine.MaterializedViewSpec{
		{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewCount},
		{Collection: "orders", Fn: metaengine.MatViewAvg, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewMin, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewMax, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
	}
}

// TestTursoMatView_PropertyServedMatchesBase: for RANDOM datasets written in
// RANDOM transaction splits, every matview-served aggregate (scalar and
// grouped) must equal the aggregate recomputed from the generated rows
// in-memory. This is the randomized counterpart of the deterministic
// serving tests: it explores group/transaction shapes the fixtures miss.
//
// Envelope note (upstream defect A, docs/research/2026-09-07_turso-go-*):
// grouped views LOSE cross-transaction deltas at scale (1,100+ rows, ~316
// groups). The draws here stay far inside the verified-exact envelope (<=60
// rows, <=8 groups, <=3 transactions — the regime the two-tx pin covers);
// the scale regime is governed by
// TestTursoMatView_GroupedSumDefectAEnvelopeGuard.
func TestTursoMatView_PropertyServedMatchesBase(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		ctx := context.Background()

		txCount := rapid.IntRange(2, 3).Draw(rt, "txCount")
		groups := rapid.IntRange(2, 8).Draw(rt, "groups")

		dir := t.TempDir()
		path := filepath.Join(dir, fmt.Sprintf("mvprop_%d.db", matViewPropertyDBCounter.Add(1)))

		eng, err := tursoengine.New(path, tursoengine.WithMaterializedViews(matViewPropertySpecs()))
		if err != nil {
			rt.Skipf("turso not available: %v", err)
		}
		t.Cleanup(func() { _ = eng.Close() })

		mb := eng.(metaengine.MapBackend)
		tx := eng.(metaengine.Transactional)
		gr := eng.(metaengine.GroupedAggregateReader)

		type row struct {
			key      string
			customer string
			amount   float64
		}

		var all []row
		var written []row
		for txIdx := 0; txIdx < txCount; txIdx++ {
			n := rapid.IntRange(5, 20).Draw(rt, fmt.Sprintf("rows_tx%d", txIdx))
			var batch []row
			for i := 0; i < n; i++ {
				idx := len(all)
				all = append(all, row{
					key: fmt.Sprintf("k%04d", idx),
					customer: fmt.Sprintf(
						"c%d",
						rapid.IntRange(0, groups-1).Draw(rt, fmt.Sprintf("grp_%d_%d", txIdx, idx)),
					),
					amount: float64(
						rapid.IntRange(0, 500).Draw(rt, fmt.Sprintf("amt_%d_%d", txIdx, idx)),
					),
				})
				batch = append(batch, all[idx])
			}

			err := tx.RunInTx(ctx, func(ctx context.Context) error {
				for _, r := range batch {
					if err := mb.MapSet(ctx, "orders", r.key, map[string]any{
						"customer": r.customer,
						"amount":   r.amount,
					}); err != nil {
						return err
					}
				}

				return nil
			})
			if err != nil {
				rt.Fatalf("tx %d: %v", txIdx, err)
			}
			written = append(written, batch...)
		}

		// Expected aggregates over exactly the committed rows.
		var expSum float64
		expCount := float64(len(written))
		expMin := written[0].amount
		expMax := written[0].amount
		expGroupSum := map[string]float64{}
		for _, r := range written {
			expSum += r.amount
			expMin = min(expMin, r.amount)
			expMax = max(expMax, r.amount)
			expGroupSum[r.customer] += r.amount
		}
		expAvg := expSum / expCount

		aggr := eng.(metaengine.AggregateReader)
		scalar := func(fn metaengine.AggregateFn, column string) float64 {
			rt.Helper()

			got, err := aggr.Aggregate(ctx, "orders", fn, column, nil)
			if err != nil {
				rt.Fatalf("Aggregate %s: %v", fn, err)
			}

			return got
		}

		if got := scalar(metaengine.MatViewSum, "amount"); got != expSum {
			rt.Fatalf("SUM served %v != expected %v", got, expSum)
		}
		if got := scalar(metaengine.MatViewCount, ""); got != expCount {
			rt.Fatalf("COUNT served %v != expected %v", got, expCount)
		}
		if got := scalar(metaengine.MatViewAvg, "amount"); got != expAvg {
			rt.Fatalf("AVG served %v != expected %v", got, expAvg)
		}
		if got := scalar(metaengine.MatViewMin, "amount"); got != expMin {
			rt.Fatalf("MIN served %v != expected %v", got, expMin)
		}
		if got := scalar(metaengine.MatViewMax, "amount"); got != expMax {
			rt.Fatalf("MAX served %v != expected %v", got, expMax)
		}

		groupsServed, err := gr.GroupedAggregate(
			ctx, "orders", metaengine.MatViewSum, "amount", "customer", nil,
		)
		if err != nil {
			rt.Fatalf("GroupedAggregate: %v", err)
		}
		if len(groupsServed) != len(expGroupSum) {
			rt.Fatalf("group count served %d != expected %d", len(groupsServed), len(expGroupSum))
		}
		for grp, want := range expGroupSum {
			got, ok := groupsServed[grp]
			if !ok {
				rt.Fatalf("group %q missing from served result", grp)
			}
			if got != want {
				rt.Fatalf("group %q served %v != expected %v", grp, got, want)
			}
		}
	})
}

// TestTursoMatView_GroupedSumDefectAEnvelopeGuard pins the LARGE-scale
// grouped-view envelope: 2,000 rows over 316 groups written in TWO
// transactions — the minimal repro shape for upstream defect A (grouped
// views lose cross-transaction deltas; identical 430.50 delta verified on
// v0.7.2 through v0.8.0-pre.10, re-checked 2026-09-11, see
// docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md).
//
// The guard SKIPS while the defect is live so CI stays green on a known
// upstream bug. When upstream fixes it, the skip becomes the loud flip:
// run with TURSO_IVM_ENFORCE_FIX=1 to assert exactness now, then remove the
// skip gate, the Doctor WARN, and the docs caveats in the same change
// (TODO_LIST "Track turso-go releases" checklist).
func TestTursoMatView_GroupedSumDefectAEnvelopeGuard(t *testing.T) {
	t.Parallel()

	if os.Getenv("TURSO_IVM_ENFORCE_FIX") == "" {
		t.Skip(
			"upstream defect A live (tursogo <= v0.8.0-pre.10): grouped views lose cross-transaction deltas at scale; set TURSO_IVM_ENFORCE_FIX=1 to enforce the fixed behavior",
		)
	}

	ctx := context.Background()

	const (
		rowsTotal    = 2000
		groupModulus = 316
		amountMod    = 97
	)

	eng, err := tursoengine.New(
		filepath.Join(t.TempDir(), "defectA_envelope.db"),
		tursoengine.WithMaterializedViews([]metaengine.MaterializedViewSpec{
			{
				Collection: "orders",
				Fn:         metaengine.MatViewSum,
				Column:     "amount",
				GroupBy:    "customer",
			},
		}),
	)
	if err != nil {
		t.Skipf("turso not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	mb := eng.(metaengine.MapBackend)
	txn := eng.(metaengine.Transactional)
	gr := eng.(metaengine.GroupedAggregateReader)

	chunk := func(from, to int) error {
		return txn.RunInTx(ctx, func(ctx context.Context) error {
			for i := from; i < to; i++ {
				value := map[string]any{
					"customer": fmt.Sprintf("c%d", i%groupModulus),
					"amount":   float64(i%amountMod) + 0.5,
				}
				if err := mb.MapSet(
					ctx,
					"orders",
					fmt.Sprintf("order-%05d", i),
					value,
				); err != nil {
					return err
				}
			}

			return nil
		})
	}

	if err := chunk(0, 1000); err != nil {
		t.Fatalf("tx1: %v", err)
	}
	if err := chunk(1000, rowsTotal); err != nil {
		t.Fatalf("tx2: %v", err)
	}

	expGroupSum := map[string]float64{}
	var expSum float64
	for i := 0; i < rowsTotal; i++ {
		amount := float64(i%amountMod) + 0.5
		expSum += amount
		expGroupSum[fmt.Sprintf("c%d", i%groupModulus)] += amount
	}

	gotGroupSum, err := gr.GroupedAggregate(
		ctx, "orders", metaengine.MatViewSum, "amount", "customer", nil,
	)
	if err != nil {
		t.Fatalf("GroupedAggregate: %v", err)
	}

	var gotSum float64
	for _, sum := range gotGroupSum {
		gotSum += sum
	}

	if gotSum != expSum {
		t.Fatalf(
			"defect A no longer reproduces: grouped view total %.2f == base %.2f — upstream fixed the IVM delta loss; flip the guard (remove skip + WARN + docs caveats)",
			gotSum,
			expSum,
		)
	}
	for grp, want := range expGroupSum {
		if got := gotGroupSum[grp]; got != want {
			t.Fatalf(
				"defect A no longer reproduces: group %q served %.2f != base %.2f — upstream fixed the IVM delta loss; flip the guard (remove skip + WARN + docs caveats)",
				grp,
				got,
				want,
			)
		}
	}
}
