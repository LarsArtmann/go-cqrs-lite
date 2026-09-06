package bboltengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// watcherTaskID is a distinct key type so Remove[V]() can unambiguously
// identify the key field in the event struct.
type watcherTaskID string

// watcherTask is a local value type for cross-engine watcher regression tests.
type watcherTask struct {
	ID     watcherTaskID
	Title  string
	Status string
}

// TestBboltWatcher_DeleteNotificationDeliversZeroValue verifies that a bbolt
// engine-backed store notifies watchers with the zero value of V on delete,
// instead of silently dropping the notification. Regression test for the
// watcher reification bug where nil.(V) failed for remove operations.
func TestBboltWatcher_DeleteNotificationDeliversZeroValue(t *testing.T) {
	g := gomega.NewWithT(t)

	eng := mustNewBboltEngine(t)

	q := metaengine.Query[watcherTask, watcherTask](
		"bbolt_watcher_tasks",
		metaengine.OnRecordTyped(
			"task_created",
			watcherTask{},
			func(_ record.Record, e watcherTask) (watcherTaskID, watcherTask) {
				return e.ID, e
			},
		),
		metaengine.OnRecordTyped("task_deleted", watcherTask{}, metaengine.Remove[watcherTask]()),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := metaengine.NewWatcher[watcherTask](store, "bbolt_watcher_tasks")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil)

	g.Expect(store.Apply(ctx, "task_created", watcherTask{ID: watcherTaskID("bt-1"), Title: "Bbolt Task"})).
		To(gomega.Succeed())

	select {
	case val := <-ch:
		g.Expect(val.ID).To(gomega.Equal(watcherTaskID("bt-1")))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for bbolt insert watcher notification")
	}

	g.Expect(store.Apply(ctx, "task_deleted", watcherTask{ID: watcherTaskID("bt-1")})).
		To(gomega.Succeed())

	select {
	case val := <-ch:
		g.Expect(val.ID).
			To(gomega.BeEmpty(), "delete notification should deliver zero value, not drop silently")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for bbolt delete watcher notification — pre-fix silent drop bug")
	}
}

// TestBboltWatcher_WithReplayRecordsTypedValue verifies that the replay
// journal captures a typed value from a bbolt-backed watcher, not seq=0 due
// to a failed type assertion in replayShim.recordValue.
func TestBboltWatcher_WithReplayRecordsTypedValue(t *testing.T) {
	g := gomega.NewWithT(t)

	eng := mustNewBboltEngine(t)

	q := metaengine.Query[watcherTask, watcherTask](
		"bbolt_replay_tasks",
		metaengine.OnRecordTyped(
			"task_created",
			watcherTask{},
			func(_ record.Record, e watcherTask) (watcherTaskID, watcherTask) {
				return e.ID, e
			},
		),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := metaengine.NewWatcher[watcherTask](store, "bbolt_replay_tasks")
	replay := watcher.WithReplay(100)
	defer watcher.Close()

	seqCh := watcher.WatchWithSeq(ctx, nil)

	g.Expect(store.Apply(ctx, "task_created", watcherTask{ID: watcherTaskID("brt-1"), Title: "Replay Task"})).
		To(gomega.Succeed())

	select {
	case sv := <-seqCh:
		g.Expect(sv.Value.ID).To(gomega.Equal(watcherTaskID("brt-1")))
		g.Expect(sv.Seq).NotTo(gomega.BeZero(), "replay seq must be recorded, not silently dropped")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for bbolt watcher seq notification")
	}

	entries := replay.Replay(0)
	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Value.ID).To(gomega.Equal(watcherTaskID("brt-1")))
}
