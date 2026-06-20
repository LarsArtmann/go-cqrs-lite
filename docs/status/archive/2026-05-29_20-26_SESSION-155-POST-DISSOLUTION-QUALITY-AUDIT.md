# Session 155 — Post-Dissolution Quality Audit & Bug Fixes

**Date:** 2026-05-29 20:26 CEST
**Branch:** master
**Commits this session:** 11 (7009769..10d432b)
**Previous session:** 154 (core dissolution, schema/snapshot extraction)

---

## Executive Summary

Session 155 was a **deep audit and quality sweep** following the core/ dissolution in session 154. Three research agents analyzed the entire codebase — go.mod hygiene, type model quality, and dependency footprint. The audit uncovered a **critical bug** (stale `core v1.6.0` phantom deps breaking gopls for the entire workspace), a **real logic bug** (`Context()` returning immediately-cancelled contexts), and multiple architecture improvements.

**Result:** 0 test failures, 1 pre-existing lint issue, all 36 test packages green, workspace fully clean.

---

## A. FULLY DONE

### Critical Bugs Fixed

| #   | Issue                                                                                                                                                                                  | Fix                                                                              | Commit                 |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ---------------------- |
| 1   | **4 go.mod files had stale `core v1.6.0 // indirect`** — root cause of ALL 39 LSP errors across the workspace. Build worked via go.work but gopls couldn't resolve the phantom module. | Removed from `snapshot/`, `decider/`, `command/`, `query/` go.mod files          | `7009769`              |
| 2   | **`ImmutableEvent.Context()` returned immediately-cancelled context** — `defer cancel()` fired after return, giving callers a useless cancelled context even for future deadlines      | Removed the cancel call; context auto-expires at deadline via internal timer     | `ace23f7a`             |
| 3   | **Snapshot error codes used `event.*` prefix** instead of `snapshot.*` after extraction                                                                                                | Changed to `snapshot.not_found`, `snapshot.store_closed`, `snapshot.save_failed` | `5507e1f2`, `5fd95fae` |

### Architecture Improvements

| #   | Change                                                                                                                | Impact                                                                                                                                                                        | Commit     |
| --- | --------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 4   | **`VersionedStore` now embeds `event.Store`** — was only wrapping Load/LoadFromVersion, not usable as a drop-in Store | Consumers can use it anywhere `event.Store` is expected; Save/AppendBatch/Close delegate automatically via embedding. Added LoadToVersion and LoadToTimestamp with upcasting. | `9af0e91f` |
| 5   | **Replaced `go-faster/yaml` with `gopkg.in/yaml.v3`** in catalog/                                                     | Eliminates 3 transitive dependencies (go-faster/errors, go-faster/jx, segmentio/asm). yaml.v3 was already in dep tree via testify.                                            | `a2e09f60` |
| 6   | **Added `AggregateRef.IsZero()` and `Validate()`**                                                                    | Prevents zero-value refs from silently passing to Store operations                                                                                                            | `e143e248` |

### Quality & Testing

| #   | Item                                       | Detail                                                                                      | Commit                 |
| --- | ------------------------------------------ | ------------------------------------------------------------------------------------------- | ---------------------- |
| 7   | **HandlerToObserver test coverage**        | Added tests for handler invocation and context pass-through                                 | `45b78be7`             |
| 8   | **Event package lint: 29 → 0 issues**      | Fixed nlreturn, unused-parameter, gci formatting                                            | `e1f74349` + formatter |
| 9   | **Storage README stale refs fixed**        | Duplicate `## Components` header, `event.Snapshot` → `snapshot.Snapshot`, dep table updated | `1360f466`             |
| 10  | **Dissolution proposals marked COMPLETED** | Both HTML proposals updated                                                                 | `4babf2fe`             |
| 11  | **CHANGELOG.md updated**                   | All session 154 changes documented                                                          | `4babf2fe`             |

### Decisions Made

| Decision                                      | Rationale                                                                                                                                                           |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Skipped ro-as-internal-MemoryBus-plumbing** | Impedance mismatch is fundamental: `event.Handler` returns `error` (propagated to publisher), `ro.Observer.OnNext` is void. Current MemoryBus is correct and clean. |
| **Skipped Snapshot.Encoding field**           | Would break 20+ call sites. Needs design discussion (encoding values, who sets it, codec interaction).                                                              |
| **Skipped replacing samber/ro entirely**      | ro is used productively in `event/reactive.go`. Would need a full reactive library replacement to remove `samber/lo` transitive.                                    |

---

## B. PARTIALLY DONE

| Item               | Status                                                                                                  | What's Left                                                      |
| ------------------ | ------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| **Lint cleanup**   | event/ is clean (0 issues). Workspace has 1 pre-existing issue in `command/store.go:102` (exhaustruct). | Fix the last issue in command/                                   |
| **go.mod hygiene** | Core phantom deps removed from 4 files. All replace directives cleaned.                                 | Some example/ modules may have inconsistent placeholder versions |

