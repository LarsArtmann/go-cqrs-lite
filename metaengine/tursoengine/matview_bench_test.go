package tursoengine_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// matviewBenchSpecs returns the views declared on the accelerated engine: a
// scalar view per aggregate fn plus grouped SUM/AVG views over "customer".
func matviewBenchSpecs() []metaengine.MaterializedViewSpec {
	return []metaengine.MaterializedViewSpec{
		{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewCount},
		{Collection: "orders", Fn: metaengine.MatViewAvg, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewMin, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewMax, Column: "amount"},
		{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
		{Collection: "orders", Fn: metaengine.MatViewAvg, Column: "amount", GroupBy: "customer"},
	}
}

// seedInTx seeds in 1000-row transactions when the engine supports them —
// one fsync per chunk instead of one per row, so setup stays fast on
// file-backed DSNs. Chunks (rather than one giant tx) also sidestep an
// upstream turso-go v0.7.2 bug: COMMIT of a very large single transaction
// that drives IVM across many materialized views intermittently fails with
// "cannot commit - no transaction is active".
func seedInTx(ctx context.Context, tb testing.TB, eng metaengine.Engine, rows []orderRow) {
	tb.Helper()

	tx, ok := eng.(metaengine.Transactional)
	if !ok {
		seedOrders(ctx, tb, eng, rows)

		return
	}

	const chunkSize = 1000

	for start := 0; start < len(rows); start += chunkSize {
		end := min(start+chunkSize, len(rows))

		if err := tx.RunInTx(ctx, func(ctx context.Context) error {
			seedOrders(ctx, tb, eng, rows[start:end])

			return nil
		}); err != nil {
			tb.Fatalf("seed tx at %d: %v", start, err)
		}
	}
}

// benchEngine is one opened+seeded engine for a benchmark phase. The
// accelerated engine is always SEEDED FIRST (before scan-heavy baseline work
// runs in this process): the upstream commit bug's failure probability grows
// with scan activity that preceded the IVM writes.
type benchEngine struct {
	eng     metaengine.Engine
	agg     metaengine.AggregateReader
	grouped metaengine.GroupedAggregateReader
}

func openBenchEngine(
	ctx context.Context,
	tb testing.TB,
	dir, name string,
	specs []metaengine.MaterializedViewSpec,
	n, customers int,
) *benchEngine {
	tb.Helper()

	eng, err := tursoengine.New( //nolint:contextcheck // constructor takes no ctx
		filepath.Join(dir, name+".db"),
		tursoengine.WithMaterializedViews(specs),
	)
	if err != nil {
		tb.Skipf("turso not available: %v", err)
	}

	seedInTx(ctx, tb, eng, orderRows(n, customers))

	return &benchEngine{
		eng:     eng,
		agg:     eng.(metaengine.AggregateReader),
		grouped: eng.(metaengine.GroupedAggregateReader),
	}
}

type aggCase struct {
	name   string
	fn     metaengine.AggregateFn
	column string
}

// specsForCase picks the accelerated engine's declared views: the full
// 7-view matrix at 1k (all aggregate fns plus grouped SUM/AVG), or exactly
// the one view the benchmarked shape needs at larger scales — multiple views
// multiply IVM updates and cross the upstream commit-bug wall.
func specsForCase(c aggCase, full bool) []metaengine.MaterializedViewSpec {
	if full {
		return matviewBenchSpecs()
	}

	switch c.name {
	case "SUM_GROUPED":
		return []metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
		}
	default:
		spec := metaengine.MaterializedViewSpec{Collection: "orders", Fn: c.fn, Column: c.column}
		if err := spec.Validate(); err != nil {
			panic(err)
		}

		return []metaengine.MaterializedViewSpec{spec}
	}
}

