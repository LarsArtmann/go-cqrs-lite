package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// maxMetaEngineSamples caps Apply and query samples so the phase stays fast.
const maxMetaEngineSamples = 500

type meBenchCounterInput struct{}

type meBenchIncrementEvent struct {
	Status string
}

func meBenchCounterQuery() metaengine.QueryDecl[meBenchCounterInput, map[string]int64] {
	return metaengine.Query[meBenchCounterInput, map[string]int64](
		"bench_counter",
		metaengine.On(meBenchIncrementEvent{}, func(e meBenchIncrementEvent) metaengine.Delta {
			return metaengine.Delta{e.Status: +1}
		}),
	)
}

// metaEnginePhase benchmarks the cost-based storage planner's overhead with a
// counter workload (M17). Measures:
//   - Apply throughput: events/sec through the planner's fold dispatch + engine write.
//   - ExecuteTyped read latency: query dispatch + engine point read.
//
// The phase creates its own memory-backed Store with a counter ADT to measure
// the planner overhead ceiling independent of the consumer's specific queries.
// This tells you whether the planner or the engine is the bottleneck.
//
// Skipped automatically when Config.SkipMetaEngine is true or the bundle has
// no metaengine registered (bundle.MetaEngine() returns nil).
func (r *runner) metaEnginePhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil // ctx done; graceful skip
	}

	if r.bundle.MetaEngine() == nil {
		return nil // no metaengine in this deployment; graceful skip
	}

	// Build a private counter store to benchmark the planner overhead.
	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, meBenchCounterQuery())
	if err != nil {
		return err
	}
	defer store.Close()

	sampleCount := min(r.config.Profile.Streams, maxMetaEngineSamples)
	if sampleCount <= 0 {
		sampleCount = 1
	}

	statuses := []string{"open", "done", "cancelled"}

	// ── Apply throughput ──
	applyColl := NewLatencyCollector(0)
	applyStart := time.Now()

	for i := range sampleCount {
		if ctx.Err() != nil {
			break
		}

		status := statuses[i%len(statuses)]
		evt := meBenchIncrementEvent{Status: status}

		start := time.Now()
		if err := store.Apply(ctx, "MeBenchIncrementEvent", evt); err != nil {
			return err
		}
		applyColl.Record(time.Since(start))
	}

	applyElapsed := time.Since(applyStart).Seconds()
	r.result.MetaEngineApplyLatency = applyColl.Stats()

	if applyElapsed > 0 {
		r.result.MetaEngineApplyThroughput = float64(applyColl.Stats().Count) / applyElapsed
	}

	// ── ExecuteTyped read latency ──
	queryColl := NewLatencyCollector(0)

	for range sampleCount {
		if ctx.Err() != nil {
			break
		}

		start := time.Now()

		_, err := metaengine.ExecuteTyped[meBenchCounterInput, map[string]int64](
			ctx, store, meBenchCounterInput{},
		)
		queryColl.Record(time.Since(start))

		if err != nil {
			return err
		}
	}

	r.result.MetaEngineQueryLatency = queryColl.Stats()

	return nil
}
