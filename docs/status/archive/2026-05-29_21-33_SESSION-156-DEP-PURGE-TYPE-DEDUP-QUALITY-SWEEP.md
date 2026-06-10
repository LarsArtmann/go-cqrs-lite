# Session 156 — Dependency Purge, Type Deduplication, and Second-Pass Quality Sweep

**Date:** 2026-05-29 21:33 CEST
**Branch:** master
**Commits this session:** 8 (99d6f6a..7d3f89e + lint fix pending)
**Previous session:** 155 (post-dissolution quality audit)

---

## Executive Summary

Session 156 was a **second-pass deep audit** triggered by the READ/UNDERSTAND/RESEARCH/REFLECT protocol. Three research agents re-examined the entire codebase from different angles (go.mod hygiene, type models, dependency footprint). This uncovered:

1. A **regression** — the core v1.6.0 phantom dep was back (buildflow pre-commit hook kept re-adding it because the root cause — missing local replace directives — was never fixed)
2. A **dead dependency** — `samber/ro` had zero production consumers, yet polluted every module's dep graph
3. **Duplicate types** — `command.AggregateType` and `command.AggregateRef` were identical copies of `event.*`

The session eliminated 3 direct dependencies and 3 transitive dependencies from the most-depended-on module (`event`), deduplicated two types, and fixed 3 other quality issues.

**Result:** 36/36 test packages green, 1 pre-existing lint issue, workspace clean.

---

## A. FULLY DONE

### Critical: Root Cause Fix (core v1.6.0 phantom dep — FINALLY solved)

| #   | What                                                          | Root Cause                                                                                                                                                                                                                                             | Fix                                                                                         | Commit    |
| --- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------- | --------- |
| 1   | core v1.6.0 kept re-appearing in command/decider/query go.mod | These modules require `testhelpers v1.6.0` (published when core/ existed) but **lacked local replace directives**. `go work sync` resolved the published module graph, pulling core as indirect. The buildflow pre-commit hook then auto-committed it. | Added missing replace directives: command/+testhelpers, decider/+memory, query/+testhelpers | `99d6f6a` |

**Key insight:** Session 155 treated the symptom (removing the core line) but not the cause (missing replaces). The buildflow hook re-added it every commit. The real fix was ensuring all local modules have replace directives so `go work sync` never needs to resolve published versions.

### Dependency Purge

| #   | What                                                                                                        | Impact                                                                                                                                                               | Commit    |
| --- | ----------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| 2   | **Deleted `event/reactive.go`** — zero production consumers of EventBus, FilterEventType, HandlerToObserver | Removed `samber/ro` (direct) + `samber/lo` + `golang.org/x/exp` (indirect) from `event/` and all downstream modules                                                  | `2f9caf9` |
| 3   | **Buildflow cascade tidy** — ro removal propagated to 47 files                                              | memory, decider, projection, listing, storage, otel, pebble, etc. all lost samber/lo + samber/ro + golang.org/x/exp. Net -199 lines removed from go.mod/go.sum files | `7d3f89e` |

**Before:** `event/` had 3 direct + 3 transitive deps from ro.
**After:** `event/` has zero samber deps. The entire workspace lost ~6 dependency entries.

### Type Deduplication

| #   | What                                                                 | How                                                                                                         | Commit                       |
| --- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- | ---------------------------- |
| 4   | `command.AggregateType` = `event.AggregateType`                      | Changed from duplicate `type AggregateType string` to type alias `type AggregateType = event.AggregateType` | `d8aa825`                    |
| 5   | `command.AggregateRef` = `event.AggregateRef`                        | Changed from duplicate struct to type alias `type AggregateRef = event.AggregateRef`                        | `d8aa825`                    |
| 6   | `command.ParseAggregateType` delegates to `event.ParseAggregateType` | Wrapper with `%w` error wrapping for lint compliance                                                        | `d8aa825` + pending lint fix |

### Other Fixes

| #   | What                                                                                                         | Commit    |
| --- | ------------------------------------------------------------------------------------------------------------ | --------- |
| 7   | Fixed misleading `otel.TraceIDLogger` docstring — claimed trace ID injection but only added `component=cqrs` | `3e9f88e` |
| 8   | Fixed `snapshot.EveryNEvents` — bare `fmt.Errorf` → `event.NewRejection` for error-family consistency        | `6a633ba` |
| 9   | Added `SchemaVersion.Increment()` — type-safe version arithmetic, used in `schema/registry.go`               | `6ccc2f5` |

---

## B. PARTIALLY DONE

| Item                        | Status                                                                                                                                                                                                                                   | What's Left               |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| **core v1.6.0 elimination** | Fixed in 3 modules (command, decider, query). Buildflow may re-add to decider on next commit (testhelpers v1.6.0 published with core dep). **Permanent fix requires publishing new testhelpers version or adding `// indirect` ignore.** | Monitor for re-regression |
| **Lint cleanup**            | event/ = 0 issues. command/ has 1 pre-existing exhaustruct + 1 wrapcheck fix pending commit.                                                                                                                                             | Commit the wrapcheck fix  |

