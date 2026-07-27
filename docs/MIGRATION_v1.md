# Migration Guide: Upgrading to go-cqrs-lite v1.0.0

This guide covers all breaking changes and migration paths from pre-v1.0.0 to v1.0.0 of go-cqrs-lite.

## Table of Contents

- [Module Import Paths](#module-import-paths)
- [Replace Directive Removal](#replace-directive-removal)
- [Error Handling: 6-Family Taxonomy](#error-handling-6-family-taxonomy)
- [Event Store: ISP Split](#event-store-isp-split)
- [Deprecated API Removals](#deprecated-api-removals)
- [Decider Pattern Over OO Aggregate](#decider-pattern-over-oo-aggregate)
- [OTel Observability](#otel-observability)
- [Event Signing](#event-signing)

---

## Module Import Paths

All modules use the `github.com/larsartmann/go-cqrs-lite/<module>` path. No changes required if you were already importing from the correct paths.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/command"
    "github.com/larsartmann/go-cqrs-lite/event"
    "github.com/larsartmann/go-cqrs-lite/decider"
    "github.com/larsartmann/go-cqrs-lite/memory"
    "github.com/larsartmann/go-cqrs-lite/storage"
    "github.com/larsartmann/go-cqrs-lite/middleware"
    "github.com/larsartmann/go-cqrs-lite/projection"
    "github.com/larsartmann/go-cqrs-lite/catalog"
    "github.com/larsartmann/go-cqrs-lite/signing"
    "github.com/larsartmann/go-cqrs-lite/listing"
    "github.com/larsartmann/go-cqrs-lite/otel"
)
```

## Replace Directive Removal

**Before (pre-v1.0.0):** You needed `replace` directives because modules weren't tagged:

```go
// go.mod
require github.com/larsartmann/go-cqrs-lite/core v1.6.0

replace github.com/larsartmann/go-cqrs-lite/core => ../core
```

**After (v1.0.0):** All modules are tagged. Remove all `replace` directives:

```go
// go.mod
require github.com/larsartmann/go-cqrs-lite/core v1.0.0
// No replace directive needed
```

### Migration Steps

1. Remove all `replace` blocks from your `go.mod`
2. Run `go get github.com/larsartmann/go-cqrs-lite/<module>@v1.0.0` for each module
3. Run `go mod tidy`

## Error Handling: 6-Family Taxonomy

v1.0.0 adopts the `go-error-family` error taxonomy. All errors returned by library functions are classified into one of six families (five at v1.0.0; Orchestration added in go-error-family v0.10.0):

| Family             | Factory                        | Meaning                    | Example                |
| ------------------ | ------------------------------ | -------------------------- | ---------------------- |
| **Rejection**      | `event.NewRejection(...)`      | Business rule violation    | "not found"            |
| **Conflict**       | `event.NewConflict(...)`       | State conflict             | "duplicate type"       |
| **Transient**      | `event.NewTransient(...)`      | Retryable failure          | "connection reset"     |
| **Infrastructure** | `event.NewInfrastructure(...)` | Non-retryable system error | "database unreachable" |
| **Corruption**     | `event.NewCorruption(...)`     | Data integrity violation   | "checksum mismatch"    |
| **Orchestration**  | `errorfamily.NewOrchestration(...)` | Workflow/saga coordination | "compensation failed" |

### Migration

If you were checking for sentinel errors, switch to the classification API:

```go
// Before: checking specific errors
if errors.Is(err, event.ErrAggregateNotFound) { ... }

// After: also check the family
if event.IsRejection(err) {
    // Business rule violation — don't retry
}
if event.IsTransient(err) {
    // Transient — safe to retry
}
```

The sentinel errors still work with `errors.Is()`, but the family check provides coarser-grained handling.

## Event Store: ISP Split

The `event.Store` interface was split into focused sub-interfaces following the Interface Segregation Principle:

| Interface           | Methods                                                       | Purpose              |
| ------------------- | ------------------------------------------------------------- | -------------------- |
| `EventSink`         | `Save`, `AppendBatch`                                         | Write events         |
| `EventSource`       | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` | Read events          |
| `Journal`           | `ReadAll`                                                     | Cross-aggregate read |
| `SeekableJournal`   | `ReadFrom`                                                    | Position-based read  |
| `BackwardsSource`   | `LoadBackwards`                                               | Reverse iteration    |
| `TransactionalSink` | `SaveWithOutbox`                                              | Transactional write  |
| `Store`             | `EventSink` + `EventSource`                                   | Full read/write      |

### Migration

```go
// Before: everything was event.Store
func ProcessEvents(store event.Store) { ... }

// After: accept the smallest interface you need
func PublishEvents(sink event.EventSink) { ... }
func ReplayHistory(journal event.Journal) { ... }
func ReadFromPosition(seekable event.SeekableJournal) { ... }
```

## Deprecated API Removals

The following methods are deprecated and will be removed in a future major version. Migrate now:

### `event.Runner` → `projection.Runner`

```go
// Before: event.Runner (deprecated)
runner := event.NewRunner(bus, handler)

// After: projection.Runner (replay + live)
runner, _ := projection.NewRunner(journal, subscriber, checkpointStore,
    projection.WithLogger(logger),
)
runner.Register(myProjection)
runner.Run(ctx)
```

### Direct Store Methods → ISP Interfaces

```go
// Before: ReadAll directly on store (deprecated)
events, _ := store.ReadAll(ctx)

// After: use Journal interface
var journal event.Journal = store
events, _ := journal.ReadAll(ctx)

// Before: ReadFrom directly on store (deprecated)
events, _ := store.ReadFrom(ctx, position, limit)

// After: use SeekableJournal interface
var seekable event.SeekableJournal = store
events, _ := seekable.ReadFrom(ctx, position, limit)
```

## Decider Pattern Over OO Aggregate

v1.0.0 recommends the `decider.Decider[State]` pattern over mutable aggregate roots. See [ADR-0001](adr/0001-decider-over-aggregate.md) for the rationale.

```go
// Decider: pure function style
decider := decider.Decider[UserState]{
    Initial: UserState{},
    Fold:    foldUserEvents,
}

repo, _ := decider.NewRepository[UserState](store, publisher, decider)

err := repo.Execute(ctx, userID, "User", func(state UserState, version event.Version) ([]event.Event, error) {
    // Pure decision function — no mutations, no I/O
    return []event.Event{
        must1(event.NewEvent("UserCreated", userID, "User", version+1, UserCreated{Name: "Alice"})),
    }, nil
})
```

### Benefits

- **Testability**: `Fold` is a pure function — test without mocks
- **Immutability**: State is never mutated in place
- **Explicitness**: All side effects (save, publish) happen in `Execute`

## OTel Observability

v1.0.0 adds OpenTelemetry tracing across all modules. Opt-in, zero overhead when no provider is configured.

```go
import (
    "go.opentelemetry.io/otel"
    cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
    "github.com/larsartmann/go-cqrs-lite/middleware"
)

// Enable OTel tracing on command dispatcher
tracer := otel.GetTracerProvider().Tracer("my-app")
cmdDispatcher.Use(middleware.CommandTracing(tracer))

// Enable OTel tracing on event bus
bus.Use(middleware.EventTracing(tracer))
bus.UsePublish(middleware.EventPublishTracing(tracer))

// All storage methods emit spans automatically
store, _ := storage.NewSQLiteEventStore(db)
// store.Save, store.Load, etc. emit OTel spans
```

### Custom Span Attributes

```go
import cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"

// Use semantic attribute helpers
attrs := cqrsotel.EventAttrs("UserCreated", userID, "User")
attrs = append(attrs, cqrsotel.AggregateAttrs("User", userID, 5)...)
```

## Event Signing

v1.0.0 adds tamper-proof event stream signing via HMAC-SHA256 or Ed25519:

```go
import "github.com/larsartmann/go-cqrs-lite/signing"

// HMAC signing
signer, _ := signing.NewHMAC([]byte("secret-key"))

// Sign on publish, verify on receive
bus.UsePublish(signing.SignMiddleware(signer))
bus.Use(signing.VerifyMiddleware(signer))
```

---

## Full Upgrade Checklist

- [ ] Remove all `replace` directives from `go.mod`
- [ ] Update all module requires to `v1.0.0`
- [ ] Run `go mod tidy`
- [ ] Replace `event.Runner` with `projection.Runner`
- [ ] Use ISP interfaces (`EventSink`, `Journal`, `SeekableJournal`) instead of direct method calls
- [ ] Adopt `decider.Decider[State]` for new aggregates
- [ ] Add error family checks (`event.IsRejection`, `event.IsTransient`) where appropriate
- [ ] Enable OTel tracing for observability (optional)
- [ ] Add event signing for tamper-proof streams (optional)