// BenchmarkMatViewRead measures scalar and grouped aggregates against the
// base tables (baseline) versus operator-declared materialized views
// (matview) at three collection sizes. Phase order per case: seed the
// accelerated engine (pristine-process IVM writes), seed + bench the
// baseline, close it, then bench the accelerated reads.
func BenchmarkMatViewRead(b *testing.B) {
	ctx := context.Background()

	// 100k is intentionally absent: seeding 100k view-maintained rows trips
	// the upstream turso-go v0.7.2 commit bug (~30k cumulative IVM writes per
	// process; see AGENTS.md). Baseline scan cost scales linearly — the 10k
	// baseline number extrapolates.
	scales := []struct {
		name      string
		n         int
		customers int
	}{
		{"1k", 1_000, 31},
		{"10k", 10_000, 99},
	}

	cases := []aggCase{
		{"SUM", metaengine.MatViewSum, "amount"},
		{"COUNT", metaengine.MatViewCount, ""},
		{"AVG", metaengine.MatViewAvg, "amount"},
		{"MIN", metaengine.MatViewMin, "amount"},
		{"MAX", metaengine.MatViewMax, "amount"},
	}

	for _, scale := range scales {
		for _, c := range cases {
			b.Run(c.name+"/scale="+scale.name, func(b *testing.B) {
				dir := b.TempDir()

				acc := openBenchEngine(ctx, b, dir, "accel", matviewBenchSpecs(), scale.n, scale.customers)

				base := openBenchEngine(ctx, b, dir, "baseline", nil, scale.n, scale.customers)

				b.Run("baseline", func(b *testing.B) {
					b.ReportAllocs()

					for b.Loop() {
						if _, err := base.agg.Aggregate(ctx, "orders", c.fn, c.column, nil); err != nil {
							b.Fatal(err)
						}
					}
				})

				if err := base.eng.Close(); err != nil {
					b.Fatal(err)
				}

				b.Run("matview", func(b *testing.B) {
					b.ReportAllocs()

					for b.Loop() {
						if _, err := acc.agg.Aggregate(ctx, "orders", c.fn, c.column, nil); err != nil {
							b.Fatal(err)
						}
					}
				})

				if err := acc.eng.Close(); err != nil {
					b.Fatal(err)
				}
			})
		}

		// Grouped aggregate (GROUP BY customer): baseline full scan + group
		// versus reading the precomputed view rows.
		b.Run("agg=SUM_GROUPED/scale="+scale.name, func(b *testing.B) {
			dir := b.TempDir()

			acc := openBenchEngine(ctx, b, dir, "accel", matviewBenchSpecs(), scale.n, scale.customers)

			base := openBenchEngine(ctx, b, dir, "baseline", nil, scale.n, scale.customers)

			b.Run("baseline", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := base.grouped.GroupedAggregate(
						ctx, "orders", metaengine.MatViewSum, "amount", "customer", nil,
					); err != nil {
						b.Fatal(err)
					}
				}
			})

			if err := base.eng.Close(); err != nil {
				b.Fatal(err)
			}

			b.Run("matview", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := acc.grouped.GroupedAggregate(
						ctx, "orders", metaengine.MatViewSum, "amount", "customer", nil,
					); err != nil {
						b.Fatal(err)
					}
				}
			})

			if err := acc.eng.Close(); err != nil {
				b.Fatal(err)
			}
		})

		// Scalar SUM served via the GROUPED view's sums (O(groups) middle path).
		b.Run("agg=SUM_VIA_GROUPED/scale="+scale.name, func(b *testing.B) {
			onlyGrouped := []metaengine.MaterializedViewSpec{
				{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
			}

			dir := b.TempDir()

			acc := openBenchEngine(ctx, b, dir, "accel", onlyGrouped, scale.n, scale.customers)

			base := openBenchEngine(ctx, b, dir, "baseline", nil, scale.n, scale.customers)

			b.Run("baseline", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := base.agg.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil); err != nil {
						b.Fatal(err)
					}
				}
			})

			if err := base.eng.Close(); err != nil {
				b.Fatal(err)
			}

			b.Run("matview", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := acc.agg.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil); err != nil {
						b.Fatal(err)
					}
				}
			})

			if err := acc.eng.Close(); err != nil {
				b.Fatal(err)
			}
		})
	}
}

// BenchmarkMatViewWrite measures steady-state MapSet (REPLACE) throughput on
// a bounded 10k-key working set with 0, 1, or 3 maintained views — the IVM
// overhead the operator pays on writes for read acceleration. Each mode runs
// its own engine sequentially (single autocommit writes; no bulk IVM txs).
func BenchmarkMatViewWrite(b *testing.B) {
	ctx := context.Background()

	const keys = 5_000 // 3 views × 5k seed writes stays under the upstream ~30k IVM-write wall

	modes := []struct {
		name  string
		specs []metaengine.MaterializedViewSpec
	}{
		{"views=0", nil},
		{"views=1", []metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
		}},
		{"views=3", []metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
			{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount"},
			{Collection: "orders", Fn: metaengine.MatViewAvg, Column: "amount", GroupBy: "customer"},
		}},
	}

	dir := b.TempDir()

	for _, mode := range modes {
		b.Run(mode.name, func(b *testing.B) {
			eng, err := tursoengine.New( //nolint:contextcheck // constructor takes no ctx
				filepath.Join(dir, "write_"+mode.name+".db"),
				tursoengine.WithMaterializedViews(mode.specs),
			)
			if err != nil {
				b.Skipf("turso not available: %v", err)
			}

			mb := eng.(metaengine.MapBackend)

			seedInTx(ctx, b, eng, orderRows(keys, 100))

			b.ReportAllocs()
			b.ResetTimer()

			i := 0

			for b.Loop() {
				key := fmt.Sprintf("order-%04d", i%keys)
				value := map[string]any{
					"customer": fmt.Sprintf("c%d", i%100),
					"amount":   float64(i%97) + 0.5,
				}

				if err := mb.MapSet(ctx, "orders", key, value); err != nil {
					b.Fatal(err)
				}

				i++
			}

			if err := eng.Close(); err != nil {
				b.Fatal(err)
			}
		})
	}
}