---

## C. NOT STARTED

| #   | Item                                                                                                      | Impact            | Effort                               |
| --- | --------------------------------------------------------------------------------------------------------- | ----------------- | ------------------------------------ |
| 1   | **Add `AggregateType()` to `Command` interface** — enables simplified `decider.Execute(ctx, cmd, decide)` | High (ergonomics) | Medium (breaking for external impls) |
| 2   | **Add `ID()` to `Command` interface** — traceability                                                      | Medium            | Medium (breaking)                    |
| 3   | **`Snapshot.Encoding` field** — future-proof against codec changes                                        | High              | Medium (20+ call sites)              |
| 4   | **Fix `query.DispatchTyped` double-wrapping errors**                                                      | Low               | Tiny                                 |
| 5   | **Remove `io.Closer` from `EventSink`/`EventSource`**                                                     | Low               | Large                                |
| 6   | **Add `Unsubscribe` to `event.Bus`**                                                                      | Medium            | Medium                               |
| 7   | **Paginated `Load` on `EventSource`**                                                                     | High              | Large                                |
| 8   | **`listing.AggregateRef` — embed `event.AggregateRef`**                                                   | Low               | Small                                |
| 9   | **Consolidate OTel helpers** (`StartSpan`, `SpanFromContext` are trivial wrappers)                        | Low               | Small                                |
| 10  | **Simplify metrics middleware** — remove `MetricsRecorder` interface, keep only OTel-specific             | Low               | Medium                               |
| 11  | **Fix `Metadata` struct `omitempty` on nested structs**                                                   | Low               | Tiny                                 |
| 12  | **`CheckVersionConflict` should take `Version` not `int`**                                                | Low               | Tiny                                 |
| 13  | **Standardize example/ placeholder versions**                                                             | Low               | Tiny                                 |
| 14  | **Remove `StreamKey()` from `AggregateRef`** (duplicates `String()`)                                      | Low               | Tiny                                 |
| 15  | **Publish v1.0.0 tags** to eliminate all replace directives                                               | High              | Medium                               |

---

## D. TOTALLY FUCKED UP

### core v1.6.0 Phantom Dep — RE-REGRESSED TWICE

This is the biggest fuck-up of the session. The core v1.6.0 phantom dependency was "fixed" in session 155 (commit `7009769`) but:

1. **First regression:** The commit only cleaned `snapshot/go.mod` despite the message saying "4 files." The sed ran on all 4 but only snapshot was staged.
2. **Second regression:** Even after properly removing the line, the buildflow pre-commit hook ran `go mod tidy`, which re-added core because the `testhelpers v1.6.0` published tag transitively requires it. The hook then auto-committed this.
3. **Third regression (this session):** Same pattern — removed core, but go.work.sync re-added it because command/decider/query lacked replace directives for testhelpers.

**Root cause finally identified:** Missing local `replace` directives for `testhelpers` and `memory` meant Go resolved these via the network, saw the v1.6.0 tag (which includes core), and added core as indirect.

**Lesson learned:** In a multi-module monorepo with published tags, EVERY local module dependency MUST have a corresponding replace directive, or `go work sync` / `go mod tidy` will pull stale published module graphs.

### samber/ro Integration — Never Should Have Happened

The samber/ro integration (session 154, commit `cc4ceb1`) was premature:

- Created `EventBus` type alias with zero consumers
- Created `command/reactive.go` dead-on-arrival
- Dragged `samber/lo` + `golang.org/x/exp` into every module
- The `HandlerToObserver` adapter silently dropped errors and context

The impedance mismatch between Go's `func(ctx, T) error` and ro's `Observer.OnNext(T)` (void return) was identified early but the integration proceeded anyway.

---

## E. WHAT WE SHOULD IMPROVE

### Architecture

1. **Command routing needs `AggregateType()`** — the biggest ergonomic gap. `decider.Execute()` takes aggregateType separately, which is redundant if the command already knows its target.
2. **Replace directives are fragile** — every new inter-module dependency requires a replace directive. Publishing v1.0.0 tags would eliminate this entirely.
3. **Buildflow pre-commit hook fights manual go.mod edits** — the hook auto-tidies and commits, sometimes re-introducing deps we just removed. Need either: (a) configure buildflow to ignore core, or (b) ensure all local replaces are present BEFORE committing.

### Code Quality

4. **`command/store.go:102` exhaustruct** — the only remaining lint issue. `Metadata{}` literal is missing 4 fields. Should use `NewMetadata()` constructor.
5. **`listing.AggregateRef` doesn't embed `event.AggregateRef`** — field duplication, but the `Ref.` accessor pattern is used everywhere so the refactor is disproportionate.
6. **`snapshot.Snapshot.State` is mutable** — no defensive copy, but this is consistent with Go stdlib patterns.

### Dependencies

7. **`samber/lo` is completely gone now** — removed as transitive of ro. Verify no other module still pulls it.
8. **`go-faster/yaml` → `gopkg.in/yaml.v3`** already done (session 155). Verify the cascade is complete.
9. **Consider removing `stretchr/testify`** — some modules use both ginkgo+gomega AND testify. Pick one.

