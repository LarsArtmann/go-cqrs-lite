# go-composable-business-types/id Integration Plan

## Executive Summary

This document outlines how `go-composable-business-types/id` should be integrated into `go-cqrs-lite` to replace plain string identifiers with branded, type-safe IDs. The integration provides compile-time safety against mixing different entity IDs while maintaining backward compatibility through gradual adoption.

## Current State Analysis

### ID Usage Patterns in go-cqrs-lite

The codebase currently uses plain `string` for all identifiers:

| Location                    | Current Type           | Purpose                    |
| --------------------------- | ---------------------- | -------------------------- |
| `event/event.go:41`         | `AggregateID() string` | Event aggregate identifier |
| `event/event.go:102`        | `aggregateID string`   | BaseEvent field            |
| `event/store.go:10`         | `aggregateID string`   | Store method parameter     |
| `aggregate/aggregate.go:11` | `ID() string`          | Aggregate root identifier  |
| `aggregate/aggregate.go:21` | `id string`            | Base aggregate field       |
| `command/command.go:9`      | `AggregateID() string` | Command interface          |
| `command/command.go:15`     | `aggregateID string`   | BaseCommand field          |

### Problem Statement

Current implementation allows dangerous runtime bugs:

```go
// Current: Compiles but is WRONG
event1, _ := event.NewEvent("UserCreated", userID, "Order", 1, nil)  // Wrong type!
event2, _ := event.NewEvent("OrderCreated", orderID, "User", 1, nil)  // Wrong type!
```

### go-composable-business-types/id Features

The library provides branded identifiers using phantom types:

```go
// Brand types (empty structs for compile-time distinction)
type UserBrand struct{}
type OrderBrand struct{}

// Type aliases for convenience
type UserID = id.ID[UserBrand, string]
type OrderID = id.ID[OrderBrand, string]

// Compile-time safety
userID := id.NewID[UserBrand]("user-123")
orderID := id.NewID[OrderBrand]("order-456")

ProcessOrder(userID)  // ERROR: cannot use UserID as OrderID
```

**Key capabilities:**

- Generic phantom type branding (`ID[Brand, ValueType]`)
- Multiple value types: `string`, `int64`, `int`, `uint64`, etc.
- Full serialization support: JSON, SQL, Binary, Text, Gob
- Comparison and sorting for ordered types
- Zero value handling (`IsZero()`, `Or()`)
- NanoId integration for cryptographically secure IDs

## Integration Strategy

### Design Philosophy

1. **Optional Enhancement**: Integration must be opt-in to maintain backward compatibility
2. **Library-Agnostic Core**: Core interfaces remain using `string` for maximum compatibility
3. **Type-Safe Wrappers**: Provide wrapper types that use branded IDs internally
4. **Zero Breaking Changes**: Existing code continues to work unchanged

### Recommended Approach: Extension Package

Create an `xtypes/` (extended types) package that provides type-safe wrappers around core types:

```
go-cqrs-lite/
├── event/           # Core - unchanged, uses string
├── command/         # Core - unchanged, uses string
├── aggregate/       # Core - unchanged, uses string
├── xtypes/          # Extension - type-safe wrappers
│   ├── event.go     # TypedEvent, TypedEventBuilder
│   ├── command.go   # TypedCommand, TypedCommandBuilder
│   ├── aggregate.go # TypedAggregate
│   └── id.go        # Type aliases for common ID types
```

### Implementation Details

#### 1. Type Aliases (xtypes/id.go)

```go
package xtypes

import (
    "github.com/larsartmann/go-composable-business-types/id"
    "github.com/larsartmann/go-composable-business-types/nanoid"
)

// AggregateID identifies aggregate roots
type AggregateBrand struct{}
type AggregateID = id.ID[AggregateBrand, string]
type AggregateIDNano = id.ID[AggregateBrand, nanoid.NanoId]

// EventID identifies domain events
type EventBrand struct{}
type EventID = id.ID[EventBrand, string]
type EventIDNano = id.ID[EventBrand, nanoid.NanoId]

// CommandID identifies commands (for idempotency)
type CommandBrand struct{}
type CommandID = id.ID[CommandBrand, string]
type CommandIDNano = id.ID[CommandBrand, nanoid.NanoId]

// CorrelationID for distributed tracing
type CorrelationBrand struct{}
type CorrelationID = id.ID[CorrelationBrand, string]

// UserID for audit trails
type UserBrand struct{}
type UserID = id.ID[UserBrand, string]
```

#### 2. Typed Event Builder (xtypes/event.go)

