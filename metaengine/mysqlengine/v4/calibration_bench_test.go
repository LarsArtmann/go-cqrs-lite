package mysqlengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func BenchmarkCalibration_MySQLSet(b *testing.B) {
	eng := mustNewMySQLEngine(b)
	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()
	var i int

	for b.Loop() {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
		i++
	}
}

func BenchmarkCalibration_MySQLGet(b *testing.B) {
	eng := mustNewMySQLEngine(b)
	mb := eng.(metaengine.MapBackend)

	ctx := context.Background()

	for i := range 1000 {
		_ = mb.MapSet(ctx, "bench", i, i*2)
	}

	var i int

	for b.Loop() {
		_, found, err := mb.MapGet(ctx, "bench", i%1000)
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}
		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}
		i++
	}
}

func BenchmarkCalibration_MySQLCounterIncrement(b *testing.B) {
	eng := mustNewMySQLEngine(b)
	cb := eng.(metaengine.CounterBackend)

	ctx := context.Background()
	var i int

	for b.Loop() {
		key := fmt.Sprintf("k%d", i%100)
		if err := cb.CounterIncrement(ctx, "bench", metaengine.Delta{key: 1}); err != nil {
			b.Fatalf("CounterIncrement %d: %v", i, err)
		}
		i++
	}
}

// mysqlCalibrationCounters is the counter-map size for the CounterGet bench
// (matches the planner's ADTCounter scale threshold of ~1K distinct keys and
// the pgengine/sqliteengine/KV-engine calibration benches).
const mysqlCalibrationCounters = 1_000

// BenchmarkCalibration_MySQL_CounterGet measures CounterGet over a 1K-key
// counter map — the actual ReadAggregate execution path (ADR-0133: the
// planner's NsPerAggregate prices ADTCounter queries, which execute
// CounterBackend.CounterGet; a SQL SUM bench would document the typed
// AggregateReader path that bypasses the planner). Feeds
// ReadCosts.NsPerAggregate (per-ROW: divide ns/op by
// mysqlCalibrationCounters). Run live with MYSQL_TEST_DSN set:
// GOWORK=off go test -run '^$' -bench '^BenchmarkCalibration_MySQL_CounterGet$'
// -benchmem -count=5.
func BenchmarkCalibration_MySQL_CounterGet(b *testing.B) {
	eng := mustNewMySQLEngine(b)
	cb := eng.(metaengine.CounterBackend)

	ctx := context.Background()

	for i := range mysqlCalibrationCounters {
		if err := cb.CounterIncrement(
			ctx,
			"aggr",
			metaengine.Delta{fmt.Sprintf("c%d", i): 1},
		); err != nil {
			b.Fatalf("seed CounterIncrement %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for b.Loop() {
		counts, err := cb.CounterGet(ctx, "aggr")
		if err != nil {
			b.Fatalf("CounterGet: %v", err)
		}

		if len(counts) != mysqlCalibrationCounters {
			b.Fatalf(
				"CounterGet: expected %d counters, got %d",
				mysqlCalibrationCounters,
				len(counts),
			)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(mysqlCalibrationCounters), "rows-scanned")
}
