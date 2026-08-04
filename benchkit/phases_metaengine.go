package benchkit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// maxMetaEngineSamples caps Apply and query samples so the phase stays fast
// while still being large enough to reveal O(N) scan costs.
const maxMetaEngineSamples = 2000

// setupMemoryMetaEngineStore creates a Memory engine, plans the given query
// args, and computes the sample count for the workload. Callers must defer
// store.Close().
func (r *runner) setupMemoryMetaEngineStore(args ...any) (*metaengine.Store, int, error) {
	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, args...)
	if err != nil {
		return nil, 0, err
	}

	sampleCount := min(r.config.Profile.Streams, maxMetaEngineSamples)
	if sampleCount <= 0 {
		sampleCount = 1
	}

	return store, sampleCount, nil
}

// Sentinel errors for metaengine correctness checks.
var (
	errMEEmptyCounter = errors.New(
		"metaengine counter: ExecuteTyped returned empty map after Apply",
	)
	errMEPointMiss = errors.New("metaengine point read: item not found")
)

// ErrMEEvent indicates a metaengine event-type mismatch (fold registrations
// don't match Apply event type strings).
var ErrMEEvent = errors.New("metaengine: event type strings may not match fold registrations")

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
		return nil //nolint:nilerr // cancelled context means skip phase, not fail
	}

	if r.bundle.MetaEngine() == nil {
		return nil
	}

	if err := r.metaEngineCounterWorkload(ctx); err != nil {
		return err
	}

	if err := r.metaEngineMapWorkload(ctx); err != nil {
		return err
	}

	return r.metaEngineSQLiteWorkload(ctx)
}

// metaEngineCounterWorkload measures planner write overhead with a simple
// counter (Delta map[string]int64 increment). This isolates the fold dispatch
// + engine write path from any scan or materialization cost.
func (r *runner) metaEngineCounterWorkload(ctx context.Context) error {
	store, sampleCount, err := r.setupMemoryMetaEngineStore(meBenchCounterQuery())
	if err != nil {
		return err
	}

	defer store.Close()

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

	// Correctness check: verify the counter has non-zero data. This catches
	// the class of bug where event type strings don't match fold registrations
	// (metaengine silently skips non-matching types, making Apply a no-op).
	counts, err := metaengine.ExecuteTyped[meBenchCounterInput, map[string]int64](
		ctx, store, meBenchCounterInput{},
	)
	if err != nil {
		return fmt.Errorf("metaengine counter correctness check: %w", err)
	}

	if len(counts) == 0 {
		return fmt.Errorf("%w after %d Apply calls: %w",
			errMEEmptyCounter, sampleCount, ErrMEEvent)
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
