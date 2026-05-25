# AggregateID Design Review

**Date:** 2026-05-25
**Status:** Pending Decision

## Context

`AggregateID` is the only ID type in `core/pkg/id/` backed by `string` instead of `ulid.ULID`. It also has exported markers, hash-based derivation, and broad API surface. This document captures the findings and tradeoffs for three related decisions.

---

## 1. Should `AggregateID` switch to `id.Of[AggregateMarker]` (ULID-backed)?

### PRO

| #   | Argument                                                                                      |
| --- | --------------------------------------------------------------------------------------------- |
| 1   | Consistency — all ID types use the same backing                                               |
| 2   | Time-sortable aggregate IDs (free timestamp extraction)                                       |
| 3   | Simpler `aggregate_id.go` — remove `ParseAggregateID`, `DeriveAggregateID`, `AggregateIDFrom` |
| 4   | Removes the YAGNI `DeriveAggregateID` and its hash nonsense                                   |

### CONTRA

| #   | Argument                                                                                                                                                                                                                                                                                                                                                                                                                                |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Breaks existing stored data** — consumers may have UUIDs, natural keys, or other non-ULID aggregate IDs in their event stores. The library can't dictate what's already persisted.                                                                                                                                                                                                                                                    |
| 2   | **Breaks interop** — `AggregateIDFrom(fmt.Stringer)` exists so consumers can pass their own branded ID types (which may not be ULIDs). `example/todo` uses `TodoID = id.Of[TodoMarker]` and passes it as an `AggregateID`. That works today because `string` backing accepts any string. With ULID backing, `AggregateIDFrom(todoID)` would create a ULID-backed ID from a non-ULID string — and `Parse` would reject it on round-trip. |
| 3   | **Wrong abstraction level** — a library doesn't own consumers' identity strategy. Aggregate IDs are the _consumer's_ domain concept. Forcing ULID is framework opinion, not library neutrality.                                                                                                                                                                                                                                         |

### Current Verdict

The `string` backing is a **deliberate design choice, not an oversight**. It's the one ID type where consumers must be free to bring their own format.

---

## 2. Should `DeriveAggregateID` be removed?

### PRO

| #   | Argument                                                                                                    |
| --- | ----------------------------------------------------------------------------------------------------------- |
| 1   | **Idempotent creation** — same inputs → same ID, no "find or create" query needed                           |
| 2   | **No collisions** — SHA-256 guarantees uniqueness across namespaces/keys                                    |
| 3   | **No external state** — don't need a lookup table or unique index to enforce "one lock per user+resource"   |
| 4   | **Event sourcing fit** — aggregates addressed by identity, not by query; fits the "load by ID" mental model |

### CONTRA

| #   | Argument                                                                                                                                                                                              |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Unnecessary hashing** — `AggregateID` is `string`-backed; `"lock:user1:resource1"` works identically and is simpler                                                                                 |
| 2   | **Kills debuggability** — `"a3f2b9..."` in logs/events is opaque; natural keys are self-documenting                                                                                                   |
| 3   | **Unused (YAGNI)** — zero production callers in the entire codebase (only its own tests use it)                                                                                                       |
| 4   | **Over-engineered API** — 3 functions for this (`DeriveAggregateID`, `AggregateIDFrom`, `ParseAggregateID` accepting any string) when `AggregateIDFrom` + a consumer-side string format would suffice |
| 5   | **Fixed-length waste** — 64-char hex string vs ~20-char natural key, bloating every event that carries it                                                                                             |
| 6   | **No reversibility** — can't extract `user1` or `resource1` from the hash; natural key preserves information                                                                                          |
| 7   | **Premature abstraction** — the library consumer should decide their ID derivation strategy, not the library                                                                                          |

### Current Verdict

The problem it solves is real (idempotent aggregates), but the solution is overbuilt and unused. A consumer can already do `id.AggregateIDFrom()` with any string they choose — including their own natural key or hash. `DeriveAggregateID` should be removed.

---

## 3. Should `AggregateMarker` be unexported?

### Current State

`AggregateMarker` is the **only** exported marker type. All others (`eventMarker`, `userMarker`, `clientMarker`, etc.) are unexported.

