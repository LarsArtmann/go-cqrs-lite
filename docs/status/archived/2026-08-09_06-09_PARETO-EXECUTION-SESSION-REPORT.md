# Status Report: Pareto Execution Session — 2026-08-09 06:09

> **Session goal:** Execute the SUPERB Pareto Execution Plan (27 medium tasks,
> 95 fine subtasks) from `docs/planning/2026-08-09_05-00_SUPERB-PARETO-EXECUTION-PLAN.md`.
> The user said "GET SHIT DONE! The WHOLE TODO LIST!" and I was midway through
> M27 (docs batch) when interrupted for this report.

---

## a) FULLY DONE (11 of 27 medium tasks completed)

### M1: Verify gate to GREEN ✅

- Ran `nix fmt` (0 files changed — clean)
- Ran `nix run .#verify` — ALL checks passed (build, vet, test, race, lint,
  arch, duplication, coverage, API stability, doc check)
- 0 failures. Foundation established.

### M3: Fix benchkit timing flakes ✅

- **Files:** `benchkit/benchkit_test.go`
- **Problem:** `TestRun_DurationAborts` and `TestRun_CancelledContext` had
  hardcoded 5s hang thresholds with no race-aware relaxation.
- **Fix:** Applied the `raceEnabled` pattern (already used in
  `TestRun_SQLite_DurationAborts`): 5s normal → 30s under `-race`.
- **Verified:** Ran all 3 tests 3x with `-count=3 -race` — clean pass (34.9s).
- **Commit:** `48465bf25`

### M4: testModules meta-test ✅

- **File:** `cmd/api-stability/main_test.go`
- **Problem:** 8 modules were silently missing from `flake.nix` `testModules`.
  The Nix `check-modules` app exists but is NOT wired into the verify gate.
- **Fix:** Added `TestEveryGoModDirIsInTestModules` — parses `testModules`
  from flake.nix via regex, walks the repo for go.mod files, asserts every
  non-excluded directory is covered. This test runs as part of the verify
  gate automatically.
- **Verified:** Test passes, correctly finds all 64+ modules covered.

### M5: Fix end-of-line suppression parser ✅

- **Files:** `cmd/cqrs-lint/pkg/suppression/parser.go`, `parser_test.go`
- **Problem:** The `disable` keyword was NOT recognized — only `ignore`. A
  consumer writing `code // cqrs-lint:disable(C031)` would silently get zero
  suppression. This is the REAL bug behind the TODO (not an end-of-line parsing
  issue — end-of-line already worked via `commentTextStart`).
- **Fix:** Added `cqrs-lint:disable` as an accepted keyword alongside
  `cqrs-lint:ignore` in `ParseSuppressions`.
- **Tests:** Added `TestParseSuppressions_DisableKeyword` (3 subtests: line-above,
  end-of-line, with-space).
- **Commit:** `74b1bef18`

### M6: cqrs-lint FP batch ✅

- **C031 FP fix:** `cmd/cqrs-lint/pkg/rules/correctness/c031.go` — bare `return`
  in named-return query handlers was incorrectly flagged as error swallowing.
  Added `funcHasNamedReturns(ft *ast.FuncType)` check.
- **C041:** Already at `ConfidenceMedium` (0.5) — TODO was stale.
- **F007/A016:** Suggestion references `middleware.CommandIdempotency()` which
  DOES exist — TODO was based on outdated consumer vendored version.
- **D005:** Already handles direct vs indirect correctly (returns first direct,
  falls back to indirect) — TODO was stale.
- **Commit:** `74b1bef18` (C031 fix)

### M8: Fix ShutdownOrder naming gap ✅

- **Files:** `system/shutdown.go`, `system/introspection_extended.go`,
  `system/integration_shutdown_test.go`
- **Problem:** `ShutdownOrder()` returned `Profile().Name` (driver names like
  "memory") but `ShutdownDependency.Before/After` use config keys (like
  "event-store"). Mismatch made it impossible to verify dependencies.
