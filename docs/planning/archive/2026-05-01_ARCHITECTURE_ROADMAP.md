# Architecture Roadmap: Error Handling, Offline-First & Next-Phase Evolution

> **Date:** 2026-05-01
> **Status:** Actionable Plan
> **Scope:** go-cqrs-lite + go-localfirst
> **Input Documents:**
>
> - `go-localfirst/docs/planning/2026-05-01_BRAINSTORM_ERROR_HANDLING_ARCHITECTURE.md` (2017 lines — error taxonomy, 9 families, 74 issues)
> - `go-cqrs-lite/docs/planning/2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md` (monorepo phases 0-10, 70% done)
> - `go-cqrs-lite/docs/planning/2026-05-01_OFFLINE_FIRST_EVERYTHING_ELSE.md` (15 dimensions of offline-first)
> - `go-cqrs-lite/docs/planning/2026-05-01_OFFLINE_FIRST_TIMING_ANALYSIS.md` (timing dimensions, metadata gaps)
> - `go-cqrs-lite/docs/research/2026-05-01_LIVESTORE_DEEP_DIVE.md` (LiveStore patterns)
> - `go-cqrs-lite/docs/research/2026-05-01_HYBRID_ARCHITECTURE_BEST_OF_BOTH_WORLDS.md` (aggregateless + aggregate hybrid)
> - `go-cqrs-lite/docs/research/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md` (Decider, DCB, functional ES)
> - `go-cqrs-lite/docs/research/2026-05-01_AGGREGATELESS_EVENT_SOURCING_DEEP_DIVE.md` (CTE-based contexts)
> - `go-cqrs-lite/docs/research/2026-05-01_INNOVATIVE_CQRS_EVENT_SOURCING_PROJECTS.md` (13 projects ranked)
> - `go-cqrs-lite/docs/research/datomic-lessons.md` (facts as values, time-travel)
> - `go-cqrs-lite/docs/research/time-travel-options.md` (temporal queries)

---

## Executive Summary

Three interlocking initiatives, ordered by dependency and value:

| #     | Initiative                                   | Scope                                  | Impact                         | Effort |
| ----- | -------------------------------------------- | -------------------------------------- | ------------------------------ | ------ |
| **A** | **Error Taxonomy in go-cqrs-lite**           | Library-level error types              | Foundation for everything else | Medium |
| **B** | **Offline-First Primitives in go-cqrs-lite** | Client metadata + sync building blocks | Enables go-localfirst          | Medium |
| **C** | **Error Handling & SSE in go-localfirst**    | Application-level error architecture   | Fixes 74 identified issues     | Large  |

**Critical dependency:** A → C (go-localfirst's error handling uses go-cqrs-lite's error types).

---

## Initiative A: Error Taxonomy in go-cqrs-lite

### Goal

Add structured error types to `core/event/` (or a new `core/pkg/errors/` package) so that consumers can classify errors without string matching.

### What Goes Into go-cqrs-lite (Library Scope)

The brainstorm identified 9 error families. Not all belong in the library. The library provides **building blocks**, not opinions about HTTP status codes or SSE broadcasting.

| Family         | Belongs in go-cqrs-lite? | Why                                                          |
| -------------- | ------------------------ | ------------------------------------------------------------ |
| Rejection      | ✅ Yes                   | Domain validation errors — universal                         |
| Conflict       | ✅ Yes                   | Version mismatch — `event.ErrVersionConflict` already exists |
| Transient      | ✅ Yes                   | Retryable infrastructure errors — middleware needs this      |
| Staleness      | ⚠️ Partial               | Library can detect it (version comparison), not resolve it   |
| Corruption     | ✅ Yes                   | Poison event detection — codec/store produce these           |
| Divergence     | ❌ No                    | Only relevant for distributed sync (go-localfirst)           |
| Pipeline       | ⚠️ Partial               | Library can produce projection errors, not manage DLQ        |
| Transport      | ❌ No                    | Application-level SSE/WS concern                             |
| Infrastructure | ✅ Yes                   | `ErrStoreClosed`, `ErrBusClosed` already exist               |

### Actionable Steps

#### Step A1: Create `core/pkg/errors/` package

```
core/pkg/errors/
├── errors.go      // Error type, Family enum, constructors
├── classify.go    // IsRejection(), IsConflict(), etc.
└── errors_test.go // 100% coverage
```

**API:**

