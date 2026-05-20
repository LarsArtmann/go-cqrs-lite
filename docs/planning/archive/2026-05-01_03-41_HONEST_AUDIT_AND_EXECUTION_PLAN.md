# Session 18 — Honest Audit & Execution Plan

**Date:** 2026-05-01 03:41  
**Branch:** master  
**Working tree:** Clean

---

## 0. CRITICAL CONTEXT: This Is a Library/SDK

**go-cqrs-lite is NOT an application.** It is a composable Go library that consumers import into their own projects.

This changes EVERYTHING about how we judge "ghost code":

| Application Lens (WRONG)              | Library/SDK Lens (CORRECT)                                     |
| ------------------------------------- | -------------------------------------------------------------- |
| "Zero internal consumers = dead code" | "Zero internal consumers = correct isolation"                  |
| "Module needs a service that uses it" | "Module needs tests + API stability, not an internal consumer" |
| "example/ should drive real traffic"  | "example/ is a usage demo, not a deployment"                   |
| "Coverage across modules matters"     | "Per-module coverage matters — consumers import individually"  |
| "Unused exports are waste"            | "Public API surface IS the product"                            |

**The real question for every module is: "Would a consumer trust this enough to import it?"**

---

## 1. BRUTALLY HONEST SELF-REFLECTION

### What did we forget?

- **FakeStore/MemoryStore key separator mismatch** — `testhelpers/fakes.go` uses `"/"`, `memory/helpers.go` uses `":"`. Same concept, different separators. A bomb waiting to go off.
- **`storage/` module has zero tests** — 346 lines of SQL code with no test verification. Consumers cannot trust it.
- **`example/user/` has no tests** — 278-line example that generates files on disk, zero test verification.
- **File size violations** — `storage/event_store.go` (346), `core/aggregate/repository.go` (268), `example/user/main.go` (278) all exceed 250-line limit.

### What is NOT a problem (library mindset)