```go
package xtypes

import (
    "context"
    "time"

    "github.com/larsartmann/go-cqrs-lite/event"
    "github.com/larsartmann/go-composable-business-types/id"
)

// TypedEvent wraps event.BaseEvent with type-safe IDs
type TypedEvent[A any, V comparable] struct {
    base        *event.BaseEvent
    aggregateID id.ID[A, V]
}

// AggregateID returns the typed aggregate ID
func (e *TypedEvent[A, V]) AggregateID() id.ID[A, V] {
    return e.aggregateID
}

// ToEvent returns the underlying event.Event (for Store operations)
func (e *TypedEvent[A, V]) ToEvent() event.Event {
    return e.base
}

// EventBuilder constructs events with type safety
type EventBuilder[A any, V comparable] struct {
    eventType     event.EventType
    aggregateID   id.ID[A, V]
    aggregateType event.AggregateType
    version       int
    payload       []byte
    opts          []event.EventOption
}

func NewEventBuilder[A any, V comparable](
    eventType event.EventType,
    aggregateID id.ID[A, V],
    aggregateType event.AggregateType,
    version int,
) *EventBuilder[A, V] {
    return &EventBuilder[A, V]{
        eventType:     eventType,
        aggregateID:   aggregateID,
        aggregateType: aggregateType,
        version:       version,
    }
}

func (b *EventBuilder[A, V]) WithPayload(payload []byte) *EventBuilder[A, V] {
    b.payload = payload
    return b
}

func (b *EventBuilder[A, V]) WithMetadata(opts ...event.EventOption) *EventBuilder[A, V] {
    b.opts = append(b.opts, opts...)
    return b
}

func (b *EventBuilder[A, V]) Build() (*TypedEvent[A, V], error) {
    if b.aggregateID.IsZero() {
        return nil, fmt.Errorf("aggregate ID is required")
    }

    base, err := event.NewEvent(
        b.eventType,
        b.aggregateID.String(),
        b.aggregateType,
        b.version,
        b.payload,
        b.opts...,
    )
    if err != nil {
        return nil, err
    }

    return &TypedEvent[A, V]{
        base:        base,
        aggregateID: b.aggregateID,
    }, nil
}
```

#### 3. Typed Command (xtypes/command.go)

```go
package xtypes

import (
    "github.com/larsartmann/go-cqrs-lite/command"
    "github.com/larsartmann/go-composable-business-types/id"
)

// TypedCommand wraps commands with branded aggregate IDs
type TypedCommand[A any, V comparable] struct {
    commandType Type
    aggregateID id.ID[A, V]
    payload     any
}

func (c *TypedCommand[A, V]) Type() command.Type {
    return c.commandType
}

func (c *TypedCommand[A, V]) AggregateID() id.ID[A, V] {
    return c.aggregateID
}

func (c *TypedCommand[A, V]) Payload() any {
    return c.payload
}

// ToCommand creates a command.Command for dispatching
func (c *TypedCommand[A, V]) ToCommand() command.Command {
    return command.New(c.commandType, c.aggregateID.String())
}
```

#### 4. Typed Aggregate (xtypes/aggregate.go)

```go
package xtypes

import (
    "github.com/larsartmann/go-cqrs-lite/aggregate"
    "github.com/larsartmann/go-cqrs-lite/event"
    "github.com/larsartmann/go-composable-business-types/id"
)

// TypedAggregate provides type-safe aggregate roots
type TypedAggregate[A any, V comparable] struct {
    base        *aggregate.Base
    id          id.ID[A, V]
    aggregateType event.AggregateType
}

func NewTypedAggregate[A any, V comparable](
    id id.ID[A, V],
    aggregateType event.AggregateType,
) *TypedAggregate[A, V] {
    return &TypedAggregate[A, V]{
        base:          aggregate.NewBase(id.String(), aggregateType),
        id:            id,
        aggregateType: aggregateType,
    }
}

func (a *TypedAggregate[A, V]) ID() id.ID[A, V] {
    return a.id
}

func (a *TypedAggregate[A, V]) Type() event.AggregateType {
    return a.aggregateType
}

func (a *TypedAggregate[A, V]) Version() int {
    return a.base.Version()
}

func (a *TypedAggregate[A, V]) Base() *aggregate.Base {
    return a.base
}

// ApplyEvent records a typed event
func (a *TypedAggregate[A, V]) ApplyEvent(evt *TypedEvent[A, V]) {
    a.base.ApplyEvent(nil, evt.ToEvent())
}
```

### Usage Examples

#### Basic Usage (with string IDs)

```go
package main

import (
    "github.com/larsartmann/go-cqrs-lite/xtypes"
    "github.com/larsartmann/go-cqrs-lite/event"
)

// Define domain-specific brands
type UserBrand struct{}
type OrderBrand struct{}

// Create type-safe IDs
type UserID = xtypes.AggregateID // Or use custom: id.ID[UserBrand, string]
type OrderID = xtypes.AggregateID

func main() {
    userID := xtypes.NewAggregateID("user-123")

    // Build event with type safety
    evt, err := xtypes.NewEventBuilder(
        "UserCreated",
        userID,
        event.AggregateType("User"),
        1,
    ).WithPayload([]byte(`{"name":"John"}`)).
        Build()

    // Pass to store (converts to string internally)
    store.Save(ctx, evt.ToEvent())
}
```

