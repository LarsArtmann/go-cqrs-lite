# Session 95 — Code Deduplication Sweep

**Date:** 2026-05-24 00:38 CEST  
**Branch:** master  
**Status:** All 26 test packages pass, zero lint, 90.2% total coverage

---

## Executive Summary

Ran `art-dupl -t 15 --semantic --sort total-tokens` and systematically deduplicated multi-line clone patterns across 10 files. Extracted 12 new test helpers, replaced ~45 inline patterns, reduced 258 lines of code. Clone groups: **366 → 363**. The small net reduction reflects that threshold 15 catches massive amounts of idiomatic Go noise (single-line assertions, interface stubs) that are architecturally correct and not meaningfully deduplicatable.

---

## A) FULLY DONE

### Previous Session (threshold 22, 141→135 groups)

| File | Helper(s) Extracted | Occurrences |
|------|-------------------|-------------|
| `storage/event_store_test.go` | `saveEvt()`, `appendBatchEvt()` | 10 |
| `storage/transactional_store_test.go` | `saveWithOutboxEvt()`, `expectOutboxInsertSuccess()`, `expectOutboxInsertError()` | 7 |
| `core/decider/decider_test.go` | `counterDecider()`, `newCounterSnapshotRepo()`, `saveSnapshot()`, `setSnapshot()` | 12 |

### This Session (threshold 15, 366→363 groups)

| File | Helper(s) Extracted | Occurrences | Lines Saved |
|------|-------------------|-------------|-------------|
| `middleware/benchmark_test.go` | `benchCommandMiddleware()` | 4 bench functions → 1 line each | ~28 |
| `core/decider/decider_test.go` | `newSnapshotSetup()`, `requireLoadState()` | 10 snapshot + 5 Load+assert | ~65 |
| `projection/runner_test.go` | `registerNoopProjection()` | 5 no-op registration patterns | ~25 |
| `core/aggregate/repository_test.go` | `newTestRootWithEvent()`, `saveUserEvent()` | 6 root+event + 5 save patterns | ~30 |
| `middleware/retry_helpers_test.go` (NEW) | `retryConfigBasic()`, `retryConfigFast()`, `retryConfigSlow()` | 10 config setups across 3 files | ~30 |
| `catalog/validate_test.go` | `expectViolationMessage()` | 5 violation+message assertions | ~30 |

**Net diff: +274 -532 = -258 lines**

### Verification

- ✅ All 26 test packages pass (`go test ./... -count=1`)
- ✅ Zero lint (`nix run .#lint` → 0 issues)
- ✅ Formatted (`nix fmt` → 3 files reformatted)
- ✅ No production code touched — all changes are test-only

---

## B) PARTIALLY DONE

### Production Code Deduplication — Analyzed, All Skipped

Every production-code clone group was reviewed and determined to be architecturally correct:

| Pattern | Why Skip |
|---------|---------|
| Catalog exporter `Option` types | Each package has its own `Option` type — type-incompatible by design |
| Decider `RepositoryOption` methods | Must match generic `Repository[State]` — method signatures are architecturally mandated |
| Todo API handler shapes | Already has `writeError`, `dispatchAndRespond` helpers extracted |
| `storage/dialect.go` `ParseTime` methods | PostgresDialect vs SQLiteDialect differ in types (`*time.Time` vs `*string`) |
| Interface method stubs (`LoadToVersion`, `LoadToTimestamp`) | Go interface implementations — must exist per mock type |
| Middleware `func(...) bool { return true }` | Lambda bodies are correct per-interface, not deduplicatable |

### Test Code — Remaining Actionable Targets Identified But Not Implemented

| Target | File | Count | Why Not Done |
|--------|------|-------|-------------|
| `core/aggregate/aggregate_test.go` version asserts | 9 clones, 3-line span | Low impact (3-line idiom) |
| `projection/builder_test.go` event.New + error check | 5 clones, 7-line span | Different event types/versions per call |
| `sync/conflict_test.go` Conflict struct literals | 5 clones, 6-line span | Different vector clock values per test |
| `example/user/main_test.go` error family asserts | 3 clones, 8-line span | Cross-module, only 3 occurrences |

---

## C) NOT STARTED

| Item | Description | Effort |
|------|------------|--------|
| Threshold 20 analysis | Run at threshold 20 to see if higher-level patterns emerge | Low |
| `example/` module deduplication | `example/user/main_test.go` has error family assertion patterns | Low |
| `integration/` module deduplication | Cross-module test patterns (BDD tests share structure) | Medium |
| Cross-module helper sharing | Can't share test helpers across go.mod boundaries without new module | High |
| Production code pattern unification | Catalog exporters could share an internal option type | Medium |

---

## D) TOTALLY FUCKED UP — Nothing

No regressions, no broken tests, no production code touched. Clean session.

---

## E) What We Should Improve

### 1. The 118-Clone and 82-Clone Groups Are Noise

- **118 clones**: `return nil` in no-op handlers — this is `testhelpers/handlers.go` doing its job. Every handler returns `nil`, `error`, or panics. The "clones" are the entire point of test helpers.
- **82 clones**: `if len(X) != N { t.Fatalf(...) }` — standard Go test assertion. The only way to deduplicate this is to add testify as a dependency (currently only indirect via ginkgo).
- **Recommendation**: Accept these as Go idioms. Art-dupl at threshold 15 is too aggressive for Go test code.

### 2. Art-dupl Threshold Calibration

| Threshold | Groups | Signal/Noise |
|-----------|--------|-------------|
| 22 | 141 | Good signal-to-noise |
| 15 | 366 | ~50% noise (single-line idioms) |
| 10 | ~800+ | Almost all noise |

