package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// maxMetaEngineSamples caps Apply and query samples so the phase stays fast
// while still being large enough to reveal O(N) scan costs.
const maxMetaEngineSamples = 2000

// ── Counter ADT types (existing workload, measures planner overhead) ──

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

// metaEnginePhase benchmarks the cost-based storage planner with two ADT
// workloads:
//
//  1. Counter ADT — Apply + ExecuteTyped (planner overhead for aggregation)
//  2. Map ADT — Apply + Scan (filtered) + PointRead + ConcurrentApply
//     (planner overhead for collection CRUD + query)
//
// The Map ADT workload is the critical benchmark: it exercises the primary
// read paths (Scan with filter, point Get) that justify the planner's
// existence. The Counter workload alone only tests write throughput.
//
// Skipped automatically when Config.SkipMetaEngine is true or the bundle has
// no metaengine registered (bundle.MetaEngine() returns nil).
func (r *runner) metaEnginePhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	if r.bundle.MetaEngine() == nil {
		return nil
	}

	if err := r.metaEngineCounterWorkload(ctx); err != nil {
		return err
	}

	return r.metaEngineMapWorkload(ctx)
}

// metaEngineCounterWorkload measures planner write overhead with a simple
// counter (Delta map[string]int64 increment). This isolates the fold dispatch
// + engine write path from any scan or materialization cost.
func (r *runner) metaEngineCounterWorkload(ctx context.Context) error {
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

	// Apply throughput (single-threaded)
	applyColl := NewLatencyCollector(0)
	applyStart := time.Now()

	for i := range sampleCount {
		if ctx.Err() != nil {
			break
		}

		status := statuses[i%len(statuses)]

		start := time.Now()

		if err := store.Apply(ctx, "meBenchIncrementEvent",
			meBenchIncrementEvent{Status: status}); err != nil {
			return err
		}

		applyColl.Record(time.Since(start))
	}

	applyElapsed := time.Since(applyStart).Seconds()
	r.result.MetaEngineApplyLatency = applyColl.Stats()

	if applyElapsed > 0 {
		r.result.MetaEngineApplyThroughput = float64(applyColl.Stats().Count) / applyElapsed
	}

	// ExecuteTyped read latency
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