```go
package errors

type Family int

const (
    Rejection      Family = iota // Bad input, not allowed, not found
    Conflict                     // Version mismatch, already exists
    Transient                    // Temporary, retryable
    Corruption                   // Poison event, damaged data
    Infrastructure               // System cannot serve
)

type Error struct {
    Code    string    // "event.version_conflict", "store.closed"
    Message string    // Human-readable
    Family  Family    // Primary classification
    Cause   error     // Wrapped error
}

// Constructors
func Reject(code, msg string) *Error
func Conflict(code, msg string) *Error
func Transient(code, msg string) *Error
func Corruption(code, msg string) *Error
func Infrastructure(code, msg string) *Error

// Fluent setter
func (e *Error) WithCause(cause error) *Error

// Classification
func IsRejection(err error) bool
func IsConflict(err error) bool
func IsTransient(err error) bool
func IsCorruption(err error) bool
func IsInfrastructure(err error) bool
```

**Size estimate:** ~120 lines + ~150 test lines. Under the 250-line file limit.

#### Step A2: Update existing sentinel errors to use the taxonomy

Map current sentinel errors to families:

| Current Error                 | Package         | Family         |
| ----------------------------- | --------------- | -------------- |
| `event.ErrVersionConflict`    | `core/event/`   | Conflict       |
| `event.ErrAggregateNotFound`  | `core/event/`   | Rejection      |
| `event.ErrStoreClosed`        | `core/event/`   | Infrastructure |
| `event.ErrBusClosed`          | `core/event/`   | Infrastructure |
| `event.ErrSnapshotNotFound`   | `core/event/`   | Rejection      |
| `command.ErrHandlerNotFound`  | `core/command/` | Infrastructure |
| `command.ErrDispatcherClosed` | `core/command/` | Infrastructure |
| `query.ErrHandlerNotFound`    | `core/query/`   | Infrastructure |
| `query.ErrDispatcherClosed`   | `core/query/`   | Infrastructure |
| `projection.ErrNilStore`      | `projection/`   | Infrastructure |

**Approach:** Keep existing sentinel errors as `errors.New(...)` for backward compat. Add a new `Classify(err) Family` function that maps known sentinels to families. This is non-breaking.

```go
func Classify(err error) Family {
    switch {
    case errors.Is(err, event.ErrVersionConflict):
        return Conflict
    case errors.Is(err, event.ErrAggregateNotFound):
        return Rejection
    case errors.Is(err, event.ErrStoreClosed),
         errors.Is(err, event.ErrBusClosed),
         errors.Is(err, command.ErrDispatcherClosed):
        return Infrastructure
    default:
        return Transient // unknown errors are retryable by default
    }
}
```

#### Step A3: Add retry-awareness to middleware

`middleware/retry.go` already retries on error. Enhance it to use `errors.IsTransient()`:

- Transient → retry with backoff
- Conflict → no retry, return immediately
- Infrastructure → no retry, return immediately
- Rejection → no retry, return immediately

This is a small behavioral change with zero API breakage.

#### Step A4: Add projection error classification

When the projection runner encounters an error:

- `IsCorruption(err)` → skip event, push to callback, continue
- `IsTransient(err)` → retry with backoff
- `IsConflict(err)` → shouldn't happen in projection, log and skip
- Default → retry, then skip

This replaces the current "stop processing on any error" behavior.

### What Does NOT Go Into go-cqrs-lite

| Concern                      | Where It Lives            | Why                     |
| ---------------------------- | ------------------------- | ----------------------- |
| SSE broadcasting of errors   | go-localfirst             | Transport-specific      |
| DLQ (dead letter queue)      | go-localfirst or consumer | Infrastructure-specific |
| HTMX error mapping           | go-localfirst             | UI-specific             |
| Divergence/CRDT errors       | go-localfirst `pkg/sync/` | Sync-specific           |
| Error notification bus       | go-localfirst             | Application pattern     |
| `system.degraded` SSE events | go-localfirst             | UI-specific             |

---

## Initiative B: Offline-First Primitives in go-cqrs-lite

### Goal

Add the minimal metadata and interfaces that enable consumers to build offline-first systems, without go-cqrs-lite itself being "offline-aware."

### Actionable Steps

#### Step B1: Add client metadata to event options

Already partially exists (`event.WithCorrelationID`). Add:

```go
// event/options.go — new option functions
func WithClientID(clientID string) Option    // which device created this event
func WithClientOccurredAt(t time.Time) Option // when it happened on the device
func WithClientTimezone(tz string) Option     // device timezone for business grouping
func WithCausationID(id string) Option        // what caused this event (for undo chains)
```

**Size:** ~20 lines per option. Trivial addition to existing `options.go`.

