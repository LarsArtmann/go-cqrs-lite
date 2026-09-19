package sqliteengine_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newResetTestEngine opens an isolated in-memory database (unique DSN per
// call, single connection) so reset tests never share cached state with the
// ginkgo suite's shared-cache DSN.
func newResetTestEngine(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:reset_"+t.Name()+"?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() {
		_ = eng.Close()
		_ = db.Close()
	})

	return eng, db
}

// ResetEngine must clear EVERY ADT surface, not just the map table: a reset
// that misses one table leaves stale state a replay folds on top of.
func TestResetEngine_ClearsEveryADT(t *testing.T) {
	t.Parallel()

	eng, _ := newResetTestEngine(t)
	ctx := context.Background()
	resetter := eng.(metaengine.EngineResetter)

	mb := eng.(metaengine.MapBackend)
	sb := eng.(metaengine.SetBackend)
	cb := eng.(metaengine.CounterBackend)
	mm := eng.(metaengine.MultimapBackend)
	lb := eng.(metaengine.LogBackend)
	sl := eng.(metaengine.StreamLogBackend)

	if err := mb.MapSet(ctx, "tasks", "t1", map[string]string{"title": "x"}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := sb.SetAdd(ctx, "seen", "t1"); err != nil {
		t.Fatalf("SetAdd: %v", err)
	}

	if err := cb.CounterIncrement(ctx, "counts", metaengine.Delta{"created": 3}); err != nil {
		t.Fatalf("CounterIncrement: %v", err)
	}

	if err := mm.MultiAdd(ctx, "index", "t1", "v1"); err != nil {
		t.Fatalf("MultiAdd: %v", err)
	}

	if err := lb.LogAppend(ctx, "audit", "entry"); err != nil {
		t.Fatalf("LogAppend: %v", err)
	}

	if err := sl.StreamAppend(ctx, "events", "s1", []any{"e1", "e2"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	if err := eng.(interface {
		GraphAddEdge(ctx context.Context, col string, edge metaengine.Edge) error
	}).GraphAddEdge(ctx, "graph", metaengine.Edge{From: "a", To: "b"}); err != nil {
		t.Fatalf("GraphAddEdge: %v", err)
	}

	if err := eng.(interface {
		SnapshotSave(ctx context.Context, collection, streamID string, version int64, data []byte) error
	}).SnapshotSave(ctx, "snaps", "s1", 5, []byte("state")); err != nil {
		t.Fatalf("SnapshotSave: %v", err)
	}

	if err := resetter.ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	if _, ok, err := mb.MapGet(ctx, "tasks", "t1"); err != nil || ok {
		t.Fatalf("map must be empty after reset (ok=%v err=%v)", ok, err)
	}

	if ok, err := sb.SetContains(ctx, "seen", "t1"); err != nil || ok {
		t.Fatalf("set must be empty after reset (ok=%v err=%v)", ok, err)
	}

	counters, err := cb.CounterGet(ctx, "counts")
	if err != nil {
		t.Fatalf("CounterGet: %v", err)
	}

	if len(counters) != 0 {
		t.Fatalf("counters must be empty after reset, got %v", counters)
	}

	values, err := mm.MultiGet(ctx, "index", "t1")
	if err != nil || len(values) != 0 {
		t.Fatalf("multimap must be empty after reset (len=%d err=%v)", len(values), err)
	}

	entries, err := lb.LogTail(ctx, "audit", 10)
	if err != nil || len(entries) != 0 {
		t.Fatalf("log must be empty after reset (len=%d err=%v)", len(entries), err)
	}

	stream, err := sl.StreamRead(ctx, "events", "s1")
	if err != nil || len(stream) != 0 {
		t.Fatalf("stream log must be empty after reset (len=%d err=%v)", len(stream), err)
	}

	neighbors, err := eng.(interface {
		GraphNeighbors(ctx context.Context, col string, node any, depth int) ([]any, error)
	}).GraphNeighbors(ctx, "graph", "a", 1)
	if err != nil {
		t.Fatalf("GraphNeighbors: %v", err)
	}

	if len(neighbors) != 0 {
		t.Fatalf("graph must be empty after reset, got %v", neighbors)
	}

	snap, version, err := eng.(interface {
		SnapshotLoad(ctx context.Context, collection, streamID string) ([]byte, int64, error)
	}).SnapshotLoad(ctx, "snaps", "s1")
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Fatalf("snapshots must be gone after reset, got data=%q v=%d err=%v", snap, version, err)
	}
}

// A reset returns the engine to its POST-PLAN state: planned tables are
// emptied but the layout survives, so writes after the reset still land in
// the planned table (not meta_map).
func TestResetEngine_KeepsPlannedLayout(t *testing.T) {
	t.Parallel()

	eng, db := newResetTestEngine(t)
	ctx := context.Background()

	planner := eng.(metaengine.LayoutPlanner)
	if err := planner.ApplyLayout("tasks", []string{"status"}, []string{"priority"}); err != nil {
		t.Fatalf("ApplyLayout: %v", err)
	}

	mb := eng.(metaengine.MapBackend)
	if err := mb.MapSet(
		ctx,
		"tasks",
		"t1",
		map[string]any{"status": "open", "priority": 1},
	); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	// Layout survived: the planned table exists and is empty; a fresh write
	// routes to it (meta_map stays untouched).
	if err := mb.MapSet(
		ctx,
		"tasks",
		"t2",
		map[string]any{"status": "done", "priority": 2},
	); err != nil {
		t.Fatalf("MapSet after reset: %v", err)
	}

	var (
		plannedRows int
		baseRows    int
	)

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM meta_planned_tasks").
		Scan(&plannedRows); err != nil {
		t.Fatalf("planned table must still exist after reset: %v", err)
	}

	_ = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM meta_map WHERE collection = 'tasks'").Scan(&baseRows)

	if plannedRows != 1 || baseRows != 0 {
		t.Fatalf(
			"post-reset write must land in the planned table: planned=%d meta_map=%d",
			plannedRows,
			baseRows,
		)
	}

	val, ok, err := mb.MapGet(ctx, "tasks", "t1")
	if err != nil || ok || val != nil {
		t.Fatalf("planned rows must be cleared (ok=%v val=%v err=%v)", ok, val, err)
	}
}

// The cached multimap sequence counters are dropped with the data: after a
// reset, appends succeed without PK collisions and ordering stays append-order.
func TestResetEngine_RestartsMultimapSequences(t *testing.T) {
	t.Parallel()

	eng, _ := newResetTestEngine(t)
	ctx := context.Background()

	mm := eng.(metaengine.MultimapBackend)
	for _, v := range []string{"a", "b", "c"} {
		if err := mm.MultiAdd(ctx, "index", "k", v); err != nil {
			t.Fatalf("MultiAdd: %v", err)
		}
	}

	if err := eng.(metaengine.EngineResetter).ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	for _, v := range []string{"x", "y"} {
		if err := mm.MultiAdd(ctx, "index", "k", v); err != nil {
			t.Fatalf("MultiAdd after reset: %v", err)
		}
	}

	got, err := mm.MultiGet(ctx, "index", "k")
	if err != nil {
		t.Fatalf("MultiGet: %v", err)
	}

	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("expected exactly the post-reset appends in order, got %v", got)
	}
}

// Store.Reset must report the sqlite engine as cleared (implements
// EngineResetter) — the one-call revert is now non-partial on the
// production-default engine.
func TestStore_Reset_ClearsSQLiteEngine(t *testing.T) {
	t.Parallel()

	eng, _ := newResetTestEngine(t)
	ctx := context.Background()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, findTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	result, err := store.Reset(ctx)
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if result.Partial() {
		t.Fatalf("sqlite engine implements EngineResetter, reset must not be partial: %s", result)
	}

	if len(result.ClearedEngines) != 1 || result.ClearedEngines[0] != "sqlite" {
		t.Fatalf("expected sqlite in ClearedEngines, got %v", result.ClearedEngines)
	}
}

// TestResetEngine_ClearsClaimkitCollections pins the ADR-0142 write-side
// collections on the ADR-0136 ladder: timers and dedup windows are derived
// data — a reset clears them like every other materialized collection,
// while the stream-log AUTOINCREMENT (journal positions) deliberately keeps
// advancing so pre-reset resumption tokens never skip replayed entries.
func TestResetEngine_ClearsClaimkitCollections(t *testing.T) {
	t.Parallel()

	eng, db := newResetTestEngine(t)
	ctx := context.Background()
	resetter := eng.(metaengine.EngineResetter)

	claimer := eng.(metaengine.DueClaimer)
	if err := claimer.ClaimInsert(
		ctx,
		"timers",
		"t1",
		time.Now().Add(-time.Second),
		[]byte("fire"),
	); err != nil {
		t.Fatalf("ClaimInsert: %v", err)
	}

	dedup := eng.(metaengine.DedupStore)
	if seen, err := dedup.DedupCheckAndRecord(
		ctx,
		"cmds",
		"c1",
		time.Minute,
		time.Now(),
	); err != nil ||
		seen {
		t.Fatalf("DedupCheckAndRecord first (seen=%v err=%v)", seen, err)
	}

	sl := eng.(metaengine.StreamLogBackend)
	if err := sl.StreamAppend(ctx, "events", "s1", []any{"e1"}); err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	preResetSeq := sqliteSeq(t, db, "meta_stream_log")
	if preResetSeq < 1 {
		t.Fatalf("sqlite_sequence must have advanced past the append, got %d", preResetSeq)
	}

	if err := resetter.ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	claims, err := claimer.ClaimDue(ctx, metaengine.ClaimDueRequest{
		Collection: "timers", Owner: "w1", Lease: time.Minute,
	})
	if err != nil || len(claims) != 0 {
		t.Fatalf("claims must be gone after reset (len=%d err=%v)", len(claims), err)
	}

	seen, err := dedup.DedupSeen(ctx, "cmds", "c1", time.Now())
	if err != nil || seen {
		t.Fatalf("dedup must be gone after reset (seen=%v err=%v)", seen, err)
	}

	if seq := sqliteSeq(t, db, "meta_stream_log"); seq < preResetSeq {
		t.Fatalf("journal seq must keep advancing across resets: pre=%d post=%d", preResetSeq, seq)
	}
}

// sqliteSeq reads the AUTOINCREMENT sequence for table from sqlite_sequence.
func sqliteSeq(t *testing.T, db *sql.DB, table string) int64 {
	t.Helper()

	var seq sql.NullInt64
	if err := db.QueryRow("SELECT seq FROM sqlite_sequence WHERE name = ?", table).
		Scan(&seq); err != nil {
		t.Fatalf("read sqlite_sequence(%s): %v", table, err)
	}

	return seq.Int64
}

// TestResetEngine_FactsSurviveReset pins the ADR-0142 §5 journal rung: task
// facts are append-only state-transition records, so a reset clears the
// derived collections (claims, dedup) but NEVER the fact journal — and fact
// positions keep advancing, exactly like stream-log seqs.
func TestResetEngine_FactsSurviveReset(t *testing.T) {
	t.Parallel()

	eng, db := newResetTestEngine(t)
	ctx := context.Background()

	resetter := eng.(metaengine.EngineResetter)
	claimer := eng.(metaengine.DueClaimer)
	sink := eng.(metaengine.FactSink)

	lister, ok := eng.(interface {
		ClaimFactsList(ctx context.Context, collection, key string) ([]metaengine.ClaimFact, error)
	})
	if !ok {
		t.Fatal("sqlite engine must expose claimkit.ClaimFactsList for fact verification")
	}

	now := time.Now()

	if err := claimer.ClaimInsert(
		ctx,
		"tasks",
		"f1",
		now.Add(-time.Second),
		[]byte("x"),
	); err != nil {
		t.Fatalf("ClaimInsert: %v", err)
	}

	if _, err := sink.ClaimDueFacts(ctx, metaengine.ClaimDueRequest{
		Collection: "tasks", Owner: "w", Now: now,
	}, func(metaengine.DueClaim) []metaengine.ClaimFact {
		return []metaengine.ClaimFact{{Type: "claimed"}}
	}); err != nil {
		t.Fatalf("ClaimDueFacts: %v", err)
	}

	before, err := lister.ClaimFactsList(ctx, "tasks", "f1")
	if err != nil || len(before) != 1 {
		t.Fatalf("facts before reset (len=%d err=%v)", len(before), err)
	}

	var maxSeqBefore int64
	if err := db.QueryRow("SELECT COALESCE(MAX(seq), 0) FROM meta_claim_facts").
		Scan(&maxSeqBefore); err != nil {
		t.Fatalf("max fact seq: %v", err)
	}

	if err := resetter.ResetEngine(ctx); err != nil {
		t.Fatalf("ResetEngine: %v", err)
	}

	after, err := lister.ClaimFactsList(ctx, "tasks", "f1")
	if err != nil || len(after) != 1 || after[0].Type != "claimed" {
		t.Fatalf(
			"fact journal must survive reset (len=%d facts=%+v err=%v)",
			len(after),
			after,
			err,
		)
	}

	// Post-reset facts land at strictly higher positions — a consumer holding
	// a pre-reset fact cursor never skips or re-reads.
	if err := claimer.ClaimInsert(
		ctx,
		"tasks",
		"f2",
		now.Add(-time.Second),
		[]byte("y"),
	); err != nil {
		t.Fatalf("ClaimInsert post-reset: %v", err)
	}

	if _, err := sink.ClaimDueFacts(ctx, metaengine.ClaimDueRequest{
		Collection: "tasks", Owner: "w", Now: now,
	}, func(metaengine.DueClaim) []metaengine.ClaimFact {
		return []metaengine.ClaimFact{{Type: "claimed"}}
	}); err != nil {
		t.Fatalf("ClaimDueFacts post-reset: %v", err)
	}

	var maxSeqAfter int64
	if err := db.QueryRow("SELECT COALESCE(MAX(seq), 0) FROM meta_claim_facts").
		Scan(&maxSeqAfter); err != nil {
		t.Fatalf("max fact seq post-reset: %v", err)
	}

	if maxSeqAfter <= maxSeqBefore {
		t.Fatalf("fact positions must keep advancing across reset: before=%d after=%d",
			maxSeqBefore, maxSeqAfter)
	}
}