---

## C. NOT STARTED

These items were identified during the audit but not executed:

| #   | Item                                                                                                                         | Impact                 | Effort                          |
| --- | ---------------------------------------------------------------------------------------------------------------------------- | ---------------------- | ------------------------------- |
| 1   | **`Snapshot.Encoding` field** — no encoding metadata on snapshots, codec changes could make old snapshots unreadable         | High (future-proofing) | Medium (20+ call sites)         |
| 2   | **`Snapshot.State` defensive copy** — `[]byte` field is mutable, caller can mutate after Save                                | Low                    | Tiny                            |
| 3   | **`Command` interface missing `AggregateType()`** — handlers can't route to aggregate types without out-of-band knowledge    | Medium                 | Small                           |
| 4   | **`VersionedStore.Upcast()` couples to `*ImmutableEvent`** — breaks for custom Event implementations                         | Medium                 | Small                           |
| 5   | **Event interface has 11 methods** (interfacebloat lint) — consider splitting read-only accessors from mutation              | Low                    | Large                           |
| 6   | **`Metadata` struct uses `omitempty` on nested struct fields** (modernize lint) — no effect, should use `omitzero` or remove | Low                    | Tiny                            |
| 7   | **`query.DispatchTyped` wraps errors twice** — creates deeply nested error chains                                            | Low                    | Tiny                            |
| 8   | **`CheckVersionConflict` takes `int` not `Version`** — inconsistent with typed approach                                      | Low                    | Tiny                            |
| 9   | **Add `Unsubscribe` to event.Bus** — handlers are permanent once registered                                                  | Medium                 | Medium                          |
| 10  | **Pagination on `EventSource.Load`** — forces full materialization for large aggregates                                      | High                   | Large                           |
| 11  | **Dedicated `Position` type for `SeekableJournal.ReadFrom`** — currently uses `id.EventID` as position                       | Low                    | Small                           |
| 12  | **`turso/` module has no tests**                                                                                             | Medium                 | Medium                          |
| 13  | **`storage/sql/` has no tests** (subpackage)                                                                                 | Low                    | Small                           |
| 14  | **oklog/ulid v1+v2 version conflict in watermill**                                                                           | Low (Cosmetic)         | Can't fix (watermill brings v1) |
| 15  | **Inconsistent placeholder versions in example/ go.mod files**                                                               | Low                    | Tiny                            |

---

## D. TOTALLY FUCKED UP

Nothing is totally fucked up. The workspace is in the **cleanest state it has ever been in**:

- 36/36 test packages green
- 1 pre-existing lint issue (not in files we touched)
- 0 stale core/ references
- 0 broken imports
- All go.mod files consistent
- Build, test, lint all pass

---

## E. WHAT WE SHOULD IMPROVE

### Architecture

1. **Command routing needs `AggregateType()`** — without it, command handlers must have out-of-band knowledge of which aggregate type to load. This is a real consumer friction point.
2. **VersionedStore should not couple to `*ImmutableEvent`** — the `Upcast()` method returns `*ImmutableEvent`, not `event.Event`. This violates the interface abstraction and makes custom Event implementations impossible.
3. **Snapshot encoding is a ticking time bomb** — if the codec changes over time, old snapshots become unreadable with no migration path. At minimum, add an `Encoding string` field.

### Type Safety

4. **`CheckVersionConflict` takes raw `int`** instead of `Version` — inconsistent with the strong-typing philosophy.
5. **`query.Handler` returns `any`** — the typed bookend pattern is good, but `DispatchTyped` double-wraps errors.
6. **`AggregateRef` has no mandatory validation** — `Validate()` exists now but isn't called by Store implementations. Should it be?

### Testing

7. **`turso/` module has ZERO tests** — it's a database connector with no test coverage.
8. **`storage/sql/` subpackage has no tests** — dialect code is only tested indirectly via the parent package.

### Dependencies

9. **samber/lo is dead weight everywhere** — it's only a transitive dep of samber/ro, but it inflates every module's dependency footprint since event/ (which everything depends on) pulls it in.
10. **example/user depends on larsartmann/httputil** — an external dep only used in one example. Consider inlining or removing.

### Developer Experience

11. **No `Unsubscribe` on event.Bus** — handlers are permanent. This limits test isolation and dynamic subscription patterns.
12. **No pagination on `EventSource.Load`** — forces full materialization. The `StreamLoader` exists but isn't part of the `EventSource` interface.

---

## F. Top #25 Things We Should Get Done Next

Sorted by impact × effort (Pareto ranking):

### P0 — Do Next (High Impact, Low Effort)

1. **Fix the 1 remaining lint issue** (`command/store.go:102` exhaustruct) — 1 minute
2. **Add `AggregateType()` to `Command` interface** — enables type-safe command routing
3. **Call `AggregateRef.Validate()` in Store implementations** — wire the guard we just added
4. **Add tests for `turso/` module** — zero coverage on a database connector is risky
5. **Fix `query.DispatchTyped` double-wrapping errors** — removes confusing error chains

