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

type benchEngines struct {
	baseline metaengine.Engine
	accel    metaengine.Engine
	baseAR   metaengine.AggregateReader
	accelAR  metaengine.AggregateReader
}

func setupBenchEngines(b *testing.B, n, customers int, specs []metaengine.MaterializedViewSpec) benchEngines {
	b.Helper()

	ctx := context.Background()

	dir := b.TempDir()

	base, err := tursoengine.New(filepath.Join(dir, "baseline.db"))
	if err != nil {
		b.Skipf("turso not available: %v", err)
	}

	acc, err := tursoengine.New(filepath.Join(dir, "accel.db"), tursoengine.WithMaterializedViews(specs))
	if err != nil {
		b.Fatalf("accel engine: %v", err)
	}

	b.Cleanup(func() {
		_ = base.Close()
		_ = acc.Close()
	})

	rows := orderRows(n, customers)

	seedOrders(ctx, b, base, rows)
	seedOrders(ctx, b, acc, rows)

	return benchEngines{
		baseline: base,
		accel:    acc,
		baseAR:   base.(metaengine.AggregateReader),
		accelAR:  acc.(metaengine.AggregateReader),
	}
}

type aggCase struct {
	name   string
	fn     metaengine.AggregateFn
	column string
}

func BenchmarkMatViewRead(b *testing.B) {
	ctx := context.Background()

	scales := []struct {
		name      string
		n         int
		customers int
	}{
		{"1k", 1_000, 31},
		{"10k", 10_000, 99},
		{"100k", 100_000, 316},
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
			b.Run(fmt.Sprintf("agg=%s/scale=%s", c.name, scale.name), func(b *testing.B) {
				eng := setupBenchEngines(b, scale.n, scale.customers, matviewBenchSpecs())

				b.Run("baseline", func(b *testing.B) {
					b.ReportAllocs()

					for b.Loop() {
						if _, err := eng.baseAR.Aggregate(ctx, "orders", c.fn, c.column, nil); err != nil {
							b.Fatal(err)
						}
					}
				})

				b.Run("matview", func(b *testing.B) {
					b.ReportAllocs()

					for b.Loop() {
						if _, err := eng.accelAR.Aggregate(ctx, "orders", c.fn, c.column, nil); err != nil {
							b.Fatal(err)
						}
					}
				})
			})
		}

		// Grouped aggregate (GROUP BY customer): baseline full scan + group vs
		// reading the precomputed view rows.
		b.Run(fmt.Sprintf("agg=SUM_GROUPED/scale=%s", scale.name), func(b *testing.B) {
			eng := setupBenchEngines(b, scale.n, scale.customers, matviewBenchSpecs())

			base := eng.baseline.(metaengine.GroupedAggregateReader)
			acc := eng.accel.(metaengine.GroupedAggregateReader)

			b.Run("baseline", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := base.GroupedAggregate(ctx, "orders", metaengine.MatViewSum, "amount", "customer", nil); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("matview", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := acc.GroupedAggregate(ctx, "orders", metaengine.MatViewSum, "amount", "customer", nil); err != nil {
						b.Fatal(err)
					}
				}
			})
		})

		// Scalar SUM served via the GROUPED view's sums (O(groups) middle path).
		b.Run(fmt.Sprintf("agg=SUM_VIA_GROUPED/scale=%s", scale.name), func(b *testing.B) {
			onlyGrouped := []metaengine.MaterializedViewSpec{
				{Collection: "orders", Fn: metaengine.MatViewSum, Column: "amount", GroupBy: "customer"},
			}

			eng := setupBenchEngines(b, scale.n, scale.customers, onlyGrouped)

			b.Run("baseline", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := eng.baseAR.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("matview", func(b *testing.B) {
				b.ReportAllocs()

				for b.Loop() {
					if _, err := eng.accelAR.Aggregate(ctx, "orders", metaengine.MatViewSum, "amount", nil); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkMatViewWrite measures steady-state MapSet (REPLACE) throughput on
// a bounded 10k-key working set with 0, 1, or 3 maintained views — the IVM
// overhead the operator pays on writes for read acceleration.
func BenchmarkMatViewWrite(b *testing.B) {
	ctx := context.Background()

	const keys = 10_000

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
			dsn := filepath.Join(dir, fmt.Sprintf("write_%s.db", mode.name))

			eng, err := tursoengine.New(dsn, tursoengine.WithMaterializedViews(mode.specs))
			if err != nil {
				b.Skipf("turso not available: %v", err)
			}

			defer func() { _ = eng.Close() }()

			mb := eng.(metaengine.MapBackend)

			seedOrders(ctx, b, eng, orderRows(keys, 100))

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
		})
	}
}
