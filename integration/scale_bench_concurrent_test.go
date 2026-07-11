package integration_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

// ---------------------------------------------------------------------------
// 8. Concurrent — parallel command dispatch + event processing
// ---------------------------------------------------------------------------

func BenchmarkScale_Concurrent_10KCommands_8Goroutines(b *testing.B) {
	b.ReportAllocs()

	dispatcher := command.NewDispatcher()
	b.Cleanup(func() { _ = dispatcher.Close() })

	err := dispatcher.Register("bench.cmd", noopCmdHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	ctx := context.Background()
	workers := 8
	opsPerWorker := 10_000

	b.ResetTimer()

	var wg sync.WaitGroup

	for b.Loop() {
		wg.Add(workers)

		for range workers {
			go func() {
				defer wg.Done()
				aggID := id.NewAggregateID()

				for range opsPerWorker {
					cmd := testutil.NewCmd(b, "bench.cmd", aggID)
					_ = dispatcher.Dispatch(ctx, cmd)
				}
			}()
		}

		wg.Wait()
	}

	b.ReportMetric(float64(b.N*workers*opsPerWorker)/b.Elapsed().Seconds(), "commands/sec")
}

func BenchmarkScale_Concurrent_DeciderExecute_4Goroutines(b *testing.B) {
	b.ReportAllocs()

	repo, ctx := newBenchDeciderRepo(b)
	workers := 4
	opsPerWorker := 1000

	b.ResetTimer()

	var wg sync.WaitGroup

	for b.Loop() {
		wg.Add(workers)

		for range workers {
			go func() {
				defer wg.Done()

				for range opsPerWorker {
					benchCreateItemConcurrent(b, repo, ctx)
				}
			}()
		}

		wg.Wait()
	}

	b.ReportMetric(float64(b.N*workers*opsPerWorker)/b.Elapsed().Seconds(), "executes/sec")
}
