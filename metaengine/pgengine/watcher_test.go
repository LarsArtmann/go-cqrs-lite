package pgengine_test

import (
	"context"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
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

// TestPostgresWatcher_DeleteNotificationDeliversZeroValue verifies that a
// Postgres engine-backed store notifies watchers with the zero value of V on
// delete, instead of silently dropping the notification. This exercises the
// reify fallback path because Postgres returns values as map[string]any.
func TestPostgresWatcher_DeleteNotificationDeliversZeroValue(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	q := metaengine.Query[watcherTask, watcherTask](
		"pg_watcher_tasks",
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

	watcher := metaengine.NewWatcher[watcherTask](store, "pg_watcher_tasks")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil)

	if err := store.Apply(
		ctx,
		"task_created",
		watcherTask{ID: watcherTaskID("pt-1"), Title: "Postgres Task"},
	); err != nil {
		t.Fatalf("apply create: %v", err)
	}

	select {
	case val := <-ch:
		if val.ID != watcherTaskID("pt-1") {
			t.Fatalf("expected insert 'pt-1', got %s", val.ID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for Postgres insert watcher notification")
	}

	if err := store.Apply(ctx, "task_deleted", watcherTask{ID: watcherTaskID("pt-1")}); err != nil {
		t.Fatalf("apply delete: %v", err)
	}

	select {
	case val := <-ch:
		if val.ID != "" {
			t.Errorf("expected zero-value watcherTask on delete, got %+v", val)
		}
	case <-time.After(5 * time.Second):
		t.Fatal(
			"timeout waiting for Postgres delete watcher notification — pre-fix silent drop bug",
		)
	}
}

// TestPostgresWatcher_WithReplayRecordsTypedValue verifies that the replay
// journal captures a typed value from a Postgres-backed watcher, not seq=0 due
// to a failed type assertion in replayShim.recordValue.
func TestPostgresWatcher_WithReplayRecordsTypedValue(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	enginetest.RunWatcherReplayTest[watcherTask](
		t, eng,
		enginetest.WatcherReplaySetup[watcherTask]{
			Collection: "pg_replay_tasks",
			Build: func(t *testing.T, eng metaengine.Engine) (*metaengine.Store, *metaengine.Watcher[watcherTask]) {
				q := metaengine.Query[watcherTask, watcherTask](
					"pg_replay_tasks",
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
				watcher := metaengine.NewWatcher[watcherTask](store, "pg_replay_tasks")
				return store, watcher
			},
			Apply: func(ctx context.Context, store *metaengine.Store, payload watcherTask) error {
				return store.Apply(ctx, "task_created", payload)
			},
		},
		watcherTask{ID: watcherTaskID("prt-1"), Title: "Postgres Replay"},
		"prt-1",
		5*time.Second,
	)
}
