package bboltengine_test

import (
	"context"
	"fmt"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// BenchmarkBboltMapScan benchmarks MapScan at various collection sizes.
// Unlike pebbleengine (which benchmarks RawScanReader), bbolt uses
// ScanBackend.MapScan — the closure-based full-scan path.
func BenchmarkBboltMapScan(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ctx := context.Background()
			eng := mustNewBboltEngine(b)

			mb := eng.(metaengine.MapBackend)
			for i := range n {
				_ = mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
					"score": float64(i),
				})
			}

			sb := eng.(metaengine.ScanBackend)

			b.ResetTimer()

			for range b.N {
				result, err := sb.MapScan(ctx, "items", nil, nil, nil, 10)
				if err != nil {
					b.Fatal(err)
				}
				if len(result.Items) == 0 {
					b.Fatal("expected non-empty results")
				}
			}
		})
	}
}

// BenchmarkBboltMapScanFiltered benchmarks MapScan with a Go-level filter
// function at various collection sizes.
func BenchmarkBboltMapScanFiltered(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			ctx := context.Background()
			eng := mustNewBboltEngine(b)

			mb := eng.(metaengine.MapBackend)
			for i := range n {
				_ = mb.MapSet(ctx, "items", fmt.Sprintf("k%d", i), map[string]any{
					"score": float64(i),
				})
			}

			sb := eng.(metaengine.ScanBackend)
			threshold := float64(n / 2)

			b.ResetTimer()

			for range b.N {
				result, err := sb.MapScan(ctx, "items",
					func(item any) bool {
						if m, ok := item.(map[string]any); ok {
							if s, ok := m["score"].(float64); ok {
								return s >= threshold
							}
						}
						return false
					},
					nil, nil, 10,
				)
				if err != nil {
					b.Fatal(err)
				}
				if len(result.Items) == 0 {
					b.Fatal("expected non-empty results")
				}
			}
		})
	}
}
