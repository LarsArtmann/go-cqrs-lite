package bench

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type counterState struct {
	Count int
}

type counterIncremented struct {
	Amount int `json:"amount"`
}

func applyCounter(state counterState, evt event.Event) (counterState, error) {
	p, err := event.DecodePayloadAuto[counterIncremented](evt)
	if err != nil {
		return state, err
	}

	state.Count += p.Amount

	return state, nil
}

func makeCounterDecider() decider.Decider[counterState] {
	return decider.Decider[counterState]{
		Initial: counterState{},
		Apply:   applyCounter,
	}
}

// BenchmarkCommandPath_Memory measures the full command write-path:
// decider.Execute → load state → decide → event creation → EventSink.Save → publish.
// This is the end-to-end CQRS command processing latency per command.
func BenchmarkCommandPath_Memory(b *testing.B) {
	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
	)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = bundle.Close() }()

	d := makeCounterDecider()

	store, ok := bundle.EventStore()
	if !ok {
		b.Fatal("bundle has no event store")
	}

	repo, err := decider.NewRepository(store, bundle.Publisher, d)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		streamID := id.NewStreamID()

		err := repo.Execute(ctx, streamID, "Counter",
			func(_ counterState, v event.Version) ([]event.Event, error) {
				evt, eerr := event.NewEvent(
					"counter.incremented", streamID, "Counter", v.Increment(),
					[]byte(`{"amount":1}`),
				)
				if eerr != nil {
					return nil, eerr
				}

				return []event.Event{evt}, nil
			})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "commands/sec")
}

// BenchmarkCommandPath_Concurrent measures command throughput under concurrent
// load across different streams. Models a realistic multi-stream workload.
func BenchmarkCommandPath_Concurrent(b *testing.B) {
	bundle, err := stack.New(
		stack.WithEventStore(memory.NewMemoryStore()),
	)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = bundle.Close() }()

	d := makeCounterDecider()

	store, ok := bundle.EventStore()
	if !ok {
		b.Fatal("bundle has no event store")
	}

	repo, err := decider.NewRepository(store, bundle.Publisher, d)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	concurrency := 8

	b.ResetTimer()

	for b.Loop() {
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for range concurrency {
			go func() {
				defer wg.Done()

				for range b.N / concurrency {
					streamID := id.NewStreamID()

					eerr := repo.Execute(ctx, streamID, "Counter",
						func(_ counterState, v event.Version) ([]event.Event, error) {
							evt, err2 := event.NewEvent(
								"counter.incremented", streamID, "Counter", v.Increment(),
								[]byte(`{"amount":1}`),
							)
							if err2 != nil {
								return nil, err2
							}

							return []event.Event{evt}, nil
						})
					if eerr != nil {
						b.Error(eerr)

						return
					}
				}
			}()
		}

		wg.Wait()
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "commands/sec")
}
