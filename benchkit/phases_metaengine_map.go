package benchkit

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ── Map ADT types (new workload: collection CRUD + query) ──

// meBenchItem is the value type for the Map ADT benchmark. It has queryable
// fields so the Scan benchmark exercises FilterOnField + SortOnField pushdown.
type meBenchItem struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

type meBenchItemCreated struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
}

// meBenchMapQuery declares a Map collection with filter+sort pushdown. The
// planner sees FilterOnField("status") + SortOnField("priority") and can
// generate an index for engines that support LayoutPlanner.
func meBenchMapQuery() metaengine.QueryDecl[struct{}, map[string]meBenchItem] {
	return metaengine.Query[struct{}, map[string]meBenchItem](
		"bench_items",
		metaengine.On(meBenchItemCreated{}, func(e meBenchItemCreated) (string, meBenchItem) {
			return e.ID, meBenchItem{
				ID:       e.ID,
				Status:   e.Status,
				Priority: e.Priority,
			}
		}),
		metaengine.FilterOnField[meBenchItem]("status", metaengine.FilterEq),
		metaengine.SortOnField[meBenchItem]("priority", true),
	)
}

// metaEngineMapWorkload benchmarks the Map ADT — the primary collection
// pattern. Measures:
//   - Apply: insert N items (single-threaded, per-item latency)
//   - Scan: filtered collection read (TypedReader.Scan with WHERE status=active)
//   - PointRead: single-item lookup (TypedReader.Get)
//   - ConcurrentApply: N goroutines writing simultaneously (contention test)
//
// This workload reveals the planner's read-path performance — the real reason
// consumers adopt metaengine instead of hand-rolling projections.
func (r *runner) metaEngineMapWorkload(ctx context.Context) error {
	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, meBenchMapQuery())
	if err != nil {
		return err
	}

	defer store.Close()

	sampleCount := min(r.config.Profile.Streams, maxMetaEngineSamples)
	if sampleCount <= 0 {
		sampleCount = 1
	}

	statuses := []string{"active", "pending", "closed", "archived"}

	// ── Apply: insert N items (single-threaded) ──
	itemIDs := make([]string, sampleCount)

	for i := range sampleCount {
		if ctx.Err() != nil {
			break
		}

		id := fmt.Sprintf("item-%04d", i)
		itemIDs[i] = id

		status := statuses[i%len(statuses)]
		priority := i % 100

		if err := store.Apply(ctx, "MeBenchItemCreated", meBenchItemCreated{
			ID:       id,
			Status:   status,
			Priority: priority,
		}); err != nil {
			return err
		}
	}

	// ── Scan: filtered collection read ──
	reader := metaengine.NewReader[meBenchItem](store, "bench_items")
	scanColl := NewLatencyCollector(0)
	scanCount := min(sampleCount/4, 200) // cap scan iterations

	if scanCount < 1 {
		scanCount = 1
	}

	for range scanCount {
		if ctx.Err() != nil {
			break
		}

		start := time.Now()

		results, scanErr := reader.Scan(
			ctx,
			metaengine.WithFilter("status", metaengine.FilterEq, "active"),
			metaengine.WithLimit(100),
		)
		scanColl.Record(time.Since(start))

		if scanErr != nil {
			return fmt.Errorf("metaengine scan: %w", scanErr)
		}

		// Record scan result count on first iteration for correctness check.
		if r.result.MetaEngineScanResults == 0 {
			r.result.MetaEngineScanResults = len(results)
		}
	}

	r.result.MetaEngineScanLatency = scanColl.Stats()

	// ── PointRead: single-item lookup via TypedReader.Get ──
	pointColl := NewLatencyCollector(0)
	pointCount := min(sampleCount/4, 200)

	if pointCount < 1 {
		pointCount = 1
	}

	for i := range pointCount {
		if ctx.Err() != nil {
			break
		}

		// Look up a real ID (round-robin through inserted items).
		id := itemIDs[i%len(itemIDs)]

		start := time.Now()

		_, found, getErr := reader.Get(ctx, id)
		pointColl.Record(time.Since(start))

		if getErr != nil {
			return fmt.Errorf("metaengine point read: %w", getErr)
		}

		if !found {
			return fmt.Errorf("metaengine point read: item %q not found", id)
		}
	}

	r.result.MetaEnginePointReadLatency = pointColl.Stats()

	// ── Concurrent Apply: contention test ──
	concurrency := r.concurrency
	if concurrency < 2 {
		concurrency = 2
	}

	if concurrency > 8 {
		concurrency = 8 // cap to avoid overwhelming the benchmark
	}

	concurrentCount := min(sampleCount, 500)

	concurrentStart := time.Now()

	err = runConcurrent(
		ctx, concurrentCount, concurrency,
		func(_ context.Context, idx int) error {
			status := statuses[idx%len(statuses)]

			return store.Apply(ctx, "MeBenchItemCreated", meBenchItemCreated{
				ID:       fmt.Sprintf("conc-%06d", idx),
				Status:   status,
				Priority: idx % 100,
			})
		},
	)
	if err != nil {
		return fmt.Errorf("metaengine concurrent apply: %w", err)
	}

	concurrentElapsed := time.Since(concurrentStart).Seconds()

	if concurrentElapsed > 0 {
		r.result.MetaEngineApplyConcurrent = float64(concurrentCount) / concurrentElapsed
	}

	return nil
}
