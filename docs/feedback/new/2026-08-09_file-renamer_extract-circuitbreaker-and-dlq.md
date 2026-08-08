# Feedback: Extract CircuitBreaker and DeadLetterStore as standalone modules

> **From**: file-and-image-renamer consumer
> **Date**: 2026-08-09
> **Priority**: Medium (would enable replacing hand-rolled implementations)

---

## Context

The `file-and-image-renamer` project is migrating to full CQRS/ES using go-cqrs-lite.
During the integration audit, two abstractions were identified as genuinely useful
but currently trapped inside heavier modules:

1. **CircuitBreaker** — trapped in `middleware/v4` (15+ deps)
2. **DeadLetterStore** — trapped in `projectionhost/v4` (17+ deps)

Both are currently hand-rolled in the consumer because the trapped versions have
APIs that don't fit standalone use cases.

---

## Request 1: `circuitbreaker/v4` standalone module

### What's needed

A zero-dependency circuit breaker module with the core state machine:

```go
package circuitbreaker

type CircuitBreaker struct { /* ... */ }

func New(config Config) *CircuitBreaker

// Core state machine
func (cb *CircuitBreaker) Allow() bool
func (cb *CircuitBreaker) RecordSuccess()
func (cb *CircuitBreaker) RecordFailure()

// State inspection
func (cb *CircuitBreaker) State() State  // Closed | Open | HalfOpen
```

### Why it can't use `middleware/v4`

The middleware version wraps the state machine inside `Middleware[M]` return types
designed for the command/event/query pipeline chain. The consumer needs the raw
state machine — `Allow()`, `RecordSuccess()`, `RecordFailure()` — to gate calls to
an external AI vision API provider chain (GLM → Synthetic fallback), not to wrap
a dispatcher middleware.

The middleware version can then compose `circuitbreaker/v4` internally, keeping
the same public API while delegating to the standalone module.

### Current hand-rolled implementation

`pkg/provider/circuit_breaker.go` — ~100 lines using `sync/atomic` for the
state machine. Works fine but duplicates logic that should live in the library.

---

## Request 2: `dlq/v4` standalone module

### What's needed

A generic dead-letter store that captures failed operations for later retry,
not tied to the projection pipeline:

```go
package dlq

type Entry[P any] struct {
    ID          string
    Payload     P
    Error       error
    Attempts    int
    Status      Status  // Pending | Retrying | Resolved | Ignored
    Timestamp   time.Time
    LastRetry   time.Time
}

type Store[P any] interface {
    Add(ctx context.Context, entry Entry[P]) error
    Get(ctx context.Context, id string) (Entry[P], error)
    List(ctx context.Context) ([]Entry[P], error)
    Update(ctx context.Context, entry Entry[P]) error
    Delete(ctx context.Context, id string) error
    Purge(ctx context.Context, status Status) error
}
```

### Why it can't use `projectionhost/v4`

The projectionhost version's `DeadLetterStore` is tied to projections:

```go
type DeadLetterEntry struct {
    ProjectionName string
    Event          event.Event   // requires event/v4
    Error          string
}
```

The consumer's dead-letter entries are domain-specific (file paths, error types,
retry counts, operator actions) and don't have an `event.Event` payload. A generic
`dlq/v4` with `Entry[P any]` would serve both the projection use case (via
`Entry[event.Event]`) and the consumer's use case (via `Entry[FilePayload]`).

The projectionhost can then embed or delegate to `dlq/v4`, the same way middleware
would compose `circuitbreaker/v4`.

### Current hand-rolled implementation

`pkg/deadletter/deadletter.go` — JSON file-backed store with `AddEntry`,
`MarkRetrying`, `MarkResolved`, `MarkIgnored`, `DeleteEntry`, `GetAll`. ~200 lines.
Works but is consumer-specific and not reusable.

---

## What's NOT needed

These modules were audited and correctly identified as not applicable to a
file-renaming CLI — no changes requested:

- `dedup/v4` — ID ring buffer; project uses content-hash dedup
- `flightrecorder/v4` — no flight recorder in this project
- `scheduling/v4` — no durable timers needed
- `metadata/v4` — event metadata is handled by the decider framework

---

## Priority

Neither request blocks the CQRS migration — the hand-rolled versions work.
But extracting them would:

1. Eliminate ~300 lines of duplicated logic in the consumer
2. Make the library more composable for non-projection use cases
3. Allow the consumer to delete `pkg/provider/circuit_breaker.go` and `pkg/deadletter/`
