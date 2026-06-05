# Module Improvement Plan

**Generated:** 2026-06-05
**Source:** Full audit of all 21 library modules (scc, coverage, lint, API surface, docs, code smells, existing plans)
**Total tasks:** 62 (all ≤12 min each)
**Sorted by:** Impact × Customer Value ÷ Effort (Pareto order)

---

## Module Quality Scorecard

| Module | LoC | Coverage | Lint | doc.go | README | errors.go | Files >250L | Score |
|---|---:|---:|---:|---|---|---|---|---|
| event | 6,658 | 89.4% | ✅ 0 | no | ⚠️ stale | ✅ | 2 (eventtest) | 🟡 |
| catalog | 9,319 | 86.0% | ✅ 0 | ✅ | ✅ | no | 3 | 🟡 |
| storage | 4,538 | 89.3% | ✅ 0 | ✅ | ✅ | no | 0 | 🟢 |
| signing | 3,123 | 94.1% | ✅ 0 | ✅ | ✅ | ✅ | 0 | 🟢 |
| integration | 3,024 | — | ✅ 0 | no | no | no | 0 | 🟡 |
| middleware | 3,033 | 98.5% | ✅ 0 | no | no | ✅ | 0 | 🟢 |
| projection | 2,634 | 90.5% | ✅ 0 | no | no | ✅ | 0 | 🟢 |
| memory | 2,568 | 99.1% | ✅ 0 | ✅ | no | ✅ | 0 | 🟢 |
| decider | 1,981 | 100.0% | ✅ 0 | no | no | ✅ | 0 | 🟢 |
| pebble | 1,339 | 86.6% | ✅ 0 | ✅ | no | ✅ | 0 | 🟢 |
| command | 1,313 | 93.8% | ✅ 0 | no | no | ✅ | 0 | 🟡 |
| id | 1,129 | 94.5% | ✅ 0 | no | no | no | 0 | 🟡 |
| listing | 1,099 | 94.9% | ✅ 0 | ✅ | ✅ | no | 0 | 🟢 |
| query | 827 | 94.3% | ✅ 0 | no | no | ✅ | 0 | 🟡 |
| schema | 798 | 89.7% | ✅ 0 | no | no | no | 0 | 🟡 |
| watermill | 752 | 92.6% | ✅ 0 | no | no | no | 2 | 🟡 |
| turso | 553 | 28.6% | ✅ 0 | ✅ | no | ✅ | 0 | 🔴 |
| otel | 403 | 96.4% | ✅ 0 | ✅ | no | no | 0 | 🟢 |
| snapshot | 335 | 92.3% | ✅ 0 | no | no | no | 0 | 🟡 |
| dispatcher | 433 | 100.0% | ✅ 0 | no | no | ✅ | 0 | 🟢 |
| codec | 268 | 93.3% | ✅ 0 | no | no | ✅ | 0 | 🟢 |

**Legend:** 🟢 good-to-excellent | 🟡 needs polish | 🔴 needs work

---

## Key Findings Summary

### What's already great
- **Zero lint issues** across ALL 21 modules
- **All modules compile and pass tests** (no red tests)
- **High coverage** — 18/21 modules above 85%, most above 90%
- **Error taxonomy** — 5-family system well-adopted in most modules
- **Clean module graph** — layered dependency structure with no cycles

### What needs improvement

#### 🔴 CRITICAL (Production correctness)
1. **turso: 28.6% coverage** — only 15 basic tests, missing edge cases for sync, error paths, concurrent access
2. **event/README.md stale** — references `go-cqrs-lite/core` (the old monolith), not `event/v2`
3. **storage/sql: 0% coverage, no test files** — shared Dialect + helpers used by all SQL backends
4. **Long functions**: `watermill/protocol.go:messageToEvent` (86L), `storage/sql_aggregate_reader.go:ListWithStatus` (115L)

#### 🟠 HIGH (Consumer-facing quality)
5. **7 modules lack doc.go**: command, query, decider, id, schema, projection, watermill — pkg.go.dev shows no package description
6. **13 modules lack README.md**: command, query, decider, id, schema, projection, memory, middleware, otel, pebble, watermill, codec, turso
7. **8 modules lack errors.go**: id, schema, snapshot, catalog, storage, otel, watermill, listing — errors scattered across files
8. **`io.Closer` embedded in 9 core interfaces** — forces consumers to implement Close() even when not needed
9. **ErrDispatcherClosed duplicated** across command, query, dispatcher (same name, same type, different packages)

