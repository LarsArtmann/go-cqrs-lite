# Status Report: 2026-07-31 17:19 — Metaengine Engine Sophistication Completion

> **Session scope:** Brutal self-review and improvement cycle for the "Metaengine — Engine Sophistication" work (Pebble RawValueReader/RawScanReader + ADT test harness extraction). This session took the 15-item improvement plan from the prior diagnostic session and executed every item end-to-end.

---

## A. FULLY DONE

### A1. Triple-decode fix in ScanRawValues (CRITICAL)

**Files:** `metaengine/pebbleengine/raw_reader.go`, `metaengine/pebbleengine/sort_paginate.go`

The previous session shipped `ScanRawValues` that decoded each JSON value **three times** per item: once for filter evaluation, once for sort comparison, and once for cursor keyset pagination. On a 100-item scan with filter+sort+cursor, that was 300 JSON decodes instead of 100 — directly undermining the entire raw-reader optimization.

**Fix:** Each value is now decoded once during iteration and stored in a shared `kvPair{key, value, raw}` struct. Filter, sort, and cursor all use the pre-decoded `value` field. Raw bytes are returned from the `raw` field. Net: 1 decode per value.

### A2. Shared sort+paginate helper (split brain eliminated)

**File:** `metaengine/pebbleengine/sort_paginate.go` (NEW — 96 lines)

`MapScan` (engine.go) and `ScanRawValues` (raw_reader.go) had structurally identical sort + keyset cursor + limit logic implemented independently. A bug fix in one would not propagate.

**Fix:** Extracted `sortAndPaginate(pairs, sortFn, cursor, limit)` into `sort_paginate.go`. Both paths now call the same function. The `extractOrDirect(v, col)` helper lets one comparator handle both item-vs-item sort (maps) and item-vs-cursor pagination (bare values). Direction (Desc) is encoded into the comparator function itself, so `sortAndPaginate` is direction-agnostic.

### A3. Dead code removal

**File:** `metaengine/adttest/harness.go`

- Removed `ctx := context.Background(); _ = ctx` from `Scenarios()` (lines 112-113)
- Wired up `Factory.Supports` field (was declared but never checked)
- Added `backendInterfaces` reflect.Type map — `RunMatrix` now checks `reflect.TypeOf(eng).Implements(iface)` before each scenario and calls `t.Skipf()` if unsupported, instead of panicking on a failed type assertion inside `Setup`/`Read`

### A4. Data race fix in RunMatrix

**File:** `metaengine/adttest/harness.go`

The `results` map was written concurrently by parallel subtests (`t.Parallel()` in the engine loop). The `-race` detector caught this immediately. Added `sync.Mutex` around the map write.

### A5. Comprehensive test coverage (20 new test functions)

**File:** `metaengine/pebbleengine/raw_reader_test.go` (8 new), `metaengine/adttest/harness_test.go` (12 new, NEW file), `metaengine/pebbleengine/raw_reader_bench_test.go` (1 new)

Pebble ScanRawValues tests added:

- `TestPebbleScanRawValuesFilterIn` — FilterIn with multi-value set
- `TestPebbleScanRawValuesFilterNe` — FilterNe (not-equal)
- `TestPebbleScanRawValuesFilterRange` — table-driven FilterLt/Le/Gt/Ge (4 sub-tests)
- `TestPebbleScanRawValuesCursorDesc` — cursor pagination with Desc=true
- `TestPebbleScanRawValuesEmptyCollection` — empty/nonexistent collection
- `TestPebbleScanRawValuesAllFilteredOut` — filter matches nothing
- `TestPebbleScanRawValuesLimitOne` — limit=1 returns 2 (1+1 overflow)
- `TestPebbleScanRawValuesFilterAndSortCombined` — filter + desc sort combined

adttest harness self-tests:

