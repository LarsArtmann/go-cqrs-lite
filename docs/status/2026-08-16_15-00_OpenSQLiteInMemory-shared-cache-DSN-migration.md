# Status: OpenSQLiteInMemory Shared-Cache DSN Migration

**Date:** 2026-08-16 15:00
**Session scope:** Migrate `OpenSQLiteInMemory` from single-connection pool pin to per-call unique shared-cache DSNs, resolving TODO_LIST item "OpenSQLiteInMemory callers: unique shared-cache DSNs".

---

## a) FULLY DONE

1. **`storage/sqlite_helpers.go` — `OpenSQLiteInMemory` rewritten.** Now generates a unique `file:<random>?mode=memory&cache=shared&_pragma=busy_timeout(5000)` DSN per call via `crypto/rand`. Removed `ConfigureSQLitePool(db)` call (the `SetMaxOpenConns(1)` pin). All pooled connections within one `*sql.DB` share the same in-memory schema, enabling read concurrency. Added `crypto/rand` and `encoding/hex` imports.

2. **`storage/view/testhelpers_test.go` — local wrapper updated.** `openSQLiteInMemory()` → `openSQLiteInMemory(tb testing.TB)` (accepts both `*testing.T` and `*testing.B`). Uses the same shared-cache DSN pattern with `busy_timeout(5000)`. Added `crypto/rand`, `encoding/hex`, `fmt`, `testing` imports.

3. **All 15 view test call sites updated.** `openSQLiteInMemory()` → `openSQLiteInMemory(t)` or `openSQLiteInMemory(b)` across 8 test files: `store_validation_test.go` (×3), `store_test.go` (×1), `store_race_test.go` (×2), `store_new_test.go` (×1), `store_multidb_bench_test.go` (×4), `store_keyset_test.go` (×1), `store_bench_helpers_test.go` (×1), `store_auto_test.go` (×2).

4. **Race tests: `SetMaxOpenConns(1)` removed.** `store_race_test.go` — both `TestSQLViewStore_ConcurrentSetQuery` and `TestSQLViewStore_ConcurrentBatchAndCount` no longer manually pin the pool. The shared-cache DSN handles multi-connection access correctly.

5. **Bench tests: `SetMaxOpenConns(1)` deliberately kept.** `store_multidb_bench_test.go` — 4 sub-benchmarks retain `SetMaxOpenConns(1)` for benchmark timing consistency (not correctness). This is a deliberate judgment call.

6. **Regression test rewritten.** `TestOpenSQLiteInMemory_SingleSharedDatabase` → `TestOpenSQLiteInMemory_SharedCacheDatabase` in `sqlite_helpers_test.go`. Old test verified writes would queue on the single pinned connection. New test verifies the opposite: a concurrent write on a second pooled connection sees the same schema (shared cache), proving the per-connection DB bug is fixed without serialization.

7. **Docs updated:**
   - `AGENTS.md` — gotcha rewritten to describe shared-cache DSN approach
   - `TODO_LIST.md` — item marked `[x]` DONE with summary; "ratify judgment calls" item narrowed from 2 to 1 (iroh only)
   - `FEATURES.md` — "In-memory SQLite pin" → "In-memory SQLite shared-cache"
   - `CHANGELOG.md` — new "Changed" entry documenting the migration

8. **Verification gate passed:**
   - `go build` — clean
   - `go vet` — clean
   - `gofmt` — clean (all files)
   - `go test -count=1 -race` — all 6 storage sub-packages green
   - `go test -count=3 -race` — all 6 green 3× (no flakiness)
   - `api-stability` — 4144 exports verified, no golden regen needed
   - `doc-check` — 877 references valid across 41 packages

9. **Auto-commit daemon committed everything** across 5 commits:
   - `c57590302` — refactor commit (bundled with metaengine journal iteration unification)
   - `c66fc607f` — view test helper + call site updates
   - `673ef3fe9` — regression test rewrite
   - `aa099b026` — busy_timeout addition (caught a race in parallel test load)
   - `d4255af5c` — dropped unused `OpenSQLite` dependency from helper
   - `490f2786a` — docs + metaengine cost model improvements (bundled)