- **"Zero consumers" for storage/** — Consumers import `storage/` into THEIR projects. No internal consumer needed.
- **"Exported but unused internally"** — Public API surface IS the product. `ContextEnricher`, `ProjectionRunner` etc. are consumer-facing APIs.
- **"example/catalog/ was a stale binary"** — Already cleaned up. Not a design issue.

### What IS actually stupid?

- **Shipping `storage/` without tests.** 346 lines of SQL, zero test coverage. A consumer importing this has NO verification it works. That's a trust failure, not a "ghost" failure.
- **`CatalogBuilder` ≈ `Registry`** — Two accumulators for the same `Catalog` type. This is a genuine split brain — consumers see two ways to build a catalog and neither is clearly canonical.
- **Key separator mismatch** — `FakeStore` uses `"/"`, `MemoryStore` uses `":"`. A consumer testing with FakeStore then deploying with a real store WILL hit this.
- **JSON v1/v2 split in storage metadata** — `json:"correlationId"` (v1 tags) in storage but `go-json-experiment/json` (v2) everywhere else. Silent data corruption risk.

### Did we lie?

- **Yes.** FEATURES.md and AGENTS.md imply `storage/` is "Done" when it has zero tests. For a library, untested code = unverified code = not done.
- **Yes.** Coverage reports average across modules, masking storage at 0%.
- **No.** "Zero consumers" is NOT a lie — it's correct library architecture.

### How can we be less stupid?

- **Every module must have tests before it's called "Done".** No exceptions.
- **Consolidate CatalogBuilder into Registry** — one canonical way to build catalogs.
- **Unify key separators** — `streamKey` should be defined once, shared across all implementations.
- **Judge modules by: "Would a consumer trust this?"** not "Does another module use it?"

---

## 2. ARCHITECTURAL DECISIONS CAUSING PROBLEMS

| Decision                                 | Problem                                                   | Fix                                            |
| ---------------------------------------- | --------------------------------------------------------- | ---------------------------------------------- |
| `CatalogBuilder` parallel to `Registry`  | Split brain — consumers see two ways to do the same thing | Consolidate: `CatalogBuilder` wraps `Registry` |
| FakeStore key `"/"` vs MemoryStore `":"` | Inconsistent behavior across store implementations        | Extract shared `streamKey`                     |
| Storage committed without tests          | Consumers cannot trust the SQL event store                | Write tests — this is the #1 priority          |
| JSON v1 tags in storage metadata         | Silent corruption risk when v2-only tags are added        | Migrate to v2                                  |
| File size violations                     | Hard to navigate, review                                  | Split files under 250 lines                    |

---

## 3. EXECUTION PLAN — Phase 1: Make Modules Trustworthy

Sorted by consumer trust impact (highest first):

| #     | Task                                                                                                 | Consumer Trust Impact | Effort | Why                                                                    |
| ----- | ---------------------------------------------------------------------------------------------------- | --------------------- | ------ | ---------------------------------------------------------------------- |
| P1-1  | **Write storage module tests** (Save, Load, LoadFromVersion, Delete, scanEvents, metadata roundtrip) | CRITICAL              | 90min  | Consumers importing `storage/` need verified SQL behavior              |
| P1-2  | **Fix FakeStore/MemoryStore key separator**                                                          | HIGH                  | 30min  | Consumers testing with FakeStore must get same behavior as MemoryStore |
| P1-3  | **Fix JSON v1/v2 split in storage metadata**                                                         | HIGH                  | 15min  | Consumers using v2 JSON must not get silent data loss                  |
| P1-4  | **Add `storage` to flake.nix CI matrix**                                                             | HIGH                  | 5min   | Consumers deserve CI-verified releases                                 |
| P1-5  | **Add `InMemoryRunner` + `UpcasterRegistry` unit tests to `core/event/`**                            | HIGH                  | 60min  | Consumers importing projections need verified behavior                 |
| P1-6  | **Consolidate `CatalogBuilder` on top of `Registry`**                                                | HIGH                  | 90min  | Consumers need ONE canonical way to build catalogs                     |
| P1-7  | **Split `storage/event_store.go` under 250 lines**                                                   | MEDIUM                | 30min  | File size convention                                                   |
| P1-8  | **Split `core/aggregate/repository.go` under 250 lines**                                             | MEDIUM                | 30min  | File size convention                                                   |
| P1-9  | **Add `example/user/main_test.go` smoke test**                                                       | MEDIUM                | 45min  | Example must verify it actually works                                  |
| P1-10 | **Review exported API surface** — keep what's consumer-facing, unexport internals                    | LOW                   | 30min  | Clean API = trustworthy API                                            |
| P1-11 | **Update FEATURES.md** — mark storage as untested                                                    | LOW                   | 10min  | Honest documentation                                                   |
| P1-12 | **Update AGENTS.md** — fix module count, coverage, library context                                   | LOW                   | 15min  | Accurate documentation                                                 |

**Total estimated: ~7 hours**

---

## 4. EXECUTION PLAN — Phase 2: Micro-tasks (≤12 min each)

| #    | Task                                                                   | Impact   | Effort | Depends On |
| ---- | ---------------------------------------------------------------------- | -------- | ------ | ---------- |
| M-1  | Add shared `streamKey` helper to `memory/helpers.go` (export it)       | HIGH     | 5min   | —          |
| M-2  | Update `FakeStore` in `testhelpers/fakes.go` to use `:` separator      | HIGH     | 5min   | —          |
| M-3  | Verify all memory + testhelpers tests pass after separator fix         | HIGH     | 5min   | M-1, M-2   |
| M-4  | Add `go-sqlmock` dep to `storage/go.mod`                               | CRITICAL | 3min   | —          |
| M-5  | Write `TestSQLEventStore_Save_Success`                                 | CRITICAL | 10min  | M-4        |
| M-6  | Write `TestSQLEventStore_Save_ConcurrencyConflict`                     | CRITICAL | 8min   | M-4        |
| M-7  | Write `TestSQLEventStore_Save_EmptyEvents`                             | CRITICAL | 3min   | M-4        |
| M-8  | Write `TestSQLEventStore_AppendBatch_Success`                          | CRITICAL | 8min   | M-4        |
| M-9  | Write `TestSQLEventStore_Load_Success`                                 | CRITICAL | 8min   | M-4        |
| M-10 | Write `TestSQLEventStore_Load_NotFound`                                | CRITICAL | 5min   | M-4        |
| M-11 | Write `TestSQLEventStore_LoadFromVersion`                              | CRITICAL | 8min   | M-4        |
| M-12 | Write `TestSQLEventStore_Delete`                                       | CRITICAL | 5min   | M-4        |
| M-13 | Write `TestSQLEventStore_Close`                                        | CRITICAL | 3min   | M-4        |
| M-14 | Write `TestMarshalMetadata_Nil` + `TestMarshalMetadata_Full`           | HIGH     | 8min   | M-4        |
| M-15 | Write `TestScanEvents_MetadataRoundtrip`                               | CRITICAL | 10min  | M-4        |
| M-16 | Fix JSON v1→v2 in marshalMetadata + scanEvents                         | HIGH     | 8min   | —          |
| M-17 | Write `InMemoryRunner.Register` nil + duplicate tests                  | HIGH     | 10min  | —          |
| M-18 | Write `InMemoryRunner.Handle` subscribe filter + checkpoint tests      | HIGH     | 10min  | M-17       |
| M-19 | Write `subscribesTo` unit test (nil/empty/specific event types)        | MEDIUM   | 10min  | —          |
| M-20 | Extract snapshot logic from `repository.go` → `repository_snapshot.go` | MEDIUM   | 10min  | —          |
| M-21 | Extract load logic from `repository.go` → `repository_load.go`         | MEDIUM   | 10min  | M-20       |
| M-22 | Extract SQL helpers from `storage/event_store.go` → `helpers.go`       | MEDIUM   | 10min  | —          |
| M-23 | Create `example/user/main_test.go` with smoke test                     | MEDIUM   | 12min  | —          |
| M-24 | Update `FEATURES.md` storage section                                   | LOW      | 5min   | M-5–M-15   |
| M-25 | Update `AGENTS.md` coverage table with current numbers                 | LOW      | 5min   | M-17–M-19  |
| M-26 | Run full test suite + lint + race                                      | HIGH     | 5min   | all        |

**Total micro-tasks: 26**  
**Estimated total: ~4 hours**

---

## 5. WHAT WE SHOULD NOT DO (Anti-Scope-Creep)

| Item                                     | Why Skip                                                                          |
| ---------------------------------------- | --------------------------------------------------------------------------------- |
| Watermill module                         | No consumer has asked for it. Storage tests first.                                |
| SQL Snapshot Store                       | Storage isn't even tested yet.                                                    |
| Circuit breaker / DLQ                    | Not needed until consumers report production issues.                              |
| HTTP handler examples                    | example/user is sufficient for a library.                                         |
| Migration CLI                            | Consumers write their own migrations.                                             |
| Documentation site                       | README + godoc is correct for a Go library.                                       |
| Remove `ProjectionRunner` interface      | It's public API — consumers may implement it. Breaking change for no gain.        |
| Unexport `ContextEnricher`/`EnrichEvent` | They're consumer-facing APIs. If nobody uses them YET, that's fine for a library. |

---

## 6. CONSUMER VALUE ANALYSIS

| Task                  | Consumer Value                                                               |
| --------------------- | ---------------------------------------------------------------------------- |
| Storage tests         | **Trust** — consumers can import `storage/` knowing SQL behavior is verified |
| Key separator fix     | **Consistency** — FakeStore and MemoryStore behave identically               |
| JSON v1/v2 fix        | **Safety** — no silent data corruption when consumers use v2 JSON            |
| CI inclusion          | **Reliability** — regressions caught before consumers hit them               |
| Coverage recovery     | **Confidence** — high coverage = safer to depend on                          |
| Catalog consolidation | **Clarity** — one canonical way to build catalogs                            |
| Example tests         | **Documentation** — verified example consumers can copy                      |
| Honest FEATURES.md    | **Trust** — don't oversell, be honest about what's tested vs. not            |
