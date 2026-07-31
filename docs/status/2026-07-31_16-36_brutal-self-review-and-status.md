# Brutal Self-Review & Status Report — Metaengine Engine Sophistication

**Date:** 2026-07-31 16:36  
**Scope:** Review of the previous session's work (Pebble RawValueReader + RawScanReader + ADT matrix extraction) and this session's diagnostic/audit pass  
**Previous report:** `2026-07-31_12-40_metaengine-engine-sophistication-comprehensive-status.md`

---

## A) FULLY DONE (Previous Session — Verified This Session)

| #   | Item                                            | Verification                                    |
| --- | ----------------------------------------------- | ----------------------------------------------- |
| 1   | Pebble `GetRawValue` (RawValueReader)           | ✅ Builds, tests pass, 2.7x benchmark confirmed |
| 2   | Pebble `ScanRawValues` (RawScanReader)          | ✅ Builds, tests pass, 4.5x benchmark confirmed |
| 3   | `exported_helpers.go` — 4 wrappers              | ✅ Builds, compiles                             |
| 4   | 12 unit tests in `raw_reader_test.go`           | ✅ All pass with `-race`                        |
| 5   | 6 benchmarks in `raw_reader_bench_test.go`      | ✅ Run successfully                             |
| 6   | `adttest/harness.go` — 480-line test harness    | ✅ Builds, tests pass transitively              |
| 7   | `adt_matrix_test.go` refactored (metaengine)    | ✅ 27 lines, delegates to harness               |
| 8   | `adt_matrix_test.go` (pebbleengine)             | ✅ 39 lines, memory↔pebble parity               |
| 9   | `sse.go` unused `strconv` import fix            | ✅ Verified: `strconv` not in sse.go imports    |
| 10  | `advanced.go` P6-1 comment update               | ✅ Says "implemented"                           |
| 11  | AGENTS.md updates (module list, test cmd, tree) | ✅ doc-check passes: "All 927 references valid" |
| 12  | api-stability golden                            | ✅ "API surface OK: 2906 exports verified"      |
| 13  | `gofumpt` / `goimports` formatting              | ✅ All files clean (no formatting needed)       |

---

## B) PARTIALLY DONE

| #   | Item                                        | What's Done                                             | What's Missing                                                                                     |
| --- | ------------------------------------------- | ------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| 1   | Test coverage for `ScanRawValues`           | 12 tests cover basic paths                              | Missing: FilterIn, FilterNe/Lt/Le/Gt/Ge, cursor+desc, empty collection, all-filtered-out, limit=1  |
| 2   | `adttest` harness                           | `RunMatrix`, `Scenarios`, `Factory` all work            | `Factory.Supports` field is dead code (never checked), no harness self-tests, no ADR               |
| 3   | `V1StabilizationChecklist` in `advanced.go` | Says "RawValueReader + RawScanReader interfaces stable" | Doesn't note WHICH engines implement them (SQLite + Pebble)                                        |
| 4   | `ContractSuite` in `advanced.go`            | Exists and works for single-engine contract testing     | Not called from pebbleengine tests; no documentation explaining ContractSuite vs adttest.RunMatrix |

---

## C) NOT STARTED