#### Step B2: Add idempotency key to Command interface

```go
// core/command/types.go
type Command interface {
    Type() Type
    AggregateID() id.AggregateID
    IdempotencyKey() string  // NEW: empty string = no dedup
}
```

This is a **breaking change** to the Command interface. All existing implementations need to add the method.

**Migration:** Provide a `BaseCommand` struct that implements `IdempotencyKey()` returning empty string:

```go
type BaseCommand struct {
    type_       Type
    aggregateID id.AggregateID
}

func (c BaseCommand) Type() Type                { return c.type_ }
func (c BaseCommand) AggregateID() id.AggregateID { return c.aggregateID }
func (c BaseCommand) IdempotencyKey() string     { return "" }
```

Existing commands can embed `BaseCommand` or implement the method directly.

#### Step B3: Add sync-related timestamp fields to event metadata

These don't need new event fields — they go into the existing `Metadata` map:

| Key                  | Set By        | Purpose                       |
| -------------------- | ------------- | ----------------------------- |
| `client.id`          | Client        | Device attribution            |
| `client.occurred_at` | Client        | Business truth timestamp      |
| `client.timezone`    | Client        | Timezone for day grouping     |
| `sync.pushed_at`     | Client        | When push was attempted       |
| `sync.acked_at`      | Server        | When server confirmed receipt |
| `sync.rebased_at`    | Server/Client | When events were reordered    |

No code change needed in go-cqrs-lite — these are convention-based metadata keys. Document them.

#### Step B4: Document the offline-first metadata convention

Create `docs/OFFLINE_FIRST_METADATA.md` with:

- Metadata key names and semantics
- Who sets each key (client vs server)
- When each key is set (creation vs sync vs ack)
- Example event with full metadata

**Size:** ~100 lines documentation.

### What Does NOT Go Into go-cqrs-lite

From the brainstorm's "Everything Else" document, these are explicitly consumer concerns:

| Concern                          | Why Not in Library                        |
| -------------------------------- | ----------------------------------------- |
| Sync protocol (push/pull/rebase) | Too opinionated                           |
| Client-side event store          | Platform-specific (SQLite, IndexedDB)     |
| Vector clock / CRDT              | Already in go-localfirst `pkg/sync/`      |
| Event signing                    | Security concern, consumer responsibility |
| Network monitor                  | Platform-specific                         |
| Auth token management            | Out of scope                              |
| Client-side projection rebuild   | Consumer's read model lifecycle           |
| Event compaction                 | Consumer's storage policy                 |

---

## Initiative C: Error Handling & SSE in go-localfirst

### Goal

Implement the full error architecture from the brainstorm, using go-cqrs-lite's error taxonomy (Initiative A) as the foundation.

### Actionable Steps (Ordered by Priority)

#### Step C1: Fix Critical Bugs (3 data races + 3 resource leaks)

From the brainstorm audit, these are correctness bugs, not design gaps:

| Bug                                                      | File                             | Fix                       |
| -------------------------------------------------------- | -------------------------------- | ------------------------- |
| SSE `BroadcastEvent` iterates `clients` map without lock | `internal/handler/sse.go`        | Add `sync.RWMutex`        |
| SSE `HandleEvents` writes to `clients` map without lock  | `internal/handler/sse.go`        | Use same mutex            |
| VectorClock map accessed from goroutines without lock    | `pkg/sync/vector_clock.go`       | Add `sync.RWMutex`        |
| RateLimiter cleanup goroutine never stops                | `pkg/middleware/rate_limiter.go` | Add `Stop()` with context |
| Sync `performPeriodicSync` spawns unbounded goroutines   | `internal/sync/manager.go`       | Use bounded pool          |
| Sync `broadcastOperation` spawns unbounded goroutines    | `internal/sync/manager.go`       | Use bounded pool          |

**Estimate:** 6 focused bug fixes. Each is 10-30 lines.

#### Step C2: Implement `pkg/errors` in go-localfirst

The full 9-family error taxonomy lives here. go-cqrs-lite provides 5 families; go-localfirst adds 4:

```go
// pkg/errors/errors.go
type Family int

const (
    // From go-cqrs-lite taxonomy
    Rejection      Family = iota
    Conflict
    Transient
    Corruption
    Infrastructure

    // go-localfirst additions
    Staleness     // Read model behind event store
    Divergence    // CRDT merge failure, multiple truths
    Pipeline      // Background processing failed (projector, SSE bridge, outbox)
    Transport     // Client connection lost
)
```

