# Status Report: Memory Store Conformance Suite Adoption + Bug Fixes

**Date:** 2026-08-08 02:47
**Session scope:** Execute 6-item backlog from paste_1.txt — adopt shared conformance suites in storage/memory, fix documentation gaps, fix pre-existing dead code.

---

## a) FULLY DONE

### 1. Removed dead `time` import in `stack/bench/durability_tiers_test.go`
- **What:** Removed `"time"` import + `var _ = time.Second` hack (lines 6, 172-173). The `time` package was genuinely unused — the entire file is benchmarks using `context`, `event`, `id`, `stack` packages.
- **Files:** `stack/bench/durability_tiers_test.go` (173→169 lines)

### 2. Added `doc.go` to `command/commandtest/`
- **What:** Created `command/commandtest/doc.go` with the package doc comment (extracted verbatim from `store_suite.go` header). Removed the package-level doc from `store_suite.go` to avoid duplicate package comments. Mirrors `query/querytest/doc.go` structure.
- **Files:** `command/commandtest/doc.go` (new, 11 lines), `command/commandtest/store_suite.go` (package doc removed from header)

### 3. Added `command/commandtest` to module tracking lists
- **AGENTS.md:** Added `command/commandtest` to the Quick Reference Modules row (between `command` and `query`, matching the `query/querytest` pattern).
- **`cmd/api-stability/main.go`:** Added `"command/commandtest"` to the `modules` slice (Layer 1 group, adjacent to `"query/querytest"`).
- **Golden regenerated:** `docs/api_surface.txt` updated (3807→3811 exports: 4 new commandtest symbols tracked — `MustCreateCommand`, `RunStoreSuite`, `StoreSuite`, `StoreFactory`).
- **Verified:** `TestEveryGoModDirIsInModulesList` passes, doc-check passes (545 references valid).