| #   | Item                                                    | Priority | Effort  | Notes                                                                                      |
| --- | ------------------------------------------------------- | -------- | ------- | ------------------------------------------------------------------------------------------ |
| 1   | Pebble LayoutPlanner (prefixed key ranges)              | 🔥 HIGH  | MED     | LayoutPlanner interface defined in engine.go:177; Pebble LSM supports key-prefix iteration |
| 2   | Postgres engine (native JSONB operators)                | MED      | HIGH    | New `metaengine/postgresengine` module                                                     |
| 3   | DuckDB analytical engine (columnar OLAP)                | LOW      | HIGH    | New `metaengine/duckdbengine` module (CGo)                                                 |
| 4   | Soak test (10M events, memory profiling)                | MED      | MED     | Long-running test harness with `runtime.MemStats`                                          |
| 5   | Chaos testing (random tx kills, error injection)        | LOW      | MED     | `FaultEngine` wrapper                                                                      |
| 6   | `metaengine-gen` code generator                         | LOW      | HIGH    | New `cmd/metaengine-gen` tool                                                              |
| 7   | Schema enforcement at `Plan()` time                     | MED      | MED     | Validate fold return types match R                                                         |
| 8   | Pebble `StreamScan` (iter.Seq2)                         | MED      | LOW     | StreamScan interface exists in engine.go:164                                               |
| 9   | Pebble `PushdownScan` hybrid                            | LOW      | MED     | Prefix scan for primary filter + Go-filter for secondary                                   |
| 10  | Pebble counter optimization (summary key)               | LOW      | LOW     | CounterGet is O(N) prefix scan                                                             |
| 11  | Property-based cross-engine parity (rapid)              | MED      | MED     | pgregory.net/rapid is already a test dep                                                   |
| 12  | Fuzz test for ScanRawValues                             | LOW      | LOW     | Fuzz filter/sort/cursor/limit parameters                                                   |
| 13  | Benchmark with filters                                  | LOW      | TRIVIAL | Current bench uses no filters                                                              |
| 14  | Benchmark with 10K/100K items                           | LOW      | TRIVIAL | Current bench uses 100 items                                                               |
| 15  | Three-way ADT matrix (memory+sqlite+pebble in one test) | MED      | LOW     | Currently transitive only                                                                  |
| 16  | ADR for adttest extraction                              | LOW      | TRIVIAL | Document the decision                                                                      |
| 17  | ADR for Pebble raw readers                              | LOW      | TRIVIAL | Document JSON tax approach + trade-off                                                     |
| 18  | Update TODO_LIST.md                                     | LOW      | TRIVIAL | Mark 🔥 items done                                                                         |
| 19  | Update FEATURES.md                                      | LOW      | TRIVIAL | Add raw readers to feature inventory                                                       |
| 20  | Update SKILL.md                                         | LOW      | TRIVIAL | Mention Pebble raw reader support                                                          |
| 21  | Engine `Capabilities()` method                          | MED      | MED     | Advertise optional interfaces without type assertions                                      |
| 22  | Pebble `MapUpdate` with raw reader                      | LOW      | LOW     | Use raw bytes for the read path                                                            |

---

## D) TOTALLY FUCKED UP

| #   | Issue                                                                | Severity | Details                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| --- | -------------------------------------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **ScanRawValues triple-decode**                                      | 🔴 HIGH  | When filters + sort + cursor are all present, each value is JSON-decoded UP TO 3 TIMES: once for filter evaluation, once for sort comparison, once for cursor comparison. This defeats the entire purpose of "raw reader" — the raw path is supposed to ELIMINATE the JSON tax, not make it worse for complex queries. A "decode-once" approach would decode each value to `any` once, evaluate filter + sort + cursor using the decoded value, then return the raw bytes. This halves the decode count for filter+sort scans and reduces it by 3x for filter+sort+cursor scans. |
| 2   | **Sort+paginate logic duplicated between MapScan and ScanRawValues** | 🟡 MED   | `pebbleengine/engine.go:320-352` (MapScan) and `raw_reader.go:97-157` (ScanRawValues) implement nearly identical sort+cursor+limit logic. The only difference is the value type (`any` vs `[]byte`). This is a split brain — a bug fix in one path won't propagate to the other.                                                                                                                                                                                                                                                                                                 |
| 3   | **`adttest.Scenarios()` has dead code**                              | 🟡 LOW   | Line 112-113: `ctx := context.Background(); _ = ctx` — the `ctx` is created, ignored, and each scenario's Setup/Read functions create their own `ctx` from the outer `RunMatrix` call. Pure dead code.                                                                                                                                                                                                                                                                                                                                                                           |
| 4   | **`Factory.Supports` field is dead**                                 | 🟡 LOW   | The `Supports func(metaengine.Engine) bool` field in `adttest.Factory` is declared but NEVER checked in `RunMatrix`. The harness always runs all scenarios against all factories, panicking if an engine doesn't implement a backend interface (via unchecked type assertion `eng.(metaengine.MapBackend)`). This is an honest API lie — the field suggests capability filtering exists, but it doesn't.                                                                                                                                                                         |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture & Type Model

1. **Decode-once in ScanRawValues** — The most impactful fix. Currently: filter decodes to `any`, sort decodes to `any` AGAIN, cursor decodes to `any` AGAIN. Fix: decode each value once, store the decoded `any` alongside the raw bytes, use the decoded value for filter+sort+cursor, return the raw bytes. O(N) decodes instead of O(3N).

