# Status Report: Bench Fold Fix, Lint Cleanup, Driver Consolidation

> **ARCHIVED 2026-08-11 — All work in this report is complete. Open items were resolved by later sessions, captured in TODO_LIST.md, or determined to be minor polish. Original content retained below for historical context.**

**Date:** 2026-08-11 06:42
**Session start:** ~06:28 (resumed from prior session handoff)
**Branch:** master
**Head commit:** `f74ef5f27` (plus uncommitted M19 driver consolidation)

---

## a) FULLY DONE

### 1. Fixed `metaengine/bench` fold reflect panic (THE verify-gate blocker)

**Commit:** `7ba946377` — fix(metaengine): reify prev value in OnRecord update folds

The prior session's verify gate had exactly 3 failing tests:
`TestPromise_CostModelAccuracy`, `TestPromise_CrossEngine_ParityAtScale`,
`TestPromise_ParityWithDuckDB` — all caused by the same root cause:

**Root cause:** `OnRecord` update folds in `record_fold.go:115` passed the raw
`prev` value directly to `reflect.ValueOf(prev)` without type reification. On SQL
engines (sqlite, duckdb, pg, mysql), `prev` arrives as `map[string]interface{}`
from JSON deserialization, causing `reflect.Call` to panic because
`map[string]interface{}` is not assignable to the typed struct (e.g. `OrderView`).

The plain `On` path already handled this correctly via `reifyReflect()` in
`fold.go:347`. The fix mirrors that pattern for `OnRecord`:

```go
// Before (broken):
args = append(args, reflect.ValueOf(prev))
// After (fixed):
args = append(args, reifyReflect(prev, prevType))
```

All 3 tests now pass. Full `nix run .#verify` is now GREEN (zero failures).

### 2. Deleted dead `dgraphengine/retry.go`

**Commit:** `7ef711eea` (auto-daemon committed alongside other changes)

`retry.go` contained `withRetry[T any]` and `isTransientError` — two functions
with **zero callers** anywhere in the codebase. Confirmed via ripgrep: the only
references to these functions were the definitions themselves and the
self-referencing call from `withRetry` to `isTransientError`. The `unused` lint
exclusion in `.golangci.yml` for this file was also removed.

### 3. Fixed `nilerr` findings in `cmd/api-stability/main_test.go`

**Commit:** `7ef711eea` (auto-daemon)

Two `filepath.Walk` callbacks in `TestMultiPackageModulesHaveArchLintConfig`
swallowed `subErr` by returning `nil` instead of propagating the error:

```go
// Before: if subErr != nil || sub == path { return nil }
// After:  if subErr != nil { return subErr }; if sub == path { return nil }
```

Applied to both walk callbacks (lines 599 and 612). Narrowed the api-stability
lint exclusion to drop `nilerr` and `nolintlint` (only `gocognit` remains).

**Commit:** `ebf09923e` — chore(lint): narrow api-stability exclusion

### 4. Documented metadataPayload serialization pattern in recipes.md

**Commit:** `9db708c11` (auto-daemon)

Added a "Metadata Serialization in KV Engines" section to `recipes.md` documenting
the `id.ActorID` → JSON-in-CBOR-envelope pattern used by `storage/pebble` and
`storage/bbolt`. Doc-check passes: 708 references valid across 42 packages.

### 5. Ran full `nix run .#verify` gate

Ran to completion with all 82+ modules passing (after the bench fix). The ONLY
failure before the fix was the 3 bench tests above. After the fix, all pass.

---

## b) PARTIALLY DONE

### 1. M19: Driver registration consolidation (UNCOMMITTED)

Created `system/main_test.go` with `TestMain` and centralized all pure-Go driver
blank imports (badger, pebble, sqlite, postgres). Created `system/main_cgo_test.go`
with the CGo-gated duckdb import. Removed scattered blank imports from 4 individual
test files. Deleted `system/engines_test.go` (content absorbed into main_test.go).

**Status:** Code changes done, tests pass (`TestIntegration_BadgerSource_HealthCheck`
PASS), but **NOT committed**. Working tree has 7 staged files.

### 2. M15: Lint exclusion audit

Done:

- Deleted dead `dgraphengine/retry.go` + removed its `unused` exclusion
- Fixed `nilerr` in `api-stability/main_test.go` + narrowed exclusion
- Narrowed api-stability exclusion (dropped `nilerr`, `nolintlint`)

NOT done:

- `flightrecorder/alias.go` (13 `deprecatedComment` findings) — still excluded
- `id/actor_id.go` (16 findings) — still excluded
- `mysqlengine` `sqlclosecheck` — still excluded
- Full categorization of all ~40 exclusion blocks as permanent vs temporary

---

## c) NOT STARTED

