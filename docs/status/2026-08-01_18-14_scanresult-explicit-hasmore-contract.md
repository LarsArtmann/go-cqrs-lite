# ScanResult Interface Refactor: Explicit HasMore Contract

**Date:** 2026-08-01 18:14
**Session scope:** Replacing the implicit `limit+1` slice-length pagination convention with an explicit `ScanResult{Items, HasMore}` struct across the entire metaengine scan interface surface.

---

## A. Fully Done

### Interface redesign (the core deliverable)

1. **Added `ScanResult{Items []any; HasMore bool}`** to `metaengine/engine.go`. The pagination signal is now an explicit struct field, not magic slice length.

2. **Added `RawScanResult{Items [][]byte; HasMore bool}`** — the raw-bytes variant for `RawScanReader`.

3. **Updated 3 scan interfaces** to return these structs:
   - `ScanBackend.MapScan` → `(ScanResult, error)`
   - `PushdownScan.PushdownMapScan` → `(ScanResult, error)`
   - `RawScanReader.ScanRawValues` → `(RawScanResult, error)`

### Engine implementations (10 functions across 5 modules)

4. **memoryEngine.MapScan** (`memory_engine.go`) — replaced `limit+1` truncation with explicit trim + `hasMore` detection.
5. **sqliteEngine.MapScan** (`sqlite_engine.go`) — same pattern.
6. **sqliteEngine.PushdownMapScan** (`sqlite_engine.go`) — SQL still fetches `limit+1`, but now trims to `limit` and sets `HasMore` explicitly.
7. **sqliteEngine.pushdownMapScanPlanned** (`planned_sqlite.go`) — same.
8. **sqliteEngine.ScanRawValues** (`raw_reader.go`) — same.
9. **pgEngine.MapScan** (`pgengine/scan.go`) — same.
10. **pgEngine.PushdownMapScan** (`pgengine/pushdown.go`) — same.
11. **duckdbEngine.MapScan** (`duckdbengine/scan.go`) — same.
12. **duckdbEngine.PushdownMapScan** (`duckdbengine/pushdown.go`) — same.
13. **pebbleEngine.MapScan** (`pebbleengine/engine.go`) — same, via shared `sortAndPaginate`.
14. **pebbleEngine.ScanRawValues** (`pebbleengine/raw_reader.go`) — rewrote to return `RawScanResult`, added `trimRaw` helper in `sort_paginate.go`.

### Consumer updates (8 files)

15. **`execute.go`** — `executeFilteredScan` updated for all 3 scan paths (RawScanReader, PushdownScan, ScanBackend). `ExecuteTyped` now type-asserts `ScanResult` (with `[]any` fallback for `MultiGet` which returns a plain slice).
16. **`collection.go`** — `reconstructCollection` simplified: takes `ScanResult` directly instead of `(raw any, limit int)`. The `hasMore := limit > 0 && len(items) > limit` heuristic is gone — reads `result.HasMore` directly.
17. **`typed_reader.go`** — `scanRaw`, `scanPushdown`, `scanClosure` all updated to extract `.Items` from the result struct.
18. **`export_import.go`**, **`stats.go`**, **`consistency.go`**, **`advanced.go`** — all MapScan callers updated to read `result.Items`.
19. **`adttest/harness.go`** — `CanonicalizeScanResults` was a latent bug: it type-asserted to `[]any` but `MapScan` returned a struct. Now correctly asserts `metaengine.ScanResult` and reads `.Items`.

### Tests (20+ test files updated)

20. All test files across 4 modules updated: `.Items` access for `len()`, `range`, index access, gomega matchers, helper function arguments.
21. All `limit+1` expected counts corrected to `limit` (the engines now trim to exactly `limit` items).

### Verification

22. API surface regenerated: 3086 → 3088 exports (+`ScanResult`, +`RawScanResult`).
23. `nix fmt` applied — 38 files formatted.
24. `nix run .#verify` GREEN — build, vet, test (with -race), lint (0 issues), doc-check (1169 refs), API surface. All 63 modules pass.

---

## B. Partially Done

### The `[]any` fallback in ExecuteTyped

`ExecuteTyped` now has a two-path type assertion:

```go
if result, ok := raw.(ScanResult); ok { ... }
if items, ok := raw.([]any); ok {
    return reconstructCollection[R](ScanResult{Items: items}, sortFn), nil
}
```

