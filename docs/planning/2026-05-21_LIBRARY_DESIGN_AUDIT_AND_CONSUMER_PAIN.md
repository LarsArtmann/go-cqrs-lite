# Library Design Audit & Consumer Pain Analysis

**Date:** 2026-05-21
**Session:** 89
**Context:** Brutal self-review triggered by "what is badly designed" → evolved into ideal API vision → real consumer analysis

---

## Table of Contents

1. [Critical Design Problems](#1-critical-design-problems)
2. [Significant Design Smells](#2-significant-design-smells)
3. [Moderate Issues](#3-moderate-issues)
4. [The Ideal Event Sourcing API](#4-the-ideal-event-sourcing-api)
5. [Real Consumer Analysis: SEC](#5-real-consumer-analysis-sec)
6. [Real Consumer Analysis: go-localsync](#6-real-consumer-analysis-go-localsync)
7. [Unified Problem Map](#7-unified-problem-map)
8. [Improvement Plan](#8-improvement-plan)
9. [Command & Query Audit for Analytics](#9-command--query-audit-for-analytics)

---

## 1. Critical Design Problems

### 1.1 `sync/` is a Ghost System

Fully implemented module (6 files + tests, 92% coverage) with **zero consumers**. Not imported by any other module, not referenced in examples, not used anywhere. Dead weight shipped as a library module.

- `sync/conflict.go`, `operation.go`, `types.go`, `vectorclock.go`, `errors.go`, `doc.go`
- **Action:** Delete entirely or extract to a separate repo.

### 1.2 `aggregate.Root` Interface is an Anti-Pattern (9 methods)

`core/aggregate/aggregate.go` — Forces every aggregate to implement infrastructure bookkeeping (`SetVersion`, `LoadEvents`, `UncommittedChanges`, `MarkChangesAsCommitted`, `ApplySnapshot`) alongside domain logic (`Apply`). The `decider` package was created to solve this exact problem, yet both coexist with duplicated options, errors, and load helpers.

| Method on `Root` | Concern | Belongs in |
|---|---|---|
| `Apply(event.Event)` | Domain | ✅ Stays |
| `ID()`, `Type()` | Identity | ✅ Stays |
| `Version()`, `SetVersion()` | Infrastructure | Repository |
| `LoadEvents([]Event)` | Infrastructure | Repository |
| `UncommittedChanges()` | Infrastructure | Repository |
| `MarkChangesAsCommitted()` | Infrastructure | Repository |
| `ApplySnapshot(Snapshot)` | Infrastructure | Repository |

**Action:** Deprecate `aggregate` package. Invest in `decider` as the single aggregate pattern.

### 1.3 `event.Store` is a Fat Interface (8 methods)

Mixes OLTP (`Save`, `Load`), batch (`AppendBatch`), time-travel (`LoadToVersion`, `LoadToTimestamp`), and lifecycle (`Delete`, `Close`). Not every store needs `AppendBatch` or time-travel.

Should be decomposed like `GlobalLoader`/`PositionalLoader` already are — small composable interfaces:

```go
type Store interface {                    // core (4 methods)
    Closer
    Save(ctx, aggType, aggID, expectedVersion, events) error
    Load(ctx, aggType, aggID) ([]Event, error)
    Delete(ctx, aggType, aggID) error
}

type BatchStore interface {               // optional
    AppendBatch(ctx, events) error
}

type TimeTravelStore interface {          // optional
    LoadToVersion(ctx, aggType, aggID, version) ([]Event, error)
    LoadToTimestamp(ctx, aggType, aggID, time) ([]Event, error)
}
```

### 1.4 Middleware Triplication

Every middleware (`Logging`, `Retry`, `Recovery`, `Validation`, `Tracing`, `Metrics`) is written **three times** for `Command`/`Event`/`Query` with identical logic, differing only in type signatures. ~200 lines of pure duplication per middleware.

The generic `dispatcher.Dispatcher[H, M]` infrastructure exists specifically to prevent this — but the middleware layer doesn't use it.

**Action:** Implement middleware once using generics over `dispatcher.Dispatcher[H, M]`.

### 1.5 `query.Handler` Returns `any`

`core/query/dispatcher.go:14` — `type Handler = func(context.Context, Query) (any, error)`. Violates the project's own "no `any`" rule at the primary query API boundary. Forces every caller into runtime type assertions via `DispatchTyped[T]`.

### 1.6 `Command.IdempotencyKey()` is Dead

Every command must implement `IdempotencyKey()`, but `Core.IdempotencyKey()` hardcodes `""`. Leaky abstraction that adds noise to every command definition without providing value.

**Action:** Remove from `Command` interface. Make it a separate optional interface or remove entirely.

### 1.7 `AggregateID` Backed by `string`, All Other IDs Backed by `ulid.ULID`

`core/pkg/id/aggregate_id.go` — Creates a split-brain ID system. `AggregateID` can be `"lock_user1_user2"` (arbitrary string) while every other ID type is a ULID. The `ULID()` helper won't work on the most important ID type in the system.

### 1.8 Storage: 12 Constructor Aliases Instead of Options

`storage/` has `NewSQLEventStore`/`NewSQLiteEventStore`/`NewTursoEventStore` — all one-line wrappers returning the same `*SQLEventStore` type. Repeated for outbox, snapshot, and checkpoint stores (4 types × 3 dialects = 12 constructors).

Should be `NewSQLEventStore(db, WithDialect(Postgres))` — one constructor, functional options.

---

## 2. Significant Design Smells

### 2.1 Deprecated `CatalogMeta` Still Shipped

Identical structs in `core/event`, `core/command`, `core/query` — all marked deprecated but still exported. Dead code in the public API surface of a library.

### 2.2 `RepositoryOption` + `With*` Duplicated Between `aggregate` and `decider`

Same four options (`WithSnapshotStore`, `WithOutbox`, `WithCodec`, `WithSnapshotStrategy`) implemented twice with identical semantics. Same for `ErrNilStore`/`ErrNilBus` sentinel errors.

### 2.3 `VectorClock` is Not Thread-Safe

`sync/vectorclock.go` — `map[NodeID]int64` with `Increment`/`Merge`/`Get` and no synchronization. Designed for distributed sync but will race if shared across goroutines.

### 2.4 Projection Live Handler Silently Swallows Errors

`projection/runner_live.go:12-16` — `dispatchToProjections` errors are logged but never returned to the bus. Replay is strict (propagates errors), live is lenient (silently continues). Inconsistent guarantees.

### 2.5 `storage` Swallows Metadata Deserialization Errors

- `storage/outbox_helpers.go:73` — `marshalMetadata` error discarded with `_`
- `storage/pebble_serialization.go:58` — same pattern

Events reconstructed without metadata silently.

### 2.6 `Metadata` Mutation Escape Hatch

`event.Metadata` has exported fields + mutable `Custom map[string]string`, but `Event.Metadata()` returns a pointer. Defensive copy of the struct doesn't prevent mutating the `Custom` map through the returned pointer.

### 2.7 Inconsistent `Parse*` Contracts

`ParseVersion`/`ParseSource`/`ParseIPAddress` return `(T, error)`. `ParseUserAgent` returns `(UserAgent, error)` but **never errors**. Same naming pattern, different contracts.

### 2.8 `RetryConfig.IsRetryable` Can Be Nil

Constructing `RetryConfig{MaxAttempts: 3}` directly gives a nil `IsRetryable` that panics at runtime. `Validate()` doesn't check for nil function fields.

### 2.9 Example/todo Reinvents `RegisterTyped`

`example/todo/commands/mixin.go` — Creates `CommandHandler` base struct + `requireCommandType[T]` helper, duplicating what `command.RegisterTyped` already provides. Confusing for library consumers — which pattern should they follow?

---

## 3. Moderate Issues

| Issue | Detail |
|---|---|
| `aggregate` uses `fmt.Errorf("%w", ErrNilStore)` unnecessarily | Wrapping sentinel in identical message — just return the sentinel |
| `reconstructEvent` has 9 positional parameters | Should use a struct or options |
| `Catalog` package IDs (`ServiceID`, etc.) are bare strings | No `Parse` functions, no format enforcement |
| `OutboxStatus` is a bare string | Could be an enum with typed constructors |
| Constructor return patterns inconsistent | Value / pointer / pointer+error with no clear rule |
| `MustParse*` only in `sync`, not in `id` package | Inconsistent convention |
| `sync` module uses `errorfamily` directly | Bypasses `event` error taxonomy |
| Storage `Save` doesn't validate event consistency | No check that versions are sequential, IDs match |
| `Register` on `projection.Runner` has no sync | Data race if called concurrently |

---

## 4. The Ideal Event Sourcing API

### The Fundamental Insight

Event sourcing has **four pure functions** and zero infrastructure concepts that belong in domain code:

```
Decide(command, state) → []Event
Fold(state, event)    → state
Project(event)        → side-effect
React(event)          → command (saga/process manager)
```

Everything else — store, bus, serialization, versioning, snapshots, outbox — is infrastructure that should disappear behind the API.

### What Consumers Write Today vs. What They Should Write

**Today: ~80 lines per command** (see `example/todo/`)

Command struct with 3 interface methods → handler struct → type assertion → decide closure → manual `json.Marshal` → `event.NewEvent(5 args)` → `repo.Execute(ctx, id, "Todo", DecideCreate(title))` → manual `bus.Subscribe` per event type

**Ideal: ~10 lines per aggregate**

```go
type Todo struct {
    Title     string
    Completed bool
}

type TodoCreated   struct { Title string }
type TodoCompleted struct {}
type TodoRenamed   struct { NewTitle string }

func Decide(_ context.Context, cmd any, s Todo, v event.Version) ([]any, error) {
    switch c := cmd.(type) {
    case CreateTodo:
        if s.Title != "" { return nil, ErrAlreadyExists }
        return []any{TodoCreated{Title: c.Title}}, nil
    case CompleteTodo:
        if !s.Completed { return nil, ErrNotCreated }
        return []any{TodoCompleted{}}, nil
    case RenameTodo:
        return []any{TodoRenamed{NewTitle: c.NewTitle}}, nil
    }
    return nil, nil
}

func Fold(s Todo, e any) (Todo, error) {
    switch ev := e.(type) {
    case TodoCreated:   return Todo{Title: ev.Title}, nil
    case TodoCompleted: return Todo{Title: s.Title, Completed: true}, nil
    case TodoRenamed:   return Todo{Title: ev.NewTitle, Completed: s.Completed}, nil
    }
    return s, nil
}
```

Wiring:

```go
todo := cqrs.Aggregate("Todo", Todo{},
    cqrs.Decide(Decide),
    cqrs.Fold(Fold),
)

app := cqrs.New(eventStore, eventBus, todo)
app.HandleCommand(ctx, CreateTodo{Title: "Buy milk"})
```

### Design Principles Behind the Ideal API

**1. Payload types ARE the API**

Today you have three separate concepts for what should be one: event type string, payload struct, and `event.Core` entity. The payload struct should BE the event. The library auto-derives event type name, ID, version, timestamp from infrastructure context, and handles serialization via default JSON codec.

```go
// Today: 5 positional args + manual marshal
evt, _ := event.NewEvent("TodoCreated", aggID, "Todo", version.Increment(), mustMarshal(payload))

// Ideal: payload IS the event
return []any{TodoCreated{Title: "Buy milk"}}, nil
```

**2. Commands are plain DTOs, not interface implementations**

```go
// Today
type CreateTodoCmd struct{ command.Core }  // must embed Core, implement 3 interface methods

// Ideal — just a struct
type CreateTodo struct {
    ID    id.AggregateID
    Title string
}
```

Library derives `Type()` from struct name and `AggregateID()` from field by convention (or struct tag). No interface, no embedding, no `MustNew` constructor.

**3. Decide is a flat function, not a closure factory**

```go
// Today: closure returning closure
func DecideCreate(title string) decider.DecideFunc[TodoState] {
    return func(state TodoState, version event.Version) ([]event.Event, error) { ... }
}
repo.Execute(ctx, id, "Todo", DecideCreate(cmd.Title))

// Ideal: flat function receives command directly
func Decide(ctx context.Context, cmd any, state Todo, ver event.Version) ([]any, error) {
    switch c := cmd.(type) { ... }
}
app.HandleCommand(ctx, cmd)  // auto-routes
```

**4. Aggregate type is bound once, not threaded everywhere**

Today `"Todo"` is passed to `NewEvent`, `Execute`, `store.Save`, etc. The aggregate type is an implementation detail of registration, not something consumers pass on every call.

**5. Versioning is invisible**

Consumers should never call `version.Increment()`. The repository knows the current version, knows how many events the decide function returned, and increments accordingly.

**6. Projections subscribe automatically**

```go
// Today: 5 manual subscribe calls
eventBus.Subscribe("TodoCreated", projection.Handle)
eventBus.Subscribe("TodoCompleted", projection.Handle)

// Ideal: projection declares what it handles
proj := projection.Build("todo_projection",
    projection.On[TodoCreated](handleCreated),
    projection.On[TodoCompleted](handleCompleted),
)
```

**7. Serialization is a default, not a parameter**

`event.JSONCodec{}` is passed to repository, projection, decode helpers — everywhere. JSON should be the default. Consumers only override when needed.

### What This Means for the Current Architecture

The `decider` package is 80% of the way there. The missing 20%:

| Today | Ideal |
|---|---|
| `DecideFunc` = closure factory | `Decide(ctx, cmd, state, version) → []any` |
| `repo.Execute(ctx, id, type, decideFn)` | `app.HandleCommand(ctx, cmd)` |
| Events are `event.Core` structs | Events are plain payload structs |
| `event.NewEvent(5 positional args)` | Auto-derived from payload type |
| `command.Core` embedding | Plain struct with `ID` field |
| Manual `bus.Subscribe` per type | Auto-subscribe from projection registration |
| `JSONCodec{}` threaded everywhere | Default JSON, override when needed |
| `aggregateType` string everywhere | Bound once at registration |
| `version.Increment()` in domain code | Invisible, handled by infrastructure |

**The shift:** stop making the domain adapt to the infrastructure. Today the domain code calls infrastructure APIs (`NewEvent`, `Increment`, `codec.Encode`). In a perfect world, the domain returns plain values and the infrastructure wraps them.

---

## 5. Real Consumer Analysis: SEC

**Project:** `/home/lars/projects/SEC/`
**Imports:** `core`, `catalog`, `memory`, `storage`
**Uses:** Command dispatcher, event store (memory + Turso/SQLite), query dispatcher, catalog + docserver
**Does NOT use:** `decider.Repository`, `projection`, middleware, outbox

### What SEC Does Badly (Their Fault)

| Issue | Detail |
|---|---|
| **Built parallel event system** | `decider.DomainEvent` with typed payloads instead of `event.Event`. `Fold` takes `DomainEvent`, so `decider.Repository[State]` can't be used. Built ~70-line `persistAndPublish()` translation layer. Cascades into ~250 lines of unnecessary code. |
| **Bus with zero subscribers** | Events published to `MemoryBus` but nothing subscribes. Dead infrastructure. |
| **Hand-rolled `foldEvents()`** | 40 lines in query handler that reimplements `Repository.Load()`. Every query loads and folds entire event history from scratch. |
| **Didn't use `command.RegisterTyped[T]`** | Wrote own `assertCmd[T]()` helper instead. |

### What's Our Fault (Library Gaps)

| Issue | Detail | Lines Wasted |
|---|---|---|
| **`event.NewEvent` requires raw `[]byte`** | SEC built `encodeJSONPayload()` + `decode.go` to work around. `NewEvents` takes `[]any` but `NewEvent` takes `[]byte` — inconsistent. | ~40 |
| **No aggregate-bound repository** | `"game"` passed on every `Execute` call. Aggregate type should be bound at registration. | noise everywhere |
| **`id.AggregateID` vs consumer's branded IDs** | SEC uses `go-branded-id` for `GameID`. Every boundary requires `gameID.String()` → `id.ParseAggregateID()`. | ~30 |
| **`storage.SQLEventStore` doesn't own `*sql.DB`** | SEC wrote 150-line `tursoStore` wrapper — pure delegation with error wrapping, just to manage DB lifecycle. | ~150 |
| **`Execute` returns only `error`** | SEC can't tell what happened (created? updated? no-op?) without reverse-engineering from state. | ~20 |

**Total library-attributable boilerplate in SEC: ~240 lines**

### Key Code Smells

**The `DomainEvent` ↔ `event.Event` translation** (`command_handler.go:193-256`):
```go
// SEC's decider returns DomainEvent (typed payloads)
// Then translates to event.Event ([]byte payloads) in persistAndPublish():
// 1. json.Marshal(payload) via encodeJSONPayload()
// 2. Manual version math: event.Version(int(expectedVersion)+i+1)
// 3. event.NewEvent(event.Type(de.EventType), aggID, aggType, version, data)
// 4. store.Save() then bus.Publish() for each event
```

This entire function (~63 lines) would disappear if the library accepted typed payloads or if SEC used `decider.Repository`.

**The branded ID conversion** (every command factory and query handler):
```go
// Converting between GameID and AggregateID — happens at every boundary
func parseAndCreateCore(gameIDStr string, cmdType command.Type) (*command.Core, error) {
    gameID, err := gameid.Parse(gameIDStr)
    aggID, err := id.ParseAggregateID(gameID.String())
    return command.New(cmdType, aggID)
}
```

---

## 6. Real Consumer Analysis: go-localsync

**Project:** `/home/lars/projects/go-localsync/`
**Imports:** `core`, `memory`, `middleware`, `projection`, `storage`
**Uses:** Decider (correctly!), event store (memory + Turso/SQLite), projections, snapshots, outbox, middleware (EventLogging only)
**Does NOT use:** `command.Dispatcher` (can't — blocked by API), `query.Dispatcher`

### What go-localsync Does Badly (Their Fault)

| Issue | Detail |
|---|---|
| **Outbox poller has a data loss bug** | Acks outbox entries even when `bus.Publish` fails: `_ = bus.Publish(ctx, evt); _ = outbox.Ack(ctx, ...)` — silent event loss. Library provides `event.OutboxPublisher` which handles this correctly. |
| **`countingDecide` hack** | Wraps decide function to count emitted events + check `state.IsNew()` to reverse-engineer what happened (created/updated/conflict/unchanged). Fragile. |
| **Outbox poller at 1ms interval** | Essentially a busy loop. Library's `OutboxPublisher` defaults to 1 second. |

### What's Our Fault (Library Gaps)

| Issue | Detail | Lines Wasted |
|---|---|---|
| **`Command.AggregateID()` required upfront** | go-localsync computes aggregate IDs *inside* decide (SHA256 hash of source+sourceID). Makes entire `command.Dispatcher` + middleware pipeline unusable. | entire command system unused |
| **`Execute` returns no domain result** | The `countingDecide` + `classifyAction` hack exists because `Repository.Execute` returns only `error`. | ~40 |
| **`NewTypedProjection[T]` assumes single payload type** | Real projections handle 3+ event types with different payloads. go-localsync forced to write manual `Handle()` with switch. | ~30 |
| **No deterministic ID generation** | SHA256 + `sync.Map` cache hand-rolled because library only offers random ULID generation. | ~25 |
| **No read model infrastructure** | `MemoryReadModel` (148 lines) + `TursoReadModel` (315 lines) fully hand-rolled. | ~460 |
| **`NewEvent` vs `NewEvents` inconsistency** | Tests use `NewEvent` (raw bytes), production uses `NewEvents` (typed payloads). Different serialization contracts. | ~15 |

**Total library-attributable boilerplate in go-localsync: ~570 lines**

### Key Code Smells

**The `countingDecide` hack** (`stack.go`):
```go
countingDecide := func(state SyncItemState, ver event.Version) ([]event.Event, error) {
    wasNew = state.IsNew()
    events, err := DecideSync(item, syncOpts...)(state, ver)
    eventCount = len(events)
    return events, err
}
action := classifyAction(err, eventCount, wasNew)
// classifyAction reverse-engineers: created (wasNew + events), updated (!wasNew + events),
// conflict (ConflictFound event), unchanged (0 events)
```

This entire pattern exists because `Execute` returns only `error`. A result type would eliminate it.

**Deterministic aggregate ID with SHA256 cache** (`aggregate_id.go`):
```go
func AggregateID(source, sourceID string) id.AggregateID {
    key := itemKey(source, sourceID)
    if cached, ok := aggIDCache.Load(key); ok { return cached.(id.AggregateID) }
    h := sha256.Sum256([]byte(key))
    aggID := id.MustParseAggregateID(hex.EncodeToString(h[:16]))
    aggIDCache.Store(key, aggID)
    return aggID
}
```

A library-provided `id.DeriveAggregateID(namespace, keys...)` would replace this.

### What go-localsync Gets Right

- **`decider.Decider[State]` + `Fold` + `Repository`** — used correctly and cleanly
- **`event.NewEvents`** with typed payloads — ergonomic batch constructor
- **`event.DecodePayload[T]`** — nice generic decode helper
- **Generic snapshot/checkpoint/outbox stores** — straightforward to wire
- **`event.EveryNEvents(10)`** for snapshot strategy — works well

---

## 7. Unified Problem Map

| Problem | Who's at fault | Impact | Should we fix? |
|---|---|---|---|
| `event.NewEvent` takes `[]byte`, `NewEvents` takes `[]any` | **Library** | Both consumers build marshal helpers | **Yes** |
| `Execute` returns only `error` | **Library** | Both consumers hack around it | **Yes** |
| `Command.AggregateID()` required upfront | **Library** | go-localsync can't use command system | **Yes** |
| `id.AggregateID` incompatible with consumer branded IDs | **Library** | SEC converts at every boundary | **Yes** |
| `storage.SQLEventStore` doesn't own DB | **Library** | 150-line wrapper in SEC | **Yes** |
| No read model base | **Library** | 460 lines in go-localsync | **Yes** |
| No deterministic ID helper | **Library** | SHA256 cache in go-localsync | **Yes** |
| `NewTypedProjection` single-payload | **Library** | Manual switch in both consumers | **Yes** |
| `aggregateType` threaded everywhere | **Library** | Noise in every call | **Yes** |
| SEC built parallel event system | **Consumer** | 250 lines waste | Make it easier not to |
| SEC bus with no subscribers | **Consumer** | Dead code | No |
| go-localsync outbox data loss bug | **Consumer** | Silent event loss | No — but they should use `OutboxPublisher` |

**Three problems cause 80% of consumer pain:**

1. **`Execute` returns only `error`** — forces every consumer to hack around knowing what happened
2. **`Command.AggregateID()` is required upfront** — blocks an entire class of use cases from using the command system
3. **`NewEvent` takes raw `[]byte`** — forces every consumer to marshal manually and build translation layers

Fix those three and **~700 lines of boilerplate across both projects disappears**.

---

## 8. Improvement Plan

### Priority 1: High Impact, Medium Effort

#### 8.1 `Execute` Should Return a Result, Not Just Error

```go
type Result struct {
    Events    []Event
    Version   Version
    Created   bool    // was initial state?
    Updated   bool    // emitted events?
    NoOp      bool    // no events emitted
}

func (r *Repository[State]) Execute(ctx, aggID, aggType, decide) (*Result, error)
```

Kills the `countingDecide` hack, the `classifyAction` workaround, and lets consumers respond meaningfully to their own commands.

#### 8.2 Make `AggregateID()` Optional on `Command`

```go
type Command interface {
    Type() string
    // AggregateID() is optional — checked via interface assertion
}
type AggregateCommand interface {
    Command
    AggregateID() id.AggregateID
}
```

Or better: make commands plain structs and derive everything from convention. go-localsync *cannot use* the library's command dispatcher because of this one method.

#### 8.3 Unify `NewEvent` and `NewEvents` — Both Should Accept Typed Payloads

```go
// Both should work:
event.New("TodoCreated", aggID, "Todo", version, TodoCreated{Title: "buy milk"})
event.NewBatch(aggID, "Todo", version, payloads, opts...)  // already exists as NewEvents
```

No consumer should ever call `json.Marshal` for event payloads.

### Priority 2: High Impact, Low Effort

#### 8.4 Accept `fmt.Stringer` or Raw Strings for IDs Alongside `id.AggregateID`

```go
// Instead of only:
func NewEvent(type string, aggID id.AggregateID, ...) (*Core, error)

// Also accept:
func NewEvent(type string, aggID string, ...) (*Core, error)
```

SEC converts between their `GameID` and `id.AggregateID` at every boundary. Pure friction.

#### 8.5 Provide `id.Derive(namespace, key)` for Deterministic IDs

```go
aggID := id.DeriveAggregateID("sync", "github", "repo-123")
// Same result every time — no cache needed
```

#### 8.6 Storage: Own Your DB Lifecycle

```go
store, err := storage.NewSQLEventStore(db, storage.WithOwnership()) // store.Close() closes db
```

One option eliminates 150 lines of wrapper code.

#### 8.7 Multi-Type Projection Builder

```go
proj := projection.Build("sync_projection",
    projection.On[Synced](handleSynced),
    projection.On[ConflictFound](handleConflict),
    projection.On[Deleted](handleDeleted),
)
```

Instead of every consumer writing manual `switch evt.Type()` with `DecodePayload[T]`.

### Priority 3: Medium Impact, Medium Effort

#### 8.8 Read Model Base Infrastructure

```go
type ReadModel[T any] interface {
    Get(ctx, id.AggregateID) (T, error)
    List(ctx, filter Filter) ([]T, error)
}

// Provided:
memory.NewReadModel[T]()    // sync.Map-backed
```

#### 8.9 Unified Storage Constructors with Functional Options

```go
// Replace 12 constructors with 4:
storage.NewEventStore(db, storage.WithDialect(storage.Postgres))
storage.NewOutbox(db, storage.WithDialect(storage.SQLite))
storage.NewSnapshotStore(db, storage.WithDialect(storage.Postgres))
storage.NewCheckpointStore(db, storage.WithDialect(storage.Turso))
```

#### 8.10 Delete Dead Code

- Remove deprecated `CatalogMeta` from 3 packages
- Delete `sync/` module (ghost system)
- Remove `IdempotencyKey()` from `Command` interface

### Quick Wins (< 30 min each)

| Issue | Fix | Effort |
|---|---|---|
| `RetryConfig.IsRetryable` nil panic | Add nil check to `Validate()` | 2 min |
| `aggregate` unnecessary `fmt.Errorf("%w", sentinel)` | Return sentinel directly | 5 min |
| `projection.Runner.Register` no synchronization | Add mutex | 10 min |
| Storage constructor aliases → `WithDialect` option | Replace 12 aliases with 4 constructors | 30 min |

---

## 9. Command & Query Audit for Analytics

### Concept

Save all commands and queries to disk for analytics, replay, debugging, and compliance. This should be a first-class middleware.

### Proposed API

```go
// Middleware registration
commandDisp.Use(middleware.CommandAudit(auditStore))
queryDisp.Use(middleware.QueryAudit(auditStore))

// Store interface
type AuditStore interface {
    Record(ctx context.Context, entry AuditEntry) error
}

type AuditEntry struct {
    ID          id.AuditID
    Type        string           // command type or query type
    Payload     []byte           // serialized command/query
    AggregateID id.AggregateID
    UserID      id.UserID
    Timestamp   time.Time
    Duration    time.Duration
    Error       error            // nil if success
    Result      []byte           // serialized result (for queries)
}
```

### Storage Backends

```go
// SQLite/Turso — structured queries, analytics SQL
storage.NewSQLiteAuditStore(db)

// In-memory — testing
memory.NewAuditStore()

// JSON files — disk-based analytics, easy to parse
audit.NewJSONFileStore("/var/log/cqrs/commands/")

// Event store — write audit entries as events (event sourcing all the way down)
auditStore := audit.NewEventAuditStore(eventStore, eventBus)
```

### Benefits

- **Analytics:** "How many CreateTodo commands failed yesterday?" "What's the p99 latency for PlayRound?"
- **Replay:** Replay commands to reproduce bugs
- **Compliance:** Full audit trail of who did what, when
- **Debugging:** "What commands were issued before this crash?"
- **Zero consumer effort:** One line of middleware registration

---

## Appendix: Complete Issue Summary by Severity

### Critical (Breaks Consumer Use Cases)

1. `Command.AggregateID()` required upfront — blocks deterministic ID use cases
2. `Execute` returns only `error` — forces every consumer to hack around it
3. `sync/` ghost module — zero consumers, dead weight
4. Middleware triplication — ~1200 lines of pure duplication

### High (Causes Significant Boilerplate)

5. `event.NewEvent` takes raw `[]byte` — forces manual marshaling
6. `aggregate.Root` 9-method interface — OO anti-pattern
7. `event.Store` fat interface — 8 methods, should decompose
8. `query.Handler` returns `any` — violates "no any" rule
9. Storage 12 constructor aliases — should use options
10. No read model infrastructure — every consumer builds from scratch

### Medium (Friction and Inconsistency)

11. Deprecated `CatalogMeta` still shipped (3 packages)
12. Duplicated `RepositoryOption` + errors between aggregate/decider
13. Projection live handler swallows errors
14. `Metadata` mutation escape hatch
15. Inconsistent `Parse*` contracts
16. `RetryConfig.IsRetryable` nil panic
17. No deterministic ID generation helper
18. `NewTypedProjection` single-payload limitation
19. `AggregateID` backed by string vs ulid

### Low (Polish)

20. `reconstructEvent` 9 parameters
21. Catalog IDs are bare strings
22. `OutboxStatus` bare string
23. Inconsistent constructor return patterns
24. `MustParse*` only in `sync`
25. Storage `Save` no event validation
26. `Register` on `Runner` no synchronization