From the 27-task Pareto plan (`docs/planning/2026-08-11_04-12_pareto-comprehensive-plan.html`):

- **M8** — Universal ADT coverage per engine (recursive CTE, brute-force vector, StreamLog)
- **M9** — Struct-composition-driven multi-collection
- **M11** — ADR-0117 command lifecycle as event streams
- **M13** — Calibration benchmarks vs baseline + CI regression check
- **M18** — Per-test database isolation for PG integration test
- **M20** — Tombstone vocab rename (NEEDS USER DECISION)
- **M21** — Per-module feature profiles for cqrs-lint
- **M22** — Redis/NATS/Dgraph actual Go integration tests
- **M23** — macOS verification of ephemeral PG
- **M24** — Move CGo DuckDB test to sub-module
- **M25** — v5 deletions (gated behind auto-projection)
- **M26** — v5 migration guide + cut v5.0.0
- **M27** — Nix apps + infra polish

---

## d) TOTALLY FUCKED UP

### 1. Forgot the status report — AGAIN

You asked for a status report. I got absorbed in executing M19 (driver
consolidation) and kept coding instead of writing the report first. This is the
same anti-pattern called out in the prior session's handoff. The report should
have been the first thing I produced when you said "status update."

### 2. Didn't commit M19 before you asked for the report

I left 7 staged files uncommitted in the working tree. The prior session's
handoff explicitly called out "9 uncommitted files" as a problem, and I
repeated the exact same mistake.

### 3. Shallow scope — only fixed 3 things this session

The session produced: 1 bug fix (reify), 1 dead-code deletion (retry.go),
2 lint fixes (nilerr), 1 doc update (recipes.md), and 1 uncommitted refactor
(M19 driver consolidation). Against a 27-task plan with 150 atomic tasks,
this is thin. The reify fix was high-value (unblocked verify) but the rest
was low-impact polish.

---

## e) WHAT WE SHOULD IMPROVE

### Process

1. **Status report FIRST, always.** When asked for a status report, write it
   immediately. Do not "just finish one more thing first." The report IS the task.

2. **Commit before reporting.** Never leave uncommitted work when producing a
   status report. The working tree state is part of the status.

3. **The `reifyReflect` pattern needs a test.** The bug existed because the
   `OnRecord` path diverged from the `On` path without a test catching the
   divergence. There should be a property test: "for every fold created via
   `OnRecord`, the update path must handle `map[string]interface{}` prev values
   identically to typed struct prev values." Without this, the same bug class
   can recur if a third fold constructor is added.

4. **The verify gate should have been run FIRST.** I started by reading the
   Pareto plan and trying to pick tasks. I should have run `nix run .#verify`
   first to establish the baseline — it would have immediately surfaced the 3
   failing bench tests and focused the session on the highest-impact fix.

### Code

5. **`reifyReflect` is duplicated knowledge.** The function exists in `reify.go`
   and is called from both `fold.go` (On path) and `record_fold.go` (OnRecord
   path). But the call sites are independent — there's no shared "invoke update
   fold" helper that enforces reification. A future fourth fold constructor
   could make the same mistake. Consider extracting a `invokeUpdateFold(handler,
   event, prev)` helper.

6. **The `dgraphengine/retry.go` dead code existed for weeks.** It was added
   "prepared for future wiring" but never wired. This is YAGNI violation. The
   `unused` lint exclusion masked it. Lesson: never add an exclusion for code
   you plan to wire "later" — wire it now or don't add it.

---

## f) Up to 50 Things to Do Next

### Immediate (uncommitted work)

1. **Commit M19 driver consolidation** — 7 staged files in working tree
2. **Run `nix run .#verify-fast`** after M19 commit to confirm system/ tests pass

### Pareto plan remaining tasks (by priority)

3. **M9: Struct-composition multi-collection** — `[]Attachment` → secondary collection
4. **M8: Universal ADT coverage** — recursive CTE graph (sqlite/pg/mysql), brute-force vector (memory/pebble), StreamLog (dgraph)
5. **M11: ADR-0117 command lifecycle** — DLQ + retries as event streams
6. **M13: Calibration benchmarks vs baseline** — add CI regression check
7. **M18: Per-test PG isolation** — wire pgtestcontainer per-test-database pattern
8. **M27: Nix apps + infra polish** — `#check-lint-config`, `#verify-ci`, register.go consolidation

### Lint audit remaining (M15)

9. **Fix `flightrecorder/alias.go`** — 13 `deprecatedComment` findings (reformat deprecation notices)
10. **Fix `id/actor_id.go`** — 16 findings (constants, receiver naming, `strings.Cut` instead of `strings.IndexByte`)
11. **Fix `mysqlengine` sqlclosecheck** — use `CloseRows` indirection like pgengine
12. **Categorize all ~40 `.golangci.yml` exclusions** — permanent vs temporary with removal conditions

