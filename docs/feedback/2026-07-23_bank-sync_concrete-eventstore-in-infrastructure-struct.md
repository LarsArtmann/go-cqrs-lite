# go-cqrs-lite — Consumer Feedback: Infrastructure Struct Exposes Concrete EventStore Instead of event.Store Interface

**Consumer:** [bank-sync](https://github.com/LarsArtmann/bank-sync) — bank transaction sync CLI (Wise API + Qonto CSV → SQLite)
**SDK version:** v4.0.x (storage/v4, event/v4, decider/v4)
**Date:** 2026-07-23
**Severity:** Low — cosmetic / API ergonomics, not a blocker
**Status:** Observation from consumer wiring. No bug, no functional gap. The question is whether the SDK can nudge consumers toward better type choices.

---

## TL;DR

The SDK ships an excellent ISP-split interface (`event.Store` = `EventSink` + `EventSource`), but consumers who manually wire their own infrastructure struct naturally end up holding the **concrete** `*storage.SQLEventStore` instead of the interface. This works fine — bank-sync is permanently bound to SQLite — but it raises a question during code review: _"Why not `event.Store`?"_ The SDK doesn't provide a named bundle struct or wiring guidance that would make holding the interface the path of least resistance.

---

## 1. The question that came up

While reviewing bank-sync's `cqrs.Infrastructure` struct (a consumer-side composition of all CQRS runtime components), a reader asked:

> ```go
> type Infrastructure struct {
>     EventStore        *cqrsstorage.SQLEventStore      // <-- why concrete?
>     Bus               *watermill.EventBus
>     DeciderRepo       *decider.Repository[BalanceSyncState]
>     CommandDispatcher *command.Dispatcher
>     QueryDispatcher   *query.Dispatcher
> }
> ```
>
> **Is there not a `cqrsstorage.EventStore`?** Why expose the concrete implementation?

The answer: yes, `event.Store` exists and is the correct interface. bank-sync's own internal wiring immediately assigns the concrete store to the interface:

```go
rawEventStore, err := cqrsstorage.NewSQLiteEventStore(db)
// ...
var eventStore event.Store = rawEventStore   // widened here
```

The struct field keeps the concrete type because bank-sync commits to SQLite permanently. This is a valid design choice for that consumer — but it prompted the question, and the question reveals an ergonomics gap.

---

## 2. What the SDK provides today

| Capability                                | SDK API                                                      | Notes                                                 |
| ----------------------------------------- | ------------------------------------------------------------ | ----------------------------------------------------- |
| Interface (read+write)                    | `event.Store` (ISP: `EventSink` + `EventSource`)             | ✅ Well-designed, segregated                          |
| Interface (read-only journal)             | `event.Journal`                                              | ✅                                                    |
| Concrete SQLite store                     | `storage.NewSQLiteEventStore(db)` → `*storage.SQLEventStore` | ✅                                                    |
| Stack presets (schema+bus+stores+Close)   | `stack/sqlite.New(...)`, `stack/postgres.New(...)`           | ✅ Returns a `Bundle` struct                          |
| **Named bundle struct for manual wiring** | —                                                            | ❌ Consumers invent their own `Infrastructure` struct |

The `stack` presets return a `Bundle` that holds concrete types. There is no SDK-provided struct or guidance for consumers who want to hold `event.Store` (the interface) in their own composition root.

---

## 3. Why consumers end up with concrete types

When a consumer needs more customization than a `stack` preset offers (bank-sync needs custom bus middleware, encryption, snapshot strategy, versioned store with upcasters), they must wire components individually. At that point:

1. The constructor `NewSQLiteEventStore` returns `*storage.SQLEventStore` (concrete)
2. The consumer builds their own struct to hold the wired components
3. The struct field type mirrors whatever the constructor returned — concrete
4. Nothing in the SDK suggests "you should store this as `event.Store`"

The concrete type is technically correct (the consumer knows their backend), but it leaks the storage choice into every place the struct is passed. If a consumer later wanted to add a second backend (e.g., Postgres for server mode, SQLite for local mode), they'd need to change the field type — even though every consumer of the field only uses interface methods.

---

## 4. What would help

### Option A: Documentation guidance (minimal)

Add a note to the `storage` package doc or an "Advanced Wiring" guide:

> **Tip:** When building a custom infrastructure struct, hold `event.Store` (the interface) rather than the concrete `*SQLEventStore`. This keeps your composition root decoupled from the storage backend and makes future backend swaps a one-line change.

This costs nothing and nudges consumers in the right direction.

### Option B: An `Infrastructure` or `Components` struct in the SDK

Provide a generic struct that consumers can embed or compose:

```go
// in the SDK
type Components struct {
    EventStore event.Store
    Bus        *watermill.EventBus
    // ... other common fields
}
```

This gives consumers a named home for wired components, with interface types by default. Consumers who need concrete access can type-assert.

### Option C: Constructors return interfaces

Have `NewSQLiteEventStore` return `event.Store` directly instead of `*SQLEventStore`. This is the most opinionated change and would break consumers who rely on concrete methods (e.g., bank-sync doesn't, but others might). Probably not worth the churn.

---

## 5. Recommendation

**Option A (documentation)** is the pragmatic fix. The ISP split is already excellent design — consumers just need a one-line nudge to hold the interface in their own structs. Options B and C are heavier changes with diminishing returns for what is ultimately a cosmetic concern.

The deeper insight: the SDK's `stack` presets already solve this for the 80% case (consumers who don't need custom wiring get a `Bundle` and never think about types). This feedback only applies to the 20% who hand-wire — and for them, a doc note is sufficient.

---

## 6. Consumer's own resolution

bank-sync accepts the concrete type as a deliberate coupling choice: SQLite is an architectural given for a local-first CLI tool, not a swappable backend. The internal wiring immediately widens to `event.Store` where polymorphism is needed. This is a valid tradeoff — the feedback here is about making the choice **visible and conscious** rather than accidental.
