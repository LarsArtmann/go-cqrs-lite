package middleware_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-idempotency"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

// idempotency_pipeline_bench_test.go — measures the overhead of idempotency
// middleware on the command path. With vs without dedup checking.

// BenchmarkIdempotency_CommandOverhead measures command dispatch latency
// with and without idempotency middleware.
func BenchmarkIdempotency_CommandOverhead(b *testing.B) {
	ctx := context.Background()
	streamID := id.NewStreamID()

	// Baseline: no idempotency middleware.
	b.Run("no-idempotency", func(b *testing.B) {
		disp := command.NewDispatcher()
		handler := func(_ context.Context, _ command.Command) error { return nil }
		_ = disp.Register("bench.cmd", handler)

		cmd, _ := command.New("bench.cmd", streamID)

		b.ResetTimer()
		for b.Loop() {
			if err := disp.Dispatch(ctx, cmd); err != nil {
				b.Fatal(err)
			}
		}
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "commands/sec")
	})

	// With idempotency middleware.
	b.Run("with-idempotency", func(b *testing.B) {
		store := idempotency.NewMemoryStore(5 * time.Minute) //nolint:staticcheck // test-only
		defer store.Close()

		disp := command.NewDispatcher()
		disp.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
		handler := func(_ context.Context, _ command.Command) error { return nil }
		_ = disp.Register("bench.cmd", handler)

		// Each iteration uses a unique stream ID to avoid dedup hits.
		// Pre-generate IDs.
		ids := make([]id.StreamID, b.N+1)
		for i := range ids {
			ids[i] = id.NewStreamID()
		}

		b.ResetTimer()
		for i := range b.N {
			cmd, _ := command.New("bench.cmd", ids[i])
			if err := disp.Dispatch(ctx, cmd); err != nil {
				b.Fatal(err)
			}
		}
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "commands/sec")
	})
}

// BenchmarkIdempotency_DuplicateDetection measures the speed of detecting a
// duplicate command (cache hit path).
func BenchmarkIdempotency_DuplicateDetection(b *testing.B) {
	ctx := context.Background()
	store := idempotency.NewMemoryStore(5 * time.Minute) //nolint:staticcheck // test-only
	defer store.Close()

	disp := command.NewDispatcher()
	disp.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
	handler := func(_ context.Context, _ command.Command) error { return nil }
	_ = disp.Register("bench.cmd", handler)

	streamID := id.NewStreamID()
	cmd, _ := command.New("bench.cmd", streamID)

	// First dispatch caches the ID.
	_ = disp.Dispatch(ctx, cmd)

	b.ResetTimer()
	for b.Loop() {
		_ = disp.Dispatch(ctx, cmd) // should be deduped
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "dedupes/sec")
}

// suppress unused import.
var _ = event.Version(0)
