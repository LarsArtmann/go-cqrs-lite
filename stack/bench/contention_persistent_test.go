package bench

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

// contention_persistent_test.go — measures same-stream contention on persistent
// backends (SQLite, Pebble). The existing contention benchmark only tests
// Memory. Real systems need to know the contention ceiling on persistent storage.

// BenchmarkContention_Persistent_SameStream measures write serialization on
// SQLite when multiple goroutines append to the SAME stream.
func BenchmarkContention_Persistent_SameStream(b *testing.B) {
	for _, backend := range []string{"sqlite", "pebble"} {
		b.Run(backend, func(b *testing.B) {
			for _, concurrency := range []int{1, 4, 8} {
				b.Run(fmt.Sprintf("workers=%d", concurrency), func(b *testing.B) {
					var bundle *stack.Bundle
					var cleanup func()

					switch backend {
					case "sqlite":
						dir := b.TempDir()
						bl, err := sqlite.New(
							dir+"/contention.db",
							sqlite.WithPragmas(sqlopt.WithOptimizations()),
						)
						if err != nil {
							b.Fatal(err)
						}
						bundle = bl
						cleanup = func() { _ = bl.Close() }
					case "pebble":
						pb := pebbleBackend()
						bl, cl := pb.create(b)
						bundle = bl
						cleanup = cl
					default:
						b.Skipf("unknown backend: %s", backend)
					}
					defer cleanup()

					store, ok := bundle.EventStore()
					if !ok {
						b.Fatal("no event store")
					}

					ctx := context.Background()
					streamID := id.NewStreamID()
					ref := id.NewStreamRef("Counter", streamID)
					var version atomic.Int64

					b.ResetTimer()

					for b.Loop() {
						var wg sync.WaitGroup
						wg.Add(concurrency)

						for range concurrency {
							go func() {
								defer wg.Done()
								for range b.N / concurrency {
									v := event.Version(version.Add(1))
									evt, err := event.NewEvent(
										"counter.incremented", streamID, "Counter", v,
										[]byte(`{"amount":1}`),
									)
									if err != nil {
										b.Error(err)

										return
									}
									if err := store.AppendBatch(
										ctx,
										ref,
										[]event.Event{evt},
									); err != nil {
										b.Error(err)

										return
									}
								}
							}()
						}
						wg.Wait()
					}

					b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
				})
			}
		})
	}
}