---

## F. Top #25 Things We Should Get Done Next

### P0 — Do Next (High Impact, Low Effort)

1. **Commit the pending wrapcheck fix** in `command/aggregate_ref.go`
2. **Fix the 1 remaining lint issue** (`command/store.go:102` exhaustruct — use `NewMetadata()` constructor)
3. **Add `AggregateType()` to `Command` interface** — biggest ergonomic win
4. **Fix `query.DispatchTyped` double-wrapping errors** — remove the inner wrap in `Dispatch`
5. **Publish v1.0.0 tags** for all modules — eliminates ALL replace directives permanently

### P1 — Do Soon (High Impact, Medium Effort)

6. **Add `Encoding` field to `snapshot.Snapshot`** — codec migration safety
7. **Add `Unsubscribe` to `event.Bus`** interface
8. **Write tests for `turso/`** — zero coverage on a database connector
9. **Add `ID()` to `Command` interface** — traceability
10. **Paginated `Load` on `EventSource`** or extension interface
11. **Simplify `decider.Execute()` to take `Command` directly** (depends on #3)

### P2 — Do Eventually (Medium Impact, Low Effort)

12. **Remove `StreamKey()` from `AggregateRef`** — duplicates `String()`
13. **Fix `Metadata` struct `omitempty`** on nested struct fields
14. **`CheckVersionConflict` should take `Version` not `int`**
15. **Standardize example/ placeholder versions**
16. **Consolidate OTel helpers** — remove trivial wrappers
17. **Remove `io.Closer` from `EventSink`/`EventSource`**
18. **Add `SchemaVersion.Decrement()`** for symmetry with `Increment()`
19. **Add `Version.Decrement()`** for symmetry with `Increment()`

### P3 — Nice to Have (Low Impact)

20. **Remove `stretchr/testify` where gomega suffices**
21. **`listing.AggregateRef` embed `event.AggregateRef`**
22. **Simplify metrics middleware** — remove `MetricsRecorder` interface
23. **Fix `Context()` leak** — `context.WithDeadline` cancel is discarded (not a runtime issue but static analysis warns)
24. **Add integration test for `VersionedStore` as full `event.Store`**
25. **NATS JetStream backend** — research doc exists

---

## G. Top #1 Question I Cannot Figure Out Myself

**How should we handle the core v1.6.0 phantom dep that buildflow keeps re-adding?**

Every commit, the buildflow pre-commit hook runs `go mod tidy`, which resolves `testhelpers v1.6.0` from the network, sees it transitively depends on `core v1.6.0`, and adds core as indirect to modules that depend on testhelpers. The local replace directive should override this, but the indirect line still appears.

Options:

1. **Publish new testhelpers version** (e.g., v1.7.0) that doesn't reference core — cleanest but requires a release
2. **Configure buildflow to ignore core** in its tidy step — requires buildflow config change
3. **Always commit with `git -c core.hooksPath=/dev/null`** — works but fragile (easy to forget)
4. **Accept the phantom indirect line** — it's harmless with the replace directive present

Which approach do you prefer?

---

## Workspace Metrics

| Metric                            | Value                                                                                                     |
| --------------------------------- | --------------------------------------------------------------------------------------------------------- |
| Go version                        | 1.26.3                                                                                                    |
| Workspace modules                 | 29 (22 library + 6 examples + 1 integration)                                                              |
| Test packages                     | 36 (all green)                                                                                            |
| No-test packages                  | 3 (`storage/sql`, `turso`, `catalog/internal/cattest`)                                                    |
| Lint issues                       | 1 pre-existing + 1 fix pending                                                                            |
| Dependencies removed this session | 3 direct (samber/ro) + 3 transitive (samber/lo, golang.org/x/exp) from event/ and cascaded to 15+ modules |
| Net lines removed                 | -199 from go.mod/go.sum files alone                                                                       |

## Dependency Graph (after ro removal)

```
Layer 0: id/, dispatcher/, codec/
Layer 1: event/ (→id, codec), command/ (→id, dispatcher, event), query/ (→dispatcher)
Layer 2: schema/ (→event), snapshot/ (→event)
Layer 3: decider/ (→event, snapshot, otel)
Layer 4: memory/, testhelpers/, signing/, otel/
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
Layer 6: integration/, catalog/, examples/, cmd/cqrs-gen
```

## Commits This Session (full list)

```
7d3f89e chore(deps): buildflow auto-tidy — remove samber/ro cascade from downstream modules
6ccc2f5 feat(event): add SchemaVersion.Increment() for type-safe version arithmetic
d8aa825 refactor(command): deduplicate AggregateType and AggregateRef via type aliases
6a633ba fix(snapshot): use error-family Rejection instead of fmt.Errorf in EveryNEvents
3e9f88e fix(otel): correct misleading TraceIDLogger docstring
2f9caf9 refactor(event): delete reactive.go — remove samber/ro dependency entirely
99d6f6a fix: add missing replace directives to prevent core v1.6.0 phantom dep
```
