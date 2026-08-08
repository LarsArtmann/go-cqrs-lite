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
package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Domain types ──

type TaskCreated struct {
	Title string
	At    time.Time
}

type TaskState struct {
	Title  string
	Status string
	Exists bool
}

func applyTask(state TaskState, evt event.Event) (TaskState, error) {
	switch evt.Type() {
	case "task.created":
		var p TaskCreated
		_ = json.Unmarshal(evt.Payload(), &p)
		state.Title = p.Title
		state.Status = "pending"
		state.Exists = true
	}
	return state, nil
}

func main() {
	ctx := context.Background()

	// Consumer: declare domain logic
	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", decider.Decider[TaskState]{
				Initial: TaskState{},
				Apply:   applyTask,
			})

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if state.Exists {
								return nil, errors.New("task already exists")
							}
							evt, err := event.New("task.created", cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "my first task", At: time.Now()})
							if err != nil {
								return nil, err
							}
							return []event.Event{evt}, nil
						})
				})
		},
	}

	// Operator: declare infrastructure
	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}

	// System wires everything
	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		log.Fatal(err)
	}
	defer sys.Close()

	// Start projections (if configured)
	if err := sys.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Dispatch a command
	taskID := id.NewStreamID()
	createCmd, _ := command.New("task.create", taskID)
	if err := sys.CommandDispatcher().Dispatch(ctx, createCmd); err != nil {
		log.Fatal(err)
	}

	// Verify events were persisted
	ref := id.NewStreamRef("Task", taskID)
	events, err := sys.EventStore().Load(ctx, ref)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created task with %d event(s)\n", len(events))
}
```

## API

### Constructor

| Symbol                              | Description                                            |
| ----------------------------------- | ------------------------------------------------------ |
| `New(ctx, domain, deployment)`      | Creates a `*System` from domain + deployment configs.  |
| `System.Start(ctx)`                 | Starts projection workers and bus listeners.           |
| `System.Close()`                    | Graceful shutdown of all infrastructure.               |
| `System.GracefulClose(ctx)`         | Drains via [Drainer]s, then context-bounded `Close()`. |
| `System.Drain(ctx)`                 | Drain in-flight work without closing (rolling deploys). |
| `System.HealthCheck(ctx)`           | Returns `nil` if all resources are healthy.            |
| `System.ResetProjection(ctx, name)` | Resets a projection checkpoint for replay.             |
| `System.RegisterDrainer(d)`         | Register a pre-close drainer for `GracefulClose`.      |
| `System.RegisterCloser(name, c)`    | Register an external `io.Closer` for lifecycle mgmt.   |

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

Built-in bus drivers: `gochannel` (in-process). Unknown driver names return
an error at construction time (no silent fallback).

### Introspection

| Symbol                          | Description                                     |
| ------------------------------- | ----------------------------------------------- |
| `System.Snapshot(ctx)`          | Returns a `Topology` snapshot of all instances. |
| `System.Health(ctx)`            | Aggregate health status string.                 |
| `System.HealthCheck(ctx)`       | Error if any resource is unhealthy.             |
| `System.HealthCheckDetailed(ctx)` | Per-engine health status ([]EngineHealth).     |
| `System.Explain(ctx)`           | Human-readable explanation of the deployment.   |
| `System.EngineNames()`          | Engine names in creation order (diagnostics).   |
| `System.ShutdownOrder()`        | Resolved close order as engine names.           |
| `System.LagPerProjection()`     | Per-projection lag map (delegates to host).     |
| `System.LagDuration()`          | Max lag across all workers.                     |
| `System.WorkerStatus()`         | Projection worker states.                       |
| `System.ProjectionPlan()`       | Serializable plan for projection engines.       |
| `System.ProjectionExplain()`    | Human-readable projection plan explanation.     |
| `System.VerifyProjections(ctx)` | Verify projection stores match source-of-truth. |

### Safety Checks

| Symbol                     | Description                                           |
| -------------------------- | ----------------------------------------------------- |
| `CheckSafety(ctx, deploy)` | Pre-construction safety report (WARN/ERROR per rule). |
| `System.ScreamReport()`    | Post-construction safety report.                      |

## Examples

### Shutdown Dependencies

Declare close-time ordering so projections drain before the event store closes:

```go
domain := system.DomainConfig{
    ShutdownDependencies: []system.ShutdownDependency{
        {Before: "eventstore", After: "projectionhost"},
    },
}
```

### Drainer (Rolling Deploy)

Register a drainer to reject new requests during a rolling deploy, then close:

```go
type httpDrainer struct{ server *http.Server }

func (d *httpDrainer) Drain(ctx context.Context) error {
    return d.server.Shutdown(ctx) // stop accepting, finish in-flight
}

sys.RegisterDrainer(&httpDrainer{server: srv})

// SIGTERM handler:
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := sys.GracefulClose(ctx); err != nil {
    log.Printf("graceful close: %v", err)
}
```

### External Closer

Register a non-engine resource (connection pool, file handle) for lifecycle management:

```go
sys.RegisterCloser("redis-pool", redisClient)
defer sys.Close() // closes engines first, then redis-pool
```

## Configuration

`LoadConfig(path)` loads a `DeploymentConfig` from YAML with env var overrides.
Env vars use the `CQRS_` prefix with double-underscore (`__`) as the nested
separator (koanf convention). Env overrides win over YAML.

### DomainConfig Fields

| Field                    | Description                                                         |
| ------------------------ | ------------------------------------------------------------------- |
| `Commands`               | Function that registers typed command handlers on the System.       |
| `Queries`                | Function that registers typed query handlers on the System.         |
| `Projections`            | Metaengine query declarations for auto-wired projections.           |
| `ProjectionDecoder`      | Decodes event payloads for projection fold handlers.                |
| `ProjectionTypeDecoder`  | Recommended: typed event decoder with stream ID access.             |
| `ProjectionEventDecoder` | Full event context decoder for projection fold handlers.            |
| `Middleware`             | Command-level domain middleware (validation, authz, etc.).          |
| `ProjectionHostOptions`  | Projection host options (batch size, DLQ, restart policy, etc.).    |
| `CheckpointStore`        | Persistent checkpoint store. If nil, in-memory (lost on restart).   |
| `ShutdownDependencies`   | Ordering constraints for `Close()` (engine names only; projection host always closes first). |

### YAML

```yaml
engines:
  primary:
    driver: sqlite
    dsn: file:events.db
    pragmas: [journal_mode=wal, foreign_keys=on]
buses:
  local:
    driver: gochannel
    mode: sync
instances:
  - role: source-of-truth
    engine: primary
    durability: normal
  - role: projections
    engine: primary
acknowledge_warnings:
  - "volatile-source-of-truth:source-of-truth"
```

### Structured Env Vars

```bash
# Nested keys use __ as separator
CQRS_ENGINES__PRIMARY__DRIVER=sqlite
CQRS_ENGINES__PRIMARY__DSN=file:events.db
CQRS_BUSES__LOCAL__DRIVER=gochannel
CQRS_INSTANCES__0__DURABILITY=strict

# Legacy shorthand (creates a "primary" engine if none exists)
CQRS_DEFAULT_DRIVER=sqlite
CQRS_DEFAULT_DSN=file:events.db
```

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