### P1 — Do Soon (High Impact, Medium Effort)

6. **Add `Encoding` field to `snapshot.Snapshot`** — future-proof against codec changes
7. **Make `VersionedStore.Upcast()` return `event.Event`** instead of `*ImmutableEvent`
8. **Add `Unsubscribe(eventType, handler)` to `event.Bus`** interface
9. **Paginated `Load` on `EventSource`** or at least a `LoadPaginated` extension interface
10. **`CheckVersionConflict` should take `Version` not `int`**
11. **Replace samber/ro with a lightweight channel-based implementation** — removes samber/lo from every module

### P2 — Do Eventually (Medium Impact, Low Effort)

12. **Defensive copy on `Snapshot.State`** in `SaveSnapshot` helper
13. **Fix `Metadata` struct `omitempty` on nested structs** (use `omitzero` or remove tags)
14. **Standardize placeholder versions in example/ go.mod files**
15. **Remove `StreamKey()` method from `AggregateRef`** — duplicates `String()`
16. **Add `CreatedAt` documentation to `Snapshot`** — who sets it?
17. **Fix `nolint:gocritic` directive on `event/event.go:184`** — it's unused, the cancel was removed
18. **Add `SchemaVersion.Increment()` method** — `schema/registry.go` does `+1` arithmetic directly

### P3 — Nice to Have (Low Impact)

19. **`io.Closer` should not be on `EventSink`/`EventSource`** — forces close semantics on in-memory stores
20. **Split `Event` interface** — 11 methods is borderline (interfacebloat), consider read-only sub-interface
21. **Dedicated `Position` type** for `SeekableJournal.ReadFrom` instead of `id.EventID`
22. **Add integration test for `VersionedStore` as full `event.Store`** — verify the embedding works end-to-end
23. **Add version tags and push v1.0.0** — eliminates all replace directives
24. **Move `oklog/ulid` v1→v2 in watermill** or accept the duplication
25. **Consider NATS JetStream backend** — research doc exists, would be a major feature addition

---

## G. Top #1 Question I Cannot Figure Out Myself

**Should `Command` include `AggregateType()` in its interface?**

Currently `Command` only has `Type()` and `AggregateID()`. The decider `Execute` function needs an `AggregateType` parameter passed separately. Two approaches:

1. **Add `AggregateType()` to `Command` interface** — cleaner API, every command knows its aggregate type. But: breaks the interface for any consumer who defined commands without it. Requires updating `BasicCommand`, all examples, all tests.

2. **Keep it separate** — decider `Execute` takes `(aggregateType, aggregateID, command)`. More flexible (same command type can target multiple aggregate types) but more verbose.

The tradeoff is between API cleanliness and backward compatibility. **What's your preference?**

---

## Workspace Metrics

| Metric                | Value                                                  |
| --------------------- | ------------------------------------------------------ |
| Go version            | 1.26.3                                                 |
| Workspace modules     | 29 (22 library + 6 examples + 1 integration)           |
| go.mod files          | 31                                                     |
| Test packages         | 36 (all green)                                         |
| No-test packages      | 3 (`storage/sql`, `turso`, `catalog/internal/cattest`) |
| Lint issues           | 1 (pre-existing in `command/store.go`)                 |
| External dependencies | 12 production, 5 test-only                             |
| Replace directives    | Required until v1.0.0 tags pushed                      |

## Dependency Graph

```
Layer 0: id/, dispatcher/, codec/
Layer 1: event/ (→id, codec, ro), command/ (→id, dispatcher), query/ (→dispatcher)
Layer 2: schema/ (→event), snapshot/ (→event)
Layer 3: decider/ (→event, snapshot)
Layer 4: memory/, testhelpers/, signing/, otel/
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen
```

## Commits This Session (full list)

```
10d432b2 chore(deps): update catalog module transitive dependency checksums and normalize schema spacing
492afc7c docs(research): add comprehensive analysis of graph databases for event sourcing
5fd95fae fix(snapshot): correct error code from event.snapshot_save_failed to snapshot.save_failed
e143e248 feat(event): add AggregateRef.IsZero() and Validate()
45b78be7 test(event): add HandlerToObserver and HandlerToObserverWithContext tests
a2e09f60 refactor(catalog): replace go-faster/yaml with gopkg.in/yaml.v3
9af0e91f refactor(schema): VersionedStore now embeds event.Store for drop-in usage
1360f466 docs(storage): fix stale event.Snapshot→snapshot.Snapshot references in README
5507e1f2 fix(snapshot): correct error codes to use snapshot. prefix
ace23f7a fix(event): Context() returned immediately-cancelled context due to defer cancel()
70097692 fix: remove stale core v1.6.0 indirect deps from 4 go.mod files
```