#### Advanced Usage (with NanoId)

```go
import (
    "github.com/larsartmann/go-composable-business-types/nanoid"
    "github.com/larsartmann/go-cqrs-lite/xtypes"
)

// Use NanoId for cryptographically secure IDs
type UserID = xtypes.AggregateIDNano

func CreateUser() {
    userID := xtypes.NewAggregateIDNano(nanoid.NewNanoId())

    // Type-safe throughout the application
    evt, _ := xtypes.NewEventBuilder(
        "UserCreated",
        userID,
        "User",
        1,
    ).Build()
}
```

#### Handler Implementation

```go
// Type-safe command handler
type CreateUserHandler struct {
    store event.Store
}

type CreateUserCommand struct {
    UserID   xtypes.AggregateIDNano
    Email    string
    Username string
}

func (h *CreateUserHandler) Handle(
    ctx context.Context,
    cmd *CreateUserCommand,
) error {
    // Compile-time safety: can't accidentally use wrong ID type
    user := xtypes.NewTypedAggregate(cmd.UserID, "User")

    evt, err := xtypes.NewEventBuilder(
        "UserCreated",
        cmd.UserID,  // Type-safe: must match aggregate ID type
        "User",
        user.Version()+1,
    ).WithPayload(payload).
        Build()

    user.ApplyEvent(evt)

    return h.store.Save(ctx, evt.ToEvent())
}
```

## Migration Path

### Phase 1: Add Extension Package (No Breaking Changes)

1. Create `xtypes/` package with all typed wrappers
2. Add comprehensive tests and examples
3. Update documentation with usage patterns
4. **Users opt-in by importing `xtypes`**

### Phase 2: Gradual Adoption (User-Driven)

```go
// Before: Plain strings (still works)
cmd := &CreateUserCommand{
    AggregateID: "user-123",  // string
}

// After: Type-safe (opt-in)
cmd := &CreateUserCommand{
    UserID: xtypes.NewAggregateID("user-123"),  // branded ID
}
```

### Phase 3: Future Considerations (v2)

If a v2 release is planned, consider:

- Making branded IDs the default in core interfaces
- Providing `compat/` package for migration
- Generic interfaces: `Event[A any, V comparable]`

## go.mod Changes

```go
module github.com/larsartmann/go-cqrs-lite

go 1.21

require (
    github.com/cockroachdb/errors v1.12.0
    github.com/google/uuid v1.6.0
)

require (
    // xtypes/ package uses this (optional dependency)
    github.com/larsartmann/go-composable-business-types v1.0.0
)
```

**Note**: The core packages (`event/`, `command/`, `aggregate/`) remain dependency-free. Only `xtypes/` imports the external library.

## Benefits

| Aspect        | Before                 | After                            |
| ------------- | ---------------------- | -------------------------------- |
| Type Safety   | Runtime errors         | Compile-time errors              |
| Refactoring   | Risky, manual checks   | Safe, compiler-assisted          |
| Documentation | Comments only          | Self-documenting types           |
| Testing       | Need integration tests | Unit tests catch type mismatches |
| ID Generation | Manual validation      | NanoId integration available     |
| Serialization | Manual handling        | Built-in JSON/SQL support        |

## Trade-offs

| Concern               | Impact          | Mitigation                     |
| --------------------- | --------------- | ------------------------------ |
| Additional dependency | Only in xtypes/ | Core remains zero-dep          |
| Learning curve        | New type syntax | Comprehensive examples         |
| Verbosity             | Generic syntax  | Type aliases reduce repetition |
| Performance           | None            | ID is just a wrapper struct    |
| Migration effort      | Opt-in only     | No forced changes              |

## Conclusion

The `go-composable-business-types/id` library provides significant value for CQRS applications where entity identity is critical. The recommended extension package approach allows users to opt into type safety while maintaining the library's zero-dependency philosophy for the core packages.

### Immediate Next Steps

1. Create `xtypes/` directory structure
2. Implement `id.go` with type aliases
3. Implement `event.go` with `TypedEvent` and `EventBuilder`
4. Implement `command.go` with `TypedCommand`
5. Implement `aggregate.go` with `TypedAggregate`
6. Add comprehensive test suite
7. Create usage examples
8. Update main README with xtypes section

### Success Criteria

- [ ] Extension package compiles without errors
- [ ] All types integrate seamlessly with core packages
- [ ] Existing tests pass without modification
- [ ] New tests cover type safety scenarios
- [ ] Examples demonstrate common usage patterns
- [ ] Documentation explains migration path
