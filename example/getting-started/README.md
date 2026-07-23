# getting-started — go-cqrs-lite in 80 Lines

The simplest useful example: a complete event-sourced counter with a materialized read model.

## What It Demonstrates

The full CQRS pipeline in 80 lines:

```
Decider → Event Store → EventBus → CatchUpSubscriber → Materialize → Read Model
```

### Key Insight: Separation of Deployer and Consumer

- **Deployer** picks infrastructure (in-memory here). One `stack.New(...)` call wires everything.
- **Consumer** defines the domain: events, state, decide function, projection callbacks.
- Swap `memory` to `sqlite`, `pebble`, or `postgres` by changing ONE line. The domain code doesn't change.

## Run It

```bash
cd example/getting-started
GOWORK=off go run main.go
```

Output:
```
Counter <ulid>: value=10 (expected 10)
```

## How It Works

1. **Deployer wires infrastructure**: `stack.New(...)` with in-memory store, bus, checkpoint store, and read-model backend.
2. **Repository executes commands**: `repo.Execute(ctx, counterID, ...)` loads state from events, runs the decide function, saves new events, publishes to the bus.
3. **CatchUpSubscriber replays the journal**: Historical events are replayed first, then the subscriber enters live mode.
4. **Materialize builds the read model**: Each event updates a `CounterView` in the KV store.
5. **Query the view**: After the projection catches up, `mat.View(ctx, counterID)` returns the materialized state.

## Swap to Persistent Storage

```go
// Change ONE line:
// bundle, err := stack.New(
//     stack.WithEventStore(memory.NewMemoryStore()),
//     ...
// )

// To:
// bundle, err := sqlite.New("counter.db")
// bundle, err := pebble.New("./data")
// bundle, err := postgres.New(dsn)
```

## Related

- [**readme-quickstart**](../readme-quickstart/) — Even simpler: just a command handler and decider, no projections
- [**taskmanager**](../taskmanager/) — Flagship example: full HTTP service with all modules wired
