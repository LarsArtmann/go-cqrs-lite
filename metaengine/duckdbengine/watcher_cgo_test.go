//go:build cgo

package duckdbengine_test

import (
	"context"
	"testing"
	"time"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// watcherTaskID is a distinct key type so Remove[V]() can identify the key
// field unambiguously.
type watcherTaskID string

// watcherTask is a local value type for cross-engine watcher regression tests.
type watcherTask struct {
	ID     watcherTaskID
	Title  string
	Status string
}

// TestDuckDBWatcher_DeleteNotificationDeliversZeroValue verifies that a DuckDB
// engine-backed store notifies watchers with the zero value of V on delete,
// instead of silently dropping the notification. This exercises the reify
// fallback path because DuckDB returns values as map[string]any.
func TestDuckDBWatcher_DeleteNotificationDeliversZeroValue(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	q := metaengine.Query[watcherTask, watcherTask](
		"duckdb_watcher_tasks",
		metaengine.OnTyped(
			"task_created",
			watcherTask{},
			func(e watcherTask) (watcherTaskID, watcherTask) {
				return e.ID, e
			},
		),
		metaengine.OnTyped("task_deleted", watcherTask{}, metaengine.Remove[watcherTask]()),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := metaengine.NewWatcher[watcherTask](store, "duckdb_watcher_tasks")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil)

	if err := store.Apply(
		ctx,
		"task_created",
		watcherTask{ID: watcherTaskID("dt-1"), Title: "DuckDB Task"},
	); err != nil {
		t.Fatalf("apply create: %v", err)
	}

	select {
	case val := <-ch:
		if val.ID != watcherTaskID("dt-1") {
			t.Fatalf("expected insert 'dt-1', got %s", val.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for DuckDB insert watcher notification")
	}

	if err := store.Apply(ctx, "task_deleted", watcherTask{ID: watcherTaskID("dt-1")}); err != nil {
		t.Fatalf("apply delete: %v", err)
	}

	select {
	case val := <-ch:
		if val.ID != "" {
			t.Errorf("expected zero-value watcherTask on delete, got %+v", val)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for DuckDB delete watcher notification — pre-fix silent drop bug")
	}
}

// TestDuckDBWatcher_WithReplayRecordsTypedValue verifies that the replay
// journal captures a typed value from a DuckDB-backed watcher, not seq=0 due
// to a failed type assertion in replayShim.recordValue.
func TestDuckDBWatcher_WithReplayRecordsTypedValue(t *testing.T) {
	t.Parallel()

	eng, err := duckdbengine.New("")
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer eng.Close()

	q := metaengine.Query[watcherTask, watcherTask](
		"duckdb_replay_tasks",
		metaengine.OnTyped(
			"task_created",
			watcherTask{},
			func(e watcherTask) (watcherTaskID, watcherTask) {
				return e.ID, e
			},
		),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := metaengine.NewWatcher[watcherTask](store, "duckdb_replay_tasks")
	replay := watcher.WithReplay(100)
	defer watcher.Close()

	seqCh := watcher.WatchWithSeq(ctx, nil)

	if err := store.Apply(
		ctx,
		"task_created",
		watcherTask{ID: watcherTaskID("drt-1"), Title: "DuckDB Replay"},
	); err != nil {
		t.Fatalf("apply create: %v", err)
	}

	select {
	case sv := <-seqCh:
		if sv.Value.ID != watcherTaskID("drt-1") {
			t.Errorf("expected 'drt-1', got %s", sv.Value.ID)
		}
		if sv.Seq == 0 {
			t.Fatal("expected non-zero seq — replayShim.recordValue silently failed (pre-fix bug)")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for DuckDB watcher seq notification")
	}

	entries := replay.Replay(0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 replay entry, got %d", len(entries))
	}
	if entries[0].Value.ID != watcherTaskID("drt-1") {
		t.Errorf("replay entry: expected 'drt-1', got %s", entries[0].Value.ID)
	}
}
