//go:build ivmrepro

// One-command reproduction suite for the three upstream turso-go IVM defects
// that block grouped materialized views (ADR-0135). Full characterization:
// docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md.
//
// Unlike TestTursoMatView_GroupedSumDefectAEnvelopeGuard, these tests ASSERT
// the defects are PRESENT: a failure here means upstream fixed or changed a
// defect — do not ship a pin bump on a red suite; flip the caveat first (the
// exact order is docs/turso-go-ivm-fix-flip-runbook.md).
//
// Run (against the go.mod-pinned driver, ~5 min):
//
//	cd metaengine/tursoengine && GOWORK=off go test \
//	  -tags "goexperiment.jsonv2 ivmrepro" -run TestIVMRepro -count=1 -timeout 30m .
//
// Checking a NEW upstream turso-go release: retarget the pin first (runbook
// section "Checking a new turso-go release"), run this suite, then either
// keep or revert the pin. Sizing env vars:
//
//	TURSO_IVM_REPRO_ROWS    cumulative rows for defects B/C (default 27000)
//	TURSO_IVM_REPRO_ROUNDS  fresh-file rounds for defect C (default 24 — the draft's methodology)
package tursoengine_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// The repro workload mirrors the research draft exactly (chunked 1,000-row
// transactions, 316 groups, i%97+0.5 amounts) so observed numbers line up
// row-for-row with the documented checkpoints (e.g. the 430.50 delta at 2k).
const (
	ivmChunkSize     = 1000
	ivmGroupModulus  = 316
	ivmAmountModulus = 97
)

func ivmAmount(i int) float64  { return float64(i%ivmAmountModulus) + 0.5 }
func ivmCustomer(i int) string { return fmt.Sprintf("c%d", i%ivmGroupModulus) }
func ivmKey(i int) string      { return fmt.Sprintf("order-%05d", i) }

func ivmExpectedSum(rows int) float64 {
	sum := 0.0
	for i := 0; i < rows; i++ {
		sum += ivmAmount(i)
	}

	return sum
}

func ivmEnvInt(t *testing.T, key string, fallback int) int {
	t.Helper()

	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		t.Fatalf("%s must be a positive integer, got %q", key, raw)
	}

	return n
}

// ivmReproEngine bundles the capability views the repro workload needs.
type ivmReproEngine struct {
	eng  metaengine.Engine
	mb   metaengine.MapBackend
	txn  metaengine.Transactional
	gr   metaengine.GroupedAggregateReader
	aggr metaengine.AggregateReader
}

// ivmOpenReproEngine opens a fresh embedded database with one SCALAR and one
// GROUPED SUM view over "orders" — the minimal shape that carries all three
// defects.
func ivmOpenReproEngine(t *testing.T, name string) *ivmReproEngine {
	t.Helper()

	eng, err := tursoengine.New(
		filepath.Join(t.TempDir(), name),
		tursoengine.WithMaterializedViews([]metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
		}),
	)
	if err != nil {
		t.Skipf("turso not available: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	return &ivmReproEngine{
		eng:  eng,
		mb:   eng.(metaengine.MapBackend),
		txn:  eng.(metaengine.Transactional),
		gr:   eng.(metaengine.GroupedAggregateReader),
		aggr: eng.(metaengine.AggregateReader),
	}
}