The previous session at threshold 22 was more productive per-group. Going below 15 would yield diminishing returns.

### 3. Cross-Module Test Helper Sharing

The project has 10+ modules with separate `go.mod` files. Test helpers can only be shared within the same module or via the `testhelpers/` module. This limits deduplication opportunities across module boundaries.

### 4. File Size Compliance

All production files remain under 250 lines. Test files are exempt from this rule per AGENTS.md.

---

## F) Top 25 Things We Should Get Done Next

### High Impact

1. **Raise threshold to 22 and re-run** — Focus on actionable multi-line patterns, not noise
2. **`core/aggregate/aggregate_test.go` version assert helper** — `requireVersion(t, core, expected)` for 8 occurrences
3. **`projection/builder_test.go` event creation helper** — `mustCreateOrderEvent(t, eventType, version, payload)` 
4. **`sync/conflict_test.go` conflict builder** — `newTestConflict(local, remote, localVC, remoteVC)` 
5. **`example/user/main_test.go` requireErrorFamily** — Extract the 3 error family assertion patterns
6. **`integration/aggregate/cqrs_bdd_test.go` event creation** — Shared event factory for BDD tests
7. **Production catalog exporter option unification** — Internal shared option type across asyncapi/d2/eventcatalog/openapi

### Medium Impact

8. **`core/event/batch_test.go` + `integration/event/timetravel_test.go`** — Shared event creation + assert helper (6-line span, 2 modules)
9. **`catalog/eventcatalog/exporter_test.go` + `catalog/asyncapi/exporter_test.go`** — Shared `mustBuildCatalog` pattern
10. **`core/query/dispatcher_test.go` dispatch helper** — `mustDispatchTyped(t, dispatcher, query)` 
11. **`core/event/outbox_publisher_test.go` event+assert pattern** — 3 occurrences of create+publish+assert
12. **`integration/event/event_sourcing_bdd_test.go` BDD helper** — Shared Given/When/Then pattern
13. **`memory/store_test.go` event creation helper** — `createTimedEvent(t, eventType, aggID, version, ts)`
14. **`storage/pebble_event_store_test.go` event batch helper** — `appendBatchWithTimestamps(store, aggID, events, timestamps)`
15. **`core/decider/benchmark_test.go` bench helper** — 3 identical bench setup functions
16. **`catalog/docserver/docserver_test.go` test catalog builder** — 3 similar test catalog constructions

### Lower Impact / Polish

17. **Threshold 20 targeted run** — Run art-dupl at 20, extract remaining 5+ clone groups
18. **`example/todo/aggregate/todo_test.go` event count assert** — 5 identical `if len(events) != N` patterns
19. **`core/pkg/id/id_test.go` parse test helper** — Shared parse+assert pattern (11 occurrences)
20. **`catalog/schema_test.go` field assert helper** — 5 field validation patterns
21. **`middleware/metrics_test.go` assert helper** — 3 metric collection assertion patterns
22. **`projection/runner_test.go` replay lifecycle helper** — 3 remaining replay goroutine patterns
23. **`integration/aggregate/repository_test.go` snapshot repo helper** — 3 snapshot repo creation patterns
24. **`core/event/clock_test.go` clock test helper** — 5 clock-related test patterns
25. **`storage/sqlite_helpers_test.go` setup helper** — Shared SQLite test setup pattern

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should we introduce a shared test assertions module (or add testify as a direct dependency) to eliminate the 82-clone `if len(X) != N { t.Fatalf(...) }` pattern across the entire project?**

Arguments for:
- 82 clones is the single largest clone group
- `require.Len(t, X, N)` is cleaner than `if len(X) != N { t.Fatalf(...) }`
- Testify is already an indirect dependency via ginkgo

Arguments against:
- The project explicitly uses `t.Errorf`/`t.Fatalf` directly (no testify in any test file)
- Gomega is the preferred assertion library for BDD tests
- Adding a new test dependency for just one assertion pattern feels heavyweight
- The 82 "clones" are idiomatic Go — many Go projects consider this the correct style

The alternative is a lightweight `assertLen[T any]` generic helper in `testhelpers/`, but it can't be imported across all modules without updating every `go.mod`.

---

## Clone Group Analysis (threshold 15)

| Metric | Before Session | After Session |
|--------|---------------|---------------|
| Total clone groups | 366 | 363 |
| Top group size | 118 | 111 |
| 82-clone group | 82 | 82 (unchanged — idiomatic Go) |
| Multi-line (≥4 lines) | ~175 | ~173 |
| Single/short-line (<4 lines) | ~191 | ~190 |
| Total lines removed | — | 258 |

## Test Coverage (unchanged)

| Package | Coverage |
|---------|----------|
| `core/query` | 100.0% |
| `core/pkg/dispatcher` | 100.0% |
| `middleware` | 100.0% |
| `catalog/adapters` | 100.0% |
| `catalog/internal/caseutil` | 100.0% |
| `memory` | 99.6% |
| `core/pkg/id` | 98.1% |
| `catalog` | 96.8% |
| `core/aggregate` | 95.9% |
| `catalog/d2` | 95.0% |
| `core/command` | 94.7% |
| `catalog/openapi` | 94.4% |
| `projection` | 94.4% |
| `catalog/asyncapi` | 93.7% |
| `core/decider` | 93.6% |
| `core/event` | 93.8% |
| `catalog/eventcatalog` | 91.3% |
| `catalog/docserver` | 90.1% |
| `storage` | 89.2% |
| `sync` | 90.2% |
| **Total** | **90.2%** |
