# Session 85 — Comprehensive Status Report

**Date:** 2026-05-21 02:26 UTC
**Branch:** master
**Commits:** 900 total | 7 this session

---

## Executive Summary

Session 85 is **in progress**. The sentinel conversion (all 57 sentinels across 11 packages) is **COMPLETE**. The error wrap migration (replacing 194 `fmt.Errorf` with structured `event.Wrap*`) is **PARTIALLY DONE** — 6/194 wraps converted in core/event. A `WrapFrom` helper was added to preserve error families when adding context.

The brutal self-review identified that the single highest-value work is structured error wraps — structured sentinels without structured wraps = structured in, garbage out. We are now executing on that.

---

## A. FULLY DONE ✅

### Sentinels (COMPLETE)

- **57/57 sentinels** converted from `errors.New()` to `errorfamily.New*()` with dot-notation codes
- **11 packages** fully converted: core/event, core/command, core/query, core/aggregate, core/decider, projection, storage, middleware, memory, catalog, sync
- **Zero `errors.New` remaining** in all library production code (dispatcher, catalog/id_parse, sync sentinels all done)
- **Zero `init()` blocks** in all library production code

### Wrap Re-exports (COMPLETE)

- `event.Wrap`, `WrapRejection`, `WrapConflict`, `WrapTransient`, `WrapCorruption`, `WrapInfrastructure`
- `event.WrapFrom(err, code, msg)` — preserves classified family via `Classify(err)`
- All available in `core/event/errors.go`

### Simple Wrap Removals (COMPLETE)

- core/event: 6 wraps removed — `ErrNilOutbox`, `ErrNilBus`, `ErrNilCheckpointStore`, `ErrNilProjection`, `ErrDuplicateProjection`, `ErrProjectionPanicked`, `ErrInvalidSnapshotInterval`
- Replaced `fmt.Errorf("%w", Sentinel)` with direct sentinel returns
- Added `.WithContext()` for structured metadata on `ErrDuplicateProjection` and `ErrProjectionPanicked`

---

## B. PARTIALLY DONE 🔶

### Structured Error Wraps (IN PROGRESS)

| Package        | Total Wraps | Converted | Remaining |
| -------------- | ----------- | --------- | --------- |
| core/event     | 22          | 7         | 15        |
| storage        | 69          | 0         | 69        |
| catalog        | 23          | 0         | 23        |
| memory         | 15          | 0         | 15        |
| projection     | 10          | 0         | 10        |
| core/aggregate | 6           | 0         | 6         |
| core/command   | 4           | 0         | 4         |
| core/query     | 4           | 0         | 4         |
| middleware     | 2           | 0         | 2         |
| **TOTAL**      | **155**     | **7**     | **148**   |

_Note: 39 wraps are in test/example code, not counted here._

### Remaining core/event wraps (15):

```
types.go:48     invalid IP address → WrapRejection
codec.go:48     decode payload → WrapCorruption
builder.go:82   build event → WrapFrom
runner.go:86    projection handle → WrapFrom
runner.go:91    checkpoint save → WrapFrom
runner.go:176   handle parallel canceled → WrapInfrastructure
runner.go:198   checkpoint save (parallel) → WrapFrom
publish_helper.go:22   stage outbox → WrapFrom
publish_helper.go:27   publish events → WrapFrom
publish_helper.go:52   save snapshot → WrapFrom
outbox_publisher.go:184   poll pending → WrapFrom
outbox_publisher.go:207   ack entries → WrapFrom
outbox_publisher.go:212   publish events → WrapFrom
codec_batch.go:29   payload marshal (dual %w) → SPECIAL CASE
codec_batch.go:41   create event → WrapFrom
```

### go.mod Updates (PARTIAL)

- `sync/go.mod`: Added `go-error-family` direct dependency ✅
- `catalog/go.mod`: Made `go-error-family` direct dependency ✅
- `core/pkg/dispatcher`: Uses `errorfamily` directly (within core module, already has dep) ✅

---

## C. NOT STARTED ⬜

### Remaining Wrap Packages (148 wraps)

- storage (69) — highest count
- catalog (23)
- memory (15)
- projection (10)
- core/aggregate (6)
- core/command (4)
- core/query (4)
- middleware (2)

### Phase 3: Dead Code Removal

