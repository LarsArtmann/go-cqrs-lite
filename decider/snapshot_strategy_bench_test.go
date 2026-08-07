package decider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// snapshot_strategy_bench_test.go — compares snapshot strategies.
// EveryN(10) vs EveryN(50) vs ReadPressure(50) vs None.
// Shows the tradeoff between snapshot write cost and load savings.

type benchSnapState struct {
	Count int
}

func foldBenchSnap(state benchSnapState, evt event.Event) (benchSnapState, error) {
	state.Count++
	return state, nil
}

func makeBenchSnapDecider() decider.Decider[benchSnapState] {
	return decider.Decider[benchSnapState]{
		Initial: benchSnapState{},
		Apply:   foldBenchSnap,
	}
}

// BenchmarkSnapshotStrategy_Comparison measures Execute throughput with
// different snapshot strategies. Each iteration appends 1 event to an existing
// stream of N events. With snapshots, only the delta is loaded.
func BenchmarkSnapshotStrategy_Comparison(b *testing.B) {
	streamLength := 100 // events already in the stream

	strategies := []struct {
		name     string
		strategy snapshot.SnapshotStrategy
	}{
		{"none", nil},
		{"every10", mustEveryN(10)},
		{"every50", mustEveryN(50)},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				b.StopTimer()
				// Create a fresh store and pre-seed with streamLength events.
				store := memory.NewMemoryStore()
				snapStore := memory.NewMemorySnapshotStore()

				streamID := id.NewStreamID()
				ref := id.NewStreamRef("Bench", streamID)

				for i := range streamLength {
					evt, _ := event.NewEvent(
						"bench.snap", streamID, "Bench", event.Version(i+1),
						[]byte(`{"n":1}`),
					)
					_ = store.AppendBatch(ctx, ref, []event.Event{evt})
				}

				// Create repository with or without snapshot strategy.
				var opts []decider.RepositoryOption[benchSnapState]
				if s.strategy != nil {
					opts = append(opts,
						decider.WithSnapshotStore[benchSnapState](snapStore),
						decider.WithCodec[benchSnapState](nil),
						decider.WithSnapshotStrategy[benchSnapState](s.strategy),
					)
				}

				d := makeBenchSnapDecider()
				repo, err := decider.NewRepository(store, nil, d, opts...)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()

				// Execute one more command (loads state + appends event).
				_ = repo.Execute(ctx, streamID, "Bench",
					func(_ benchSnapState, v event.Version) ([]event.Event, error) {
						evt, err := event.NewEvent(
							"bench.snap", streamID, "Bench", v.Increment(),
							[]byte(`{"n":1}`),
						)
						if err != nil {
							return nil, err
						}
						return []event.Event{evt}, nil
					})
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "executes/sec")
		})
	}
}

func mustEveryN(n int) snapshot.SnapshotStrategy {
	s, err := snapshot.EveryNEvents(n)
	if err != nil {
		panic(err)
	}
	return s
}

// suppress unused import
var _ = fmt.Sprintf