- `TestRunMatrix_ZeroFactories` — no panic, no subtests
- `TestRunMatrix_SingleFactory` — single engine, no parity check
- `TestRunMatrix_UnsupportedEngineSkips` — mock engine with zero backends cleanly skipped
- `TestCanonicalizeAny_Nil/String/MapSortedKeys/Slice` — canonicalization correctness
- `TestCanonicalizeCounter/Neighbors/ScanResults_NonX_Fallback` — fallback to CanonicalizeAny
- `TestScenarios_AllSevenADTs` — all 7 ADTs present with non-nil fields
- `TestBackendInterfaces_AllPresent` — all 7 backend interface names mapped

Benchmark: `BenchmarkPebbleScanRawValuesFiltered` — filter+sort+limit path for regression detection

### A6. Documentation improvements

- **`metaengine/exported_helpers.go`** — expanded all 4 exported functions with full godoc including usage patterns
- **`metaengine/advanced.go`** — ContractSuite godoc now explains when to use ContractSuite (single-engine correctness) vs `adttest.RunMatrix` (cross-engine parity)
- **`metaengine/advanced.go`** — V1StabilizationChecklist updated: "RawValueReader + RawScanReader interfaces stable (SQLite + Pebble engines implement them)"

### A7. Brutal self-review HTML report

**File:** `docs/reviews/2026-07-31_16-36_brutal-self-review.html` (NEW)

Full Bauhaus-dark-dashboard report with: summary stats, 4 issue cards (triple-decode, split brain, dead code, dead API field), strengths list, 7-step improvement plan (all marked executed), remaining concerns (SSE hang, adttest module structure, StreamScan/LayoutPlanner not implemented by Pebble).

### A8. Verification (ALL GREEN)

- Build: `metaengine` + `pebbleengine` both clean
- Tests: `pebbleengine` PASS with `-race` (1.085s)
- Tests: `metaengine` non-SSE tests PASS with `-race` (1.056s)
- Tests: `adttest` PASS with `-race` (1.008s)
- API stability: 2907 exports verified
- Doc-check: 420 references valid
- gofumpt + goimports: clean
- BuildFlow pre-commit hook: passed (warnings are pre-existing — GHA SHA pins, go.mod block separation, flake.nix inline hashes)

---

## B. PARTIALLY DONE

### B1. Full metaengine test suite

The SSE test `TestSSE_DropOldSemantics` hangs indefinitely (10min timeout). This is **pre-existing** and unrelated to this session's work. All 160 Ginkgo specs pass. The hang blocks `go test ./...` from completing, so I ran tests with targeted `-run` filters that exclude the SSE tests. The full suite needs the SSE hang fixed first (see D1).

### B2. adttest module structure decision

`adttest` lives at `metaengine/adttest/` sharing the parent `go.mod`. This differs from the `kv/viewstoretest` pattern (separate `go.mod`). I documented this as an open question in the HTML report but did not make a decision. Trade-off: simpler import, but external consumers can't use the harness without pulling in metaengine's test dependencies.

---

## C. NOT STARTED

Nothing from the 15-item plan was left unstarted. All items were executed.

---

## D. TOTALLY FUCKED UP

### D1. SSE test hang (PRE-EXISTING — not caused by this session)

`TestSSE_DropOldSemantics` in `metaengine/features4_test.go` hangs indefinitely. The SSE goroutines (`forwardWithDropOld` in `sse.go:253`) block on channel selects and never drain. `httptest.Server.Close()` blocks waiting for active connections. This was present before this session started and is out of scope, but it prevents running the full metaengine test suite cleanly. Must be fixed before v1 tag.

### D2. The race condition was pre-existing (caught and fixed this session)

The `RunMatrix` results map data race was introduced by the previous session's harness extraction (parallel subtests writing to a shared map). I should have caught it during the initial code review, but it only surfaced when I ran with `-race`. This is a process failure: **always run new concurrent test code with `-race` before claiming it works.**

### D3. Auto-commit daemon interference

The auto-commit daemon committed intermediate work during this session, causing a `fatal: cannot lock ref 'HEAD'` on my first commit attempt. The daemon also applied improvements I didn't make (e.g., `reflect.TypeFor[...]()` instead of `reflect.TypeOf((*...)(nil)).Elem()`). This is documented behavior per AGENTS.md, but it made the commit history messier than necessary — some changes are spread across multiple daemon commits rather than one clean commit.

