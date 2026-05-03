# User Service Example

A reference-quality integration demonstrating the full go-cqrs-lite stack using the **Decider pattern** — pure functions for state reconstruction and command decisions, no mutable aggregate required.

## What It Demonstrates

| Capability                                                         | File                                |
| ------------------------------------------------------------------ | ----------------------------------- |
| **Decider pattern** — pure `fold` + `decide` functions             | `state.go`, `decide.go`             |
| **Command dispatch** with typed command structs                    | `commands.go`, `handlers.go`        |
| **Event sourcing** — load → fold → decide → save → publish         | `decider.go` (library)              |
| **Event bus subscription** — real-time projection updates          | `handlers.go`                       |
| **Read model projection** — builds queryable state from events     | `projection.go`                     |
| **Query dispatch** with typed results                              | `queries.go`, `handlers.go`         |
| **Middleware chain** — Recovery → Logging → Metrics → Retry        | `main.go`, `middleware_adapters.go` |
| **Error classification** — Rejection, Conflict, Transient families | `decide.go`, `main.go`              |
| **EventCatalog generation** — AsyncAPI-compatible docs             | `catalog.go`                        |
| **Branded IDs** — `id.AggregateID`, `id.EventID`, etc.             | throughout                          |

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
                              (reads from model)
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
└── middleware_adapters.go # Logger + Metrics adapters
```

## Run

```bash
cd example/user
go run .
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
dispatcher.Register("CreateUser", func(ctx context.Context, cmd command.Command) error {
    c := cmd.(*CreateUserCmd)
    return deciderRepo.Execute(ctx, c.AggregateID(), "User",
        decideCreateUser(c.AggregateID(), c.email, c.name),
    )
})
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

### Error Classification (decide.go + main.go)

Decide functions return typed errors using `event.NewRejection` / `event.NewConflict`:

```go
return nil, event.NewRejection("user.create.email_required", "email is required")
```

Consumers classify them:

```go
family := event.Classify(err)      // → Rejection
retryable := event.IsRetryable(err) // → false
```
