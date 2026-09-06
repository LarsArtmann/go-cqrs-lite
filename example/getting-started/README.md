# getting-started — go-cqrs-lite in a Single File

The simplest useful example: a complete event-sourced counter with a
metaengine-backed read model.

## What It Demonstrates

The full CQRS pipeline:

```
Command → Decider → Event Store → Projection Host → metaengine → Read Model
```

### Key Insight: Separation of Deployer and Consumer

- **Deployer** picks infrastructure in ONE place: `DeploymentConfig` engines
  and roles (in-memory here).
- **Consumer** defines the domain: events, state, decide function, and folds.
- Swap `memory` to `sqlite`, `postgres`, or `pebble` by changing one
  `EngineConfig` line. The domain code doesn't change.

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

1. **Deployer declares engines**: `DeploymentConfig` maps a `primary` engine
   (memory here) and binds it to the `source-of-truth` and `projections`
   roles. `buildSystem(dsn)` is the only function an operator changes.
2. **Consumer registers the decider + command**: `system.RegisterDecider`
   wires the fold; `system.RegisterCommand` maps `IncrementCmd` to a
   `system.Execute` Op. The dispatcher routes by command type.
3. **Consumer declares the read model as folds**: `metaengine.Query` with
   create/update folds per event type, registered via
   `system.DomainConfig.Projections`.
4. **Projection host applies events**: started explicitly, it replays the
   journal and then follows live events into the metaengine store.
5. **Query the view**: `metaengine.NewReader.Get(ctx, streamID)` returns the
   materialized `CounterView` once the projection converges.

## Swap to Persistent Storage

```go
// buildSystem(ctx, dsn) — change ONE EngineConfig line:
engine := system.EngineConfig{Driver: "memory"}
// To:
engine := system.EngineConfig{Driver: "sqlite", DSN: "counter.db"}
// Backends self-register via init(); each backend the deployment might use
// needs one blank import (see the sqliteengine import in main.go).
```

## Related

- [**readme-quickstart**](../readme-quickstart/) — Even simpler: just a command handler and decider, no projections
- [**metaengine-quickstart**](../metaengine-quickstart/) — Map, graph, and vector projections over the metaengine pipeline
- [**taskmanager**](../taskmanager/) — Flagship example: full HTTP service with all modules wired
