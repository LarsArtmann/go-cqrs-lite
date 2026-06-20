// Package decider implements the pure-function aggregate pattern for event sourcing.
//
// A Decider[State] replaces mutable aggregate roots with two pure functions:
//   - DecideFunc: takes current state + command, returns events
//   - Apply: takes state + event, returns new state
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
// calls for the same aggregate into a single store.Load query. When N goroutines
// execute commands targeting the same aggregate simultaneously, only one DB read
// occurs — all callers receive the same immutable event slice. This is transparent:
// no API change, no configuration needed. Only load is coalesced; Save and Publish
// still execute independently per caller.
//
// To disable coalescing (e.g. when the store provides its own caching), pass
// WithLoadCoalescing[State](false):
//
//	repo, _ := decider.NewRepository(store, bus, d,
//	    decider.WithLoadCoalescing[MyState](false))
package decider
