package metaengine_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// These benchmarks cover the metaengine SQLite engine's PLANNER PATH
// end-to-end: Plan → Apply (fold dispatch → backend write) → Execute
// (backend read → reify). The existing calibration benchmarks only measure
// raw backend ops (MapSet/MapGet); none exercised the Store dispatch layer
// that real consumers hit. See ADR-0061 (SQLite engine), ADR-0067
// (tx-atomic MapUpdate), ADR-0066 (ExecuteTyped reify).
//
// They live in metaengine (not cmd/cqrs-bench) deliberately: cqrs-bench's
// Factory returns a *stack.Bundle and models an event-store write/read
// workload, whereas the metaengine is an ADT planner with a fundamentally
// different abstraction. Forcing it into a cqrs-bench profile would produce a
// meaningless benchmark; the real coverage gap is this dispatch-path bench.

// newSQLiteEngineBench opens an in-memory SQLite database and wraps it in a
// metaengine engine. The returned cleanup closes the *sql.DB (the engine's
// own Close is a no-op).
func newSQLiteEngineBench(tb testing.TB) (metaengine.Engine, func()) {
	tb.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		tb.Fatalf("sql.Open: %v", err)
	}

	eng, err := metaengine.NewSQLiteEngine(db)
	if err != nil {
		_ = db.Close()
		tb.Fatalf("NewSQLiteEngine: %v", err)
	}

	return eng, func() { _ = db.Close() }
}

// populateTasks applies n TaskCreated events through the store, exercising the
// Map fold (FoldInsert → MapSet) and — when the counter query is registered —
// the Counter fold (CounterIncrement).
func populateTasks(tb testing.TB, store *metaengine.Store, n int) {
	tb.Helper()
	ctx := context.Background()

	for i := 0; i < n; i++ {
		if err := store.Apply(ctx, "TaskCreated", TaskCreated{
			ID:       TaskID(fmt.Sprintf("t-%d", i)),
			Title:    "bench task",
			Assignee: UserID("u-1"),
			Status:   "open",
			Priority: i % 10,
			At:       time.Unix(int64(i), 0),
		}); err != nil {
			tb.Fatalf("populate Apply %d: %v", i, err)
		}
	}
}

// --- Write path ---

// BenchmarkSQLitePlanner_ApplyInsert measures the full Apply dispatch for a
// FoldInsert (MapSet) on the SQLite engine: Store.Apply → fold selection →
// MapBackend.MapSet → INSERT. This is the per-event write cost consumers pay.
func BenchmarkSQLitePlanner_ApplyInsert(b *testing.B) {
	eng, cleanup := newSQLiteEngineBench(b)
	defer cleanup()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		b.Fatalf("Plan: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := store.Apply(ctx, "TaskCreated", TaskCreated{
			ID:       TaskID(fmt.Sprintf("t-%d", i)),
			Title:    "bench task",
			Assignee: UserID("u-1"),
			Status:   "open",
			Priority: i % 10,
			At:       now,
		}); err != nil {
			b.Fatalf("Apply %d: %v", i, err)
		}
	}
}

// BenchmarkSQLitePlanner_ApplyUpdate measures the full Apply dispatch for a
// FoldUpdate on the SQLite engine: Store.Apply → MapUpdater.MapUpdate → the
// transactional read-modify-write (ADR-0067). Updating an existing key is the
// worst-case write path (BEGIN tx; SELECT; INSERT; COMMIT).
func BenchmarkSQLitePlanner_ApplyUpdate(b *testing.B) {
	eng, cleanup := newSQLiteEngineBench(b)
	defer cleanup()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		b.Fatalf("Plan: %v", err)
	}

	const tasks = 1000
	populateTasks(b, store, tasks)

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := store.Apply(ctx, "TaskCompleted", TaskCompleted{
			ID: TaskID(fmt.Sprintf("t-%d", i%tasks)),
			At: now,
		}); err != nil {
			b.Fatalf("Apply update %d: %v", i, err)
		}
	}
}

// --- Read path ---

// BenchmarkSQLitePlanner_ExecutePointLookup measures a point-lookup query
// (ReadPointLookup → MapGet → reify) on the SQLite engine, the most common
// read pattern.
func BenchmarkSQLitePlanner_ExecutePointLookup(b *testing.B) {
	eng, cleanup := newSQLiteEngineBench(b)
	defer cleanup()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		b.Fatalf("Plan: %v", err)
	}

	const tasks = 1000
	populateTasks(b, store, tasks)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := metaengine.ExecuteTyped[FindTask, FindTaskResult](
			ctx, store, FindTask{ID: TaskID(fmt.Sprintf("t-%d", i%tasks))},
		); err != nil {
			b.Fatalf("ExecuteTyped %d: %v", i, err)
		}
	}
}

// BenchmarkSQLitePlanner_ExecuteScan measures a filtered+sorted scan
// (ReadFilteredScan → MapScan → Go-side filter+sort) on the SQLite engine.
// This is the path ADR-0063 demoted to O(NlogN) until sort-column pushdown.
func BenchmarkSQLitePlanner_ExecuteScan(b *testing.B) {
	eng, cleanup := newSQLiteEngineBench(b)
	defer cleanup()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, listTasksByStatusQuery())
	if err != nil {
		b.Fatalf("Plan: %v", err)
	}

	const tasks = 1000
	populateTasks(b, store, tasks)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := metaengine.ExecuteTyped[ListTasksByStatus, ListTasksByStatusResult](
			ctx, store, ListTasksByStatus{Status: "open", Limit: 50},
		); err != nil {
			b.Fatalf("ExecuteTyped scan %d: %v", i, err)
		}
	}
}

// --- End-to-end ---

// BenchmarkSQLitePlanner_EndToEnd measures the realistic steady-state
// dispatch cost: one Apply (write) + one Execute (read) per iteration, both
// routed through the Store's query runtime on the SQLite engine. This is the
// single benchmark that proves the whole planner path is wired and bounded.
func BenchmarkSQLitePlanner_EndToEnd(b *testing.B) {
	eng, cleanup := newSQLiteEngineBench(b)
	defer cleanup()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		b.Fatalf("Plan: %v", err)
	}

	const tasks = 1000
	populateTasks(b, store, tasks)

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.Apply(ctx, "TaskCompleted", TaskCompleted{
			ID: TaskID(fmt.Sprintf("t-%d", i%tasks)),
			At: now,
		})
		_, _ = metaengine.ExecuteTyped[FindTask, FindTaskResult](
			ctx, store, FindTask{ID: TaskID(fmt.Sprintf("t-%d", i%tasks))},
		)
	}
}

// BenchmarkMemoryPlanner_EndToEnd is the in-memory engine equivalent of the
// SQLite end-to-end benchmark, for relative comparison of the dispatch
// overhead minus disk/SQL I/O.
func BenchmarkMemoryPlanner_EndToEnd(b *testing.B) {
	eng := metaengine.NewMemoryEngine()
	defer func() { _ = eng.Close() }()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		b.Fatalf("Plan: %v", err)
	}

	const tasks = 1000
	populateTasks(b, store, tasks)

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = store.Apply(ctx, "TaskCompleted", TaskCompleted{
			ID: TaskID(fmt.Sprintf("t-%d", i%tasks)),
			At: now,
		})
		_, _ = metaengine.ExecuteTyped[FindTask, FindTaskResult](
			ctx, store, FindTask{ID: TaskID(fmt.Sprintf("t-%d", i%tasks))},
		)
	}
}
