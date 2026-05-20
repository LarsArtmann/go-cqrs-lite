# Migration Guide

This guide covers breaking changes and migration paths for go-cqrs-lite consumers.

## v1.4.0 (core)

### `Command.IdempotencyKey()` added to interface

The `command.Command` interface now requires an `IdempotencyKey() string` method.

**Before:**
```go
type CreateOrder struct { /* ... */ }
func (c *CreateOrder) Type() command.Type          { return "create_order" }
func (c *CreateOrder) AggregateID() id.AggregateID { return c.aggregateID }
```

**After:**
```go
type CreateOrder struct { /* ... */ }
func (c *CreateOrder) Type() command.Type          { return "create_order" }
func (c *CreateOrder) AggregateID() id.AggregateID { return c.aggregateID }
func (c *CreateOrder) IdempotencyKey() string      { return c.idempotencyKey }
```

For quick migration, embed `command.Core` which provides a default `IdempotencyKey()` returning `""`:

```go
type CreateOrder struct {
    *command.Core
    // ...
}
```

### `Event.Version()` returns `event.Version` instead of `int`

**Before:**
```go
version := evt.Version() // int
```

**After:**
```go
version := evt.Version() // event.Version
v := version.Int()       // int, when needed
```

### `event.SchemaVersion` is a distinct type

`SchemaVersion` is no longer a bare `int`. Use `event.SchemaVersion(1)` or `event.ParseSchemaVersion("1")`.

### `event.EveryNEvents` returns `(SnapshotStrategy, error)` instead of `SnapshotStrategy`

**Before:**
```go
strategy := event.EveryNEvents(10)
```

**After:**
```go
strategy, err := event.EveryNEvents(10)
// or
strategy := event.MustEveryNEvents(10) // panics on invalid input
```

### New `Store` interface methods: `LoadToVersion`, `LoadToTimestamp`

If you implement `event.Store` directly, add these methods:

```go
func (s *MyStore) LoadToVersion(ctx context.Context, aggType event.AggregateType, aggID id.AggregateID, maxVersion event.Version) ([]event.Event, error)
func (s *MyStore) LoadToTimestamp(ctx context.Context, aggType event.AggregateType, aggID id.AggregateID, maxTime time.Time) ([]event.Event, error)
```

## v1.2.0 (memory)

Implements `event.PositionalLoader` with `LoadAllFromPosition`.
Implements `LoadToVersion` and `LoadToTimestamp` on `MemoryStore`.

## v1.1.0 (middleware)

### `RetryConfig.IsRetryable` defaults to `event.IsRetryable`

Previously, `DefaultRetryConfig().IsRetryable` always returned `false`. Now it uses the error classification system. Override if you need custom behavior:

```go
cfg := middleware.DefaultRetryConfig()
cfg.IsRetryable = func(err error) bool { return false } // disable retry
```

## New APIs (non-breaking)

### `command.RegisterTyped[T]`

Type-safe command handler registration:

```go
// Before
dispatcher.Register("create_user", func(ctx context.Context, cmd command.Command) error {
    c := cmd.(*CreateUserCmd) // manual type assertion
    return handleCreate(ctx, c)
})

// After
command.RegisterTyped(dispatcher, "create_user", func(ctx context.Context, c *CreateUserCmd) error {
    return handleCreate(ctx, c)
})
```

### `query.RegisterTyped[T]`

Type-safe query handler with typed return:

```go
query.RegisterTyped(dispatcher, "get_user", func(ctx context.Context, q query.Query) (*User, error) {
    return &User{Name: "Alice"}, nil
})

result, err := query.DispatchTyped[*User](ctx, dispatcher, &GetUserQuery{})
```

### `event.NewEvents` / `event.DecodePayloads`

Batch event creation and decoding:

```go
// Batch create
events, err := event.NewEvents(aggID, aggType, version,
    []event.Type{"user.created", "user.activated"},
    []any{CreatedPayload{Email: "a@b.com"}, ActivatedPayload{}},
)

// Batch decode
payloads, err := event.DecodePayloads[CreatedPayload](events, event.JSONCodec{})
```

### `event.NewTypedProjection[T]`

Auto-decoding projection handler:

```go
proj := event.NewTypedProjection[UserCreatedPayload]("user-projection",
    []event.Type{"user.created"},
    func(ctx context.Context, evt event.Event, payload UserCreatedPayload) error {
        // payload is already decoded, no manual json.Unmarshal
        return updateReadModel(ctx, payload)
    },
)
```
