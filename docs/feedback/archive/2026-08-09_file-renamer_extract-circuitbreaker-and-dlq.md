# Feedback: Extract CircuitBreaker and DeadLetterStore as standalone modules

> **From**: file-and-image-renamer consumer
> **Date**: 2026-08-09
> **Priority**: Medium (would enable replacing hand-rolled implementations)

---

## Prependix — Review & Disposition (2026-08-09)

**Decision: No new modules.** Both requests resolve to "the library is correct as-is; the consumer's need is met by existing tooling or is out of scope."

### CircuitBreaker → point at failsafe-go directly (docs, not a module)

The consumer assumes a zero-dependency state machine must be extracted because the library version is "trapped" in `middleware/v4`. This misses that `middleware/circuit_breaker.go` **delegates the entire state machine to [failsafe-go](https://github.com/failsafe-go/failsafe-go)** — it does not implement one. The library deliberately migrated away from a hand-rolled `sync/atomic` breaker to failsafe-go ([AGENTS.md principle #17](../../../AGENTS.md)).

failsafe-go already exposes the exact API the consumer wants:

| Consumer asks for | failsafe-go equivalent         |
| ----------------- | ------------------------------ |
| `Allow() bool`    | `TryAcquirePermit() bool`      |
| `RecordSuccess()` | `RecordSuccess()`              |
| `RecordFailure()` | `RecordError(err)`             |
| `State() State`   | `State() circuitbreaker.State` |

**No module.** A thin facade would add naming cosmetics (`Allow` vs `TryAcquirePermit`) at the cost of a leaky abstraction: consumers would inevitably want failsafe-go's real features (sliding windows, state-transition listeners, metrics hooks) exposed, forcing the facade to grow into the thing it wraps. `middleware/circuit_breaker.go` itself is ~20 lines of integration glue — the right reference implementation to point at.

**Action**: add a SKILL.md FAQ entry directing standalone circuit-breaker consumers to failsafe-go, with a cross-reference to `middleware/circuit_breaker.go` as the integration pattern.

### DLQ → don't extract; the consumer's use case is a different abstraction

The consumer's `pkg/deadletter/deadletter.go` stores **failed file-rename operations** (file paths, error types, retry counts, operator actions). That is an **application-level retry queue**, not a dead-letter queue. This is a CQRS/ES library; dead-lettering is a _message_ concept:

- **Projection poison events** → `projectionhost.DeadLetterStore` (already exists, correctly event-specific)
- **Commands that exhaust retries** → would be CQRS-aligned, but the consumer is not asking for this

The projectionhost DLQ is tightly coupled to `event.Event` _by design_:

- `DeadLetterEntry` carries `ProjectionName`, `EventID`, `EventType`, `StreamID`, `ErrorCode`, `ErrorFamily`
- The SQLite impl reconstructs events via `event.ReconstructEventFromFields` for replay
- The `Store`/`List`/`Delete`/`Purge` surface assumes message-quarantine semantics

Genericizing this into `Entry[P any]` would either lose that richness or force projectionhost to maintain a parallel typed layer on top — the exact coupling the consumer is trying to escape, now inside the library. A generic failed-work queue is general-purpose application infrastructure, not CQRS/ES building material.

**No module.** The consumer's 200-line JSON-backed retry store is the right shape for _their_ domain. It is bespoke application logic, not duplicated library logic.

### Summary

| Request             | Disposition             | Rationale                                                                               |
| ------------------- | ----------------------- | --------------------------------------------------------------------------------------- |
| `circuitbreaker/v4` | Docs pointer, no module | failsafe-go IS the standalone breaker; a facade is a leaky abstraction                  |
| `dlq/v4`            | No module, out of scope | Consumer needs an app-level retry queue; projectionhost DLQ is event-specific by design |

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
