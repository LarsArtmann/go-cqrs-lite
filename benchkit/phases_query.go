package benchkit

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// maxQueryDispatches caps the number of query-dispatch samples per path so the
// phase stays fast even on large profiles. The dispatcher overhead is stable
// after a few hundred samples.
const maxQueryDispatches = 500

// queryPhase benchmarks typed query dispatch latency against a pre-populated
// read model (M15). Measures three paths:
//   - Hit: registered handler found and invoked (getCountQueryType).
//   - Miss: unregistered type returns handler-not-found (missQueryType).
//   - Paginated: paginated result construction (listCountsQueryType).
//
// Requires ReadModels (kv.Store); gracefully skips otherwise.
func (r *runner) queryPhase(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil //nolint:nilerr // ctx done; graceful skip
	}

	if r.bundle.ReadModels == nil {
		return nil
	}

	store := r.bundle.ReadModels
	profile := r.config.Profile

	// Pre-populate per-stream counters so hit queries read real data.
	// Reuses the first dispatchCount aggregate IDs (already generated in setup).
	dispatchCount := min(profile.Streams, maxQueryDispatches)
	if dispatchCount <= 0 {
		dispatchCount = 1
	}

	streamIDs := r.aggIDs[:dispatchCount]

	for i, sid := range streamIDs {
		if err := writeCount(ctx, store, countKey(sid.String()), uint64(i+1)); err != nil {
			return err
		}
	}

	disp := newBenchQueryDispatcher(store, streamIDs)
	defer disp.Close()

	// ── Hit path ──
	hitColl := NewLatencyCollector(0)

	for i, sid := range streamIDs {
		if ctx.Err() != nil {
			break // ctx done; report partial results
		}

		q := getCountQuery{streamID: sid.String()}

		start := time.Now()

		result, err := query.DispatchTyped[CountResult](ctx, disp, q)

		hitColl.Record(time.Since(start))

		if err != nil {
			r.result.QueryCorrectnessErrors++

			continue
		}

		// Correctness: the pre-populated value is i+1.
		if result.Count != uint64(i+1) {
			r.result.QueryCorrectnessErrors++
		}
	}

	r.result.QueryHitLatency = hitColl.Stats()

	// ── Miss path ──
	missCount := min(dispatchCount, 100)
	missColl := NewLatencyCollector(0)

	for range missCount {
		if ctx.Err() != nil {
			break // ctx done; report partial results
		}

		start := time.Now()

		// Expected error: handler not found. We measure the dispatch overhead
		// of the miss path, not the error itself.
		_, _ = disp.Dispatch(ctx, missQuery{})

		missColl.Record(time.Since(start))
	}

	r.result.QueryMissLatency = missColl.Stats()

	// ── Paginated path ──
	pageColl := NewLatencyCollector(0)
	pageSize := uint(20)

	for page := uint(0); int(page)*int(pageSize) < dispatchCount; page++ {
		if ctx.Err() != nil {
			break // ctx done; report partial results
		}

		q := listCountsQuery{page: page, pageSize: pageSize}

		start := time.Now()

		result, err := query.DispatchTyped[query.PaginatedResult[CountResult]](ctx, disp, q)

		pageColl.Record(time.Since(start))

		if err != nil {
			r.result.QueryCorrectnessErrors++

			continue
		}

		// Correctness: total count should equal dispatchCount.
		if result.TotalCount != uint(dispatchCount) {
			r.result.QueryCorrectnessErrors++
		}
	}

	r.result.QueryPaginatedLatency = pageColl.Stats()

	return nil
}
