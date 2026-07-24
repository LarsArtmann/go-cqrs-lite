# decider — Pure-Function Aggregate Pattern

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/decider/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/decider/v4)

The Decider replaces mutable aggregate roots with two pure functions: `DecideFunc` (command + state + version -> events) and `Apply`/`Fold` (state + event -> state). The `Repository` orchestrates load-fold-decide-save-publish.

```bash
go get github.com/larsartmann/go-cqrs-lite/decider/v4
```

## Quick Start

```go
d := decider.Decider[CounterState]{
    Initial: CounterState{},
    Apply:   applyCounter, // fold: (state, event) -> state
}

strategy, _ := snapshot.EveryNEvents(100)

repo, err := decider.NewRepository(store, bus, d,
    decider.WithSnapshotStore(snapStore),
    decider.WithSnapshotStrategy(strategy),
)
if err != nil { log.Fatal(err) }

// Execute: load → fold → decide → save → publish
err = repo.Execute(ctx, aggID, "Counter", increment(aggID, 5))

// Load (replay from store or snapshot)
state, version, err := repo.Load(ctx, aggID, "Counter")

// Time travel
state, ver, _ := repo.LoadAtVersion(ctx, aggID, "Counter", 3)
```

## API

### Decider[State]

| Field     | Type                                      | Description                               |
| --------- | ----------------------------------------- | ----------------------------------------- |
| `Initial` | `State`                                   | The starting state before any events.     |
| `Apply`   | `func(State, event.Event) (State, error)` | Fold function: applies an event to state. |

### Repository[State]

| Method                                  | Description                                              |
| --------------------------------------- | -------------------------------------------------------- |
| `NewRepository(store, bus, d, opts...)` | Creates a repository.                                    |
| `Execute(ctx, aggID, aggType, decide)`  | Load → fold → decide → save → publish.                   |
| `Load(ctx, aggID, aggType)`             | Returns `(state, version, error)` from replaying events. |
| `LoadAtVersion(ctx, aggID, aggType, v)` | Time travel: state at a specific version.                |

### Options

| Option                             | Description                                                               |
| ---------------------------------- | ------------------------------------------------------------------------- |
| `WithSnapshotStore(s)`             | Enables snapshot-based loading (skip full replay).                        |
| `WithSnapshotStrategy(s)`          | When to create snapshots: `EveryNEvents(n)`, `NewReadPressure(n)`.        |
| `WithCodec(c)`                     | Codec for snapshot serialization (default: CBOR).                         |
| `WithStateCache(c)`                | LRU-bounded cache for incremental loads (7.4x faster for hot aggregates). |
| `WithLoadCoalescing[State](false)` | Disable singleflight load coalescing.                                     |

### TypedDecider[State, Cmd]

Command type bound at compile time (ADR-0001):

```go
d := decider.TypedDecider[CounterState, IncrementCmd]{
    Initial: CounterState{},
    Decide:  decideIncrement,
    Apply:   foldCounter,
}
repo, _ := decider.NewTypedRepository(store, bus, d)
err := repo.ExecuteCommand(ctx, aggID, "Counter", IncrementCmd{Amount: 5})
```

## Design

- **Pure functions**: `DecideFunc` and `Apply` have no side effects. State transitions are deterministic and testable.
- **Singleflight load coalescing**: Concurrent `Load` calls for the same aggregate coalesce into one `store.Load` query. Events are immutable, so sharing is safe. Disable via `WithLoadCoalescing(false)`.
- **Hot-state cache**: `NewStateCache[State](256)` enables incremental loads. On cache hit: `LoadFromVersion(cachedVer)` + fold delta. On miss: full `Load` + cache populate.
- **Snapshot strategies**: `EveryNEvents(n)` snapshots every N events. `NewReadPressure(loads)` snapshots after N loads + next write. Combine both with `WithInnerStrategy`.
- **Version-based optimistic concurrency**: The repository stamps each new event with the expected version, preventing concurrent writes from corrupting state.

## Related Modules

- [**event**](../event/README.md) — Event store/bus interfaces consumed by the repository
- [**snapshot**](../snapshot/README.md) — Snapshot strategies (`EveryNEvents`, `ReadPressure`)
- [**id**](../id/README.md) — Branded `StreamID` for aggregates
- [**command**](../command/README.md) — Dispatch typed commands into `repo.Execute`
- [**scenario**](../scenario/README.md) — BDD test DSL for deciders
- [**schema**](../schema/README.md) — Upcast old events on load