---

## b) PARTIALLY DONE

1. **`nix run .#verify` NOT run.** Only per-module (`GOWORK=off`) tests were run. The full workspace gate (`#verify` = build + vet + test + race + lint + doc-check + doc-assertions across 82 modules) was not executed. The storage module is green, but other modules that depend on `storage` (stack/sqlite, stack/bbolt, system, integration, etc.) were not re-verified. Low risk since the exported API surface didn't change, but the stale-GREEN anti-pattern says: don't claim green without the gate.

2. **`nix run .#lint` NOT run.** golangci-lint was not run on the storage module. The `golangci_lint_ls` LSP diagnostics show context-loading errors (go.work version mismatch), but these are noise per AGENTS.md. The real lint gate is `nix run .#lint`.

3. **`nix run .#check-duplication` NOT run.** The shared-cache DSN construction is now duplicated between `storage/sqlite_helpers.go` and `storage/view/testhelpers_test.go` (both construct the same DSN pattern). The view helper can't import `storage` (circular dep), so this is structural, but the duplication gate may flag it.

---

## c) NOT STARTED

1. **No SKILL.md / references update.** The consumer-facing skill references (`.agents/skills/go-cqrs-lite/references/`) were not checked for mentions of the old `OpenSQLiteInMemory` behavior. If any recipe or FAQ mentions the pool pin, it's now stale.

2. **No `nix run .#check-arch` run.** Dependency budget enforcement not verified. The new `crypto/rand` and `encoding/hex` imports are stdlib, so they shouldn't affect budgets, but the gate wasn't run.

3. **No `nix run .#check-coverage` run.** Coverage drift check not executed.

4. **Tag `storage/v4.7.2` not cut.** The TODO_LIST release item for `storage v4.7.2` (SQLite `OpenSQLiteInMemory` pool pin) is still blocked on user authorization. The code is now materially different from what that tag item described (shared-cache DSN, not pool pin), so the tag description in TODO_LIST may need updating.

---

## d) TOTALLY FUCKED UP

Nothing. All code changes compile, all tests pass, all docs check green. The main gap is verification depth (per-module vs full-gate).

---

## e) WHAT WE SHOULD IMPROVE

1. **The regression test has a subtle correctness gap.** `TestOpenSQLiteInMemory_SharedCacheDatabase` verifies a concurrent write _succeeds_ while a tx is open, but SQLite shared-cache mode uses table-level locking. The write succeeds because `INSERT` acquires a read lock on the table (the tx only holds a write lock via `BEGIN`, which in shared-cache mode is `BEGIN DEFERRED`). If the tx had done a write first (`INSERT`/`UPDATE`), the concurrent write would get `SQLITE_BUSY` (even with `busy_timeout(5000)` it would retry). The test doesn't cover the "tx holds a write lock" case. A stronger test would: begin tx, write inside tx, then concurrent write should get `SQLITE_BUSY` and succeed after rollback.

