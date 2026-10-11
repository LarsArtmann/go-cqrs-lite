//go:build ivmrepro

// Defect-A onset-boundary bisect: the principled rows × groups × tx matrix
// TODO_LIST:208 asks for before any upstream filing. Opt-in via
// TURSO_IVM_BISECT=1 so the routine ivmrepro suite does not pay for it:
//
//	cd metaengine/tursoengine && GOWORK=off go test -tags ivmrepro \
//	  -run TestIVMReproDefectAOnsetBisect -count=1 -timeout 30m .
//
// Each configuration opens a FRESH database, inserts rows in chunk-sized
// transactions, and after every committed transaction compares the grouped
// view total against the exactly-expected sum of the inserted rows. The
// onset is the first transaction boundary where the view diverges.
package tursoengine_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/samber/lo"
)

type bisectOutcome struct {
	rows    int
	groups  int
	chunk   int
	onsetTx int // 1-based index of the first diverging transaction; 0 = never
	wallTx  int // 1-based index where defect C's commit wall aborted; 0 = none
	view    float64
	base    float64
}

// runBisectConfig drives one fresh-database workload and returns the onset.
func runBisectConfig(t *testing.T, rows, groups, chunk int) bisectOutcome {
	t.Helper()

	ctx := context.Background()

	eng, err := tursoengine.New(
		filepath.Join(t.TempDir(), fmt.Sprintf("bisect_%d_%d_%d.db", rows, groups, chunk)),
		tursoengine.WithMaterializedViews([]metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
			{
				Collection: "orders",
				Fn:         metaengine.MatViewSum,
				Column:     "amount",
				GroupBy:    "customer",
			},
		}),
		tursoengine.WithKnownGroupedViewBug(),
	)
	if err != nil {
		t.Skipf("turso not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	mb := eng.(metaengine.MapBackend)
	txn := eng.(metaengine.Transactional)
	gr := eng.(metaengine.GroupedAggregateReader)

	out := bisectOutcome{rows: rows, groups: groups, chunk: chunk}

	expected := 0.0
	txIndex := 0

	for from := 0; from < rows; from += chunk {
		to := min(from+chunk, rows)
		txIndex++

		stepErr := txn.RunInTx(ctx, func(ctx context.Context) error {
			for i := from; i < to; i++ {
				key := fmt.Sprintf("order-%05d", i)
				val := map[string]any{
					"customer": fmt.Sprintf("c%d", i%groups),
					"amount":   float64(i%97) + 0.5,
				}
				if err := mb.MapSet(ctx, "orders", key, val); err != nil {
					return err
				}
				expected += float64(i%97) + 0.5
			}

			return nil
		})
		if stepErr != nil {
			// Defect C's commit wall is DATA here (prior scan activity
			// shrinks it — documented): record where it hit and stop the
			// config; post-abort view state absorbs aborted deltas.
			if out.onsetTx == 0 {
				out.wallTx = txIndex
				out.base = expected

				return out
			}

			out.wallTx = txIndex

			return out
		}

		perGroup, err := gr.GroupedAggregate(
			ctx,
			"orders",
			metaengine.MatViewSum,
			"amount",
			"customer",
			nil,
		)
		if err != nil {
			t.Fatalf("GroupedAggregate: %v", err)
		}

		view := 0.0
		for _, sum := range perGroup {
			view += sum
		}

		if view != expected && out.onsetTx == 0 {
			out.onsetTx = txIndex
			out.view = view
			out.base = expected
		}
	}

	if out.onsetTx == 0 {
		perGroup, err := gr.GroupedAggregate(
			ctx,
			"orders",
			metaengine.MatViewSum,
			"amount",
			"customer",
			nil,
		)
		if err != nil {
			t.Fatalf("final GroupedAggregate: %v", err)
		}

		for _, sum := range perGroup {
			out.view += sum
		}
		out.base = expected
	}

	return out
}

func ceilDiv(a, b int) int {
	if a%b == 0 {
		return a / b
	}
	return a/b + 1
}

// TestIVMReproDefectAOnsetBisect sweeps the three dimensions and reports the
// onset matrix. Assertions are SOFT: the test only fails when the harness
// itself breaks — onset positions are data, not expectations.
func TestIVMReproDefectAOnsetBisect(t *testing.T) {
	if os.Getenv("TURSO_IVM_BISECT") != "1" {
		t.Skip("opt-in: set TURSO_IVM_BISECT=1 to run the onset bisect")
	}

	report := func(label string, outcomes []bisectOutcome) {
		for _, o := range outcomes {
			verdict := "EXACT"
			if o.onsetTx > 0 {
				verdict = fmt.Sprintf("DIVERGES at tx#%d (view %.2f vs base %.2f, delta %.2f)",
					o.onsetTx, o.view, o.base, o.base-o.view)
			}
			if o.wallTx > 0 {
				verdict += fmt.Sprintf(
					"; defect-C WALL at tx#%d (base by then %.2f)",
					o.wallTx,
					o.base,
				)
			}
			t.Logf("BISECT %s rows=%d groups=%d chunk=%d txs=%d → %s",
				label, o.rows, o.groups, o.chunk, ceilDiv(o.rows, o.chunk), verdict)
		}
	}

	// Dimension 1 — transaction size at fixed rows/groups (2k rows, 316
	// groups): does a SINGLE-transaction load stay exact, and where is the
	// smallest cross-tx boundary that loses its delta?
	var chunkSweep []bisectOutcome
	chunkSweep = lo.Map([]int{2000, 1000, 500, 100, 10}, func(chunk int, _ int) bisectOutcome { return runBisectConfig(t, 2000, 316, chunk) })
	report("chunk", chunkSweep)

	// Dimension 2 — groups at fixed rows/chunk (2k rows, 500-row txs):
	// is the delta loss group-count-sensitive below the draft's 316?
	var groupSweep []bisectOutcome
	groupSweep = lo.Map([]int{1, 2, 8, 64, 316, 2000}, func(groups int, _ int) bisectOutcome { return runBisectConfig(t, 2000, groups, 500) })
	report("groups", groupSweep)

	// Dimension 3 — rows at fixed groups/chunk (316 groups, 500-row txs):
	// is the SECOND transaction already enough, and does the loss scale?
	var rowSweep []bisectOutcome
	rowSweep = lo.Map([]int{500, 1000, 2000, 4000}, func(rows int, _ int) bisectOutcome { return runBisectConfig(t, rows, 316, 500) })
	report("rows", rowSweep)
}