#### 🟡 MEDIUM (Code health)
10. **3 files exceed 250-line limit**: `catalog/schema/reflect.go` (281), `catalog/internal/cattest/builders.go` (377 — test helper), `event/eventtest/fake_store.go` (273)
11. **25+ functions exceed 30-line limit** — mostly in storage, signing/multisig, watermill, catalog
12. **`any` used 40+ times in catalog** — expected for generic schema types, but some could use typed alternatives
13. **event/eventtest shows 18.4% coverage** — false alarm: test helpers are tested by their consumers, not internally

#### 🟢 LOW (Polish)
14. **event README badge points to old `core` package** — cosmetic but confuses new consumers
15. **No example demonstrates watermill or schema modules**
16. **integration/ has no doc.go or README**

---

## Execution Plan — 62 Tasks

### Phase 1: Critical Correctness (6 tasks, ~60 min)

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 1 | Add turso tests: Save+Load edge cases, AppendBatch, LoadFromVersion, ReadAll, concurrent access | turso | 12m | 28.6% → ~80% coverage, only basic happy-path tested |
| 2 | Add turso tests: error paths (nil DB, closed store, invalid args) + InitSchema idempotency | turso | 10m | Error paths completely untested |
| 3 | Add turso tests: SnapshotStore + CheckpointStore full coverage | turso | 10m | Only SaveAndLoad tested, missing version/empty/error cases |
| 4 | Fix event/README.md: replace all `go-cqrs-lite/core` references with `/event/v2` | event | 5m | Stale import path misleads consumers |
| 5 | Add storage/sql test file: test Dialect implementations (Placeholder, FormatTime, ScanTimeDest, ParseTime, schemas) | storage | 12m | 0% coverage on shared code used by all SQL backends |
| 6 | Decompose `storage/sql_aggregate_reader.go:ListWithStatus` (115L → 3 funcs: buildQuery, scanRows, buildResult) | storage | 12m | Longest function in codebase, hard to review |

### Phase 2: Package Documentation (13 tasks, ~65 min)

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 7 | Add `command/doc.go` — describe dispatch, handlers, middleware, typed handlers | command | 5m | No package docs on pkg.go.dev |
| 8 | Add `query/doc.go` — describe dispatch, typed dispatch, pagination | query | 5m | No package docs on pkg.go.dev |
| 9 | Add `decider/doc.go` — describe pure-function pattern, Repository, Execute/Load | decider | 5m | Core pattern, must have docs |
| 10 | Add `id/doc.go` — describe branded IDs, Of[T], built-in types, custom markers | id | 5m | Key differentiator module |
| 11 | Add `schema/doc.go` — describe upcasting, VersionedStore, schema evolution | schema | 5m | No docs at all currently |
| 12 | Add `projection/doc.go` — describe Runner, replay+live, checkpoints, dead letters | projection | 5m | Complex module needs overview |
| 13 | Add `watermill/doc.go` — describe protocol adapter, publisher/subscriber adapters | watermill | 5m | No docs at all currently |
| 14 | Add `snapshot/doc.go` — describe SnapshotStore, strategies, codec integration | snapshot | 5m | No docs at all currently |
| 15 | Add `memory/README.md` — testing-only disclaimer, list implementations, quick example | memory | 10m | Used by every consumer but no README |
| 16 | Add `middleware/README.md` — table of 24 middleware factories, quick examples | middleware | 10m | Most consumer-facing module after event |
| 17 | Add `storage/README.md` update: add sql/ subpackage section, Dialect docs | storage | 5m | sql/ subpackage not documented |
| 18 | Add `turso/README.md` — Open/OpenInMemory/OpenSync, constructor reference | turso | 5m | No README for a production module |
| 19 | Add `codec/README.md` — JSONCodec, RawCodec, Codec interface | codec | 5m | Simple module, easy docs |