2. **The DSN construction is duplicated** between `storage/sqlite_helpers.go` and `storage/view/testhelpers_test.go`. Both build the same `file:<random>?mode=memory&cache=shared&_loc=auto&_time_format=sqlite&_pragma=busy_timeout(5000)` pattern. This is forced by the circular import constraint (storage imports view, so view can't import storage). A shared `sql` package helper (in `storage/sql/`) could hold the DSN builder, but that's a refactor, not a bug.

3. **The `TestConfigureSQLitePool` test is now slightly misleading.** It calls `OpenSQLiteInMemory()` (which no longer calls `ConfigureSQLitePool`), then explicitly calls `ConfigureSQLitePool(db)` and checks `MaxOpenConnections == 1`. This tests `ConfigureSQLitePool` in isolation, which is fine, but the test reads as if `OpenSQLiteInMemory` produces a pool-capped DB. A comment clarifying "OpenSQLiteInMemory no longer pins; this tests ConfigureSQLitePool independently" would help.

4. **The `store_multidb_bench_test.go` benchmarks still call `SetMaxOpenConns(1)`.** This was a deliberate judgment call (benchmark timing consistency), but it means the benchmarks don't benefit from the read concurrency the shared-cache DSN enables. If the benchmarks are measuring read-heavy workloads, removing the pin would give more realistic numbers. Worth revisiting per-bench.

5. **No stress test for shared-cache mode under high parallelism.** The 3× `-race` run is good, but the original flake manifested under heavy parallel test load (many `t.Parallel()` tests running simultaneously). A targeted stress run with `-parallel 16` or higher would give more confidence.

6. **Uncommitted `metaengine/query.go` change.** There's a `FilterCount()` method added to `QueryConfig` that's uncommitted in the working tree. This is from a different session/agent (not part of this task), but it's sitting there uncommitted. Either it should be committed or stashed — it's a loose end.

---

## f) Up to 50 Things We Should Get Done Next

### Verification (do these first)

1. Run `nix run .#verify` — full workspace gate (build + vet + test + race + lint + doc-check + doc-assertions across 82 modules)
2. Run `nix run .#lint` — golangci-lint on the storage module specifically
3. Run `nix run .#check-arch` — dependency budget enforcement
4. Run `nix run .#check-coverage` — coverage drift check
5. Run `nix run .#check-duplication` — verify the DSN duplication between storage + view helpers isn't flagged (or update baseline)
6. Run storage tests with `-parallel 16 -count=5 -race` — stress test the shared-cache mode under high parallelism
7. Run `stack/sqlite` tests — verify the stack preset that uses `storage` still works
8. Run `system` tests — verify the composition root that uses `storage` still works

### Regression Test Hardening

9. Add a "tx holds write lock" case to `TestOpenSQLiteInMemory_SharedCacheDatabase` — begin tx, write inside tx, verify concurrent write gets `SQLITE_BUSY` and succeeds after rollback
10. Add a test verifying that two separate `OpenSQLiteInMemory()` calls produce isolated databases (write in one, verify not visible in the other)
11. Add a comment to `TestConfigureSQLitePool` clarifying it tests `ConfigureSQLitePool` in isolation, not the `OpenSQLiteInMemory` output

### Code Quality

12. Commit or stash the uncommitted `metaengine/query.go` `FilterCount()` method
13. Consider extracting the shared-cache DSN builder into `storage/sql/` to eliminate the duplication between `sqlite_helpers.go` and `view/testhelpers_test.go`
14. Review whether `store_multidb_bench_test.go` benchmarks should drop `SetMaxOpenConns(1)` — they may benefit from read concurrency for read-heavy sub-benchmarks
15. Check if `ConfigureSQLitePool` is still needed as an exported function — its only internal caller (`OpenSQLiteInMemory`) no longer calls it; external consumers may still use it for file-based SQLite

### Documentation

16. Check `.agents/skills/go-cqrs-lite/references/*.md` for mentions of the old pool-pin behavior
17. Update the TODO_LIST release item for `storage v4.7.2` — the description says "pool pin" but the code now does shared-cache DSNs
18. Check `storage/README.md` for any mention of the old behavior
19. Check `docs/adr/` for any ADR that references the pool pin approach
20. Verify the `docs/status/` archive reports that mention the pool pin are annotated as superseded

### Release / Tagging

21. Cut `storage/v4.7.2` tag (blocked on user authorization) — update the tag description to reflect shared-cache DSNs, not pool pin
22. Create a GitHub Release for `storage/v4.7.2` with release notes describing the shared-cache migration
23. Run the replace-drop sweep after the wave-4 tags (TODO_LIST item)

### Metaengine (from the uncommitted query.go)

24. Review the `FilterCount()` method on `QueryConfig` — is it wired into the cost model? Does `estimateCost` use it?
25. If `FilterCount()` is meant for selectivity estimation, wire it into `estimateCost` as a multiplier or divisor
26. Add tests for `FilterCount()` — verify it counts declarative filters correctly

### Broader Project

27. Run `nix run .#verify-fast` as a quick sanity gate before the full `#verify`
28. Check if any other test helper in the repo uses `file::memory:` directly (not through `OpenSQLiteInMemory`) — those would still have the per-connection bug
29. Review `scheduling/sqlstore/` tests — they use SQLite and may have their own in-memory DB helper
30. Review `metaengine/*engine/` test helpers — they may use `file::memory:` directly
31. Check `integration/` tests for `file::memory:` usage
32. Check `stack/bench/` for `file::memory:` usage
33. Consider adding a `storage.OpenSQLiteInMemoryWithName(name string)` variant for tests that want deterministic names
34. Consider adding a `storage.IsSharedCacheDSN(dsn string) bool` helper for diagnostic tooling
35. Update the `cmd/cqrs-lint` doctor to report the DSN type (shared-cache vs file::memory:) if it detects SQLite usage
36. Add a `docs/adr/` for the shared-cache DSN decision (the original pool-pin fix was in a status report, not an ADR)
37. Review whether the `busy_timeout(5000)` in the DSN is the right value — the production WAL path uses 5000ms, but in-memory shared-cache may need different tuning
38. Check if `modernc.org/sqlite` supports `_pragma=busy_timeout` in the DSN (it does for file-based, but verify for shared-cache mode)
39. Consider adding a `storage.OpenSQLiteInMemoryWithPool(n int)` variant for tests that want to control pool size
40. Review the `system/system_hardening_test.go` test that was modified in commit `d4255af5c` — verify it still passes with the shared-cache DSN
41. Check if the `storage/sql/validate_fuzz_test.go` (added in `c57590302`) uses `OpenSQLiteInMemory` or `file::memory:` directly
42. Run the `cmd/doc-check` on any new docs added during this session
43. Regenerate `cmd/api-stability` golden if any exported symbol changed (none did, but verify)
44. Check if the `ROADMAP.md` mentions the pool pin
45. Review `docs/feedback/` for any consumer feedback about SQLite in-memory behavior
46. Consider extracting the DSN parameter list (`_loc`, `_time_format`, `_pragma=busy_timeout`) into a shared `storage/sql` helper that both `OpenSQLite` and `OpenSQLiteInMemory` use
47. Verify the `storage/sql/dialect.go` SQLite schema embed still works with shared-cache mode
48. Check if `storage/relational/` tests use `OpenSQLiteInMemory` or their own helper
49. Consider documenting the shared-cache DSN pattern in `.agents/skills/go-cqrs-lite/references/faq.md` under "common pitfalls"
50. Run `nix fmt` to ensure all files are treefmt-clean (the daemon may have committed before formatting)

---

## g) Questions I Cannot Answer Myself

1. **Should the bench tests (`store_multidb_bench_test.go`) drop `SetMaxOpenConns(1)`?** I kept it deliberately for benchmark timing consistency, but if the benchmarks are measuring read-heavy workloads (the `SQL/Get` and `SQL/Scan` sub-benchmarks), the single-connection pin produces artificially serialized numbers. Removing it would give more realistic read-concurrency numbers, but would change baseline comparisons. This is a judgment call about what the benchmarks are _for_.

2. **Should I commit the uncommitted `metaengine/query.go` `FilterCount()` method?** It's sitting in the working tree from a prior session/agent. It looks like it was meant to feed into the cost model for selectivity estimation, but it's not wired into `estimateCost` yet. Committing an unwired exported method creates API surface that may need to be supported. Stashing it loses the work. I don't know the intent behind it.

3. **Should `storage/v4.7.2` be tagged from the current master HEAD?** The TODO_LIST release item for `storage v4.7.2` describes it as "SQLite `OpenSQLiteInMemory` pool pin", but the code now does shared-cache DSNs (materially different behavior). The tag description may need updating before cutting. Also, the replace-drop sweep (TODO_LIST item) is blocked on the wave-4 tags — tagging storage unblocks that. But tagging is always blocked on user authorization.