- Deprecate `aggregate` package
- Remove 21 deprecated catalog exports
- Remove `CatalogMeta` from event/command/query
- Remove deprecated `CatalogBuilder`

### Phase 4: Example Unification

- Update example/user to use `catalog.Command[T]()` API
- Align example patterns
- Add catalog exports to examples

### Phase 5: Type Safety

- Brand `OutboxID`
- Brand catalog ID types
- Add `ErrorCode` branded type

---

## D. TOTALLY FUCKED UP 💥

**Nothing.** All 24 test packages pass. Zero lint. Zero build errors.

**Gopls false positives** (stale cache, not real errors):

- `camelCaseToHuman` "undefined" in `message_config.go` — function IS defined in same file
- `TursoInitSchema` "undefined" in `sqlite_bench_test.go` — function IS defined in `turso_connector.go`
- `go-error-family not in go.mod` for catalog/id_parse.go — IS in go.mod

---

## E. WHAT WE SHOULD IMPROVE

1. **Continue wrap migration** — 148 production wraps still use `fmt.Errorf`
2. **The dual-%w case** in `codec_batch.go:29` needs careful handling
3. **storage wraps** (69) are the biggest batch — tackle next
4. **WithContext integration** — add aggregate_id/version context to storage errors
5. **Test coverage for new code paths** — WrapFrom, WithContext

---

## F. TOP 10 NEXT TASKS (sorted by impact/effort)

| #   | Task                                   | Effort | Impact |
| --- | -------------------------------------- | ------ | ------ |
| 1   | Replace 15 remaining core/event wraps  | 30min  | HIGH   |
| 2   | Replace 69 storage wraps               | 90min  | HIGH   |
| 3   | Replace 15 memory wraps                | 20min  | MED    |
| 4   | Replace 10 projection wraps            | 15min  | MED    |
| 5   | Replace 23 catalog wraps               | 30min  | LOW    |
| 6   | Replace 6 core/aggregate wraps         | 10min  | MED    |
| 7   | Replace 4 core/command + 4 query wraps | 15min  | MED    |
| 8   | Add WithContext to storage errors      | 20min  | MED    |
| 9   | Deprecate aggregate package            | 10min  | MED    |
| 10  | Brand OutboxID                         | 20min  | MED    |

---

## G. TOP #1 QUESTION

**How do we handle `fmt.Errorf` with multiple `%w` verbs?**

In `codec_batch.go:29`: `fmt.Errorf("%w: %s: %w", ErrPayloadMarshal, eventType, err)` wraps BOTH a sentinel AND an arbitrary error. `errorfamily.Wrap` only wraps one error. Options:

1. **Wrap `err`**, add `ErrPayloadMarshal` as context → loses `errors.Is(err, ErrPayloadMarshal)`
2. **Wrap `ErrPayloadMarshal`**, set `err` as cause via `WithCause` → `errors.Is` works for sentinel but not for `err`
3. **Create custom multi-cause error** → complex, not supported by errorfamily
4. **Leave as `fmt.Errorf`** → preserves dual-unwrap but loses structured metadata

Current leaning: Option 2 (wrap sentinel, set cause) since `ErrPayloadMarshal` is the classification and `err` is the detail.

---

## Project Metrics

| Metric               | Value            |
| -------------------- | ---------------- |
| Total commits        | 900              |
| Production LOC       | ~15,470          |
| Test LOC             | ~30,431          |
| Test packages        | 24 (all passing) |
| Benchmarks           | 53               |
| Structured sentinels | 57               |
| Converted wraps      | 7/155            |
| Lint issues          | 0                |

---

## Session 85 Commit History

| Commit    | Description                                                               |
| --------- | ------------------------------------------------------------------------- |
| `7d6d2c2` | refactor(event): remove redundant fmt.Errorf wraps around sentinels       |
| `0bde2af` | feat(event): add WrapFrom helper that preserves error family              |
| `6a945c7` | refactor(errors): convert remaining 9 bare sentinels to structured errors |
| `6437ae7` | feat(core/event): add Wrap\* helper functions for errorfamily             |
| `9aca05f` | refactor(dispatcher): migrate sentinels to errorfamily                    |
| `d62e222` | refactor(catalog): migrate ID sentinels to errorfamily                    |
| `367dcb5` | refactor(sync): migrate sentinels to errorfamily                          |