### Phase 3: Error Hygiene (8 tasks, ~45 min)

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 20 | Create `id/errors.go` — consolidate 4 fmt.Errorf calls into named sentinels | id | 5m | Errors scattered across id.go, aggregate_id.go |
| 21 | Create `schema/errors.go` — consolidate 6 fmt.Errorf calls | schema | 5m | 6 error sites, no central file |
| 22 | Create `snapshot/errors.go` — consolidate 4 fmt.Errorf calls | snapshot | 5m | Missing convention |
| 23 | Create `catalog/errors.go` — consolidate 31 fmt.Errorf calls into named sentinels | catalog | 12m | Largest module, most error sites |
| 24 | Create `storage/errors.go` — re-export sql/errors.go + consolidate storage-level errors | storage | 5m | Errors in sql/ but not storage/ root |
| 25 | Create `watermill/errors.go` — consolidate 21 fmt.Errorf calls | watermill | 10m | Second most error-dense module |
| 26 | Create `listing/errors.go` — consolidate 4 fmt.Errorf calls | listing | 5m | Quick win |
| 27 | Rename `command.ErrHandlerNotFound` → don't (it's different from dispatcher/query versions) — add doc comments explaining the distinction | command, query, dispatcher | 5m | ErrHandlerNotFound x3 is confusing but intentional |

### Phase 4: Function Decomposition (8 tasks, ~80 min)

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 28 | Decompose `watermill/protocol.go:messageToEvent` (86L → 4 funcs: parseHeaders, parseTimestamp, buildMetadata, constructEvent) | watermill | 12m | Longest function outside storage |
| 29 | Decompose `watermill/protocol.go:buildMetadata` (54L → 2 funcs) | watermill | 8m | Second longest in watermill |
| 30 | Decompose `storage/event_store.go:Save` (55L → 2 funcs: validateVersion, persistEvents) | storage | 10m | Core write path, hard to review |
| 31 | Decompose `storage/event_store.go:AppendBatch` (46L → 2 funcs) | storage | 8m | Batch write path |
| 32 | Decompose `storage/event_store_global.go:ReadFrom` (59L → 2 funcs: buildQuery, scanResults) | storage | 10m | Projection-critical path |
| 33 | Decompose `signing/multisig/middleware.go:RequireMultiSigMiddleware` (55L → 2 funcs) | signing | 8m | Complex verification logic |
| 34 | Decompose `signing/payload.go:canonicalPayload` (41L → 2 funcs: sortKeys, encodePayload) | signing | 8m | Core signing logic |
| 35 | Decompose `catalog/schema/reflect.go:FromType` (281L file → split into reflect.go + reflect_struct.go + reflect_primitive.go) | catalog | 12m | Only file >250L in production code |

### Phase 5: Coverage Gaps (6 tasks, ~55 min)

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 36 | Add catalog/asyncapi edge-case tests: empty registry, nil schemas, concurrent builds | catalog | 10m | AsyncAPI is least-tested exporter |
| 37 | Add catalog/d2 edge-case tests: empty services, circular deps, missing messages | catalog | 10m | D2 exporter has fewer tests than others |
| 38 | Add pebble edge-case tests: concurrent Save, Close-then-access, large batches | pebble | 10m | 86.6% → 90%+ |
| 39 | Add storage/sql/helpers.go tests: SharedInsertEvents, SharedCheckpointLoad, SharedEventLoad error paths | storage | 10m | Shared SQL helpers only tested via integration |
| 40 | Add event/codec tests: decode malformed JSON, nil payload, encoding roundtrip | event | 8m | Codec is critical infrastructure |
| 41 | Add integration/ signing cross-module tests: HMAC sign→verify→middleware chain | integration | 10m | Signing middleware integration gaps |

### Phase 6: io.Closer Removal Design (3 tasks, ~30 min)

> These are DESIGN tasks only — actual removal is a v3 breaking change.

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 42 | Write ADR "Remove io.Closer from core interfaces" — propose `Lifecycle` interface with `Close() error` as standalone, not embedded | event | 10m | 9 interfaces force Close() on consumers |
| 43 | Write ADR "Unify ErrDispatcherClosed" — propose single sentinel in dispatcher/ re-exported by command/ and query/ | dispatcher | 10m | 3 copies of same error |
| 44 | Write ADR "Split catalog into catalog/core + per-exporter modules" — propose the 5-module split from earlier analysis | catalog | 10m | 9,319 LoC in single module |