func (r *ivmReproEngine) insertChunk(ctx context.Context, from, to int) error {
	return r.txn.RunInTx(ctx, func(ctx context.Context) error {
		for i := from; i < to; i++ {
			if err := r.mb.MapSet(ctx, "orders", ivmKey(i), map[string]any{
				"customer": ivmCustomer(i),
				"amount":   ivmAmount(i),
			}); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *ivmReproEngine) groupedSumTotal(t *testing.T) (total float64, groups int) {
	t.Helper()

	perGroup, err := r.gr.GroupedAggregate(
		context.Background(), "orders", metaengine.MatViewSum, "amount", "customer", nil,
	)
	if err != nil {
		t.Fatalf("GroupedAggregate: %v", err)
	}

	for _, sum := range perGroup {
		total += sum
	}

	return total, len(perGroup)
}

func (r *ivmReproEngine) scalarSum(t *testing.T) float64 {
	t.Helper()

	got, err := r.aggr.Aggregate(
		context.Background(), "orders", metaengine.MatViewSum, "amount", nil,
	)
	if err != nil {
		t.Fatalf("Aggregate SUM: %v", err)
	}

	return got
}

// TestIVMReproDefectA_GroupedDeltaLossAt2k reproduces defect A at its minimal
// documented checkpoint: 2,000 rows in two transactions, where the grouped
// view total must LOSE the cross-transaction delta (documented view 95,459.50
// vs base 95,890.00 — a 430.50 loss, stable across v0.7.2 … pre.10).
func TestIVMReproDefectA_GroupedDeltaLossAt2k(t *testing.T) {
	ctx := context.Background()
	r := ivmOpenReproEngine(t, "defectA_2k.db")

	for _, chunk := range [][2]int{{0, 1000}, {1000, 2000}} {
		if err := r.insertChunk(ctx, chunk[0], chunk[1]); err != nil {
			t.Fatalf("tx %d..%d: %v", chunk[0], chunk[1], err)
		}
	}

	want := ivmExpectedSum(2000)
	got, groups := r.groupedSumTotal(t)
	if got == want {
		t.Fatalf(
			"defect A did not reproduce: grouped view total %.2f == base %.2f — upstream may have fixed the delta loss; confirm with TURSO_IVM_ENFORCE_FIX=1 go test -run TestTursoMatView_GroupedSumDefectAEnvelopeGuard, then flip per docs/turso-go-ivm-fix-flip-runbook.md",
			got,
			want,
		)
	}
	t.Logf("defect A present: view %.2f vs base %.2f over %d groups (delta %.2f; draft documents 430.50)",
		got, want, groups, want-got)

	if scalar := r.scalarSum(t); scalar != want {
		t.Fatalf(
			"scalar SUM view diverged at 2k rows (%.2f != %.2f) — NEW upstream regression beyond the characterized defects; investigate before any pin bump",
			scalar,
			want,
		)
	}
}

// TestIVMReproDefectB_GroupedViewCollapsesAtScale drives the same workload to
// the collapse regime (default 27,000 rows, every transaction committing) and
// pins BOTH halves of the documented behavior: the grouped view must diverge
// and collapse to less than half the base total, while the SCALAR view stays
// exact at the same scale (the currently-safe shape — if scalars ever drift,
// that is a new defect and flips this test loudly).
//
// View reads happen ONLY at the raw repro's milestones: every extra scan of
// the grouped view shrinks defect C's write budget ("prior scan activity in
// the process shrinks the wall" — docs/agents/gotchas-tooling-build.md), and
// per-chunk checkpointing pulls the COMMIT abort down to ~24k rows, killing
// the run before the documented 27k collapse point. The scalar-exactness pin
// is evaluated at the LAST SUCCESSFUL milestone: after a COMMIT abort the
// view state absorbs the aborted transaction's deltas (the documented
// post-abort anomaly), so a post-abort scalar read proves nothing about the
// pre-abort exactness that matters.
func TestIVMReproDefectB_GroupedViewCollapsesAtScale(t *testing.T) {
	ctx := context.Background()
	rows := ivmEnvInt(t, "TURSO_IVM_REPRO_ROWS", 27000)
	r := ivmOpenReproEngine(t, "defectB_collapse.db")

	milestones := map[int]bool{
		1000: true, 1100: true, 2000: true, 13000: true, 26000: true, rows: true,
	}

	firstDivergence := 0
	committed := 0
	commitAborted := false
	scalarExactThrough := 0
	for start := 0; start < rows; start += ivmChunkSize {
		err := r.insertChunk(ctx, start, start+ivmChunkSize)
		if err != nil {
			commitAborted = true
			t.Logf(
				"COMMIT abort at %d cumulative rows — defect C firing inside defect B's run: %v",
				start+ivmChunkSize, err,
			)
			break
		}

		committed = start + ivmChunkSize
		if milestones[committed] {
			if got, _ := r.groupedSumTotal(t); got != ivmExpectedSum(committed) && firstDivergence == 0 {
				firstDivergence = committed
				t.Logf("first grouped divergence at %d rows", committed)
			}
			if r.scalarSum(t) == ivmExpectedSum(committed) {
				scalarExactThrough = committed
			}
		}
	}
	if commitAborted && committed < rows {
		t.Logf(
			"defect C wall onset moved earlier (%d < %d rows) — see TestIVMReproDefectC for the deterministic position",
			committed, rows,
		)
	}

	want := ivmExpectedSum(committed)
	got, groups := r.groupedSumTotal(t)
	if firstDivergence == 0 {
		t.Fatalf(
			"defect B did not reproduce: grouped view stayed exact through %d rows — upstream may have fixed it; flip per docs/turso-go-ivm-fix-flip-runbook.md",
			committed,
		)
	}
	if loss := want - got; loss <= want/2 {
		t.Fatalf(
			"defect B signature changed: view diverged (first at %d rows) but did NOT collapse at %d rows (view %.2f vs base %.2f, %.1f%% loss) — investigate before any pin bump",
			firstDivergence, committed, got, want, 100*loss/want,
		)
	}
	t.Logf("defect B present: view %.2f vs base %.2f over %d groups at %d rows (first divergence at %d)",
		got, want, groups, committed, firstDivergence)

	if scalarExactThrough == 0 {
		t.Fatalf(
			"scalar SUM view diverged BEFORE the first successful milestone — NEW upstream regression beyond the characterized defects; investigate before any pin bump",
		)
	}
	t.Logf("scalar SUM view exact through %d rows (the currently-SAFE shape)", scalarExactThrough)
}

// TestIVMReproDefectC_CommitAbortsAtRowWall reproduces the COMMIT-abort wall
// (tracked upstream in PR #8257): past ~27,000 cumulative view-maintained
// rows, write transactions abort at COMMIT (through tursoengine the wall has
// been observed at 24k-27k — the position varies with in-process scan
// activity, so it is LOGGED per round, not pinned). Each round uses a FRESH
// database file; the draft established the wall at 24/24 rounds (default
// here). Post-abort probes check the two documented follow-ons: the aborted
// chunk's base rows must NOT persist once the file is reopened fresh (the
// in-process zombie-transaction readback that DOES show them is the PR
// #8257 mechanism, not persistence), and the file must reject further
// view-maintaining writes.
func TestIVMReproDefectC_CommitAbortsAtRowWall(t *testing.T) {
	ctx := context.Background()
	rows := ivmEnvInt(t, "TURSO_IVM_REPRO_ROWS", 27000)
	rounds := ivmEnvInt(t, "TURSO_IVM_REPRO_ROUNDS", 24)
	bound := rows + 3*ivmChunkSize

	abortRows := make([]int, 0, rounds)
	for round := 1; round <= rounds; round++ {
		r := ivmOpenReproEngine(t, fmt.Sprintf("defectC_round%02d.db", round))

		abortedAt := 0
		for start := 0; start < bound && abortedAt == 0; start += ivmChunkSize {
			err := r.insertChunk(ctx, start, start+ivmChunkSize)
			if err == nil {
				continue
			}

			abortedAt = start + ivmChunkSize
			t.Logf("round %02d: COMMIT aborted at %d cumulative rows: %v", round, abortedAt, err)
		}

		if abortedAt == 0 {
			t.Fatalf(
				"round %d: no COMMIT abort through %d cumulative view-maintained rows — defect C did not reproduce; upstream may have fixed it; flip per docs/turso-go-ivm-fix-flip-runbook.md",
				round, bound,
			)
		}
		if abortedAt <= ivmChunkSize {
			t.Fatalf(
				"round %d: COMMIT aborted at %d rows — before even one full chunk committed; the wall signature changed, investigate before any pin bump",
				round, abortedAt,
			)
		}
		abortRows = append(abortRows, abortedAt)

		// The aborted chunk (rows abortedAt-ivmChunkSize .. abortedAt-1) must
		// not persist. Reading through the SAME engine only proves the
		// zombie-transaction visibility artifact — close and reopen the file
		// fresh first.
		_ = r.eng.Close()
		reopened := ivmOpenReproEngine(t, fmt.Sprintf("defectC_round%02d.db", round))
		if _, found, getErr := reopened.mb.MapGet(ctx, "orders", ivmKey(abortedAt-ivmChunkSize)); getErr != nil {
			t.Fatalf("round %d: post-abort base read: %v", round, getErr)
		} else if found {
			t.Fatalf(
				"round %d: COMMIT failed at %d rows but the aborted chunk's first row PERSISTED (visible after fresh reopen) — clean-rollback contract broken; NEW upstream defect, do not pin-bump",
				round, abortedAt,
			)
		}

		if err := reopened.insertChunk(ctx, 0, 1); err == nil {
			t.Fatalf(
				"round %d: post-abort file accepted a view-maintaining write — the documented poisoned-file follow-on did not reproduce; investigate before any pin bump",
				round,
			)
		}
	}

	t.Logf(
		"defect C present in %d/%d rounds; abort rows span %d..%d (draft: deterministic at chunk 27000)",
		len(abortRows), rounds, abortRows[0], abortRows[len(abortRows)-1],
	)
}
