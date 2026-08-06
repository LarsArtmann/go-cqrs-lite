# system — Deployer-Driven Composition Root

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/system/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/system/v4)

The next-generation composition root for go-cqrs-lite. The consumer provides
only domain types (`DomainConfig`); the operator provides only infrastructure
decisions (`DeploymentConfig`). The two are separate types, compiler-enforced.
The `System` owns all infrastructure wiring internally.

```bash
go get github.com/larsartmann/go-cqrs-lite/system/v4
```

## Why?

`stack.Bundle` is a bag of peer capability fields — flexible, but the consumer
must wire them together manually. `system` goes further: the consumer declares
domain intent (deciders, commands, queries, projections), the operator declares
infrastructure (engines, bus, durability), and `System.New` wires everything
together automatically. This separation means domain code never touches
infrastructure choices.

## Quick Start

```go
import "github.com/larsartmann/go-cqrs-lite/system/v4"

// Consumer: declare domain
domain := system.DomainConfig{
    StreamTypes: []string{"User"},
}

// Operator: declare infrastructure
deployment := system.DeploymentConfig{
    Engines: []system.EngineConfig{
        {Name: "primary", Driver: "memory", Role: system.RoleSourceOfTruth},
    },
    Bus: system.BusConfig{Driver: "memory"},
}

// System wires everything
sys, err := system.New(ctx, domain, deployment)
if err != nil { log.Fatal(err) }
defer sys.Close()

// Register domain logic
system.RegisterDecider[UserState](sys, "User", decider)
system.RegisterCommand[CreateUser, UserState](sys, "user.create", decideFunc)

// Start projections and bus
sys.Start(ctx)

// Dispatch
sys.Dispatch(ctx, cmd)
```

## API

### Constructor

| Symbol                         | Description                                           |
| ------------------------------ | ----------------------------------------------------- |
| `New(ctx, domain, deployment)` | Creates a `*System` from domain + deployment configs. |
| `System.Start(ctx)`            | Starts projection workers and bus listeners.          |
| `System.Close()`               | Graceful shutdown of all infrastructure.              |

### Domain Registration

| Symbol                                  | Description                                          |
| --------------------------------------- | ---------------------------------------------------- |
| `RegisterDecider[State](sys, ...)`      | Register an aggregate decider with snapshot support. |
| `RegisterCommand[Cmd, State](sys, ...)` | Bind a command type to a decider.                    |
| `RegisterQuery[Q, R](sys, ...)`         | Register a typed query handler.                      |
| `DispatchQuery[Q, R](ctx, sys, q)`      | Dispatch a typed query.                              |

### Driver Registry

| Symbol                       | Description                                              |
| ---------------------------- | -------------------------------------------------------- |
| `RegisterDriver(name, f)`    | Register a storage engine factory (like `database/sql`). |
| `RegisterBusDriver(name, f)` | Register a bus driver factory.                           |
| `RegisteredDrivers()`        | List registered engine driver names.                     |
| `RegisteredBusDrivers()`     | List registered bus driver names.                        |

Built-in drivers: `memory`, `sqlite`.

### Introspection

| Symbol                          | Description                                     |
| ------------------------------- | ----------------------------------------------- |
| `System.Snapshot(ctx)`          | Returns a `Topology` snapshot of all instances. |
| `System.Health(ctx)`            | Aggregate health status string.                 |
| `System.Explain(ctx)`           | Human-readable explanation of the deployment.   |
| `System.ProjectionPlan()`       | Serializable plan for projection engines.       |
| `System.ProjectionExplain()`    | Human-readable projection plan explanation.     |
| `System.VerifyProjections(ctx)` | Verify projection stores match source-of-truth. |

### Safety Checks

| Symbol                     | Description                                           |
| -------------------------- | ----------------------------------------------------- |
| `CheckSafety(ctx, deploy)` | Pre-construction safety report (WARN/ERROR per rule). |
| `System.ScreamReport()`    | Post-construction safety report.                      |

## Design

- **Separate config types**: `DomainConfig` (consumer) and `DeploymentConfig`
  (operator) are distinct types — the compiler enforces the separation (D11).
- **Driver registry**: Modeled after `database/sql` — drivers register
  themselves via `init()`, the system looks them up by name.
- **Metaengine integration**: Projections use `metaengine.Store` instances
  connected via `StreamLogBackend` — the source-of-truth layer feeds the
  projection layer.
- **Scream store**: Pre-flight safety checks warn about dangerous
  configurations (volatile engines in production, missing durability, etc.).

## Related Modules

- [**metaengine**](../metaengine/README.md) — Engine abstraction and cost planner
- [**decider**](../decider/README.md) — Aggregate pattern
- [**projectionhost**](../projectionhost/README.md) — Managed projection lifecycle
- [**stack**](../stack/README.md) — The simpler `Bundle` composition alternative
