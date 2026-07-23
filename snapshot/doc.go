// Package snapshot provides snapshot persistence for event-sourced aggregates.
//
// Snapshots capture the full aggregate state at a given version, eliminating
// the need to replay the entire event history on each load.
//
// # Quick Start (Typed — Recommended)
//
// Use [TypedStore] to get compile-time type safety on snapshot state:
//
//	store, _ := memory.NewSnapshotStore()
//	ts := snapshot.NewTypedStore[UserState](store, codec.JSONCodec{})
//
//	_ = ts.Save(ctx, snapshot.TypedSnapshot[UserState]{
//	    StreamID:   aggID,
//	    StreamType: "User",
//	    Version:       10,
//	    State:         UserState{Name: "Alice"},
//	    CreatedAt:     time.Now(),
//	})
//	got, _ := ts.LoadAtVersion(ctx, ref, 10)
//	// got.State is UserState, not []byte — no manual decode needed
//
// # Untyped API (Low-Level)
//
// [Snapshot] stores State as []byte. This is the storage-level interface that
// [TypedStore] adapts. Use it directly only when you need raw bytes access
// (custom codecs, migration tooling, test infrastructure).
//
//	snap := snapshot.Snapshot{
//	    StreamID:   aggID,
//	    StreamType: "User",
//	    Version:       10,
//	    State:         []byte(`{"name":"Alice"}`),
//	    CreatedAt:     time.Now(),
//	}
//	_ = store.Save(ctx, snap)
//
// # Strategies
//
// Use EveryNEvents to snapshot automatically based on write count:
//
//	strategy, _ := snapshot.EveryNEvents(100)
//
// Use ReadPressure to snapshot based on read count — ideal for hot-read,
// cold-write aggregates that never hit EveryNEvents thresholds:
//
//	rp, _ := snapshot.NewReadPressure(50)
//	// Snapshots after the next write when an aggregate has been loaded
//	// at least 50 times since its last snapshot.
//
// Combine both via the Inner option:
//
//	rp, _ := snapshot.NewReadPressure(50,
//	    snapshot.WithInnerStrategy(everyN100))
//
// # ISP Split
//
//	SnapshotSink   — write side (Save)
//	SnapshotSource — read side (LoadAtVersion)
//	SnapshotStore  — Sink + Source combined
//
//	TypedSnapshotSink[State]   — typed write side
//	TypedSnapshotSource[State] — typed read side
//	TypedStore[State]          — typed adapter over any SnapshotStore
package snapshot