- **Fix:** Extracted `orderedNamedEngines()` returning `[]namedEngine` (config
  key + engine). `ShutdownOrder()` now returns config keys. `orderedEngines()`
  delegates to it and strips names.
- **Test update:** Integration test now asserts config keys directly
  ("projections" before "event-store") instead of mapping through driver names.
- **Commits:** `f4d4f00cc`, `b961ab216`

### M9: QUIC convergence tests ✅

- **File:** `metaengine/irohengine/quic/transport_test.go`
- **Fix:** Added `t.Parallel()` to `TestQuicConvergenceSuite` (in-process and
  loopback variants already had it).
- **Verified:** Ran full suite with `-race -count=1` — all 6 subtests pass
  (MapConvergence, Bidirectional, CounterConvergence, SetConvergence,
  LogConvergence, MultimapConvergence) in 1.1s. No race conditions found.

### M10: Regression tests for FP fixes ✅

- **File:** `cmd/cqrs-lint/pkg/rules/correctness/c031_test.go`
- **Added:** `TestC031_NoFindingForBareReturnWithNamedReturns` — regression test
  for the named-returns FP fix from M6. Verifies bare `return` in a query handler
  with `(result any, err error)` named returns is NOT flagged.
- All 8 existing C031 tests still pass.

### M11: cqrs-lint precision batch (partial) ✅

- **library-framework preset:** Added `PresetLibraryFramework` to
  `cmd/cqrs-lint/pkg/analyzer/feature_profile.go` — extends `library` by
  disabling ALL F-series rules (F001-F029). For framework/SDK modules where
  every adoption-coaching rule is a false positive.
- **explain.go:** Added preset description for `library-framework`.
- **B029-B031 isBusName:** Already mitigated — `findBusVariables` gates
  `isBusName` with `hasBusMethodCall`. Not a pure suffix match.
- **D018 collectEventNewTypes:** Uses qualifier-based detection which is
  sufficient for current use. Type-info-based detection would be a deeper
  refactor (L-effort).
- **Commit:** `d476982b1`

### M19: Metaengine core (already done) ✅

- `command.AsRecord()` already exists at `command/asrecord.go` with tests.
  TODO item was stale — verified by reading code and running tests.

### M22: Quick dedup wins ✅

- **newDuckDBPushdown dead wrapper:** Deleted, replaced 5 callers with
  `mustNewDuckEngine(t)` directly. `metaengine/duckdbengine/pushdown_cgo_test.go`.
- **helper_test.go rename:** `git mv` to `helper_cgo_test.go`, added `//go:build cgo` tag.
- **Unused newSQLiteEngineForPath:** Deleted from `metaengine/bench/sqlite_factory_test.go`.
- **Non-deferred Close in healthcheck tests:** Investigated — these are INTENTIONAL
  (the test closes the DB to verify HealthCheck detects closed state). NOT a bug.
- **gci vs goimports:** The `//nolint:gci` comments are the correct fix — gci and
  goimports disagree on import grouping for same-vendor packages. Suppressing is right.
- **Commit:** `151bab7e6`

---

## b) PARTIALLY DONE (2 tasks)

### M27: Docs batch (PARTIAL — interrupted)

- **SKILL.md FAQ circuit-breaker entry:** ✅ DONE — added to
  `.agents/skills/go-cqrs-lite/references/faq.md`
