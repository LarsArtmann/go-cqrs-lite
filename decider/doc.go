// Package decider implements the pure-function stream pattern for event sourcing.
//
// A Decider[State] replaces a mutable stream with a pure fold: the
// Apply function takes the current state and an event, returning the new state.
// The decision logic (command → events) is supplied separately as a
// [DecideFunc] to Repository.Execute — it is NOT a field on Decider[State].
//
// The Repository[State] handles the full lifecycle: load → apply → decide → save → publish.
//
// # Quick Start
//
//	d := decider.Decider[UserState]{
//	    Initial: UserState{},
//	    Apply: func(s UserState, evt event.Event) (UserState, error) {
//	        switch evt.Type() {
//	        case "user.created":
//	            return applyCreated(s, evt)
//	        }
//	        return s, nil
//	    },
//	}
//
//	repo, _ := decider.NewRepository[UserState](store, bus, d)
//
//	err := repo.Execute(ctx, aggID, "User",
//	    func(state UserState, version event.Version) ([]event.Event, error) {
//	        return event.NewEvents(aggID, "User", version,
//	            []event.Type{"user.created"},
//	            []any{UserCreated{Name: "Alice"}},
//	        )
//	    },
//	)
//
// # Time Travel
//
//	state, version, _ := repo.Load(ctx, aggID, "User")
//	state, version, _ = repo.LoadAtVersion(ctx, aggID, "User", 3)
//	state, version, _ = repo.LoadAtTime(ctx, aggID, "User", cutoff)
//
// # Schema Evolution (Upcasting)
//
// Connect decider with the schema/ module by wrapping the store before passing
// it to NewRepository. The VersionedStore implements event.Store transparently —
// the decider loads upcasted events without any code change:
//
//	store := schema.NewVersionedStore(rawStore,
//	    schema.NewUpcaster("user.created", 1, upcastV1ToV2),
//	)
//	repo, _ := decider.NewRepository[UserState](store, bus, d)
//	// repo.Load now returns events with schema version 2, even if stored as v1
//
// # Load Coalescing
//
// Repository uses singleflight.Group internally to coalesce concurrent Load
// calls for the same stream into a single store.Load query. When N goroutines
// execute commands targeting the same stream simultaneously, only one DB read
// occurs — all callers receive the same immutable event slice. This is transparent:
// no API change, no configuration needed. Only load is coalesced; Save and Publish
// still execute independently per caller.
//
// To disable coalescing (e.g. when the store provides its own caching), pass
// WithLoadCoalescing[State](false):
//
//	repo, _ := decider.NewRepository(store, bus, d,
//	    decider.WithLoadCoalescing[MyState](false))
//
// # Hot-State Cache
//
// WithStateCache enables an in-memory LRU cache of folded stream state.
// On a cache hit, Load fetches only events since the cached version
// (O(new events)) instead of replaying the full history (O(total events)).
// Execute updates the cache after every successful write, keeping it fresh.
//
//	cache := decider.NewStateCache[MyState](256)
//	repo, _ := decider.NewRepository(store, bus, d,
//	    decider.WithStateCache[MyState](cache))
//
// Benchmark: 7.4x faster Load (2090→283 ns/op) with 500-event history.
// The cache is process-local and best-effort — misses fall back to the normal
// load path. Profile before enabling; for small streams the fold cost may
// be negligible.
//
// # Read-Pressure Snapshot Strategy
//
// Combine WithSnapshotStrategy with snapshot.NewReadPressure to snapshot
// streams that are read frequently but written rarely. EveryNEvents only
// triggers on write count; ReadPressure triggers on read count:
//
//	rp, _ := snapshot.NewReadPressure(50)
//	repo, _ := decider.NewRepository(store, bus, d,
//	    decider.WithSnapshotStore[MyState](snapStore),
//	    decider.WithCodec[MyState](codec.JSONCodec{}),
//	    decider.WithSnapshotStrategy[MyState](rp))
//	// After 50 Loads, the next Execute saves a snapshot automatically.
package decider