2. **Extract `sortAndPaginate` helper** — The sort+cursor+limit logic in both `MapScan` and `ScanRawValues` is structurally identical. Extract a generic helper that takes a `sortFunc` and returns the paginated slice. Eliminates the split brain.

3. **`Factory.Supports` should be wired or removed** — Either implement capability checking (skip scenarios whose `Requires` field doesn't match the engine's interfaces) or remove the field. Currently it's an honest API lie.

4. **Engine `Capabilities()` method** — Instead of runtime type assertions scattered across `TypedReader` and `ExecuteTyped`, engines should advertise their capabilities via a `Capabilities()` method returning a struct/bitset. This would make the planner's job easier and eliminate the "try every interface" pattern.

5. **`adttest` should be a separate module with its own `go.mod`** — Currently it's a sub-package of `metaengine`, meaning external consumers can't import it without importing all of metaengine's test deps (ginkgo, gomega, sqlite). The `kv/viewstoretest` pattern has its own go.mod. This should be revisited for consistency.

### Testing

6. **Only `FilterEq` is tested for ScanRawValues** — `FilterNe`, `FilterLt`, `FilterLe`, `FilterGt`, `FilterGe`, and `FilterIn` are all untested through the raw scan path. The `FilterIn` operator has special `[]any` handling that could break silently.

7. **Cursor pagination with desc sort is untested** — The cursor logic for desc sort differs (reverses the comparison direction). Untested means unknown.

8. **No edge case tests** — Empty collection, all items filtered out, limit=1 (boundary). These are trivial to add and catch real bugs.

9. **No adttest harness self-tests** — The harness has zero dedicated tests. `RunMatrix` with zero factories, `CanonicalizeAny` with nil/edge cases, etc.

10. **Benchmarks don't test the filter path** — The 4.5x speedup claim is for the no-filter path only. The filter path (which double-decodes) may be slower than MapScan for complex queries. This should be benchmarked.

11. **Property-based testing for cross-engine parity** — `pgregory.net/rapid` is already a test dep in the event package. Could generate random operation sequences and verify all engines produce identical results.

### Documentation

12. **No ADR for the adttest extraction** — The decision to create a shared test harness (vs keeping tests per-module) is undocumented.

13. **No ADR for the Pebble raw readers** — The JSON tax reduction approach and the Go-side filter evaluation trade-off are undocumented.

14. **`ContractSuite` vs `adttest.RunMatrix` relationship is unclear** — Both exist in the codebase. `ContractSuite` tests a single engine; `RunMatrix` tests cross-engine parity. No doc explains when to use which.

15. **Exported helpers have minimal godoc** — `PassesFilterSpecs`, `ItemFieldByName`, `CompareValues`, `EvalFilterOp` have one-line comments. Full godoc with usage examples would help external engine implementors.

16. **TODO_LIST.md and FEATURES.md not updated** — The two 🔥 items are done but not marked in the project's tracking docs.

### Code Quality

17. **`ScanRawValues` sorts ALL rows then applies cursor** — O(N log N) regardless of cursor position. For large collections, this is wasteful. An incremental approach isn't possible with Pebble's LSM (no secondary index), but a counter summary key could help for the common case.

18. **`ScanRawValues` materializes all rows** — No `StreamScan` equivalent. The `StreamScan` interface exists in `engine.go:164` but Pebble doesn't implement it. For large collections, this is OOM-risky.

19. **Pebble `CounterGet` is O(N) prefix scan** — Every counter read scans all counter keys. A "counter summary" key with the total, updated atomically alongside individual counters, would make it O(1).

---

## F) UP to 50 Things to Get Done Next

### Immediate (This Session's TODO, Already Planned)