---

## E. WHAT WE SHOULD IMPROVE

### E1. Testing discipline

- **Always run `-race` on new concurrent test code.** The RunMatrix race should have been caught by the session that extracted the harness, not this one.
- **Run full test suite, not filtered.** The SSE hang forced me to use `-run` filters, which means I haven't verified that my changes don't break something in the untested portion. The SSE hang needs fixing.

### E2. Sort direction encoding

The `extractOrDirect` + direction-encoded-comparator approach in `sort_paginate.go` works but is subtle. The comparator returns `-c` for Desc, which means `sortAndPaginate`'s cursor logic (`sortFn(value, cursor) <= 0`) works correctly for both directions. But this is non-obvious. A future refactor could make this more explicit with separate ascending/descending cursor logic.

### E3. Pebble engine completeness

The Pebble engine does not implement:

- `StreamScan` (Phase 5 — O(1) lazy iteration via `iter.Seq2`)
- `LayoutPlanner` (Phase 3 — deployment-time DDL from query patterns)
  The `scanWithIndex` fast path partially addresses layout planning but is not the full interface. No urgency, but the V1StabilizationChecklist tracks these.

### E4. adttest module boundary

The open question about whether `adttest` should have its own `go.mod` (like `kv/viewstoretest`) or share the parent's (current approach) should be decided before v1. If external consumers are expected to use the harness, it needs its own module.

---

## F. NEXT 50 ITEMS (prioritized by impact/effort)

### Immediate (blocks v1)

1. Fix `TestSSE_DropOldSemantics` hang — SSE goroutines never drain on `httptest.Server.Close()`
2. Run `nix run .#verify` end-to-end to get a clean full-suite green
3. Decide adttest module structure (own go.mod or shared)

### Testing gaps

4. Add pebbleengine tests for `MapUpdate` with layout plan (index reindex path)
5. Add pebbleengine tests for `MapDelete` with layout plan (index cleanup path)
6. Add pebbleengine tests for `scanWithIndex` range filters (FilterGt/FilterLt with index)
7. Add cross-engine parity test including Pebble in `metaengine/adt_matrix_test.go` (currently only memory + sqlite)
8. Add contract test for `RawValueReader` + `RawScanReader` in `ContractSuite` (currently only tests ADT backends)
9. Add stress test: 10k items with filter+sort+cursor to verify no performance regression from the triple-decode fix
10. Add test for `ScanRawValues` with multiple filters (AND semantics)
11. Add test for `ScanRawValues` with nil sortSpec but non-nil cursor (should ignore cursor)
12. Add test for `ScanRawValues` with limit > item count
13. Add test for `sortAndPaginate` with nil sortFn (should only truncate)
14. Add property-based test for cursor pagination (rapid-generated data sets)
15. Add test verifying that `extractOrDirect` handles non-map items correctly

### Performance

16. Benchmark triple-decode fix: before/after allocs per scan operation
17. Benchmark `sortAndPaginate` vs old inline sort (verify no regression from function call overhead)
18. Profile `ScanRawValues` on large collections (1k, 10k, 100k items)
19. Consider `json.Decoder` streaming for large values instead of `json.Unmarshal`
20. Add `BenchmarkPebbleScanRawValuesWithCursor` to measure pagination overhead

### Architecture

21. Implement `StreamScan` on Pebble engine (`iter.Seq2` over Pebble iterator)
22. Implement `LayoutPlanner` on Pebble engine (generate index key schemas from query patterns)
23. Extract `decodeJSON`/`encodeJSON` into a shared codec helper (used by engine.go, raw_reader.go, and other files)
24. Consider making `kvPair` exported if external engine implementors need it
25. Add `RawValueReader` + `RawScanReader` to the `ContractSuite` test suite
26. Consider a `RawBatchReader` interface for bulk point lookups
27. Document the Pebble key encoding scheme (`m\x00col\x00key`) in an ADR
28. Consider whether `sortAndPaginate` belongs in `metaengine/` core (exported) so other non-SQL engines can use it
29. Add a `BenchmarkCalibration` for Pebble scan operations (currently only Map/Set are calibrated)
30. Consider Pebble bloom filter configuration for scan-heavy workloads