The stated reason in the comment: _"Export it so domain packages can create domain-specific IDs interoperable with AggregateID."_

### Actual Usage

One external consumer embeds it:

```go
// example/todo/domain/ids.go
type TodoMarker struct {
    id.AggregateMarker
}

type TodoID = id.Of[TodoMarker]
```

### Problem

The embedding provides **zero actual interoperability**:

- `AggregateID` = `cbid.ID[AggregateMarker, string]`
- `TodoID` = `cbid.ID[TodoMarker, ulid.ULID]`

Different markers, different backing types. `TodoID` is not assignable to `AggregateID` regardless of the embedding. The conversion goes through `AggregateIDFrom(fmt.Stringer)` which only calls `.String()` — no marker relationship needed.

### PRO unexporting

| #   | Argument                                                             |
| --- | -------------------------------------------------------------------- |
| 1   | Consistency with all other marker types                              |
| 2   | Removes a misleading API that implies type interop through embedding |
| 3   | Simpler public surface                                               |

### CONTRA unexporting

| #   | Argument                                                                                             |
| --- | ---------------------------------------------------------------------------------------------------- |
| 1   | Breaking change if any external consumer relies on `AggregateMarker` (currently only `example/todo`) |

### Current Verdict

Should be unexported. The `example/todo` embedding is decorative and should be removed.

---

## Usage Scope

`AggregateID` is used **everywhere** — 255 references across 66 files:

| Layer         | Usage                                                                                |
| ------------- | ------------------------------------------------------------------------------------ |
| `event.Store` | `Save`, `Load`, `LoadFromVersion`, `LoadFromTimestamp` — all keyed by `AggregateID`  |
| `event.Event` | `AggregateID()` return type on every event                                           |
| `decider`     | `Execute`, `Load`, `ExecuteWithResult` — all take `AggregateID` as the aggregate key |
| `command`     | `Command.AggregateID()` return type                                                  |
| `memory`      | Store/bus/snapshot all use it as the primary key                                     |
| `storage`     | SQL, SQLite, Pebble — all SQL columns, key encodings, lookups by `AggregateID`       |
| `testhelpers` | `FakeStore`, `NewEvent` helpers, snapshot fakes                                      |
| `examples`    | Both `example/user` and `example/todo` create and pass `AggregateID` throughout      |

## All ID Types in the Project

### Core library — `id.Of[T]` (ULID-backed)

| Type            | File                              | Backing                            |
| --------------- | --------------------------------- | ---------------------------------- |
| `AggregateID`   | `core/pkg/id/aggregate_id.go:26`  | `cbid.ID[AggregateMarker, string]` |
| `EventID`       | `core/pkg/id/event_id.go:8`       | `id.Of[eventMarker]`               |
| `UserID`        | `core/pkg/id/user_id.go:8`        | `id.Of[userMarker]`                |
| `ClientID`      | `core/pkg/id/client_id.go:8`      | `id.Of[clientMarker]`              |
| `RequestID`     | `core/pkg/id/request_id.go:8`     | `id.Of[requestMarker]`             |
| `CorrelationID` | `core/pkg/id/correlation_id.go:8` | `id.Of[correlationMarker]`         |
| `CausationID`   | `core/pkg/id/causation_id.go:8`   | `id.Of[causationMarker]`           |

### Catalog module — plain `string`-based

| Type        | File                  |
| ----------- | --------------------- |
| `ServiceID` | `catalog/types.go:9`  |
| `DomainID`  | `catalog/types.go:35` |
| `MessageID` | `catalog/types.go:41` |
| `ChannelID` | `catalog/types.go:47` |

### Other

| Type       | File                            | Note                |
| ---------- | ------------------------------- | ------------------- |
| `OutboxID` | `core/event/outbox.go:9`        | plain `string`      |
| `TodoID`   | `example/todo/domain/ids.go:11` | `id.Of[TodoMarker]` |

---

## Proposed Actions (if approved)

1. **Keep `AggregateID` as `string`-backed** — the design is correct for a library
2. **Remove `DeriveAggregateID`** — YAGNI, consumers can derive their own IDs
3. **Unexport `AggregateMarker`** — remove `// Export it so...` comment, unexport, and remove the embedding from `example/todo/domain/ids.go`