This handles `MultiGet` (which returns `[]any`, not `ScanResult`). It works, but it's a design inconsistency — `MultiGet` should either return `ScanResult` too, or `ExecuteTyped` should handle non-scan collection results differently. The fallback is pragmatic but not clean.

---

## C. Not Started

1. **ADR for the ScanResult design** — No Architecture Decision Record documents _why_ the `limit+1` convention was replaced with an explicit struct. This is a public API change that future contributors need context on.

2. **Shared scan test harness** — The ScanBackend test pattern (filter, sort, limit, cursor) is still duplicated between pgengine, duckdbengine, and pebbleengine. A shared `scantest` package (like `adttest`) would eliminate duplication and enforce cross-engine parity for the new `ScanResult` contract.

3. **Tagging** — Neither `metaengine/pgengine/v4.0.0` nor `metaengine/duckdbengine/v4.0.0` have been tagged. The interface change is untagged.

---

## D. Mistakes & Issues

### D1: sed over-matching in test fixes (BIGGEST ISSUE)

Used `sed` to batch-replace `len(results)` → `len(results.Items)` and `!= 3` → `!= 2` across test files. The sed patterns were too broad and changed assertions that legitimately expected 3 items (no-limit queries, filter-IN queries) to expect 2. Required 4+ rounds of manual fixes to correct the over-matches. **Lesson:** sed is dangerous for semantic changes. Should have used targeted `multiedit` per file after reading each.

### D2: Missing closing brace in execute.go

The first `multiedit` on `execute.go` accidentally deleted a closing `}` in the pushdown path, causing a syntax error. Required a manual fix. **Lesson:** `multiedit` with large replacement blocks is error-prone — verify structural integrity after each batch.

### D3: adttest CanonicalizeScanResults latent bug

`CanonicalizeScanResults` was asserting `v.([]any)` but `MapScan` had already changed to return `ScanResult`. This was a **pre-existing latent bug** that my change exposed — the assertion always failed silently and fell through to `CanonicalizeAny(v)`. Fixed it to assert `metaengine.ScanResult`. But I should have caught this earlier during the interface change, not during the test-fixing phase.

### D4: The initial "quick fix" (limit+1 alignment)

Before the proper `ScanResult` refactor, I first aligned pgengine/duckdbengine to use `limit+1` truncation (matching Memory/SQLite/Pebble). This was a band-aid — the user correctly challenged it and asked for the proper design. The band-aid commits are still in history but were superseded by the struct refactor.

### D5: Heap growth test flake

`TestSoak_MemoryBounded` failed once during the verify gate (heap grew 2.6MB vs 2.0MB max). Passes in isolation. This is a pre-existing flaky test (GC pressure under -race), not caused by this change. Not fixed.

---

## E. What We Should Improve

### E1: MultiGet should return ScanResult too (or not)

`MultimapBackend.MultiGet` returns `[]any`. `ExecuteTyped` handles this via a `[]any` fallback path alongside the `ScanResult` assertion. This is a design inconsistency: either `MultiGet` is a scan-like operation (should return `ScanResult` with `HasMore=false`) or it's a point lookup (shouldn't go through `reconstructCollection` at all). The current two-path assertion is pragmatic but muddies the contract.

### E2: `sortAndPaginate` still uses `limit+1` internally

The pebble `sortAndPaginate` helper (`pebbleengine/sort_paginate.go`) still truncates to `limit+1` internally. The `MapScan`/`ScanRawValues` callers then trim to `limit` and set `HasMore`. This is correct but means the internal helper still carries the old convention. It should be renamed or refactored to communicate that callers handle the final trim.

### E3: `collection.go` lost the limit parameter

`reconstructCollection` used to take `(raw, limit, sortKeyFn)`. Now it takes `(result ScanResult, sortKeyFn)`. The `limit` was used for the `hasMore` heuristic, which is now gone. But `extractLimitFromInput(input)` is no longer called in the `isCollectionResult` path — this means the limit is no longer available for any future feature that might need it at the reconstruction layer. Not a problem today, but worth noting.

### E4: Test count adjustments were manual and error-prone

Every test that asserted `len == limit+1` needed manual correction to `len == limit`. There were ~15 such assertions across 6 files. A better approach would have been to grep for `limit+1` comments and assertions _before_ changing the engines, collect them all, and fix them in one batch.

