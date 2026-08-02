package metaengine

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestWatcher_ReceivesDeleteNotification verifies that remove/delete
// operations notify watchers with the zero value of V, instead of silently
// dropping the notification (the pre-fix behavior where nil.(V) failed).
func TestWatcher_ReceivesDeleteNotification(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()

	deleteQuery := Query[testFindTask, testTask](
		"del_tasks",
		OnTyped("task_created", testTask{}, func(e testTask) (testTaskID, testTask) {
			return e.ID, e
		}),
		OnTyped("task_deleted", testTask{}, Remove[testTask]()),
	)

	store, err := Plan([]Engine{eng}, deleteQuery)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher[testTask](store, "del_tasks")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil)

	// Insert a task.
	_ = store.Apply(ctx, "task_created", testTask{ID: "del-1", Title: "To Delete"})

	// Receive the insert notification.
	select {
	case val := <-ch:
		if val.ID != "del-1" {
			t.Fatalf("expected insert 'del-1', got %s", val.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for insert notification")
	}

	// Delete the task.
	_ = store.Apply(ctx, "task_deleted", testTask{ID: "del-1"})

	// Receive the delete notification — pre-fix this was silently dropped.
	select {
	case val := <-ch:
		if val.ID != "" {
			t.Errorf("expected zero-value testTask on delete, got %+v", val)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for delete notification — was silently dropped (pre-fix bug)")
	}
}

// TestWatcherWithSeq_ReceivesDeleteNotification verifies that WatchWithSeq
// also delivers delete notifications with the zero value.
func TestWatcherWithSeq_ReceivesDeleteNotification(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngine()

	deleteQuery := Query[testFindTask, testTask](
		"del_seq_tasks",
		OnTyped("task_created", testTask{}, func(e testTask) (testTaskID, testTask) {
			return e.ID, e
		}),
		OnTyped("task_deleted", testTask{}, Remove[testTask]()),
	)

	store, err := Plan([]Engine{eng}, deleteQuery)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher[testTask](store, "del_seq_tasks")
	replay := watcher.WithReplay(100)
	defer watcher.Close()

	seqCh := watcher.WatchWithSeq(ctx, nil)

	// Insert.
	_ = store.Apply(ctx, "task_created", testTask{ID: "ds-1", Title: "Insert"})

	select {
	case sv := <-seqCh:
		if sv.Value.ID != "ds-1" {
			t.Fatalf("expected insert 'ds-1', got %s", sv.Value.ID)
		}

		if sv.Seq == 0 {
			t.Error("expected non-zero seq for insert")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for insert seq notification")
	}

	// Delete — should still produce a seq notification with zero value.
	_ = store.Apply(ctx, "task_deleted", testTask{ID: "ds-1"})

	select {
	case sv := <-seqCh:
		if sv.Value.ID != "" {
			t.Errorf("expected zero-value testTask on delete, got %+v", sv.Value)
		}

		if sv.Seq == 0 {
			t.Error("expected non-zero seq for delete")
		}

		if sv.Seq != replay.LatestSeq() {
			t.Errorf("seq mismatch: got %d, latest %d", sv.Seq, replay.LatestSeq())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for delete seq notification — was silently dropped (pre-fix bug)")
	}
}

// TestSQLiteWatcher_ReceivesValue_WithReplay verifies that the SQLite engine
// + replay journal correctly records values. The pre-fix replayShim.recordValue
// used value.(V) which could silently fail and return seq=0.
func TestSQLiteWatcher_ReceivesValue_WithReplay(t *testing.T) {
	t.Parallel()

	store := newSQLiteTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher[testTask](store, "tasks")
	replay := watcher.WithReplay(100)
	defer watcher.Close()

	seqCh := watcher.WatchWithSeq(ctx, nil)

	_ = store.Apply(ctx, "task_created", testTask{ID: "sqlite-replay-1", Title: "Replay Test"})

	select {
	case sv := <-seqCh:
		if sv.Value.ID != "sqlite-replay-1" {
			t.Errorf("expected 'sqlite-replay-1', got %s", sv.Value.ID)
		}

		if sv.Seq == 0 {
			t.Fatal("expected non-zero seq — replayShim.recordValue silently failed (pre-fix bug)")
		}

		if sv.Seq != replay.LatestSeq() {
			t.Errorf("seq mismatch: got %d, latest %d", sv.Seq, replay.LatestSeq())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SQLite watcher seq notification")
	}

	// Verify the replay journal has the recorded entry.
	entries := replay.Replay(0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 replay entry, got %d", len(entries))
	}

	if entries[0].Value.ID != "sqlite-replay-1" {
		t.Errorf("replay entry: expected 'sqlite-replay-1', got %s", entries[0].Value.ID)
	}
}

// TestSQLiteWatcher_ReceivesDeleteNotification verifies that delete
// notifications work with the SQLite engine too.
func TestSQLiteWatcher_ReceivesDeleteNotification(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	defer db.Close()

	eng, err := NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	deleteQuery := Query[testFindTask, testTask](
		"sqlite_del_tasks",
		OnTyped("task_created", testTask{}, func(e testTask) (testTaskID, testTask) {
			return e.ID, e
		}),
		OnTyped("task_deleted", testTask{}, Remove[testTask]()),
	)

	store, err := Plan([]Engine{eng}, deleteQuery)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWatcher[testTask](store, "sqlite_del_tasks")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil)

	// Insert.
	_ = store.Apply(ctx, "task_created", testTask{ID: "sdel-1", Title: "SQLite Delete"})

	select {
	case val := <-ch:
		if val.ID != "sdel-1" {
			t.Fatalf("expected insert 'sdel-1', got %s", val.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SQLite insert notification")
	}

	// Delete.
	_ = store.Apply(ctx, "task_deleted", testTask{ID: "sdel-1"})

	// Pre-fix: nil.(testTask) fails silently, notification dropped.
	select {
	case val := <-ch:
		if val.ID != "" {
			t.Errorf("expected zero-value testTask on delete, got %+v", val)
		}
	case <-time.After(2 * time.Second):
		t.Fatal(
			"timeout waiting for SQLite delete notification — was silently dropped (pre-fix bug)",
		)
	}
}

// TestReifyWatcherValue_NilReturnsZero verifies that nil values (from delete
// operations) produce the zero value of V instead of (zero, false).
func TestReifyWatcherValue_NilReturnsZero(t *testing.T) {
	t.Parallel()

	v, ok := reifyWatcherValue[testTask](nil)
	if !ok {
		t.Fatal("expected ok=true for nil input (delete notification)")
	}

	if v.ID != "" || v.Title != "" || v.Status != "" {
		t.Errorf("expected zero testTask, got %+v", v)
	}
}

// TestReifyWatcherValue_TypedFastPath verifies that values already of type V
// are returned directly without a JSON round-trip (fast path).
func TestReifyWatcherValue_TypedFastPath(t *testing.T) {
	t.Parallel()

	original := testTask{ID: "fp-1", Title: "Fast Path", Status: "open"}
	v, ok := reifyWatcherValue[testTask](original)
	if !ok {
		t.Fatal("expected ok=true for typed input")
	}

	if v != original {
		t.Errorf("expected identical value (fast path), got %+v", v)
	}
}

// TestReifyWatcherValue_MapStringAnyFallback verifies that map[string]any
// values (as produced by SQLite JSON decode) are reified to V via JSON
// round-trip instead of silently failing the type assertion.
func TestReifyWatcherValue_MapStringAnyFallback(t *testing.T) {
	t.Parallel()

	// Simulate what SQLite engine produces: map[string]any from json.Unmarshal.
	raw := map[string]any{
		"ID":     "map-1",
		"Title":  "From Map",
		"Status": "active",
	}

	v, ok := reifyWatcherValue[testTask](raw)
	if !ok {
		t.Fatal("expected ok=true for map[string]any input (reify fallback)")
	}

	if v.ID != "map-1" {
		t.Errorf("expected ID 'map-1', got %s", v.ID)
	}

	if v.Title != "From Map" {
		t.Errorf("expected Title 'From Map', got %s", v.Title)
	}

	if v.Status != "active" {
		t.Errorf("expected Status 'active', got %s", v.Status)
	}
}

// TestReifyWatcherValue_IncompatibleTypeReturnsFalse verifies that a genuinely
// incompatible value returns (zero, false) — not a panic.
func TestReifyWatcherValue_IncompatibleTypeReturnsFalse(t *testing.T) {
	t.Parallel()

	// chan is not JSON-serializable, so reify will fail.
	bad := make(chan int)

	v, ok := reifyWatcherValue[testTask](bad)
	if ok {
		t.Fatal("expected ok=false for incompatible type")
	}

	if v.ID != "" {
		t.Errorf("expected zero testTask on failure, got %+v", v)
	}
}

// TestReifyWatcherValue_JSONValueFastPath verifies that a jsonValue wrapper
// (raw JSON bytes from a SQL engine) is decoded directly to V without an
// intermediate map[string]any round-trip. This pins the fast-path inside
// reify[R] that watcher notifications rely on.
func TestReifyWatcherValue_JSONValueFastPath(t *testing.T) {
	t.Parallel()

	jv := jsonValue(`{"ID":"jv-1","Title":"JSON Value","Status":"active"}`)

	v, ok := reifyWatcherValue[testTask](jv)
	if !ok {
		t.Fatal("expected ok=true for jsonValue input (fast-path decode)")
	}

	if v.ID != "jv-1" {
		t.Errorf("expected ID 'jv-1', got %s", v.ID)
	}

	if v.Title != "JSON Value" {
		t.Errorf("expected Title 'JSON Value', got %s", v.Title)
	}

	if v.Status != "active" {
		t.Errorf("expected Status 'active', got %s", v.Status)
	}
}

// TestWorkloadMeter_ReificationFailures pins the counter that Watch/WatchWithSeq
// increment whenever a watcher value cannot be reified to V. A non-zero count
// indicates an engine bug or a planned-type/stored-shape mismatch.
func TestWorkloadMeter_ReificationFailures(t *testing.T) {
	t.Parallel()

	m := newWorkloadMeter()

	if got := m.ReificationFailures(); got != 0 {
		t.Fatalf("expected 0 reification failures initially, got %d", got)
	}

	m.IncReificationFailure()
	m.IncReificationFailure()

	if got := m.ReificationFailures(); got != 2 {
		t.Fatalf("expected 2 reification failures, got %d", got)
	}

	// The counter is independent of workload-rate stats.
	stats := m.Stats()
	if stats.WriteRatePerSec != 0 || stats.ReadRatePerSec != 0 {
		t.Errorf("expected zero read/write rates, got write=%f read=%f",
			stats.WriteRatePerSec, stats.ReadRatePerSec)
	}
}
