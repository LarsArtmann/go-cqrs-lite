# User Service Example

A reference-quality integration demonstrating the full go-cqrs-lite stack using the **Decider pattern** — pure functions for state reconstruction and command decisions, no mutable aggregate required.

## Quick Start

```bash
cd example/user
go run .    # runs the full demo with signing, middleware, and catalog generation
go test ./...  # runs all tests including the signing pipeline
```

## What It Demonstrates

| Capability                                            | File                                | API                                  |
| ----------------------------------------------------- | ----------------------------------- | ------------------------------------ |
| **Decider pattern** — pure `fold` + `decide`          | `state.go`, `decide.go`             | `decider.Decider[State]`             |
| **Command dispatch** with typed structs               | `commands.go`, `handlers.go`        | `command.RegisterTyped`              |
| **Event sourcing** — load→fold→decide→save→publish    | via `decider.Repository`            | `deciderRepo.Execute`                |
| **Bus subscriptions** — real-time projections         | `handlers.go`                       | `bus.SubscribeAll`                   |
| **Read model projection** from events                 | `projection.go`                     | `ReadModelStore.Handle`              |
| **Query dispatch** with typed results                 | `queries.go`, `handlers.go`         | `query.DispatchTyped[T]`             |
| **Middleware chain** — Recovery→Logging→Metrics→Retry | `main.go`, `middleware_adapters.go` | `dispatcher.Use`                     |
| **Event signing** — HMAC-SHA256 sign/verify           | `main.go`                           | `signing.SignMiddleware`             |
| **Error classification** — Rejection, Conflict        | `decide.go`                         | `event.NewRejection` / `NewConflict` |
| **EventCatalog generation**                           | `catalog.go`                        | `catalog.NewBuilder`                 |

## Architecture

```
Command → Dispatcher → Handler → DeciderRepository.Execute()
                                         │
                                    load events
                                    fold to State
                                    call decide(state, version)
                                    save new events
                                    publish to Bus
                                         │
                                    ┌────┴────┐
                                    ▼         ▼
                              Projection   EventCatalog
                              (read model)  (docs)
                                    │
                                    ▼
                              QueryDispatcher
```

### Module Dependencies

```
example/user
├── core/command     → Dispatcher, RegisterTyped
├── core/decider     → Decider[State], Repository
├── core/event       → NewEvent, Version, errors
├── core/query       → Dispatcher, DispatchTyped
├── memory           → MemoryStore, MemoryBus
├── middleware        → Recovery, Logging, Metrics, Retry
├── signing          → HMAC, SignMiddleware, VerifyMiddleware
└── catalog          → Builder, EventCatalog exporter
```

## File Structure

```
example/user/
├── main.go                # Wiring + demo flow
├── events.go              # Shared event payload types
├── state.go               # UserState + fold function
├── decide.go              # decideCreateUser, decideChangeName
├── commands.go            # CreateUserCmd, ChangeUserNameCmd
├── queries.go             # GetUserQuery, ListUsersQuery
├── projection.go          # ReadModelStore (projection + query source)
├── handlers.go            # Command + query handler wiring
├── catalog.go             # EventCatalog generation
├── middleware_adapters.go # Logger + Metrics adapters
├── main_test.go           # Unit + integration tests
└── smoke_test.go          # Full-stack signing + error tests
```

## Key Patterns

### Decider (state.go + decide.go)

State reconstruction is a pure function — no mutable aggregate:

```go
func foldUser(state UserState, evt event.Event) (UserState, error) { ... }
func decideCreateUser(aggID, email, name) func(UserState, event.Version) ([]event.Event, error) { ... }
```

### Command Handler (handlers.go)

Handlers are thin — they bridge command structs to the decider:

```go
command.RegisterTyped(dispatcher, cmdCreateUser,
    func(ctx context.Context, c *CreateUserCmd) error {
        return deciderRepo.Execute(ctx, c.AggregateID(), aggregateType,
            decideCreateUser(c.AggregateID(), c.email, c.name))
    })
```

### Read Model Projection (projection.go)

Subscribes to the event bus and builds queryable state:

```go
bus.SubscribeAll(func(ctx context.Context, evt event.Event) error {
    return readModel.Handle(ctx, evt)
})
```

### Query with Typed Results (handlers.go)

```go
result, err := query.DispatchTyped[ReadModel](ctx, qryDisp, &GetUserQuery{aggregateID: userID})
```

### Middleware Composition (main.go)

```go
cmdDispatcher.Use(
    middleware.CommandRecovery(),
    middleware.CommandLogging(newLogger()),
    middleware.CommandMetrics(&printMetricsRecorder{}),
    middleware.CommandRetry(middleware.DefaultRetryConfig()),
)
```

### Event Signing (main.go)

```go
signer, _ := signing.NewHMAC(hmacSecret)
bus.UsePublish(signing.SignMiddleware(signer))
bus.Use(signing.VerifyMiddleware(signer))
```

### Error Classification (decide.go)

Decide functions return typed errors:

```go
return nil, event.NewRejection("user.create.email_required", "email is required")
```

Consumers classify them:

```go
family := event.Classify(err)      // → "rejection"
retryable := event.IsRetryable(err) // → false
```

## Tests

| Test                                   | What it covers                           |
| -------------------------------------- | ---------------------------------------- |
| `TestDecider_CreateUser`               | Pure decide function, event creation     |
| `TestDecider_CreateUser_EmptyEmail`    | Validation rejection                     |
| `TestDecider_CreateUser_AlreadyExists` | Conflict detection                       |
| `TestDecider_ChangeName`               | Update decide function                   |
| `TestFoldUser`                         | State reconstruction from events         |
| `TestReadModel_Projection`             | Event → read model mapping               |
| `TestFullCQRS_Lifecycle`               | End-to-end: create→changeName→query→list |
| `TestQueryDispatcher`                  | Query dispatch with typed results        |
| `TestEventCatalog_Generation`          | AsyncAPI + EventCatalog export           |
| `TestErrorClassification`              | Error family classification              |
| `TestFullStack_WithSigning`            | Full pipeline with HMAC sign/verify      |
| `TestFullStack_DuplicateUserRejection` | Conflict error path                      |

```bash
go test ./... -v         # run all tests
go test -run FullStack   # run only smoke tests
```