| #   | Task                                          | Impact  | Effort  | Status  |
| --- | --------------------------------------------- | ------- | ------- | ------- |
| 1   | Fix ScanRawValues triple-decode → decode-once | 🔴 HIGH | LOW     | Planned |
| 2   | Extract shared sort+paginate helper           | MED     | LOW     | Planned |
| 3   | Remove dead `ctx` in Scenarios()              | LOW     | TRIVIAL | Planned |
| 4   | Remove dead `Factory.Supports` field          | LOW     | TRIVIAL | Planned |
| 5   | Add FilterIn test                             | HIGH    | LOW     | Planned |
| 6   | Add FilterNe/Lt/Le/Gt/Ge tests                | HIGH    | LOW     | Planned |
| 7   | Add cursor+desc sort test                     | HIGH    | LOW     | Planned |
| 8   | Add empty/all-filtered/limit=1 tests          | MED     | LOW     | Planned |
| 9   | Add adttest harness self-tests                | MED     | LOW     | Planned |
| 10  | Add godoc to exported helpers                 | MED     | LOW     | Planned |
| 11  | Document ContractSuite vs RunMatrix           | MED     | LOW     | Planned |
| 12  | Update V1StabilizationChecklist               | LOW     | TRIVIAL | Planned |

### Short-Term (Metaengine Engine Sophistication)

| #   | Task                                                    | Impact | Effort  |
| --- | ------------------------------------------------------- | ------ | ------- |
| 13  | Pebble `StreamScan` (iter.Seq2) implementation          | MED    | LOW     |
| 14  | Benchmark with filters (show filter evaluation cost)    | MED    | TRIVIAL |
| 15  | Benchmark with 10K/100K items (scaling behavior)        | MED    | TRIVIAL |
| 16  | Three-way ADT matrix (memory+sqlite+pebble in one test) | MED    | LOW     |
| 17  | Call `ContractSuite` from pebbleengine tests            | MED    | TRIVIAL |
| 18  | Pebble `MapUpdate` with raw reader path                 | LOW    | LOW     |
| 19  | Pebble counter optimization (summary key)               | LOW    | LOW     |
| 20  | Pebble `PushdownScan` hybrid (prefix scan + Go-filter)  | LOW    | MED     |

### Mid-Term (New Engines & Features)

| #   | Task                                                          | Impact  | Effort |
| --- | ------------------------------------------------------------- | ------- | ------ |
| 21  | Pebble LayoutPlanner (prefixed key ranges for indexed fields) | 🔥 HIGH | MED    |
| 22  | Postgres engine (native JSONB operators, GIN indexes)         | HIGH    | HIGH   |
| 23  | Postgres LayoutPlanner (CREATE INDEX CONCURRENTLY)            | MED     | MED    |
| 24  | DuckDB analytical engine (columnar OLAP)                      | LOW     | HIGH   |
| 25  | DuckDB StreamingScan (arrow-based)                            | LOW     | MED    |
| 26  | Engine `Capabilities()` method                                | MED     | MED    |
| 27  | `metaengine-gen` code generator                               | LOW     | HIGH   |
| 28  | Schema enforcement at Plan() time                             | MED     | MED    |
| 29  | Multi-engine `Plan()` with Pebble cost model                  | MED     | MED    |
| 30  | Pebble LayoutPlanner integration with Plan()                  | MED     | MED    |

### Testing & Reliability

| #   | Task                                                   | Impact | Effort  |
| --- | ------------------------------------------------------ | ------ | ------- |
| 31  | Soak test (10M events, memory profiling)               | MED    | MED     |
| 32  | Chaos testing (FaultEngine wrapper)                    | LOW    | MED     |
| 33  | Fuzz test for ScanRawValues                            | LOW    | LOW     |
| 34  | Property-based cross-engine parity (rapid)             | MED    | MED     |
| 35  | CI: add pebbleengine to race test matrix               | MED    | TRIVIAL |
| 36  | CI: add raw reader benchmarks with regression tracking | LOW    | LOW     |
| 37  | CI: verify raw_reader.go coverage >90%                 | LOW    | TRIVIAL |

### Documentation

| #   | Task                                                | Impact | Effort  |
| --- | --------------------------------------------------- | ------ | ------- |
| 38  | ADR for adttest extraction                          | LOW    | TRIVIAL |
| 39  | ADR for Pebble raw readers                          | LOW    | TRIVIAL |
| 40  | Update TODO_LIST.md (mark 🔥 items done)            | LOW    | TRIVIAL |
| 41  | Update FEATURES.md (add raw readers)                | LOW    | TRIVIAL |
| 42  | Update SKILL.md (mention Pebble raw reader support) | LOW    | TRIVIAL |

