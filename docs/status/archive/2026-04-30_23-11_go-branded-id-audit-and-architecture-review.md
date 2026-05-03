# Session 15: go-branded-id Audit & Architecture Review

**Date:** 2026-04-30 23:11
**Focus:** Verify proper usage of `github.com/larsartmann/go-branded-id`, eliminate duplication, identify architectural gaps

---

## Executive Summary

The `id` package was **not** properly using `go-branded-id`. It re-implemented all 8 serialization methods that the library already provides through interface detection. Fixed by replacing 175 lines of hand-rolled serialization with 32 lines of delegation. Deeper audit revealed a **critical data-loss bug** in `storage/scanEvents` and several lower-priority improvements.

---

## Work Completed

### a) FULLY DONE

| #   | Item                                                                                                                                                                              | Commit    |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| 1   | **Delegation refactor**: Replaced 8 re-implemented serialization methods with one-line delegations to `cbid.ID[T, ulid.ULID]`                                                     | `6511003` |
| 2   | **Dead code removal**: Removed `errNilReceiver`, `errUnsupportedType` (never referenced outside deleted code), `MaxULIDsPerMs` (zero references outside tests), `math` import     | `6511003` |
| 3   | **Full audit**: Searched all 7 modules for: raw `string` where branded IDs belong, direct `ulid` imports outside `id` package, manual ULID parsing, unnecessary `.String()` calls | —         |
| 4   | **All tests pass**: 17 packages, 0 failures                                                                                                                                       | —         |
| 5   | **Zero lint**: All 6 linted modules clean                                                                                                                                         | —         |

### b) PARTIALLY DONE

Nothing partially done this session.

### c) NOT STARTED

See "Top 25 Things to Do Next" below.

### d) TOTALLY FUCKED UP (Found, Not Introduced)

| Severity    | File                                            | Bug                                                                                                                                                                                                                                     | Impact                                                                                           |
| ----------- | ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| 🔴 CRITICAL | `storage/event_store.go:246-289` (`scanEvents`) | Event ID scanned from DB (`idStr`) is **never passed** to `event.NewEvent()`. Every call to `event.NewEvent()` generates `id.NewEventID()` (line 187 of `event.go`). Events loaded from SQL get **new IDs instead of their originals**. | Data loss in production: audit trails break, idempotency breaks, event replay produces wrong IDs |
| 🟡 MEDIUM   | `storage/event_store.go` (throughout)           | SQL store manually calls `.String()` on write and `id.ParseAggregateID()` on read, completely ignoring the `sql.Scanner`/`driver.Valuer` implementations that now properly delegate to `cbid.ID`                                        | Unnecessary code, missed type safety                                                             |
| 🟢 LOW      | `core/event/event.go:194`                       | `NewEvent` uses `time.Now()` — not injectable, makes testing harder                                                                                                                                                                     | Test flakiness                                                                                   |

---

## Architecture Analysis

### Current State of go-branded-id Integration

```
go-cqrs-lite/core/pkg/id/
├── id.go          (125 lines) — Of[T] wrapping cbid.ID[T, ulid.ULID]
│                    Delegates: New, Parse, MustParse, IsZero, Equal,
│                    Compare, Get, Or, Reset, String, GoString, ULID
├── id_encoding.go (32 lines)  — Pure delegation to cbid.ID methods
│                    MarshalJSON, UnmarshalJSON, MarshalBinary,
│                    UnmarshalBinary, MarshalText, UnmarshalText,
│                    Scan, Value
├── aggregate_id.go, event_id.go, user_id.go, etc. (6 files)
│                    Brand markers + convenience constructors
└── id_test.go, fuzz_test.go, benchmark_test.go (992 lines)
```

### What go-branded-id Provides vs What We Add