This package also provides the **notification bus** and **DLQ** interfaces:

```go
// pkg/errors/notify.go
type NotificationBus interface {
    Publish(ctx context.Context, event Notification) error
    Subscribe(handler func(ctx context.Context, event Notification)) error
}

type Notification struct {
    Type    string         // "command.failed", "system.degraded", "data.diverged"
    Payload map[string]any
    Family  Family         // Which error family triggered this
}

// pkg/errors/dlq.go
type DeadLetterQueue interface {
    Push(ctx context.Context, entry DLQEntry) error
    List(ctx context.Context, filter DLQFilter) ([]DLQEntry, error)
    Replay(ctx context.Context, id string) error  // re-process a dead letter
}
```

#### Step C3: Implement the Dual Bus Pattern

From the brainstorm (Dimension 1, Option B):

```
event.Bus     → domain events → event store → projectors → SSE
notify.Bus    → notifications → SSE + logging + metrics (ephemeral, no persistence)
```

The `NotificationBus` lives in `pkg/errors/notify.go`. It's in-memory, ephemeral, and separate from `event.Bus`.

**Wiring:**

```go
// cmd/api/main.go
notifyBus := errors.NewNotificationBus(logger)
eventBus := memory.NewMemoryBus()  // go-cqrs-lite

// SSE handler subscribes to BOTH buses
sseHandler := handler.NewSSE(eventBus, notifyBus, logger)

// Error middleware publishes to notify bus
cmdDispatcher.Use(errors.ErrorNotificationMiddleware(notifyBus))
```

#### Step C4: Implement Error Notification Middleware

```go
// pkg/middleware/error_notify.go
func ErrorNotificationMiddleware(notifyBus *errors.NotificationBus) command.Middleware {
    return func(next command.Handler) command.Handler {
        return command.HandlerFunc(func(ctx context.Context, cmd command.Command) error {
            err := next(ctx, cmd)
            if err == nil { return nil }

            family := errors.Classify(err)
            switch family {
            case errors.Conflict:
                notifyBus.Publish(ctx, errors.Notification{
                    Type:   "todo.conflict",
                    Family: family,
                    Payload: map[string]any{"aggregateId": cmd.AggregateID()},
                })
            case errors.Infrastructure, errors.Corruption:
                notifyBus.Publish(ctx, errors.Notification{
                    Type:   "system.error",
                    Family: family,
                })
            case errors.Transient:
                notifyBus.Publish(ctx, errors.Notification{
                    Type:   "system.degraded",
                    Family: family,
                })
            // Rejection: no broadcast (private to requester)
            }
            return err
        })
    }
}
```

#### Step C5: Implement SSE Connection Manager

Fix all SSE issues from the brainstorm:

```
pkg/middleware/sse_manager.go (or internal/handler/sse.go rewrite)
├── Mutex-protected client registry
├── Per-client send timeout (5s → close)
├── Event ring buffer for Last-Event-ID replay
├── Jittered reconnect advisory (retry: 5000)
├── Proxy-friendly headers (X-Accel-Buffering: no)
└── Connection rate limiter
```

#### Step C6: Implement HTMX Error Response Handler

```go
// internal/handler/error_handler.go
type ErrorHandler struct {
    notifyBus *errors.NotificationBus
}

func (h *ErrorHandler) Respond(c *gin.Context, err error) {
    family := errors.Classify(err)

    switch family {
    case errors.Rejection:
        c.Header("HX-Retarget", "#error-message")
        c.Header("HX-Reswap", "innerHTML")
        c.String(422, `<div class="error">%s</div>`, err.Error())

    case errors.Conflict:
        c.Header("HX-Trigger", fmt.Sprintf(`{"conflictDetected":{"message":"%s"}}`, err.Error()))
        c.String(409, "")

    case errors.Transient:
        c.Header("HX-Trigger", `{"systemDegraded":{"message":"Connection issue, retrying..."}}`)
        c.String(503, "")

    case errors.Infrastructure:
        c.Header("HX-Trigger", `{"systemError":{"message":"Something went wrong. Your data is safe."}}`)
        c.String(500, "")
    }
}
```

#### Step C7: Eliminate the Dual-Write Anti-Pattern

From the brainstorm (Dimension 10.2) — the CQRS/CRDT dual-write race:

**Strategy: Single write path.** All state mutations go through CQRS event sourcing. The CRDT sync manager becomes a transport layer:

```
Sync Peer → OpCreate → SyncCreateCommand → CommandDispatcher → Event Store → Read Model
```

The sync manager dispatches commands instead of writing to `repo` directly. This eliminates:

- Dual-write race (10.2)
- Ghost aggregates (10.5)
- CQRS/CRDT identity crisis (10.14)

#### Step C8: Implement Graceful Lifecycle

From brainstorm Pattern 6:

```go
// pkg/lifecycle/lifecycle.go
type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

type Manager struct {
    components []Lifecycle
    wg         sync.WaitGroup
}
```

Each long-running component (SSE handler, sync manager, projector) implements `Lifecycle`. The manager orchestrates startup ordering and graceful shutdown with drain timeout.

---

## Execution Order

```
Phase 1: Foundation (go-cqrs-lite)          [~2-3 sessions]
├── A1: Create core/pkg/errors/ package
├── A2: Map existing sentinels to families
├── A3: Enhance retry middleware
├── A4: Add projection error handling
├── B1: Add client metadata options
├── B2: Add IdempotencyKey to Command
├── B3: Document metadata conventions
└── Test everything, zero lint

Phase 2: Critical Bug Fixes (go-localfirst)  [~1-2 sessions]
├── C1: Fix 3 data races + 3 resource leaks
└── Verify with race detector

Phase 3: Error Architecture (go-localfirst)  [~2-3 sessions]
├── C2: Implement pkg/errors with 9 families
├── C3: Implement notification bus
├── C4: Implement error notification middleware
├── C5: Implement SSE connection manager
├── C6: Implement HTMX error handler
└── Integration tests

Phase 4: Architecture Fixes (go-localfirst)  [~2-3 sessions]
├── C7: Eliminate dual-write (single write path)
├── C8: Implement graceful lifecycle
└── End-to-end tests

Phase 5: Future Innovation (go-cqrs-lite)    [~3-5 sessions]
├── Hybrid architecture (aggregate + context queries)
├── Decider pattern (pure decide/apply functions)
├── Time-travel queries (Store.AsOf)
└── sqlc-based multi-engine storage
```

---

## What We're NOT Doing (Explicit Non-Goals)

| Non-Goal                             | Why                                                      |
| ------------------------------------ | -------------------------------------------------------- |
| Sync protocol implementation         | Too opinionated for a library; consumer's responsibility |
| Client-side event store              | Platform-specific (SQLite, IndexedDB, OPFS)              |
| Event signing / encryption           | Security concern, consumer's responsibility              |
| WASM / mobile client SDK             | Needs separate TypeScript/Dart SDK                       |
| Full CRDT implementation             | Already in go-localfirst `pkg/sync/`                     |
| Multi-engine storage (MySQL, SQLite) | sqlc is ready but PostgreSQL-only for now                |
| Watermill module                     | Planned Phase 6 — not started                            |
| Tag v1.0.0 releases                  | After all modules stabilize                              |
| GraphQL, gRPC transport              | Framework-level, not library                             |
| LLM/AI integration                   | Research only, no implementation planned                 |

---

## Risk Assessment

| Risk                                          | Probability | Impact | Mitigation                                                              |
| --------------------------------------------- | ----------- | ------ | ----------------------------------------------------------------------- |
| `IdempotencyKey()` breaks Command interface   | High        | Medium | Provide `BaseCommand` embed, update all implementations in one pass     |
| Error taxonomy too rigid for all consumers    | Medium      | Low    | Families are additive; `Classify()` has a default (Transient)           |
| Dual-bus pattern adds complexity              | Medium      | Low    | Notification bus is optional; system works without it                   |
| Single-write-path migration breaks sync       | Medium      | High   | Incremental: first add command dispatch, then remove direct repo writes |
| Metadata convention not followed by consumers | Low         | Low    | Document clearly; it's convention-based, not enforced                   |

---

## Success Metrics

After Phase 1-4 completion:

- [ ] `core/pkg/errors/` package with 100% test coverage
- [ ] All existing sentinel errors classified into families
- [ ] `IdempotencyKey()` on Command interface
- [ ] Client metadata options (`WithClientID`, `WithClientOccurredAt`, etc.)
- [ ] Zero data races (`go test -race ./...`)
- [ ] Zero goroutine leaks (all long-running components implement Lifecycle)
- [ ] Notification bus operational (SSE broadcasts errors)
- [ ] HTMX error responses classified by family
- [ ] Single write path (sync operations go through command dispatcher)
- [ ] SSE supports `Last-Event-ID` reconnect replay
- [ ] All tests pass across both projects

---

_This plan synthesizes 2000+ lines of brainstorm, 7 research documents, and the current state of both codebases into a concrete execution sequence._