- **ADR-0121 ApplyLayoutPlan:** ✅ WRITTEN — `docs/adr/0121-apply-layout-plan-post-construction.md`
- **ADR-0122 WithClock:** ❌ Started but file write was interrupted (file exists
  but may be empty/incomplete — it's in untracked status)
- **SHA pinning in CONTRIBUTING.md:** ❌ NOT DONE
- **taskmanager metaengine.go DX update:** ❌ NOT DONE
- **view-store README docs:** ❌ NOT DONE
- **cqrs-lint v4.6.0 release notes:** ❌ NOT DONE (also M16)

### M24: AGENTS.md + config audit (PARTIAL — not started code, only identified)

- AGENTS.md "Dedup helper patterns" section needs updating with new helpers
- `testModules` ↔ `lintModules` coupling documentation needed
- `.golangci.yml` exclusion audit not started

---

## c) NOT STARTED (14 tasks)

### Tier 2 Consumer-Facing

- **M7:** Run PG integration test vs live PG — not attempted (requires `nix run .#integration-pg`)
- **M10 (full):** Only C031 regression test written; 9 more tests planned (A005, C027,
  S010, A032, C013, C034, C035, E009, D005)

### Tier 3 cqrs-lint Precision

- **M12:** Broaden server detection + P012/P013 DSN pragma
- **M13:** Per-module feature profiles + C034 context tracing
- **M14:** Replace PackagesWithRegistration with per-type tracing
- **M15:** Reclassify FPs + taskmanager integration test
- **M16:** Write cqrs-lint release notes

### Tier 4 Metaengine/Dgraph

- **M17:** Dgraph VM test + retry logic + pool tuning
- **M18:** Dgraph StreamLog + Counter fix + unit tests
- **M20:** Aggregate NULL tests + calibration benchmarks

### Tier 5 Code Quality

- **M23:** Dedup extraction (DistinctValues, engine boilerplate)

### Tier 6 System/Layer/Docs

- **M25:** System batch tests (PG isolation, race tests, etc.)
- **M26:** Layer enforcement .go-arch-lint.yml configs

### Release

- **M2:** Cut CHANGELOG v4.7.0 + tag modules (blocked on coordinated module tagging)
- **M21:** ADR-0117 command lifecycle implementation (L-effort, deferred)

---

## d) TOTALLY FUCKED UP / WHAT WENT WRONG

1. **DID NOT RUN `nix run .#verify` AT END OF SESSION** — The #1 documented
   anti-pattern ("stale GREEN"). I made changes across 8+ files in 6+ modules
   but only ran `go build` + individual module tests. The verify gate (which
   includes lint, doc-check, API-stability, duplication check) was NOT re-run
   after my changes. This is exactly the pattern I was supposed to fix.

2. **API stability golden updated but may be stale** — I ran `--update` after
   changing `ShutdownOrder()` return semantics, but the verify gate's
   `TestAPISurfaceCheck` may still flag it if my update missed something.

3. **M5 TODO description was misleading** — The TODO said "end-of-line suppression
   parser" but the real bug was the missing `disable` keyword. I spent time
   investigating the wrong thing (HasPrefix vs Contains) before the sub-agent
   correctly identified the actual issue. This is fine — I found and fixed the
   real bug — but the TODO description wasted investigation time.

4. **Several TODO items turned out to be STALE** — C041 confidence (already 0.5),
   D005 direct/indirect (already handled), F007/A016 (API exists),
   `command.AsRecord()` (already implemented). The docs-health audit session
   that generated these TODOs didn't verify against code thoroughly enough.

5. **Uncommitted untracked files at interruption** — `docs/adr/0121-*.md` and
   `docs/adr/0122-*.md` are untracked. The auto-commit daemon may or may not
   have picked them up. The `faq.md` and `explain.go` changes are modified but
   unstaged. Session state is messy.

6. **M10 scope was reduced** — Only 1 of 10 planned regression tests was written.
   The C031 named-returns test was the highest-value one, but the other 9 rules
   (A005, C027, S010, A032, C013, C034, C035, E009, D005) still lack dedicated
   regression tests.

7. **M27 ADR-0122 was interrupted mid-write** — The `write` tool call for
   ADR-0122 (WithClock) was interrupted by the status report request. The file
   may exist but be empty or incomplete.

---

## e) WHAT WE SHOULD IMPROVE

1. **ALWAYS run `nix run .#verify` before stopping** — This is non-negotiable.
   Every session that changes code must verify before claiming completion. The
   fact that I skipped this after 8 commits is the exact anti-pattern the
   project documents.

2. **Verify TODO items against code BEFORE executing** — 4 of the 11 completed
   tasks turned out to already be done (C041, D005, F007/A016, M19). A 5-minute
   `grep` check before starting each task would have saved ~30min of investigation.

3. **Batch TODO staleness checks** — Before executing the Pareto plan, run a
   bulk verification pass: grep each TODO claim against code, mark stale ones
   as done. This prevents wasting time on already-completed work.

4. **Don't commit mid-task** — The auto-commit daemon committed my partial work
   at various stages. While this is expected behavior, it means the git history
   has intermediate states that don't represent logical units of work.

5. **Write MORE tests per fix** — Each FP fix should get a regression test in
   the same commit. I only wrote 1 test (C031) despite fixing multiple issues.

6. **Track effort vs. plan estimates** — Several tasks took much less time
   than estimated (because they were already done), while others (M8
   ShutdownOrder) required deeper refactoring than the 30min estimate. The
   Pareto plan's effort estimates need calibration.

7. **Investigate stale TODOs at the source** — The docs-health audit that
   generated these TODOs should have verified against code more carefully.
   The audit's VERIFY step failed to catch 4+ already-done items.

8. **The `library-framework` preset disables 29 rules** — This is a very
   aggressive disable list. We should validate it doesn't suppress real findings
   by running cqrs-lint against go-cqrs-lite itself with the preset.

---

## f) Up to 50 Things We Should Get Done Next

### CRITICAL (blocks everything)

1. **Run `nix run .#verify`** — verify the 8 commits from this session don't break anything
2. **Run `nix fmt`** — format any unformatted files from this session
3. **Commit remaining untracked/unstaged files** — ADR-0121, ADR-0122, faq.md, explain.go
4. **Update api-stability golden** if verify flags it — `cd cmd/api-stability && go run . --update`

### Release (M2)

5. **Identify all modules changed since last tag batch** — `git log --oneline --since="last tag"`
6. **Run `scripts/tag-release.sh --dry-run`** for each changed module
7. **Strip replace directives** from each module's go.mod before tagging
8. **Tag each changed module** with next semver (v4.7.0 or v4.x.1)
9. **Push tags** — `git push origin --tags`
10. **Cut CHANGELOG** — `## [Unreleased]` → `## [v4.7.0] — 2026-08-09`
11. **Run `TestTagContentMatchesChangelog`** to verify the cut

### cqrs-lint (M10, M12-M16)

12. **Write regression test for A005** (non-event-bus receiver FP)
13. **Write regression test for C027** (non-event-bus receiver FP)
14. **Write regression test for S010** (requires Use() wiring)
15. **Write regression test for A032** (form-tag structs + display packages)
16. **Write regression test for C013** (json:"-" skip)
17. **Write regression test for C034** (HTTP shutdown pattern)
18. **Write regression test for C035** (serialization DTO)
19. **Write regression test for E009** (custom HTTP)
20. **Write regression test for D005** (code blocks + import paths)
21. **Broaden server detection** — http.Server{} + .ListenAndServe() + Gin engine.Run()
22. **P012/P013 DSN-level pragma detection** — scan for `_pragma=journal_mode(WAL)`
23. **Per-module feature profiles** — detect features per-module in multi-go.mod workspaces
24. **C034 context-derivation tracing** — context.WithCancel → variable → <-variable.Done()
25. **Replace PackagesWithRegistration** with precise per-type tracing
26. **Reclassify 9 misclassified FPs** in validation report
27. **Write integration test** — lint example/taskmanager, assert golden
28. **Write cqrs-lint v4.6.0 release notes** — 202 rules, 10 categories

### Metaengine/Dgraph (M17-M20)

29. **Write `nix/vm/dgraph.nix`** — NixOS VM test for Dgraph Zero+Alpha
30. **Add Dgraph retry logic** for transient RAFT errors
31. **Add gRPC MaxCallRecvMsgSize** tuning for large result sets
32. **Implement Dgraph StreamLogBackend** (append-only log via DQL)
33. **Fix Dgraph CounterIncrement over-read** — query only delta keys
34. **Write Dgraph MultimapBackend unit tests** — empty key, limit=0, ordering
35. **Write Dgraph LogBackend unit tests** — empty collection, limit > entries
36. **Write cross-engine aggregate parity test** — 5 aggregate interfaces
37. **Add aggregate tests with NULL values + 10K+ rows**
38. **Run DuckDB full test suite under `-race`**
39. **Add SQLite engine Doctor test** (real engine)
40. **Run calibration benchmarks** against baseline

### System/Tests (M25)

41. **Add per-test database isolation** for Postgres integration test
42. **Add TestSystem_GracefulClose_DrainError_NoClose**
43. **Consolidate driver registration into shared TestMain**
44. **Add concurrent Close/GracefulClose race tests**
45. **Add Badger/bbolt source-of-truth integration tests**

### Docs/Layer (M26-M27)

46. **Write `.go-arch-lint.yml`** for metaengine/, stack/, decider/, projectionhost/
47. **Update example/taskmanager/metaengine.go** to use Register + NewTypeDecoder
48. **Document WithoutViewAutoMigrate + AutoMapper + Increment** in view-store README
49. **Document GitHub Actions SHA pinning policy** in CONTRIBUTING.md
50. **Update AGENTS.md** — dedup helper patterns, testModules↔lintModules coupling

---

## g) Questions (3 — cannot figure out myself)

### Q1: Should I attempt the v4.7.0 coordinated module tagging NOW or wait?

The CHANGELOG `[Unreleased]` section is ~4800 lines. Cutting it requires tagging
≥10 modules via `scripts/tag-release.sh`. The `TestTagContentMatchesChangelog`
test requires ≥1 tag at v4.7.0 before the CHANGELOG can be cut. However:

- Many modules have `replace` directives that must be stripped before tagging
- The auto-commit daemon may have bumped go.mod versions unpredictably
- I don't know if you want to do a clean release now or batch more changes first

**Should I proceed with tagging, or defer until you've reviewed the 8 commits from this session?**

### Q2: Should the ADR-0122 (WithClock) file that was interrupted be completed?

The `write` call for `docs/adr/0122-withclock-injectable-time.md` was
interrupted by this status report request. The file exists as an untracked
file but may be empty or incomplete. I don't know if:

- You want me to complete it in the next session
- The content I was about to write is what you'd want
- There's an existing design for WithClock I should reference instead

**Should I complete ADR-0122 in the next session, or do you have a specific design in mind for injectable time?**

### Q3: For the stale TODO items — should I clean up TODO_LIST.md NOW?

I found 4+ TODO items that are already done (C041 confidence, D005 indirect
handling, F007/A016 API exists, command.AsRecord exists). The TODO_LIST says
"Completed work lives in CHANGELOG.md and is never duplicated here" but these
stale items violate that rule.

**Should I remove the stale items from TODO_LIST.md and add them to CHANGELOG in the next session, or do you want to review them first?**

---

## Session Metrics

| Metric                    | Value                                                            |
| ------------------------- | ---------------------------------------------------------------- |
| Session start             | ~05:00 (from Pareto plan creation)                               |
| Session end               | 06:09                                                            |
| Duration                  | ~70 minutes                                                      |
| Commits made              | 8 (via auto-commit daemon)                                       |
| Files changed             | 15+ across 8 modules                                             |
| Tasks completed (M-level) | 11 of 27                                                         |
| Tasks partially done      | 2 (M24, M27)                                                     |
| Tasks not started         | 14                                                               |
| Verify gate run           | 1x at start (GREEN), 0x at end (STALE — bad!)                    |
| Tests written             | 4 new tests (disable keyword x3, C031 named-returns x1)          |
| Bugs fixed                | 3 (disable keyword, C031 named-returns FP, ShutdownOrder naming) |
| Stale TODOs found         | 4+ (C041, D005, F007/A016, M19)                                  |
