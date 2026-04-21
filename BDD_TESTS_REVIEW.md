# BDD Tests — go-cqrs-lite

**Last updated:** 2026-04-22
**Framework:** Ginkgo v2 + Gomega
**Status:** Active

---

## Test Files

| File | Specs | Focus |
|---|---|---|
| `event/event_sourcing_bdd_test.go` | 22 | Event Store, Event Bus, Event Creation |
| `aggregate/cqrs_bdd_test.go` | 11 | Full CQRS roundtrip, Repository lifecycle, concurrency, invariants |
| `query/query_bdd_test.go` | 6 | Query dispatch, typed results, middleware |

**Total: 39 specs**

### Supporting files

| File | Purpose |
|---|---|
| `event/event_bdd_suite_test.go` | `TestEventBDD` — Ginkgo suite entry point |
| `aggregate/cqrs_bdd_suite_test.go` | `TestCQRSBDD` — Ginkgo suite entry point |
| `query/query_bdd_test.go` | `TestQueryBDD` — Ginkgo suite entry point |

---

## Running

```bash
# BDD suites only
go test ./event/ -v -run TestEventBDD
go test ./aggregate/ -v -run TestCQRSBDD
go test ./query/ -v -run TestQueryBDD

# All tests (BDD + unit)
go test ./...
```

---

## Scenarios

### Event Store (`event/event_sourcing_bdd_test.go`)

**Persona:** "As a developer building an event-sourced system"

- Save events for a new aggregate, verify persistence and version tracking
- Append events to an existing aggregate, verify ordering and version increments
- Detect version conflicts when expected version mismatches
- Load events for a non-existent aggregate → `ErrAggregateNotFound`
- Load events from a specific version onward (partial replay)
- Load from a version beyond current state → empty slice
- Delete an aggregate's entire event history
- `AppendBatch` for bulk imports without version checks
- Reject operations on a closed store

### Event Bus (`event/event_sourcing_bdd_test.go`)

**Persona:** "As a developer reacting to domain events"

- Subscribe to a specific event type, receive only matching events
- `SubscribeAll` receives every published event
- Middleware wraps handlers in correct FIFO order
- Handler failure stops processing and returns wrapped error
- Publishing to a closed bus → `ErrBusClosed`
- Subscribing to a closed bus → `ErrBusClosed`
- **Subscribe + SubscribeAll on same event** → event delivered twice (once per subscription)
- **Subscribe to wrong type** → no event received

### Event Creation (`event/event_sourcing_bdd_test.go`)

**Persona:** "As a developer creating domain events"

- Create event with full metadata (correlation, causation, user, request, source)
- Preserve every field including tracing IDs and timestamps
- Reject empty aggregate ID, empty aggregate type, negative version
- Custom metadata key-value pairs survive through the metadata map

### CQRS Roundtrip (`aggregate/cqrs_bdd_test.go`)

**Persona:** "As a developer building a CQRS application"

Uses a real `expense` aggregate with Submit → Approve → Pay workflow:

- Submit new expense: persist event + publish to bus
- Approve and pay: multi-step workflow with complete audit trail
- Replay state from event store alone (no cached state)
- Dispatch unregistered command → handler not found error
- Close dispatcher → reject registration and dispatch

### Repository Lifecycle (`aggregate/cqrs_bdd_test.go`)

**Persona:** "As a developer managing aggregate lifecycle"

- Save aggregate with no changes → no-op
- Load non-existent aggregate → error
- Multiple save/load cycles accumulate events correctly, clear uncommitted changes
- **Re-creation after delete** — delete events, submit new → version 1
- **Pay without approve** — aggregate allows pay (no approval invariant enforced)

### CQRS Concurrency (`aggregate/cqrs_bdd_test.go`)

**Persona:** "As a developer verifying domain correctness"

- **Concurrent saves on same aggregate** — 20 goroutines race, version conflicts detected, final state consistent

### Query Dispatcher (`query/query_bdd_test.go`)

**Persona:** "As a developer building read-side queries"

- Register handler and dispatch matching query → typed result
- `DispatchTyped[T]` with correct type → strongly typed result
- `DispatchTyped[T]` with wrong type → descriptive type mismatch error
- Dispatch unregistered query → query not supported error
- Close dispatcher → reject further dispatch and registration
- Query middleware wraps handler in order

---

## Test Aggregate

The `cqrs_bdd_test.go` file defines a complete `expense` aggregate (Submit/Approve/Pay) with:

- `HistoryLoader` implementation for proper version replay
- JSON payloads for `ExpenseSubmitted`
- Three command types: `submitExpenseCmd`, `approveExpenseCmd`, `payExpenseCmd`
- Full command handlers wired through `EventSourcedRepository`

This serves as both a test fixture and a reference implementation for library users.

---

## Known Issues Discovered

- **No approval invariant**: The `expense` aggregate allows `Pay` without `Approve`. This is by design (no domain rules enforced at aggregate level) but worth documenting for library consumers.