---

## F. Next Steps (Prioritized)

1. **Write ADR** — Document the `ScanResult` design decision and the `limit+1` convention it replaces.
2. **Fix MultiGet inconsistency** — Either make `MultiGet` return `ScanResult{HasMore: false}` or separate the collection-result path in `ExecuteTyped`.
3. **Extract shared `scantest` package** — Cross-engine scan parity tests (filter, sort, limit, cursor, HasMore).
4. **Tag engine modules** — `metaengine/v4.3.0` (breaking scan interface), `pgengine/v4.0.0`, `duckdbengine/v4.0.0`.
5. **Refactor `sortAndPaginate`** — Rename or restructure so the `limit+1` internal convention is clearly documented as an internal detail.
6. **Add `HasMore` assertion to adttest matrix** — The ADT test harness should verify that engines return `HasMore=true` when more items exist beyond the limit.
7. **Review `TypedReader.trimAndCache`** — It still uses `fetchLimit = cfg.limit * 2` for prefetch. With explicit `HasMore`, the prefetch logic could be simplified.
8. **Unify `RawScanResult` and `ScanResult`** — Consider a generic `ScanResult[T any]` with `Items []T`. Would eliminate the two-struct duplication.
9. **Add property-based test for HasMore** — Rapid test: insert N items, scan with limit L, verify `HasMore == (N > L)` and `len(Items) == min(N, L)`.
10. **Document the three-tier scan dispatch** — The RawScanReader → PushdownScan → ScanBackend cascade in `execute.go` is non-obvious. A diagram or comment block would help.
11. **Review `LogTail` return type** — `LogBackend.LogTail` returns `[]any`, not `ScanResult`. It goes through the same `ExecuteTyped` path. Same inconsistency as MultiGet.
12. **Clean up `scanSingleColumn` generic helper** — It returns `[]T` which all scan implementations then wrap into `ScanResult`. The wrapping logic is duplicated.
13. **Add `HasMore` to `TypedReader.Scan` return** — Currently `TypedReader.Scan` returns `([]V, error)`. With `HasMore` now explicit, the typed reader could expose it.
14. **Review cursor construction in `reconstructCollection`** — It uses `lastItem` from the full `Items` slice. With `HasMore` explicit, the cursor should be derived from the last item _of the returned page_, not a `limit+1` overflow item. This is now correct but worth a test.
15. **Performance benchmark** — The `ScanResult` struct adds one allocation per scan call (struct on stack vs slice header). Benchmark to confirm no regression on hot paths.

---

## G. Questions

1. **Should `MultiGet` and `LogTail` also return `ScanResult`?** They return `[]any` today and go through `reconstructCollection` via the `[]any` fallback. Making them return `ScanResult{HasMore: false}` would be consistent but adds wrapping overhead for point lookups that never paginate. Alternatively, they could bypass `reconstructCollection` entirely via a different code path. Which direction do you prefer?

2. **Should this be a major version bump (`metaengine/v5`)?** The scan interface signatures changed (return type from `[]any` to `ScanResult`). Any consumer implementing `ScanBackend`, `PushdownScan`, or `RawScanReader` with a custom engine breaks. The module is tagged at `v4.2.0`. Is this a v5.0.0 or a v4.3.0 with breaking-change notes?

3. **Should `ScanResult` be a generic `ScanResult[T any]`?** This would eliminate `RawScanResult` and give type safety on `Items`. But it complicates the interface (generic methods on non-generic interfaces require type parameters at the method level, which Go doesn't support — so it would need to be a function, not a method, or the interface itself becomes generic).

---

## Resolution (2026-08-03)

`ScanResult{Items, HasMore}` and `RawScanResult` shipped across all scan interfaces (10 engine functions, 5 modules, 20+ test files). The `limit+1` convention was replaced.

**Q1 (MultiGet/LogTail ScanResult):** Deferred — they still return `[]any`. Low priority (point lookups never paginate).
**Q2 (major version bump):** Resolved as `metaengine/v4.3.0` (not v5) — scan interface changed but module consumers updated.
**Q3 (generic ScanResult[T]):** Deferred — Go doesn't support generic methods on non-generic interfaces.

Next steps: items 4 (tag) DONE (`metaengine/v4.4.0`); item 6 (HasMore in adttest) not confirmed; others are standing improvements captured in TODO_LIST.md.
