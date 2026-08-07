# Status: Test Suite Consolidation — bbolt contract, durability bench, commandtest/querytest extraction, for-loop modernization

**Date:** 2026-08-08 01:39
**Session start:** ~01:17
**Commits:** `cb5df1038`, `431c252c9` (auto-committed by daemon)

---

## a) FULLY DONE

### 1. `stack/bbolt/contract_test.go` — CREATED ✅
- Mirrors the pebble pattern exactly: `contracttest.RunSuite(t, factory)`
- `bbolt.New` returns `*stack.Bundle` directly (no `.Bundle` unwrap needed, unlike pebble/turso)
- All 5 subtests pass: BundleFields, EventRoundtrip, CommandRoundtrip, ReadModelRoundtrip, CloseIdempotent
- Verified: `go test -count=1 ./...` PASS

### 2. `stack/bench/durability_tiers_test.go` — bbolt benchmark ADDED ✅
- `BenchmarkDurabilityTiers_Bbolt` tests all 3 tiers (Strict, Normal, Relaxed) via `bbolt.WithDurability`
- Import `github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4` added
- `stack/bbolt/v4` was already in `stack/bench/go.mod` (line 85, direct dep)
- Smoke-tested with `-benchtime=1x`: Strict=6173, Normal=30470, Relaxed=38942 events/sec

### 3. Shared test packages extracted ✅
- **`command/commandtest/store_suite.go`** (283 lines) — `RunStoreSuite`, `MustCreateCommand`, `StoreSuite` interface, `StoreFactory` type, 6 subtests (SaveAndLoad, DuplicateDetection, AppendBatch, ReadAll, ReadFrom, LoadFromTimestamp)
- **`query/querytest/store_suite.go`** (191 lines) — `RunStoreSuite`, `MustCreateQuery`, `StoreSuite` interface, `StoreFactory` type, 4 subtests (SaveAndLoadQueries, DuplicateDetection, ReadAllQueries, ReadQueriesFrom)
- Both are sub-packages within their parent module (no separate go.mod needed) — `commandtest` is `command/v4/commandtest`, `querytest` is `query/v4/querytest`
- Pebble `command_store_test.go`: 259 → 39 lines (thin wrapper)
- Pebble `query_store_test.go`: 183 → 34 lines (thin wrapper)
- Bbolt `command_store_test.go`: 287 → 71 lines (thin wrapper + 2 bbolt-specific tests: AppendBatchDuplicate, LoadEmptyStream)
- Bbolt `query_store_test.go`: 163 → 12 lines (thin wrapper)
- **Net: 892 lines → 136 lines** across the 4 consumer files

### 4. For-loop modernization in `storage/bbolt/contract_test.go` ✅
- 4 instances of `for i := 0; i < N; i++` → `for i := range N`
- All 4 gopls `rangeint` hints cleared
- Tests pass

---

## b) PARTIALLY DONE

### Nothing partial — all 4 tasks were completed and verified

---

## c) NOT STARTED (things I noticed but didn't do)

1. **`storage/memory/` has the same duplicated tests** — `command_store_test.go` (316 lines), `query_store_test.go` (248 lines). These could ALSO be refactored to use `commandtest.RunStoreSuite` / `querytest.RunStoreSuite`. They have additional tests beyond the shared suite (journal-specific, closed-store behavior), but the core suite tests are duplicated.
2. **`commandtest` has no `doc.go`** — `querytest` has one (`doc.go` with package doc). I put the package doc in `store_suite.go` header instead. Consistency issue.
3. **`commandtest` has no test file of its own** — `[no test files]`. A meta-test that instantiates the suite against a memory store would validate the suite itself (mirrors how `eventtest` is tested).
4. **No `nix fmt` / `nix run .#lint` run** — I relied on `go build` + `go test` + `go vet`. The daemon may have formatted.
5. **AGENTS.md module list not updated** — `command/commandtest` is missing from the Modules row in the Quick Reference table (it lists `id/idtest`, `query/querytest` but not `command/commandtest`).
6. **TODO_LIST.md not updated** — The 4 items I completed are still listed as pending in TODO_LIST.md.

---

## d) TOTALLY FUCKED UP — Nothing