### 4. Refactored `storage/memory` command store tests — adopted `commandtest.RunStoreSuite`
- **What:** Replaced 6 hand-written tests with a single `TestMemoryCommandStore_Suite` delegating to `commandtest.RunStoreSuite`. Kept 5 memory-specific tests not covered by the suite (batch-internal duplicate, Load_NotFound, LoadToTimestamp, Close lifecycle, MultipleStreams).
- **Removed (now covered by suite):** `TestMemoryCommandStore_SaveAndLoad`, `TestMemoryCommandStore_DuplicateCommand`, `TestMemoryCommandStore_AppendBatch`, `TestMemoryCommandStore_LoadFromTimestamp`
- **`command_journal_test.go`:** Removed `TestMemoryCommandStore_Journal` and `TestMemoryCommandStore_Journal_ReadFromZeroID` (both fully covered by suite's `ReadAll` + `ReadFrom` subtests). Kept 4 edge-case tests: EmptyStore, NonExistentID, ClosedStore, OrderingByReceivedAt.
- **Line reduction:** command_store_test.go 317→220 lines, command_journal_test.go 236→146 lines. Total: 553→366 lines (34% reduction).

### 5. Refactored `storage/memory` query store tests — adopted `querytest.RunStoreSuite`
- **What:** Replaced 3 hand-written tests with a single `TestMemoryQueryStore_Suite` delegating to `querytest.RunStoreSuite`. Kept 3 edge-case tests (NonExistentID, EmptyStore, ClosedStore).
- **Removed (now covered by suite):** `TestMemoryQueryStore` (basic save/read/cursor), `TestMemoryQueryStore_LoadQueriesAfterTime`, `TestMemoryQueryStore_ReadQueriesFromZeroID`
- **Line reduction:** query_store_test.go 248→139 lines (44% reduction).

### 6. Added self-test to `commandtest` package
- **What:** Created `command/commandtest/store_suite_test.go` — runs `RunStoreSuite` against `memory.NewMemoryCommandStore()` to validate the suite itself (mirrors how `eventtest` is tested).
- **Files:** `command/commandtest/store_suite_test.go` (new, 21 lines)

### 7. Fixed 2 real bugs discovered by the conformance suite
The suite immediately caught divergences between `MemoryCommandStore`/`MemoryQueryStore` and every other backend (pebble, bbolt, SQL):

**Bug 1: `limit=0` semantics divergence**
- `MemoryCommandStore.ReadFrom` and `MemoryQueryStore.ReadQueriesFrom` treated `limit=0` as "return zero results" (via `min(startIdx+0, len(...))` → 0). All other backends treat `limit=0` as "no limit" (return everything). The suite's `ReadFrom` subtest calls `ReadFrom(ctx, zeroID, 0)` expecting all records.
- **Fix:** Changed to `end := len(...)` when `limit > 0`, otherwise use `min(startIdx+limit, len(...))`.
- **Files:** `storage/memory/command_store.go:214`, `storage/memory/query_store.go:134`

**Bug 2: `MemoryQueryStore.SaveQuery` had no duplicate detection**
- Unlike `MemoryCommandStore` (which checks `commandIDIndex`), `bbolt` (`query_store.go:43`), `pebble` (`query_store.go:92`), and SQL (`query_store_save.go:79`), `MemoryQueryStore.SaveQuery` silently accepted duplicate query IDs, overwriting the index entry. The suite's `DuplicateDetection` subtest expects `ErrDuplicateQuery`.
- **Fix:** Added `idIndex` existence check + `errorfamily.WrapConflict(query.ErrDuplicateQuery, ...)` before append.
- **Files:** `storage/memory/query_store.go:71-78`

---

## b) PARTIALLY DONE

Nothing partially done — all 6 backlog items were completed.

---

## c) NOT STARTED

Nothing from the backlog was skipped.

---

## d) TOTALLY FUCKED UP

Nothing. All changes compile, pass `-race` tests, and the api-stability golden is regenerated.

---

## e) WHAT WE SHOULD IMPROVE

1. **Auto-commit daemon mixed my changes with metaengine changes** — Commit `0448a257f` ("refactor(metaengine): remove GraphBackend and harden memory stores") bundles my `storage/memory` fixes with GraphBackend removal I didn't do. This makes `git bisect` unreliable. The daemon should either be smarter about commit grouping or we should disable it during active sessions.

2. **`command/commandtest` is untagged** — It was added in a prior session but never tagged. `GOWORK=off go test` fails for `storage/memory`, `storage/pebble`, and `storage/bbolt` because `command/v4@v4.3.0` doesn't include the `commandtest` subpackage. This is a pre-existing issue (pebble/bbolt had it before this session), but my refactoring of memory's tests means memory now has it too. Needs a `command/v4.4.0` tag (or whatever the next version is).

3. **Pre-existing integration test failure** — `TestBundle_RunProjections_GraphProjection` in `integration/` fails with `jsontext: invalid character 'X' at start of value` — a CBOR/JSON codec mismatch in the graph projection handler. Unrelated to this session but should be investigated.

4. **Uncommitted `metaengine/graphadapter/adapter_test.go`** — Has 64 uncommitted lines adding a `TestAdapter_StoreIntegration_RecordAware` test. Not from this session. Another agent or the daemon left it dangling.

5. **The conformance suite found bugs in seconds** — These bugs existed in `MemoryQueryStore` for weeks/months. The lesson: adopt shared suites IMMEDIATELY when they exist, not as a backlog item. Every day without suite adoption is a day bugs hide.

---

## f) Up to 50 things we should get done next

### Critical / Blocking
1. **Tag `command/v4.4.0`** — includes `commandtest` subpackage so `GOWORK=off` tests pass for storage/memory, storage/pebble, storage/bbolt
2. **Fix `TestBundle_RunProjections_GraphProjection`** — CBOR/JSON decode mismatch in integration test (pre-existing failure)
3. **Commit or discard `metaengine/graphadapter/adapter_test.go`** — 64 uncommitted lines from unknown source
4. **Tag `storage/memory/v4.3.0`** — includes the `limit=0` fix + `SaveQuery` duplicate detection (breaking bug fixes)

### High Priority
5. **Add `limit=0` test to `commandtest`/`querytest` suites** — The suite currently tests this implicitly via `ReadFrom(zeroID, 0)` but doesn't document the contract explicitly. Add a dedicated subtest: "ReadFrom with limit=0 returns all".
6. **Audit `MemoryCommandStore.Load` for `ErrCommandNotFound` vs empty-slice behavior** — The suite doesn't test this (Load on a non-existent stream). Memory returns error; SQL might return empty slice. Check parity.
7. **Adopt conformance suites in `storage/sql`** — The SQL command/query stores don't import `commandtest`/`querytest` yet. Same ~90% duplication pattern.
8. **Verify `LoadToTimestamp` exists in the suite** — Currently it's a memory-specific test. If it's part of the `CommandSource` interface, all backends should test it via the suite.
9. **Add `AppendBatch_DuplicateInBatch` to the suite** — Currently memory-specific. Pebble/bbolt/SQL all need this behavior; should be in the shared suite.
10. **Add `ClosedStore` tests to the suite** — Currently memory-specific for both command and query stores. Close-after-write should be tested for all backends.
11. **Add `MultipleStreams` to the suite** — Currently memory-specific. All backends must isolate streams.
12. **Run `nix run .#verify` gate** — This session ran targeted tests only. A full verify cycle confirms nothing else broke.

### Medium Priority
13. **Add querytest self-test** — `command/commandtest` now has `store_suite_test.go` (self-test). `query/querytest` does NOT have one (it only tests `New()`, not `RunStoreSuite`). Add one for parity.
14. **Consider a `commandtest.StoreSuiteExtended` interface** — For optional methods like `LoadToTimestamp`, `LoadToTimestamp`, `Close`. Backends that support them get extra subtests.
15. **Document the `limit=0 = unlimited` contract** — In the `SeekableCommandJournal`/`SeekableQueryJournal` interface doc comments.
16. **Add a conformance test for `ReadFrom` with non-existent ID** — Currently memory-specific. Behavior should be standardized (return nil, nil — not an error).
17. **Add a conformance test for ordering preservation** — Currently memory-specific (`Journal_OrderingByReceivedAt`). All backends should preserve insertion order in the journal.
18. **Check if `MemoryEventStore` has the same `limit=0` bug** — If event store's `ReadFrom` uses the same pattern, it has the same bug. Audit all `ReadFrom`/`ReadQueriesFrom`/`ReadFrom` implementations.
19. **Consider extracting a `limitClause(limit int, len int) int` helper** — The `if limit > 0 { end = min(...) }` pattern will be duplicated. Extract once.
20. **Update AGENTS.md Test row** — Add `./command/commandtest/...` to the `go test` command in the Quick Reference.
21. **Check `kv/viewstoretest` for similar suite adoption gaps** — It's listed in AGENTS.md but might have the same unadopted pattern.

### Low Priority / Cleanup
22. **Run `gofumpt -w` on changed files** — Ensure formatting is pristine before next verify gate.
23. **Add `commandtest` to cqrs-lint module catalog** — The linter's `ModuleCatalog` tracks modules for scorecard/adoption features.
24. **Check if `scenario` package can use `commandtest` store factories** — Reduce test setup boilerplate.
25. **Audit all `var _ = <pkg>.<symbol>` patterns** — Dead import suppression across the repo. The one I fixed was pre-existing; there may be more.
26. **Consider a `querytest.MustCreateQuery` that accepts `time.Time`** — Currently doesn't support `WithQueryReceivedAt`. Tests that need timestamp control create queries manually.
27. **Add `commandtest.MustCreateCommand` with `WithReceivedAt` option** — Same gap as above for command tests.
28. **Document the suite adoption process in CONTRIBUTING.md** — "When adding a new backend, import commandtest/querytest and call RunStoreSuite. Keep only backend-specific edge-case tests."
29. **Check if `storage/turso` needs conformance suite adoption** — Turso connector may have command/query stores.
30. **Consider a shared `CloseableStoreSuite`** — Both command and query stores test Close behavior. Extract to a shared helper.
31. **Run dedup analysis on the memory test files** — `art-dupl` may find remaining duplication in the memory-specific tests.
32. **Add benchmarks to the conformance suite** — Optional performance regression detection for backends.
33. **Consider fuzzing `ReadFrom` with random limit values** — The `limit=0` bug would have been caught by a fuzzer.
34. **Update `.art-dupl-baseline.json`** — After the test consolidation, duplication baseline may need regenerating.
35. **Check `stack/*` presets for direct memory store usage** — Stack presets may bypass the suite; verify they test the right things.
36. **Consider adding `commandtest` to the `go.work` summary** — It's already a package within `command/`, but verify workspace visibility.
37. **Review all `ReadFrom` callers for `limit=0` assumptions** — Some callers may pass 0 expecting unlimited; others may pass 0 expecting empty. Audit the codebase.
38. **Add a CI gate that fails when a backend doesn't adopt the suite** — `cqrs-lint` rule: if a directory has `command_store_test.go` and doesn't import `commandtest`, warn.
39. **Consider extracting `testutil.NewCommandStore`** — A cross-module factory for creating command stores in tests, reducing boilerplate further.
40. **Add `querytest` to the Test row in AGENTS.md** — The `go test` command should include `./query/querytest/...` explicitly.
41. **Document the `limit=0` convention in ADR** — This is a cross-cutting convention; an ADR prevents future divergence.
42. **Run `nix run .#check-coverage`** — The test consolidation may have changed coverage numbers. Verify we didn't lose coverage.
43. **Consider property-based testing for store limits** — Use `pgregory.net/rapid` to generate random limit values and verify behavior.
44. **Add `commandtest.StoreSuite` to `catalog/` registry** — Track it as a test infrastructure module in the catalog.
45. **Review `bbolt` command/query store tests** — They adopted the suite in a prior session. Verify they don't have leftover hand-written tests that duplicate suite coverage.
46. **Consider a `backendtest` meta-package** — Combines `eventtest` + `commandtest` + `querytest` + `snapshottest` into one import for full-stack backend testing.
47. **Add memory store test for concurrent Save + ReadFrom** — Race condition testing for the global journal.
48. **Review if `MemoryQueryStore` needs an `AppendBatch` equivalent** — Currently queries are saved one at a time. Is there a batch use case?
49. **Consider adding `ReadAll` to the suite** — Currently tests it as part of `testReadAll` but doesn't explicitly verify empty-store behavior.
50. **Add a session log entry** — Record this session's bug discoveries in `docs/sessions/SESSION_MILESTONES.md`.

---

## g) Questions I cannot figure out myself

1. **Should `command/v4.4.0` be tagged now?** The `commandtest` subpackage has existed for multiple sessions but is untagged. Every session that touches `storage/memory` or `storage/pebble` command store tests will hit the `GOWORK=off` failure until it's tagged. But tagging requires verifying ALL command module changes since `v4.3.0` are stable. Should I run the full verify gate and tag, or is there a reason this hasn't been tagged yet?

2. **Is the `TestBundle_RunProjections_GraphProjection` failure known?** It fails with a CBOR/JSON decode error (`invalid character 'X' at start of value`). This looks like the event payload is CBOR-encoded but the graph projection handler tries to JSON-unmarshal it. Is this a known issue from the CBOR default codec adoption, or is it new?

3. **The `metaengine/graphadapter/adapter_test.go` has 64 uncommitted lines adding `TestAdapter_StoreIntegration_RecordAware` — should I commit this or discard it?** It imports `record/v4` and tests Record-aware store integration. It's not my change but it's in the working tree. The auto-commit daemon seems to have skipped it (possibly because it doesn't compile yet due to missing `record` types).
