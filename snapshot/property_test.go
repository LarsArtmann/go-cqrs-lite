package snapshot_test

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// Property tests for snapshot.TypedStore[State] — Save/Load round-trip fidelity.
// Mirrors the kv property-test pattern (rapid.Check per codec).

type testState struct {
	Count int
	Label string
	Flags [8]byte
}

func genState() *rapid.Generator[testState] {
	return rapid.Custom(func(t *rapid.T) testState {
		var flags [8]byte
		for i := range flags {
			flags[i] = rapid.Byte().Draw(t, "flag")
		}

		return testState{
			Count: rapid.IntRange(0, 1_000_000).Draw(t, "count"),
			Label: rapid.StringN(1, 20, 100).Draw(t, "label"),
			Flags: flags,
		}
	})
}

func genVersion() *rapid.Generator[event.Version] {
	return rapid.Custom(func(t *rapid.T) event.Version {
		return event.Version(rapid.IntRange(0, 100000).Draw(t, "version"))
	})
}

// TestProperty_SaveLoadRoundTrip — Save then Load returns the same state.
func TestProperty_SaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := snapshot.NewTypedStore[testState](newFakeStore(), nil) // nil → CBOR
		ctx := context.Background()

		state := genState().Draw(rt, "state")
		ver := genVersion().Draw(rt, "version")
		streamID := id.NewStreamID()
		streamType := id.StreamType("TestAggregate")
		ref := id.NewStreamRef(streamType, streamID)

		snap := snapshot.TypedSnapshot[testState]{
			StreamID:   streamID,
			StreamType: streamType,
			Version:    ver,
			State:      state,
			CreatedAt:  time.Now().UTC(),
		}

		err := store.Save(ctx, snap)
		if err != nil {
			rt.Fatalf("Save failed: %v", err)
		}

		got, err := store.Load(ctx, ref)
		if err != nil {
			rt.Fatalf("Load failed: %v", err)
		}

		if got.State != state {
			rt.Fatalf("state mismatch: saved %+v, loaded %+v", state, got.State)
		}

		if got.Version != ver {
			rt.Fatalf("version mismatch: saved %d, loaded %d", ver, got.Version)
		}
	})
}

// TestProperty_LoadAtVersionExactMatch — LoadAtVersion succeeds only when
// versions match exactly.
func TestProperty_LoadAtVersionExactMatch(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := snapshot.NewTypedStore[testState](newFakeStore(), nil)
		ctx := context.Background()

		savedVer := genVersion().Draw(rt, "savedVer")
		queryVer := genVersion().Draw(rt, "queryVer")
		streamID := id.NewStreamID()
		streamType := id.StreamType("TestAggregate")
		ref := id.NewStreamRef(streamType, streamID)

		_ = store.Save(ctx, snapshot.TypedSnapshot[testState]{
			StreamID:   streamID,
			StreamType: streamType,
			Version:    savedVer,
			State:      testState{Label: "snap"},
			CreatedAt:  time.Now().UTC(),
		})

		got, err := store.LoadAtVersion(ctx, ref, queryVer)

		// fakeStore's LoadAtVersion returns the snapshot only if savedVer == queryVer.
		if savedVer == queryVer {
			if err != nil {
				rt.Fatalf("LoadAtVersion should succeed on exact match: %v", err)
			}

			if got.Version != queryVer {
				rt.Fatalf("returned version %d != requested %d", got.Version, queryVer)
			}
		} else {
			if err == nil {
				rt.Fatalf(
					"LoadAtVersion should fail on mismatch (saved %d, query %d)",
					savedVer,
					queryVer,
				)
			}
		}
	})
}

// TestProperty_DeleteThenLoadFails — after Delete, Load returns ErrSnapshotNotFound.
func TestProperty_DeleteThenLoadFails(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := snapshot.NewTypedStore[testState](newFakeStore(), nil)
		ctx := context.Background()

		streamID := id.NewStreamID()
		streamType := id.StreamType("TestAggregate")
		ref := id.NewStreamRef(streamType, streamID)

		_ = store.Save(ctx, snapshot.TypedSnapshot[testState]{
			StreamID:   streamID,
			StreamType: streamType,
			Version:    event.Version(1),
			State:      testState{Label: "snap"},
			CreatedAt:  time.Now().UTC(),
		})

		_ = store.Delete(ctx, ref)

		_, err := store.Load(ctx, ref)
		if err == nil {
			rt.Fatalf("Load after Delete should return ErrSnapshotNotFound")
		}
	})
}

// TestProperty_OverwriteReplacesState — a second Save on the same ref replaces
// both State and Version.
func TestProperty_OverwriteReplacesState(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		store := snapshot.NewTypedStore[testState](newFakeStore(), nil)
		ctx := context.Background()

		first := genState().Draw(rt, "first")
		second := genState().Draw(rt, "second")
		streamID := id.NewStreamID()
		streamType := id.StreamType("TestAggregate")
		ref := id.NewStreamRef(streamType, streamID)

		_ = store.Save(ctx, snapshot.TypedSnapshot[testState]{
			StreamID: streamID, StreamType: streamType,
			Version: event.Version(1), State: first, CreatedAt: time.Now().UTC(),
		})
		_ = store.Save(ctx, snapshot.TypedSnapshot[testState]{
			StreamID: streamID, StreamType: streamType,
			Version: event.Version(2), State: second, CreatedAt: time.Now().UTC(),
		})

		got, err := store.Load(ctx, ref)
		if err != nil {
			rt.Fatalf("Load after overwrite failed: %v", err)
		}

		if got.State != second {
			rt.Fatalf("state after overwrite: expected %+v, got %+v", second, got.State)
		}

		if got.Version != event.Version(2) {
			rt.Fatalf("version after overwrite: expected 2, got %d", got.Version)
		}
	})
}
