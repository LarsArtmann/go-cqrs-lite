// Package snapshot provides snapshot persistence for event-sourced aggregates.
//
// Snapshots capture the full aggregate state at a given version, eliminating
// the need to replay the entire event history on each load.
//
// # Quick Start
//
//	store, _ := memory.NewMemorySnapshotStore()
//	snap := snapshot.Snapshot{
//	    AggregateID:   aggID,
//	    AggregateType: "User",
//	    Version:       10,
//	    State:         []byte(`{"name":"Alice"}`),
//	    CreatedAt:     time.Now(),
//	}
//	_ = store.Save(ctx, snap)
//	loaded, _ := store.LoadAtVersion(ctx, ref, 10)
//
// # Strategies
//
// Use EveryNEvents to snapshot automatically:
//
//	strategy, _ := snapshot.EveryNEvents(100)
//
// # ISP Split
//
//	SnapshotSink   — write side (Save)
//	SnapshotSource — read side (LoadAtVersion)
//	SnapshotStore  — Sink + Source combined
package snapshot