### Regression prevention

13. **Add property test for `OnRecord` update fold reification** — catch `map[string]any` → typed struct mismatches
14. **Extract `invokeUpdateFold` helper** — centralize the reify+call pattern to prevent future divergence
15. **Add a test that exercises update folds on sqlite engine specifically** — the bench tests caught this, but only because they happened to use sqlite; a focused unit test would be faster

### Verify gate hardening

16. **Run full `nix run .#verify` after M19 commit** — confirm zero failures
17. **Run `nix run .#check-arch`** — dependency budget after system/ changes
18. **Run `nix run .#check-duplication`** — no-new-clones gate
19. **Run `nix run .#check-coverage`** — coverage drift check

### Metaengine feature work

20. **M9 design: how `[]Attachment` maps to a secondary collection** — collection naming, key derivation, join-aware read path
21. **M9 implementation: extend TypeInspector** — detect slice fields, generate secondary collection plan
22. **M8 design: recursive CTE graph traversal on sqlite** — schema, query template, degraded cost
23. **M8 implementation: brute-force vector search on memory engine** — cosine similarity scan
24. **M8 implementation: StreamLog fallback on dgraph** — append-ordered nodes
25. **M11 design: DLQ as event streams** — event types, fold, projection
26. **M11 design: retries as event streams** — scheduler integration

### Testing infrastructure

27. **M18: Wire pgtestcontainer into system PG test** — per-test database isolation
28. **M13: Run calibration benchmarks** — capture baseline numbers for all engines
29. **M13: Compare to `calibration-baseline.md`** — fix any drift
30. **M13: Add CI regression check** — bench threshold alert

### Documentation

31. **Update TODO_LIST.md** — mark M15 items done, add M19 completion
32. **Update CHANGELOG.md** — add bench fix, lint cleanup, driver consolidation
33. **Update AGENTS.md** — document the `reifyReflect` pattern as a required call for any new fold constructor
34. **Add `TestMain` convention to AGENTS.md** — system/ uses centralized TestMain; new integration tests don't need blank imports

### Polish (Tier 3)

35. **M27: Add `#check-lint-config` nix app** — validate `.golangci.yml` excluded paths exist
36. **M27: Add `#verify-ci` nix app** — mirror GH Actions GOWORK=off per-module
37. **M27: Consolidate engine `register.go` boilerplate** — 7 modules share the same pattern
38. **M27: Audit indirect deps in `metaengine/go.mod`** — modernc sqlite chain
39. **M27: Add property-based tests for metadataPayload CBOR roundtrip**

### Engine parity gaps

40. **Add `CounterIncrement` benchmark to pebbleengine** — badger and bbolt have it
41. **Add bboltengine parity tests** — `edge_cases_test.go`, `fuzz_test.go`, `stream_log_test.go`, `watcher_test.go`
42. **Write dgraphengine ProbeEngine integration test** — mirrors pgengine probe_live_test.go

### Live-latency polish

43. **Add `sync.Once` dedup to ProbeEngine warning** — prevents repeated warnings for same broken engine
44. **Add Prometheus counter for missing TrackerHost** — `probe_engine_missing_tracker_host_total`
45. **Test `Store.StartAutoReplan` lifecycle** — start, drift detection, stop, cleanup

### v5 preparation (gated on M9)

46. **M25: Delete `stack.Materialize`** — after auto-projection is production-ready
47. **M25: Delete `storage.RelationalProjection` + `storage/view`**
48. **M25: Delete `stack.Bundle` + all 8 stack presets**
49. **M26: Write v5 migration guide** — v4 tiers → v5 system.System
50. **M26: Cut v5.0.0** — tag all modules, run full verify

---

## g) Questions

### Q1: Should I commit the M19 driver consolidation as-is?

The code changes are done (7 files: 1 deleted, 4 modified, 2 new) and the
badger integration test passes. But I haven't run the full `nix run .#verify`
after M19. Should I commit now and verify after, or verify first?

### Q2: Should the `reifyReflect` fix get a dedicated regression test?

The bug was caught by existing bench parity tests, but those are slow (DuckDB
test takes 6.6s). A focused unit test that creates an `OnRecord` update fold,
feeds it a `map[string]interface{}` prev value, and asserts no panic would be
faster and more targeted. Should I add this before moving on?

### Q3: M20 (tombstone vocab rename) is marked NEEDS-DECISION — should I defer it?

The rename has large blast radius (`OnTombstone`, `OnRebirth`,
`isMaterializedTombstoned`, `kv.TombstoneQuerier`, `AutoMapperWithTombstone`,
`TombstoneColumn`, `IsTombstoned`). It's a breaking change. The plan says
"unblock now with aliases or defer to v5." Which path do you prefer?
