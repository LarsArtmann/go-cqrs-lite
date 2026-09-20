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

ref := id.NewStreamRef("Counter", aggID)

// Execute: load → fold → decide → save → publish
err = repo.ExecuteRef(ctx, ref, increment(aggID, 5))

// Load (replay from store or snapshot)
state, version, err := repo.LoadRef(ctx, ref)

// Time travel
state, ver, _ := repo.LoadAtVersionRef(ctx, ref, 3)
```

> The pair forms (`Execute`/`Load`/`LoadAtVersion` taking `streamID,
> streamType` separately) are **deprecated** forwarders — removed in v5.
> Build an `id.StreamRef` via `id.NewStreamRef(streamType, streamID)` instead.

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
| `ExecuteRef(ctx, ref, decide)`          | Load → fold → decide → save → publish.                   |
| `LoadRef(ctx, ref)`                     | Returns `(state, version, error)` from replaying events. |
| `LoadAtVersionRef(ctx, ref, v)`         | Time travel: state at a specific version.                |
| `LoadAtTimeRef(ctx, ref, t)`            | Time travel: state as of a timestamp.                    |
| `WaitForVersionRef(ctx, ref, v)`        | Block until the stream reaches a version.                |

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
err := repo.ExecuteCommandRef(ctx, id.NewStreamRef("Counter", aggID), IncrementCmd{Amount: 5})
```

## Design

- **Pure functions**: `DecideFunc` and `Apply` have no side effects. State transitions are deterministic and testable.
- **Singleflight load coalescing**: Concurrent loads for the same aggregate coalesce into one `store.Load` query. Events are immutable, so sharing is safe. Disable via `WithLoadCoalescing(false)`.
- **Hot-state cache**: `NewStateCache[State](256)` enables incremental loads. On cache hit: `LoadFromVersion(cachedVer)` + fold delta. On miss: full replay + cache populate.
- **Snapshot strategies**: `EveryNEvents(n)` snapshots every N events. `NewReadPressure(loads)` snapshots after N loads + next write. Combine both with `WithInnerStrategy`.
- **Version-based optimistic concurrency**: The repository stamps each new event with the expected version, preventing concurrent writes from corrupting state.

## Related Modules

- [**event**](../event/README.md) — Event store/bus interfaces consumed by the repository
- [**snapshot**](../snapshot/README.md) — Snapshot strategies (`EveryNEvents`, `ReadPressure`)
- [**id**](../id/README.md) — Branded `StreamID` for aggregates
- [**command**](../command/README.md) — Dispatch typed commands into `repo.Execute`
- [**scenario**](../scenario/README.md) — BDD test DSL for deciders
- [**schema**](../schema/README.md) — Upcast old events on load
