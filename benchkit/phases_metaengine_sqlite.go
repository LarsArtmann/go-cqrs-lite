package benchkit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

// metaEngineSQLiteWorkload runs the same Map ADT workload as
// metaEngineMapWorkload, but against a SQLite engine. This gives a direct
// Memory-vs-SQLite comparison:
//
//   - Memory engine: measures planner+fold overhead with zero I/O
//   - SQLite engine: measures SQL query execution + json_extract cost
//
// The SQLite engine exercises PushdownScan (WHERE status=... pushed to SQL)
// which is the planner's primary value proposition — the whole reason
// consumers adopt metaengine instead of hand-rolling in-memory maps.
//
// Uses a smaller sample count than the Memory workload to keep the phase
// fast. SQLite writes are ~10x slower than memory writes.
func (r *runner) metaEngineSQLiteWorkload(ctx context.Context) error {
	dbName := fmt.Sprintf("bench_me_%d", time.Now().UnixNano())

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName))
	if err != nil {
		return fmt.Errorf("metaengine sqlite: open db: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite serializes writes
	defer db.Close()

	eng, err := sqliteengine.NewSQLiteEngine(db) //nolint:contextcheck // no ctx param
	if err != nil {
		return fmt.Errorf("metaengine sqlite: create engine: %w", err)
	}

	store, err := metaengine.Plan([]metaengine.Engine{eng}, meBenchMapQuery())
	if err != nil {
		return fmt.Errorf("metaengine sqlite: plan: %w", err)
	}

	defer store.Close()

	// Use fewer samples for SQLite — it's ~10x slower than memory.
	sqliteSamples := min(r.config.Profile.Streams, maxMetaEngineSamples/4)
	sqliteSamples = max(sqliteSamples, 10)

	statuses := []string{"active", "pending", "closed", "archived"}

	// ── Apply: insert N items ──
	itemIDs := make([]string, sqliteSamples)

	for i := range sqliteSamples {
		if ctx.Err() != nil {
			break
		}

		id := fmt.Sprintf("sql-item-%04d", i)
		itemIDs[i] = id

		status := statuses[i%len(statuses)]
		priority := i % 100

		if err := store.Apply(ctx, "meBenchItemCreated", meBenchItemCreated{
			ID:       id,
			Status:   status,
			Priority: priority,
		}); err != nil {
			return fmt.Errorf("metaengine sqlite apply: %w", err)
		}
	}

	// Correctness check: verify the store has data.
	reader := metaengine.NewReader[meBenchItem](store, "bench_items")

	_, found, getErr := reader.Get(ctx, itemIDs[0])
	if getErr != nil {
		return fmt.Errorf("metaengine sqlite correctness check: %w", getErr)
	}

	if !found {
		return fmt.Errorf("%w: item %q: %w", errMEPointMiss, itemIDs[0], ErrMEEvent)
	}

	// ── Scan: filtered collection read ──
	scanColl := NewLatencyCollector(0)
	scanCount := min(sqliteSamples/4, 50)
	scanCount = max(scanCount, 1)

	for range scanCount {
		if ctx.Err() != nil {
			break
		}

		start := time.Now()

		_, scanErr := reader.Scan(
			ctx,
			metaengine.WithFilter("status", metaengine.FilterEq, "active"),
			metaengine.WithLimit(100),
		)

		scanColl.Record(time.Since(start))

		if scanErr != nil {
			return fmt.Errorf("metaengine sqlite scan: %w", scanErr)
		}
	}

	r.result.MetaEngineSQLiteScanLatency = scanColl.Stats()

	// ── PointRead: single-item lookup ──
	pointColl := NewLatencyCollector(0)
	pointCount := min(sqliteSamples/4, 50)
	pointCount = max(pointCount, 1)

	for i := range pointCount {
		if ctx.Err() != nil {
			break
		}

		id := itemIDs[i%len(itemIDs)]

		start := time.Now()

		_, _, getErr := reader.Get(ctx, id)

		pointColl.Record(time.Since(start))

		if getErr != nil {
			return fmt.Errorf("metaengine sqlite point read: %w", getErr)
		}
	}

	r.result.MetaEngineSQLitePointReadLatency = pointColl.Stats()

	// ── Apply throughput ──
	applyColl := NewLatencyCollector(0)
	applyStart := time.Now()

	throughputSamples := min(sqliteSamples, 200)

	for i := range throughputSamples {
		if ctx.Err() != nil {
			break
		}

		status := statuses[i%len(statuses)]

		start := time.Now()

		if err := store.Apply(ctx, "meBenchItemCreated", meBenchItemCreated{
			ID:       fmt.Sprintf("sql-tput-%04d", i),
			Status:   status,
			Priority: i % 100,
		}); err != nil {
			return fmt.Errorf("metaengine sqlite throughput apply: %w", err)
		}

		applyColl.Record(time.Since(start))
	}

	applyElapsed := time.Since(applyStart).Seconds()

	if applyElapsed > 0 {
		r.result.MetaEngineSQLiteApplyThroughput = float64(applyColl.Stats().Count) / applyElapsed
	}

	return nil
}
