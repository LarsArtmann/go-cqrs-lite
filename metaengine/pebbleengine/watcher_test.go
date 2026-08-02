package pebbleengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// watcherTask is a local value type for cross-engine watcher regression tests.
// It mirrors metaengine's internal testTask but is defined in the engine package
// so the watcher pipeline can be exercised without importing test internals.
type watcherTask struct {
	ID     string
	Title  string
	Status string
}

// TestPebbleWatcher_DeleteNotificationDeliversZeroValue verifies that a Pebble
// engine-backed store notifies watchers with the zero value of V on delete,
// instead of silently dropping the notification. This is a regression test
// for the watcher reification bug where nil.(V) failed for remove operations.
func TestPebbleWatcher_DeleteNotificationDeliversZeroValue(t *testing.T) {
	g := gomega.NewWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	q := metaengine.Query[watcherTask, watcherTask](
		"pebble_watcher_tasks",
		metaengine.OnTyped("task_created", watcherTask{}, func(e watcherTask) (string, watcherTask) {
			return e.ID, e
		}),
		metaengine.OnTyped("task_deleted", watcherTask{}, metaengine.Remove[watcherTask]()),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := metaengine.NewWatcher[watcherTask](store, "pebble_watcher_tasks")
	defer watcher.Close()

	ch := watcher.Watch(ctx, nil)

	g.Expect(store.Apply(ctx, "task_created", watcherTask{ID: "pt-1", Title: "Pebble Task"})).To(gomega.Succeed())

	select {
	case val := <-ch:
		g.Expect(val.ID).To(gomega.Equal("pt-1"))
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Pebble insert watcher notification")
	}

	g.Expect(store.Apply(ctx, "task_deleted", watcherTask{ID: "pt-1"})).To(gomega.Succeed())

	select {
	case val := <-ch:
		g.Expect(val.ID).To(gomega.BeEmpty(), "delete notification should deliver zero value, not drop silently")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Pebble delete watcher notification — pre-fix silent drop bug")
	}
}

// TestPebbleWatcher_WithReplayRecordsTypedValue verifies that the replay
// journal captures a typed value from a Pebble-backed watcher, not seq=0 due
// to a failed type assertion in replayShim.recordValue.
func TestPebbleWatcher_WithReplayRecordsTypedValue(t *testing.T) {
	g := gomega.NewWithT(t)

	eng, err := pebbleengine.NewPebbleEngine("")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer eng.Close()

	q := metaengine.Query[watcherTask, watcherTask](
		"pebble_replay_tasks",
		metaengine.OnTyped("task_created", watcherTask{}, func(e watcherTask) (string, watcherTask) {
			return e.ID, e
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := metaengine.NewWatcher[watcherTask](store, "pebble_replay_tasks")
	replay := watcher.WithReplay(100)
	defer watcher.Close()

	seqCh := watcher.WatchWithSeq(ctx, nil)

	g.Expect(store.Apply(ctx, "task_created", watcherTask{ID: "prt-1", Title: "Replay Task"})).To(gomega.Succeed())

	select {
	case sv := <-seqCh:
		g.Expect(sv.Value.ID).To(gomega.Equal("prt-1"))
		g.Expect(sv.Seq).NotTo(gomega.BeZero(), "replay seq must be recorded, not silently dropped")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Pebble watcher seq notification")
	}

	entries := replay.Replay(0)
	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Value.ID).To(gomega.Equal("prt-1"))
}