All changes compile, all tests pass (including `-race`), `go vet` is clean.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (this session's work)

1. **Run `nix fmt` on new files** — `command/commandtest/store_suite.go`, `query/querytest/store_suite.go`, `stack/bbolt/contract_test.go` were never formatted through treeformatter/gofumpt.
2. **Add `command/commandtest` to AGENTS.md** module list and api-stability modules list.
3. **Refactor `storage/memory/command_store_test.go` + `query_store_test.go`** to use the shared suites — same ~90% duplication pattern.
4. **Add `doc.go` to `commandtest`** for consistency with `querytest`.
5. **Add a self-test to `commandtest`** — run the suite against `storage/memory`'s `MemoryCommandStore` to validate the suite itself (not just consumer backends).
6. **Run `nix run .#verify`** or at least `nix run .#verify-fast` — I never ran the full verification gate. All claims are based on per-module `go test`, not the workspace-level gate.
7. **Check api-stability golden** — I added new exported symbols (`StoreSuite`, `StoreFactory`, `RunStoreSuite`, `MustCreateCommand`, `MustCreateQuery`). If `commandtest`/`querytest` are tracked by api-stability, the golden needs regen. (`querytest` IS in the api-stability modules list; `commandtest` is NOT — needs to be added first.)

### Architectural observations

8. **The `time` import in `durability_tiers_test.go`** is still suppressed with `var _ = time.Second` (line 123, pre-existing). The import is genuinely unused now — it should be removed, not suppressed.
9. **`commandtest.StoreSuite` embeds `command.Store` + `command.SeekableCommandJournal`** — but neither pebble nor bbolt's concrete `*CommandStore` type explicitly declares implementing this. It works because Go has structural typing, but if a backend implements `command.Store` without `SeekableCommandJournal`, the suite will fail at the type assertion. This is a feature (compile-time enforcement), not a bug — but it should be documented.
10. **`LoadToTimestamp` is not tested in the shared suite** — both pebble and bbolt implement it, but I only included `LoadFromTimestamp` in the suite. The original pebble test didn't test `LoadToTimestamp` either, so this is a pre-existing gap, not a regression.

---

## f) Up to 50 things to get done next

### Test consolidation (direct continuation)
1. Refactor `storage/memory/command_store_test.go` → use `commandtest.RunStoreSuite`
2. Refactor `storage/memory/query_store_test.go` → use `querytest.RunStoreSuite`
3. Check if `storage/` (SQL backend) has command/query store tests that could use the shared suites
4. Add `doc.go` to `command/commandtest/`
5. Add self-test to `commandtest` (run suite against `storage/memory.MemoryCommandStore`)
6. Add self-test to `querytest` (run suite against `storage/memory.MemoryQueryStore`)
7. Add `LoadToTimestamp` test case to `commandtest.RunStoreSuite`
8. Add `LoadEmptyStream` test case to the shared suite (currently bbolt-specific)

### Formatting & verification
9. Run `nix fmt` on all new/changed files
10. Run `nix run .#lint` — check for lint issues in new files
11. Run `nix run .#verify` or `nix run .#verify-fast` — full gate
12. Run `nix run .#check-duplication` — verify no new clones introduced
13. Run `nix run .#check-coverage` — verify coverage didn't regress

### Documentation
14. Update AGENTS.md Quick Reference module list (add `command/commandtest`)
15. Update TODO_LIST.md (mark the 4 items as done)
16. Add `command/commandtest` to `cmd/api-stability/main.go` modules list
17. Regen api-stability golden if needed (`cd cmd/api-stability && GOWORK=off go run main.go -update`)
18. Remove `var _ = time.Second` in `durability_tiers_test.go` + drop unused `time` import
19. Update the `stack/bench` README (if exists) to mention bbolt benchmark

### bbolt backend improvements
20. Run `nix run .#integration-all` to verify bbolt stack in cross-module integration
21. Check if `storage/bbolt` has a `go.sum` entry for `stack/bolt/v4` (shouldn't need it — stack depends on storage, not vice versa)
22. Add bbolt to `stack/bench` parity tests (if any exist beyond durability tiers)
23. Verify bbolt contract test runs in CI (`nixos-vm-tests` or GitHub Actions)

### Broader test infrastructure
24. Check if `storage/turso/` command/query store tests exist and could use shared suites
25. Create an `eventtest`-style package for snapshot store conformance tests (if duplication exists)
26. Create an `eventtest`-style package for checkpoint store conformance tests
27. Check if `kv/` store tests could benefit from a similar `kvtest` shared suite
28. Audit all `_test.go` files across modules for `for i := 0; i < N; i++` patterns (gopls rangeint hints)
29. Run `gopls` diagnostics workspace-wide to find remaining rangeint hints

### System package (uncommitted changes seen in git status)
30. Investigate the uncommitted `system/` changes (constructor.go, introspection.go, shutdown.go, system_hardening_test.go, system_internal_test.go) — these appeared between sessions, not from this session's work
31. Verify system package builds and tests pass with those changes

### Meta
32. Tag `command/v4` with the new `commandtest` sub-package (after go.mod is stable)
33. Tag `query/v4` with the updated `querytest` sub-package
34. Tag `stack/bbolt/v4` with the contract test
35. Tag `storage/pebble/v4` and `storage/bbolt/v4` with the refactored tests
36. Verify all module tags are monotonically increasing (AGENTS.md version-sequence rule)

---

## g) Questions I cannot figure out myself

1. **Should `storage/memory/` command/query store tests be refactored in this same pass, or is that a separate task?** The memory backend has 316+248 lines of command/query store tests with the same ~90% duplication. The shared suites are ready. But the memory tests also have additional test cases (journal-specific, closed-store behavior) that go beyond the shared suite — should I add those to the shared suite, or keep them as memory-specific?

2. **Should `commandtest` get its own go.mod (like `event/v4/eventtest` does) or stay as a sub-package of `command/v4`?** The `eventtest` pattern uses a nested go.mod because it has different dependency constraints (test-only deps). `commandtest` and `querytest` currently have zero extra deps beyond what `command/v4` and `query/v4` already require — so a nested go.mod seems unnecessary. But for consistency with `eventtest`, it might be expected. Which is preferred?

3. **The uncommitted `system/` changes in git status (constructor.go, shutdown.go, etc.) — are those yours from another session, or should I investigate/verify them?** I didn't touch those files, but they show as modified. I don't want to accidentally include them in a future commit.