### Code Quality & Cleanup

| #   | Task                                                                        | Impact | Effort  |
| --- | --------------------------------------------------------------------------- | ------ | ------- |
| 43  | Wire `Factory.Supports` or remove it                                        | LOW    | TRIVIAL |
| 44  | Add per-engine setup/teardown hooks to adttest                              | LOW    | LOW     |
| 45  | Deduplicate canonicalization logic (harness vs cross_engine_adt_test)       | LOW    | LOW     |
| 46  | Pebble sorted map optimization (key-prefix sort)                            | LOW    | MED     |
| 47  | Remove the `encoding/json/v2` stdversion gopls warnings (suppression)       | LOW    | TRIVIAL |
| 48  | Consolidate pebble engine compile-time assertions (add to engine.go header) | LOW    | TRIVIAL |
| 49  | Add `FilterSpec` and `SortSpec` to the godoc in engine.go                   | LOW    | TRIVIAL |
| 50  | Document the `jsonValue` internal type with a clear example                 | LOW    | TRIVIAL |

---

## G) Questions I Cannot Answer Myself

### 1. Should `adttest` be a separate Go module with its own `go.mod`?

Currently it's a sub-package of `metaengine` (no separate go.mod), which means external consumers can't import it without importing all of metaengine's test deps (ginkgo, gomega, sqlite). The `kv/viewstoretest` pattern has its own go.mod and is in go.work and api-stability. Should `adttest` follow the same pattern?

**Why I can't answer this:** This is a product/API decision. A separate module gives cleaner dependency isolation and allows external engine implementors to import the test harness without pulling in metaengine's test deps. But it adds module-management overhead (another go.mod, go.work entry, api-stability entry). The trade-off depends on whether external consumers are expected to implement engines.

### 2. Should the `ScanRawValues` filter path use `encoding/json/v2`'s streaming decoder instead of `decodeJSON`?

Currently `decodeJSON` calls `json.Unmarshal` which allocates a full `map[string]any`. The `json/v2` package has a streaming decoder (`json.Decoder`) that could decode field-by-field, stopping after the filter/sort fields are found. This would reduce allocations for wide JSON objects (objects with many fields but only 1-2 filter/sort fields). But it adds significant complexity and may not be faster for small objects.

**Why I can't answer this:** This requires benchmarking with realistic data shapes (wide vs narrow objects). The right answer depends on the typical payload size in consumer projects, which I don't have visibility into.

### 3. Should the three-way ADT matrix test (memory + sqlite + pebble in a single test) live in a new `metaengine/integration` test module?

Currently parity is verified transitively (memory↔sqlite in metaengine, memory↔pebble in pebbleengine). A direct three-way test would be stronger but requires a module that depends on both `metaengine` and `metaengine/pebbleengine`. The existing `metaengine/projectionadapter` could be a model, but it's a library module, not a test-only module. Should we create a `metaengine/adttest/integration` test module?

**Why I can't answer this:** This is an architecture decision about test module boundaries. The transitive verification is logically sound (if A=B and A=C then B=C), but a direct test catches integration bugs that transitive verification misses. The trade-off is module complexity vs test confidence.

---

## Session Summary

### What This Session Did

1. **Verified all previous session work** — Build, test, race, doc-check, api-stability: all green (except pre-existing `TestA032_NoFindingForBrandedID` failure in cqrs-lint)
2. **Identified 4 critical issues** — Triple-decode, sort duplication, dead code, dead API field
3. **Created comprehensive 15-item execution plan** — Sorted by impact/effort, all items scoped to ≤12 min each
4. **Verified formatting** — `gofumpt` and `goimports` confirm all new files are clean (no `nix fmt` needed)
5. **Verified api-stability golden** — 2906 exports verified, no regen needed
6. **Verified doc-check** — All 927 references valid across 38 packages

### What This Session Did NOT Do

- Did not start executing the 15-item plan (interrupted by status report request)
- Did not write the brutal self-review HTML report (planned as task 13)
- Did not commit the untracked status report from the previous session

### Current State of the Working Tree

```
?? docs/status/2026-07-31_12-40_metaengine-engine-sophistication-comprehensive-status.md
```

One untracked file — the previous session's status report. All code changes were committed by the auto-commit daemon.
