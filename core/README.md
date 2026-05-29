# core — CQRS Types, Event Sourcing, and the Decider Pattern

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/core.svg)](https://pkg.go.dev/github.com/LarsArtmann/go-cqrs-lite/core)

Core types for CQRS + Event Sourcing: commands, queries, events, branded IDs, and the Decider pattern. Zero infrastructure dependencies (no HTTP, no database, no message broker).

```bash
go get github.com/larsartmann/go-cqrs-lite/core
```

## Packages

| Package               | Purpose                                                           |
| --------------------- | ----------------------------------------------------------------- |
| `core/command`        | Command dispatch, typed handlers, middleware                      |
| `core/query`          | Query dispatch, typed results, pagination                         |
| `core/event`          | Event sourcing, store/bus interfaces, codec, upcasters, snapshots |
| `core/decider`        | Functional aggregate pattern (recommended)                        |
| `core/pkg/id`         | Branded IDs: `id.Of[T]` backed by ULID                            |
| `core/pkg/dispatcher` | Generic dispatcher with lifecycle management                      |

## The Decider Pattern (Recommended)

The Decider replaces mutable aggregate roots with pure functions. No 9-method interface, no mutable state, zero-infrastructure testing.

```go
type UserState struct {
    Name  string
    Email string
}

type UserCreated struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

d := decider.Decider[UserState]{
    Initial: UserState{},
    Fold: func(s UserState, evt event.Event) (UserState, error) {
        switch evt.Type() {
        case "user.created":
            p, err := event.DecodePayload[UserCreated](evt, event.JSONCodec{})
            if err != nil {
                return s, err
            }
            s.Name = p.Name
            s.Email = p.Email
        }
        return s, nil
    },
}
```

### Repository: Load → Fold → Decide → Save → Publish

```go
repo, _ := decider.NewRepository[UserState](store, bus, d)

// Execute a command
err := repo.Execute(ctx, aggID, "User",
    func(state UserState, version event.Version) ([]event.Event, error) {
        if state.Name != "" {
            return nil, fmt.Errorf("user already exists")
        }
        return event.NewEvents(aggID, "User", version,
            []event.Type{"user.created"},
            []any{UserCreated{Name: "Alice", Email: "alice@example.com"}},
        )
    },
)

// Load current state
state, version, _ := repo.Load(ctx, aggID, "User")
```

## Commands

```go
type CreateUserCmd struct {
    *command.BasicCommand
    Email string
}

// Type-safe handler — cmd is *CreateUserCmd, not command.Command
cmds := command.NewDispatcher()
command.RegisterTyped(cmds, "user.create",
    func(ctx context.Context, cmd *CreateUserCmd) error {
        return repo.Execute(ctx, cmd.AggregateID(), "User", decideCreate(cmd))
    },
)

// Dispatch
cmd := &CreateUserCmd{
    BasicCommand: command.MustNew("user.create", aggID),
    Email:        "alice@example.com",
}
cmds.Dispatch(ctx, cmd)
```

## Queries

```go
queries := query.NewDispatcher()
query.RegisterTyped[*GetUserResult](queries, "user.get",
    func(ctx context.Context, q query.Query) (*GetUserResult, error) {
        return &GetUserResult{Name: "Alice"}, nil
    },
)

result, err := query.DispatchTyped[*GetUserResult](ctx, queries, q)
```

## Events

```go
// Single event
evt, _ := event.NewEvent("user.created", aggID, "User", 1,
    UserCreated{Name: "Alice"},
    event.WithCorrelationID(correlationID),
    event.WithUserID(userID),
)

// Batch creation
events, _ := event.NewEvents(aggID, "User", 0,
    []event.Type{"user.created", "user.email.verified"},
    []any{UserCreated{Name: "Alice"}, EmailVerified{At: time.Now()}},
)

// Decode payload
payload, _ := event.DecodePayload[UserCreated](evt, event.JSONCodec{})
```

### Store & Bus Interfaces

```go
// Store = EventSink + EventSource (ISP split)
type EventSink interface {
    Save(ctx, aggType, aggID, events, expectedVersion) error
    AppendBatch(ctx, aggType, aggID, events) error
}

type EventSource interface {
    Load(ctx, aggType, aggID) ([]Event, error)
    LoadFromVersion(ctx, aggType, aggID, fromVersion) ([]Event, error)
    LoadToVersion(ctx, aggType, aggID, maxVersion) ([]Event, error)
    LoadToTimestamp(ctx, aggType, aggID, maxTime) ([]Event, error)
}

type Store interface { EventSink; EventSource }

type Bus interface {
    Publish(ctx, ...Event) error
    Subscribe(eventType, handler) error
    SubscribeAll(handler) error
}
```

## Branded IDs

Prevents mixing up different ID types at compile time:

```go
type OrderID = id.Of[orderMarker]
type UserID = id.Of[userMarker]

orderID := id.New[OrderID]()
userID := id.New[UserID]()
// store.Save(ctx, "Order", userID, ...) // won't compile — expects AggregateID
```

## Key Types

| Type                   | Package        | Purpose                                             |
| ---------------------- | -------------- | --------------------------------------------------- |
| `event.Version`        | `core/event`   | Strong-typed event version (int)                    |
| `event.Type`           | `core/event`   | Event type string                                   |
| `event.AggregateType`  | `core/event`   | Aggregate type string                               |
| `event.SchemaVersion`  | `core/event`   | Event schema version for upcasting                  |
| `command.BasicCommand` | `core/command` | Embed in command structs for interface satisfaction |
| `query.Pagination`     | `core/query`   | Page size + cursor for paginated queries            |
| `id.AggregateID`       | `core/pkg/id`  | `id.Of[aggregateMarker]` — the primary ID type      |

## Dependencies

| Dependency        | Purpose                                                                                |
| ----------------- | -------------------------------------------------------------------------------------- |
| `oklog/ulid/v2`   | Binary-sortable, time-ordered identifiers                                              |
| `go-branded-id`   | Generic branded ID type backing `id.Of[T]`                                             |
| `go-error-family` | Error classification taxonomy (Rejection/Conflict/Transient/Infrastructure/Corruption) |

## Error Classification

```go
// Domain rejections (client error, not retryable)
return event.NewRejection("user.create.empty_email", "email is required")

// Conflicts (optimistic concurrency, duplicate)
return event.NewConflict("user.create.duplicate", "user already exists")

// Transient (retryable)
return event.NewTransient("user.create.timeout", "operation timed out")
```