| Capability                            | go-branded-id | Our `id` package | Rationale                                                          |
| ------------------------------------- | :-----------: | :--------------: | ------------------------------------------------------------------ |
| `ID[B, V]` core type                  |      ✅       |        —         | Generic branded type                                               |
| JSON marshal/unmarshal                |      ✅       |  ✅ (delegated)  | cbid handles via interface detection                               |
| Binary marshal/unmarshal              |      ✅       |  ✅ (delegated)  | cbid handles ulid.ULID's BinaryMarshaler                           |
| Text marshal/unmarshal                |      ✅       |  ✅ (delegated)  | cbid handles ulid.ULID's TextMarshaler                             |
| SQL Scan/Value                        |      ✅       |  ✅ (delegated)  | cbid handles ulid.ULID's TextMarshaler → string                    |
| IsZero/Equal/Reset/Or                 |      ✅       |  ✅ (delegated)  | Direct delegation                                                  |
| `NewID[B, V](v)`                      |      ✅       |        —         | We use `New[T any]()` with ULID auto-generation                    |
| ULID-specific generation              |       —       |        ✅        | `newULID()` with crypto/rand + timestamp                           |
| ULID-specific parsing                 |       —       |        ✅        | `Parse[T](s)` with `ulid.Parse()` validation                       |
| ULID timestamp extraction             |       —       |        ✅        | `ULID(id)` returns `time.Time`                                     |
| Direct `ulid.ULID.Compare()`          |       —       |        ✅        | Bypasses `cbid.ID.Compare()` (returns error for non-primitives)    |
| Pre-defined brand types               |       —       |        ✅        | AggregateID, EventID, UserID, CorrelationID, etc.                  |
| Convenience constructors              |       —       |        ✅        | `NewAggregateID()`, `ParseAggregateID()`, `MustParseAggregateID()` |
| `fmt.Formatter` (%s, %d, %v, %#v, %q) |      ✅       |        ❌        | Not forwarded — our `String()` uses `ulid.ULID.String()` directly  |
| `Ptr()`/`FromPtr()`                   |      ✅       |        ❌        | Not forwarded — could be useful                                    |
| `GobEncode`/`GobDecode`               |      ✅       |        ❌        | Not forwarded — rarely needed                                      |

### Type Model Assessment

**Good patterns (keep):**

- Named field `wrapped cbid.ID[T, ulid.ULID]` — correct, embedding would leak wrong method signatures
- `Compare` bypassing `cbid.ID.Compare()` — avoids `(int, error)` return for a type we know is ordered
- `String()` using `ulid.ULID.String()` directly — skips cbid's type switch

**Missing opportunities:**

- `fmt.Formatter` not forwarded → `%#v` shows raw ULID instead of `id(01H...)`
- `Ptr()`/`FromPtr()` not forwarded → useful for optional ID fields in API payloads
- No `FromULID[T](ulid.ULID)` constructor for reconstruction from storage

---

## e) WHAT WE SHOULD IMPROVE

### Critical

1. **Storage event ID preservation** — `scanEvents` must pass the original event ID through, not generate new ones. Requires either a `WithEventID` option on `NewEvent` or a separate `ReconstructEvent` constructor.

### High Impact / Low Effort

2. **Remove unnecessary `.String()` in `fmt.Errorf`** — 5 locations where branded IDs are passed to `fmt.Errorf` with explicit `.String()`. Since `Of[T]` implements `fmt.Stringer`, `%s` and `%q` format verbs call `.String()` automatically.
3. **Use `sql.Scanner`/`driver.Valuer` in storage** — The `id` package now properly delegates these. The SQL store can pass `id.AggregateID` directly as query parameters instead of calling `.String()` manually.
4. **Forward `Ptr()`/`FromPtr()` from cbid** — One-line delegations, enables optional ID fields.

### Medium Impact / Medium Effort

5. **Add `FromULID[T](ulid.ULID)` constructor** — Internal method for reconstructing IDs from storage without string round-trip.
6. **Forward `fmt.Formatter`** — Enables `%#v` to show `id(01H...)` and `%d` for numeric IDs.
7. **Brand `OutboxID`** — `core/event/outbox.go` defines `type OutboxID string`. Should be `type OutboxID = id.Of[outboxMarker]` for consistency.
8. **Add `WithEventID` option or `ReconstructEvent` function** — Needed to fix the storage bug without breaking `NewEvent`'s auto-ID-generation contract.

### Strategic / Larger Effort

9. **Watermill module** — Planned since session 14. Pub/sub integration for production event bus.
10. **SQL-backed snapshot store** — Planned since session 14. Currently only `MemorySnapshotStore`.
11. **Tag v0.1.0 releases** — All 9 modules ready for initial tagging.
12. **CI integration tests with real PostgreSQL** — `storage/` module only tested with compile checks, not against a real DB.

---

## f) Top 25 Things We Should Do Next

Sorted by **impact × ease** (highest first):

| #   | Item                                                                  | Effort | Impact                       | Area                                  |
| --- | --------------------------------------------------------------------- | ------ | ---------------------------- | ------------------------------------- |
| 1   | **Fix scanEvents to preserve original event ID**                      | S      | 🔴 Critical                  | storage bug                           |
| 2   | **Remove 5 unnecessary `.String()` calls in `fmt.Errorf`**            | XS     | Cleanliness                  | core/aggregate, core/command, storage |
| 3   | **Use `sql.Scanner`/`driver.Valuer` in storage module**               | S      | Type safety                  | storage                               |
| 4   | **Forward `Ptr()`/`FromPtr()` from cbid**                             | XS     | API completeness             | core/pkg/id                           |
| 5   | **Forward `fmt.Formatter` from cbid**                                 | XS     | Debugging UX                 | core/pkg/id                           |
| 6   | **Add `FromULID[T](ulid.ULID)` constructor**                          | XS     | Internal API                 | core/pkg/id                           |
| 7   | **Brand `OutboxID` as `id.Of[outboxMarker]`**                         | S      | Consistency                  | core/event                            |
| 8   | **Add `WithEventID` option or `ReconstructEvent`**                    | S      | 🔴 Critical fix prerequisite | core/event                            |
| 9   | **Add ` occurredAt` injectable option to `NewEvent`**                 | S      | Testability                  | core/event                            |
| 10  | **Upgrade go-branded-id to latest HEAD** (post-v0.1.0 changes)        | XS     | Maintenance                  | all modules                           |
| 11  | **Add GoDoc examples for `id.New`, `id.Parse`, `id.Of`**              | S      | Documentation                | core/pkg/id                           |
| 12  | **Add integration test for storage module with testcontainers**       | M      | Confidence                   | storage                               |
| 13  | **Create Watermill module (event bus)**                               | L      | Production readiness         | watermill/                            |
| 14  | **Create SQL-backed snapshot store**                                  | M      | Production readiness         | storage/                              |
| 15  | **Add `id.Of[T].MarshalJSONv2` support for go-json-experiment/json**  | S      | Compatibility                | core/pkg/id                           |
| 16  | **Add `contextEnricher` option to `NewEvent` for automatic metadata** | S      | DX                           | core/event                            |
| 17  | **Audit catalog module for branded ID usage**                         | S      | Consistency                  | catalog                               |
| 18  | **Add projection checkpoint SQL store**                               | M      | Production readiness         | storage/                              |
| 19  | **Tag v0.1.0 releases for all modules**                               | M      | Distribution                 | all                                   |
| 20  | **Add godoc for all exported types in storage module**                | S      | Documentation                | storage                               |
| 21  | **Example app with HTTP handlers + SQL storage**                      | M      | Documentation                | example/                              |
| 22  | **Add `encoding.BinaryMarshaler` usage in storage for payload**       | S      | Efficiency                   | storage                               |
| 23  | **Create `CHANGELOG.md` entry for session 15**                        | XS     | Documentation                | root                                  |
| 24  | **Update AGENTS.md with session 15 findings**                         | S      | Memory                       | root                                  |
| 25  | **Add `go-branded-id` replace directive for local dev**               | XS     | DX                           | go.work                               |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `event.NewEvent` support preserving the original event ID during reconstruction from storage?**

Currently `NewEvent()` always generates `id.NewEventID()` (line 187). The storage module's `scanEvents` needs to reconstruct events with their original IDs. There are two approaches:

**Option A: `WithEventID` option**

```go
event.NewEvent("user.created", aggID, "User", 1, payload, event.WithEventID(originalID))
```

Pro: Minimal change. Con: Exposes internal concern (ID generation) as public API.

**Option B: Separate `ReconstructEvent` constructor**

```go
event.ReconstructEvent(idStr, eventType, aggID, aggType, version, payload, occurredAt)
```

Pro: Explicit intent. Con: New public function, more parameters.

**Option C: Unexported `core` constructor, expose via `storage`**
Keep `NewEvent` for creation, add an unexported path for reconstruction.

This is a domain design decision that affects the public API contract of the `event` package. What's your preference?

---

## Test Coverage Impact

| Package       | Before | After | Note                                                |
| ------------- | ------ | ----- | --------------------------------------------------- |
| `core/pkg/id` | 97.1%  | 97.1% | Same tests pass — delegation is behavior-preserving |

---

## Files Changed This Session

```
core/pkg/id/id.go          131 → 125 lines (-6)
core/pkg/id/id_encoding.go 175 → 32 lines  (-143)
```

**Net: -149 lines of code, 0 behavior changes for consumers.**