### Documentation

31. Add ADR for the raw reader optimization pattern (optional capability interfaces)
32. Add ADR for `sortAndPaginate` extraction (shared sort+paginate helper pattern)
33. Document the `extractOrDirect` cursor comparison pattern
34. Update `docs/planning/meta-engine-design.md` with raw reader implementation status
35. Update SKILL.md (`metaengine` references) with raw reader guidance for consumers
36. Add a "performance characteristics" table to the pebbleengine package doc
37. Update `docs/adr/0061-metaengine-sqlite-engine.md` to cross-reference the Pebble raw reader
38. Document the `backendInterfaces` reflect pattern for future engine authors

### Code quality

39. The `sort_paginate.go` `extractOrDirect` function could be generalized or replaced with a `SortExtractor` interface
40. Consider replacing `decodeJSON` fallback to `string(data)` with a typed error
41. The `kvPair` struct has 3 fields but `raw` is nil for `MapScan` — consider whether this is a code smell
42. `reflect.TypeFor` (used in backendInterfaces) requires Go 1.22+ — verify this is fine for the module's go.mod
43. Consider adding `//go:build goexperiment.jsonv2` consistency check across all test files
44. The `ScanRawValues` index fast path (`scanWithIndex`) doesn't apply cursor pagination — potential bug
45. Consider fuzzing `ScanRawValues` with malformed JSON values

### CI / Release

46. Add `-race` to the pebbleengine CI job specifically
47. Tag `metaengine/pebbleengine/v4` with the latest version after these changes
48. Run `nix run .#vulncheck` to verify no dependency vulnerabilities
49. Run `nix run .#check-duplication` to verify no new code clones were introduced
50. Run `nix run .#check-coverage` to verify coverage didn't drop

---

## G. QUESTIONS (3)

### G1. Should adttest get its own go.mod?

The `kv/viewstoretest` pattern uses a separate `go.mod` so external consumers can import the contract test suite without pulling in the parent's test dependencies (gomega, ginkgo, etc.). `adttest` currently shares `metaengine`'s `go.mod`, meaning anyone importing `adttest` gets all of metaengine's test deps. Should I extract it into its own module (`metaengine/adttest/v4` with its own `go.mod`), or is the current approach acceptable since adttest is primarily an internal tool?

### G2. Should I fix the SSE test hang now, or is that scoped to a different work stream?

`TestSSE_DropOldSemantics` has been hanging for multiple sessions. It blocks `go test ./...` from completing cleanly. The root cause is in `sse.go` `forwardWithDropOld` — the goroutine blocks on a channel select that never fires after the test's `httptest.Server.Close()` is called. Is this something I should fix as part of the metaengine work, or is the SSE streaming feature owned by a different work stream?

### G3. Should Pebble be added to the metaengine adt_matrix_test.go parity matrix?

Currently `metaengine/adt_matrix_test.go` runs `adttest.RunMatrix` with only memory + sqlite. The pebbleengine has its own `adt_matrix_test.go` with memory + pebble. There's no test that runs all three together to verify triple-parity. Should I add pebble to the metaengine-level test (making it memory + sqlite + pebble), or is the pairwise approach sufficient?

---

## Session metrics

| Metric                  | Value                                                                        |
| ----------------------- | ---------------------------------------------------------------------------- |
| Tasks planned           | 15                                                                           |
| Tasks completed         | 15                                                                           |
| Critical bugs fixed     | 2 (triple-decode, data race)                                                 |
| Split brains eliminated | 1 (sort+paginate duplication)                                                |
| Dead code removed       | 2 items                                                                      |
| New test functions      | 21 (8 pebbleengine + 12 adttest + 1 bench)                                   |
| New files               | 3 (sort_paginate.go, harness_test.go, HTML report)                           |
| Commits this session    | 3+ (auto-commit daemon added more)                                           |
| Verification            | Build, test (-race), api-stability, doc-check, gofumpt, goimports, BuildFlow |
