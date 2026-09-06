package dgraphengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Scaled calibration benches: measure the per-ROW marginal cost of Dgraph's
// single-RPC scan reads. NsPerScan / NsPerFilteredScan feed the cost model
// as PER-ROW constants (estimateCost multiplies them by volume), so they
// must hold the marginal per-row cost measured ACROSS result sizes — not
// the per-RPC total of a single small run. The RPC fixed cost does not
// scale with volume and is bounded by the NetworkRTT prior instead.
//
// Per-row slope: (ns(R2) - ns(R1)) / (rows2 - rows1) over 100/1000/10000.
// Run live only:
//
//	nix run .#ephemeral-dgraph -- bash -c \
//	  'cd metaengine/dgraphengine && GOWORK=off go test -tags goexperiment.jsonv2 \
//	   -run "^$" -bench BenchmarkCalibration_DgraphScaled -benchtime 20x -count 3 .'

func seedScaledDocs(b *testing.B, eng metaengine.Engine, col string, rows int) {
	b.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement MapBackend")
	}

	ctx := context.Background()

	for i := 0; i < rows; i++ {
		doc := map[string]any{"i": i, "mod": i % 10}

		if err := mb.MapSet(ctx, col, fmt.Sprintf("k%06d", i), doc); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
	}
}

func BenchmarkCalibration_DgraphScaled(b *testing.B) {
	eng := mustNewDgraphEngine(b)
	ctx := context.Background()

	sb, ok := eng.(metaengine.ScanBackend)
	if !ok {
		b.Fatal("dgraph engine does not implement ScanBackend")
	}

	for _, rows := range []int{100, 1000, 10000} {
		col := uniqueCollection(b, fmt.Sprintf("scaled_%d", rows))
		seedScaledDocs(b, eng, col, rows)

		b.Run(fmt.Sprintf("FullScan/rows=%d", rows), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				res, err := sb.MapScan(ctx, col, nil, nil, nil, 0)
				if err != nil {
					b.Fatalf("MapScan: %v", err)
				}

				if len(res.Items) != rows {
					b.Fatalf("got %d items, want %d", len(res.Items), rows)
				}
			}
		})

		b.Run(fmt.Sprintf("FilteredScan/rows=%d", rows), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				res, err := sb.MapScan(
					ctx, col,
					func(item any) bool {
						m, ok := item.(map[string]any)
						if !ok {
							return false
						}

						mod, _ := m["mod"].(float64)

						return int(mod) == 0 // ~10% selectivity
					},
					nil, nil, 0,
				)
				if err != nil {
					b.Fatalf("MapScan filtered: %v", err)
				}

				if len(res.Items) == 0 {
					b.Fatal("filtered scan matched nothing")
				}
			}
		})
	}
}
