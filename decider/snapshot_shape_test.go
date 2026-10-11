package decider_test

import (
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// stampSnapshot plants a snapshot with an explicit StateShape — the
// hand-planted stand-in for a snapshot written by a previous binary whose
// State shape differed. A stale-shaped snapshot carries a WRONG value on
// purpose: if the repository ever loads it, the loaded state lies.
func stampSnapshot(
	t *testing.T,
	snapshotStore *eventtest.FakeSnapshotStore,
	codec codec.Codec,
	streamID id.StreamID,
	value int,
	version event.Version,
	shape string,
) {
	t.Helper()

	snap := makeSnapshot(t, codec, streamID, value, version)
	snap.StateShape = shape
	snapshotStore.SetSnapshot(&snap)
}

// TestSnapshotStateVersion_MismatchDiscardsAndRebuilds is the core T4b
// behavior: a declared state version plus a stored snapshot stamped with a
// DIFFERENT version must discard the snapshot (never decode its bytes into
// State) and rebuild from the journal — observable as the correct folded
// state plus a counted discard.
func TestSnapshotStateVersion_MismatchDiscardsAndRebuilds(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()
	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
		makeEvent(t, "CounterIncremented", streamID, 2),
		makeEvent(t, "CounterIncremented", streamID, 3),
	})

	// Stale-shaped snapshot: stamped "1" while the repository declares "2",
	// and carrying a poisoned value that must never surface.
	stampSnapshot(t, snapshotStore, codec, streamID, 99, event.Version(2), "1")

	repo := newCounterSnapshotRepo(
		t, store, bus, snapshotStore, codec,
		decider.WithSnapshotStateVersion[counterState]("2"),
	)

	requireLoadState(t, repo, streamID, 3, 3)

	if got := repo.SnapshotShapeDiscards(); got != 1 {
		t.Fatalf("expected 1 shape discard, got %d", got)
	}
}

// TestSnapshotStateVersion_MatchingStampLoads pins that a MATCHING stamp
// keeps the fast path: the snapshot decodes and only post-snapshot events
// fold on top. No discard is counted.
func TestSnapshotStateVersion_MatchingStampLoads(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()
	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
		makeEvent(t, "CounterIncremented", streamID, 2),
		makeEvent(t, "CounterIncremented", streamID, 3),
	})

	// Consistent snapshot: value 2 after two events, stamped with the
	// declared version.
	stampSnapshot(t, snapshotStore, codec, streamID, 2, event.Version(2), "2")

	repo := newCounterSnapshotRepo(
		t, store, bus, snapshotStore, codec,
		decider.WithSnapshotStateVersion[counterState]("2"),
	)

	requireLoadState(t, repo, streamID, 3, 3)

	if got := repo.SnapshotShapeDiscards(); got != 0 {
		t.Fatalf("expected 0 shape discards, got %d", got)
	}
}

// TestSnapshotStateVersion_UnstampedAccepted pins absent = accept: a
// snapshot written before the stamp existed loads normally under a
// repository that declares a version.
func TestSnapshotStateVersion_UnstampedAccepted(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	streamID := id.NewStreamID()
	mustAppendBatch(t, store, "Counter", streamID, []event.Event{
		makeEvent(t, "CounterCreated", streamID, 1),
		makeEvent(t, "CounterIncremented", streamID, 2),
		makeEvent(t, "CounterIncremented", streamID, 3),
	})

	saveSnapshot(t, snapshotStore, codec, streamID, 2, event.Version(2))

	repo := newCounterSnapshotRepo(
		t, store, bus, snapshotStore, codec,
		decider.WithSnapshotStateVersion[counterState]("2"),
	)

	requireLoadState(t, repo, streamID, 3, 3)

	if got := repo.SnapshotShapeDiscards(); got != 0 {
		t.Fatalf("expected 0 shape discards on unstamped accept, got %d", got)
	}
}

// TestSnapshotStateVersion_SaveStamps declares "2", lets the strategy save a
// snapshot, and asserts the saved snapshot carries the stamp — the write-side
// half without which the read-side guard could never fire on fresh data.
func TestSnapshotStateVersion_SaveStamps(t *testing.T) {
	t.Parallel()

	store, bus, snapshotStore, codec := newSnapshotSetup(t)

	everyTwo, err := snapshot.EveryNEvents(2)
	if err != nil {
		t.Fatalf("EveryNEvents: %v", err)
	}

	repo := newCounterSnapshotRepo(
		t, store, bus, snapshotStore, codec,
		decider.WithSnapshotStateVersion[counterState]("2"),
		decider.WithSnapshotStrategy[counterState](everyTwo),
	)

	streamID := id.NewStreamID()

	err = executeAndIncrement(t, repo, streamID, "CounterCreated")
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	err = executeAndIncrement(t, repo, streamID, "CounterIncremented")
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}

	saved := snapshotStore.Saved()
	if len(saved) == 0 {
		t.Fatal("expected snapshot to be saved")
	}

	if saved[len(saved)-1].StateShape != "2" {
		t.Fatalf("saved snapshot StateShape = %q, want \"2\"", saved[len(saved)-1].StateShape)
	}
}