### Phase 7: Code Quality Polish (10 tasks, ~60 min)

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 45 | Move `catalog/internal/cattest/builders.go` test helpers to `catalog/internal/cattest/testdata_builder.go` pattern (split 377L → 2 files) | catalog | 5m | Only test file >250L |
| 46 | Split `event/eventtest/fake_store.go` (273L → fake_store.go 180L + fake_store_helpers.go 90L) | event | 5m | Slightly over limit |
| 47 | Add `// Deprecated: Use decider instead` to aggregate package doc.go (if not already) | aggregate | 3m | Formal deprecation trail |
| 48 | Add example_test.go to `schema/` — show Upcaster + VersionedStore usage | schema | 10m | No example for schema evolution |
| 49 | Add example_test.go to `watermill/` — show basic publisher/subscriber adapter usage | watermill | 10m | No example exists |
| 50 | Add example_test.go to `projection/` — show Runner with Builder pattern | projection | 10m | Complex API, example helps adoption |
| 51 | Add `integration/README.md` — explain what cross-module tests exist and how to run them | integration | 3m | No docs for integration module |
| 52 | Verify all 21 modules have consistent `go 1.26.3` in go.mod | all | 3m | Consistency check |
| 53 | Run `go mod tidy` on all 21 modules and verify no changes needed | all | 5m | Housekeeping |
| 54 | Add `t.Parallel()` to turso tests (currently missing) | turso | 3m | Convention compliance |

### Phase 8: Consumer Experience (8 tasks, ~45 min)

| # | Task | Module | Est | Why |
|---|---|---|---|---|
| 55 | Add quickstart example to `event/doc.go` or new `event/example_test.go` — NewEvent + Save + Load roundtrip | event | 8m | Most imported module, needs copy-paste example |
| 56 | Add quickstart example to `command/example_test.go` — New + Register + Dispatch | command | 5m | No example_test.go |
| 57 | Add quickstart example to `query/example_test.go` — New + Register + DispatchTyped | query | 5m | No example_test.go |
| 58 | Add quickstart example to `decider/example_test.go` — Decider + Repository + Execute | decider | 8m | Core pattern example |
| 59 | Add quickstart example to `id/example_test.go` — Of[T] custom branded type + New + Parse | id | 5m | Key differentiator, needs example |
| 60 | Add quickstart example to `signing/example_test.go` — HMAC sign + verify + middleware | signing | 8m | Security feature needs clear example |
| 61 | Update `storage/README.md` — add Turso section, update quickstart with v2 import paths | storage | 3m | Turso section missing |
| 62 | Add 1-line `README.md` to each of the 6 example/ dirs explaining what it demonstrates | example | 3m | Examples have no README |

---

## Summary

| Phase | Theme | Tasks | Est | Priority |
|---|---|---|---|---|
| 1 | Critical Correctness | 6 | 60m | 🔴 DO NOW |
| 2 | Package Documentation | 13 | 65m | 🟠 HIGH |
| 3 | Error Hygiene | 8 | 45m | 🟠 HIGH |
| 4 | Function Decomposition | 8 | 80m | 🟡 MEDIUM |
| 5 | Coverage Gaps | 6 | 55m | 🟡 MEDIUM |
| 6 | io.Closer / Architecture ADRs | 3 | 30m | 🟡 MEDIUM (design only) |
| 7 | Code Quality Polish | 10 | 60m | 🟢 LOW |
| 8 | Consumer Experience | 8 | 45m | 🟠 HIGH (adoption) |
| **TOTAL** | | **62** | **~440m** | |

## Critical Path (80/20 — highest impact, do first)

```
Phase 1 (correctness) → Phase 2 (docs) → Phase 8 (examples) → Phase 3 (errors)
```

Phase 1 fixes real gaps (turso coverage, storage/sql tests, stale docs).
Phase 2 makes every module discoverable on pkg.go.dev.
Phase 8 gives consumers copy-paste starting points.
Phase 3 brings error convention consistency.

Phases 4–7 can run in any order and are safe to defer.
