# Session 6: cqrs-lint Metaengine Follow-up + Daemon Drift Repair

**Date:** 2026-08-07 21:36
**Session start commit:** `9d690a9f4`
**Session end commit:** `0ea5c2a82`
**Branch:** master

---

## Context

This session continued work from session 5 (`docs/status/2026-08-06_20-10_cqrs-lint-metaengine-session5-execution.md`).
The user asked to "execute the prioritized next 50 things" and then requested a
comprehensive self-review status report.

The session 5 report listed 4 next steps:

1. Fix 14 metaengine pre-existing test failures
2. Implement P2.11 (F021 per-query fold analysis)
3. Scorecard metaengine section
4. Full `nix run .#verify` pass

---

## What Actually Happened (The Brutal Truth)

### The #1 Lesson: Status Reports Are Point-in-Time

The session 5 report said "14 metaengine tests fail from ADR-0115 SQLite engine extraction."
AGENTS.md explicitly warns: _"Status reports are point-in-time, not living documents. When a prior
session's report says 'X is broken', re-verify before treating that as current truth."_

I re-verified FIRST. Result: **all 14 metaengine tests were already fixed** by the auto-commit
daemon's refactoring commits (engine test extraction, keycodec consolidation, pushdown test
harness extraction, etc.). The "blocking" issue was already gone.

### What I Actually Found and Fixed

The daemon had introduced NEW failures while fixing the old ones:

- **Module catalog drift**: `metaengine/bench` and `testutil/pgtestcontainer` added to `go.work`
  but not registered in cqrs-lint's catalog exclusion list or api-stability's modules list
- **Golden file drift**: 7 modules had missing/stale golden test fixtures (JSON format changes
  from `encoding/json/v2` adoption)
- **Dependency version drift**: `retry/go.mod` pinned `go-retry` v0.1.0, but the daemon had
  refactored `go-retry` to return `(Duration, error)` from `Backoff`/`ComputeDelay` (v0.2.0)
- **Pre-existing lint issues**: `system/config_loader.go` depguard violations, `quic/transport_test.go`
  `containedctx` violation (NOT mine, left untouched)

---

## a) FULLY DONE

### 1. Module Catalog Drift Fix

**Files:** `cmd/cqrs-lint/pkg/analyzer/module_catalog_test.go`

- Added `metaengine/bench` and `testutil/pgtestcontainer` to the exclusion list
- Test `TestCatalogEveryGoWorkModuleCovered` now passes

### 2. API Stability Coverage Fix

**Files:** `cmd/api-stability/main.go`, `docs/api_surface.txt`

- Added `testutil/pgtestcontainer` to the modules slice
- Regenerated api_surface.txt (3647→3728 exports, +81 from pgtestcontainer)
- Test `TestEveryGoModDirIsInModulesList` now passes

### 3. Golden File Regeneration (7 modules)

**Files:** 24 golden JSON/SQL files across 7 modules

- `listing/`: page.json, stream-status.json (NEW — `.json` variant alongside `.snap`)
- `schema/`: upcaster-output.json
- `signing/`: hmac-signed-metadata.json, signature-json.json
- `snapshot/`: snapshot-structure.json, every-n-events-strategy.json
- `storage/`: 12 SQL golden files (postgres/sqlite/duckdb × events/commands/snapshots/checkpoints)
- `storage/pebble/`: event-store-roundtrip.json
- `watermill/`: message-metadata.json

### 4. Retry Dependency Bump

**Files:** `retry/go.mod`, `retry/go.sum`

- Bumped `github.com/larsartmann/go-retry` from v0.1.0 → v0.2.0
- Aligns with the daemon's API change: `Backoff`/`ComputeDelay` now return `(time.Duration, error)`
- `retry/` module builds clean in both workspace and GOWORK=off mode

### 5. F021 Per-Query Fold Analysis (P2.11)

**Files:** `cmd/cqrs-lint/pkg/rules/adoption/f020_f021.go`, `f018_f021_test.go`

- Rewrote F021 to inspect each `metaengine.Query` call individually
- New `findQueriesWithFolds()` scans for `metaengine.Query` calls and counts direct `On`/`OnTyped`
  fold arguments per query
- Only queries with 3+ folds trigger a finding (not global fold count)
- Fallback: if folds exist but no direct `Query` calls are found (indirect passing), uses old
  global count with `ConfidenceLow`
- Finding message now includes the query name and fold count:
  `"metaengine query "q" has 5 fold declarations — high write amplification..."`
- **2 new tests:**
  - `TestF021_PerQueryPrecision` — 2 queries × 2 folds each (4 total) → NO finding (old code false-positived)
  - `TestF021_MultipleQueriesOneAmplified` — 1 query with 1 fold + 1 with 3 folds → exactly 1 finding

### 6. Scorecard Metaengine Section

**Files:** `cmd/cqrs-lint/scorecard.go`, `scorecard_render.go`, `scorecard_test.go`

- New struct: `ScorecardMetaengine{Detected, Engines, PushdownAdopted, Suggestion}`
- `ComputeScorecard()` populates it from `FeatureProfile.HasMetaengine/Engines/Pushdown`
- Rendered in all 3 output formats:
  - **Text**: `METAENGINE` section with Detected/Engines/Pushdown lines
  - **Markdown**: `### Metaengine` section with bold key-value pairs
  - **JSON**: `metaengine` field in scorecard result object
- Suggests pushdown adoption when `FilterOnField`/`SortOnField` not detected
- **4 new tests:** DetectedWithPushdown, DetectedWithoutPushdown, NotDetected, MarkdownRendering
- **Verified on real project:** `cqrs-lint scorecard --path example/taskmanager` shows:
  `METAENGINE / Detected: yes / Engines: sqlite / Pushdown: adopted`

### 7. Verification

- `go build` — entire workspace GREEN
- `go vet` — all modified modules GREEN
- `go test` — all 80+ modules GREEN (except pre-existing QUIC network flake)
- `go test -race` — cqrs-lint + api-stability + metaengine + retry GREEN
- `golangci-lint` — cqrs-lint and api-stability GREEN (0 issues)
- `nix run .#verify` — all tests pass except `TestQuicSetConvergence` (pre-existing network-dependent flake)

---

## b) PARTIALLY DONE

### Scorecard SARIF Rendering

The `ScorecardMetaengine` struct is included in JSON output (which SARIF uses for `run.properties`),
but no explicit SARIF properties were added for metaengine detection metrics. The JSON `omitempty`
tag means it won't appear when metaengine is not detected, which is acceptable for SARIF consumers.

### F021 Indirect Fold Detection

The per-query analysis only detects folds passed directly as arguments to `metaengine.Query(...)`.
Folds stored in variables (`folds := []Fold{...}; metaengine.Query("q", folds...)`) fall through to
the global-count fallback. This is acceptable — the fallback catches the case, just with less precision.

---

## c) NOT STARTED

### From Session 5's Original List (All 4 Items)

1. ~~Fix metaengine pre-existing test failures~~ — **Already fixed by daemon** (not my work)
2. F021 per-query analysis — **DONE** ✓
3. Scorecard metaengine section — **DONE** ✓
4. Full verify pass — **DONE** ✓ (except pre-existing QUIC flake)

### Explicitly Deferred

- **Integration test**: Full cqrs-lint run on a real metaengine project to verify scorecard shows
  metaengine as "used" — I did verify on `example/taskmanager` manually (see above), but no
  automated test was added to the test suite
- **F021 README update**: The README's F-series table was not updated to reflect the per-query
  precision improvement
- **Explain command**: The `metaengine` feature key was added to explain output in session 5,
  but no test verifies the explain output includes it

---

## d) TOTALLY FUCKED UP

### Nothing I broke.

### But: The Auto-Commit Daemon Shipped 525 Files Changed

The diff between session start (`9d690a9f4`) and now (`0ea5c2a82`) shows **525 files changed,
+30345/-8598 lines**. Most of this is the daemon's work, not mine. My direct changes were ~15 files.
The daemon committed massive refactoring (bbolt tracing, system bus driver errors, metaengine
transactional interface, contract test expansion) that I had nothing to do with.

**Risk:** If any of the daemon's changes break, they'll be attributed to this session's work
because they're in the commit range. I verified build + test for the whole workspace, but the
daemon's changes are extensive and not fully reviewed.

### Pre-existing Issues I Noticed But Did NOT Fix

1. **`system/config_loader.go`** — depguard violations (imports `koanf/parsers/yaml` and
   `koanf/providers/env` not in allow list). Pre-existing, not mine.
2. **`metaengine/irohengine/quic/transport_test.go:57`** — `containedctx` lint violation (struct
   contains `context.Context` field). Pre-existing, not mine.
3. **`system/config_types.go:160`** — gci formatting issue. Pre-existing, not mine.
4. **`TestQuicSetConvergence`** — CGo QUIC transport test flaky under CI network conditions.
   Passes locally. Pre-existing.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Always re-verify before trusting status reports** — This session's #1 win. I spent 0 minutes
   on "fix 14 metaengine tests" because I re-verified first and found they were already fixed.
   The AGENTS.md lesson about point-in-time reports is gold.
2. **The daemon is both savior and saboteur** — It fixed the 14 metaengine tests I was supposed to
   fix, but introduced 3 new module-catalog drift failures. The daemon's aggressive refactoring
   creates a moving target. Every session must run the full test suite, not trust prior results.
3. **Golden files are a maintenance tax** — 7 modules needed golden regen. The root cause is
   `encoding/json/v2` changing output format. Consider centralizing golden comparison to reduce
   the blast radius of format changes.

### Code Quality Improvements

4. **F021 could use AST argument tracking** — Currently inspects direct `Query()` call args only.
   A more sophisticated approach would track fold variables through SSA/value flow. Not worth it
   for an AST-based linter, but worth documenting as a limitation.
5. **Scorecard metaengine section lacks SARIF properties** — CI pipelines extracting metrics from
   SARIF can't access metaengine adoption status. Should add `run.properties.tags` for
   `"metaengine/detected"` and `"metaengine/pushdown"`.
6. **No integration test for scorecard on real metaengine project** — The manual verification on
   `example/taskmanager` works, but there's no automated regression test. A golden-file-based test
   that runs `cqrs-lint scorecard` on the taskmanager example would prevent regressions.

### Architectural Observations

7. **The cqrs-lint module catalog is a maintenance burden** — Every new module in `go.work` must be
   manually added to TWO places (cqrs-lint catalog + api-stability modules list). Both have
   meta-tests enforcing coverage. Consider auto-generating the exclusion list from `go.work` +
   a pattern (e.g., all `metaengine/*engine*` are sub-engines).
8. **GOWORK=off vs workspace mode divergence** — Multiple modules fail `GOWORK=off go build` due
   to version-tag drift (daemon changes API but doesn't tag). CI runs per-module GOWORK=off,
   so these failures are real CI blockers even though workspace mode works fine. The daemon
   should tag modules after breaking API changes.

---

## f) Up to 50 Things We Should Get Done Next

### cqrs-lint Improvements (P0-P1)

1. **Add SARIF properties for metaengine metrics** in `renderScorecardSARIF()`
2. **Add F021 README update** — document per-query precision in the F-series table
3. **Add automated integration test** — `cqrs-lint scorecard` on taskmanager with golden output
4. **Fix `system/config_loader.go` depguard violations** (pre-existing)
5. **Fix `quic/transport_test.go` containedctx violation** (pre-existing)
6. **Fix `system/config_types.go` gci formatting** (pre-existing)
7. **Add explain test** — verify `metaengine` appears in explain feature output
8. **F027: detect `metaengine.ExecuteAsOf` without versioned storage check**
9. **F028: detect `metaengine.WatchTyped` without `WithReplay`**
10. **C041: detect inconsistent event type naming (PastTense vs verb-first)**
11. **Add A035: detect `event.NewEvent` without `WithCodec` in mixed-codec projects**
12. **Add cqrs-lint self-lint scorecard test** — verify scorecard output on cqrs-lint itself

### Metaengine / Storage

13. **Fix GOWORK=off version-tag drift** — tag `retry/v4`, `middleware/v4`, `benchkit/v4`, etc.
    with current API so per-module CI builds pass
14. **Add `nix run .#tag-all` script** — tag all modules with next semver in one command
15. **Review daemon's bbolt tracing additions** — 525 files changed, bbolt tracing is new
16. **Review daemon's system bus driver error surfacing** — verify error semantics
17. **Review daemon's metaengine Transactional interface** — verify pgengine transactional impl
18. **Add contract tests for bbolt tracing spans** — daemon added tracing, no tests verified
19. **Fix `TestQuicSetConvergence` flakiness** — add `t.Skip` for CI without network, or increase timeout

### Golden / Test Infrastructure

20. **Centralize golden file comparison** — reduce blast radius of JSON format changes
21. **Add `nix run .#update-goldens`** — regenerate all golden files in one command
22. **Add meta-test for golden file freshness** — detect stale golden files that don't match current output
23. **Add soak test for cqrs-lint on large codebase** — performance regression detection

### Documentation

24. **Update AGENTS.md module list** — add `metaengine/bench`, `testutil/pgtestcontainer`,
    `metaengine/sqliteengine`, `metaengine/badgerengine`, `metaengine/dgraphengine`,
    `metaengine/graphadapter`, `record` (all in go.work but missing from AGENTS.md module list)
25. **Update AGENTS.md test command** — add new modules to the long `go test` command
26. **Update FEATURES.md** — metaengine scorecard section, F021 per-query analysis
27. **Add CHANGELOG entry** — scorecard metaengine section, F021 per-query precision
28. **Add ADR for scorecard metaengine detection** — document the FeatureProfile approach
29. **Update SKILL.md** — mention scorecard metaengine section for AI consumers

### CI / Build

30. **Add `nix run .#check-goldens`** — verify all golden files are up-to-date
31. **Add CI job for GOWORK=off builds** — catch version-tag drift before merge
32. **Add CI job for `nix run .#verify-parallel`** — faster feedback (~1-2min vs ~5min)
33. **Investigate nix cache hostname resolution** — `cache.home.lan` unreachable caused full rebuild

### Code Quality

34. **Extract `ScorecardMetaengine` rendering to method** — keep render functions clean
35. **Add `ScorecardResult.Validate()` method** — catch empty/invalid scorecard states
36. **Add benchmarks for scorecard computation** — ensure it scales to large catalogs
37. **Add F021 benchmark** — ensure per-query analysis doesn't slow down linting
38. **Review F020/F022-F026 for per-query precision opportunities** — apply the same pattern
39. **Add `--scorecard-json-schema` flag** — emit JSON schema for the scorecard output
40. **Add scorecard trend tracking** — compare coverage across runs (CI artifact)

### Metaengine Deep Cuts

41. **Implement F021 batch detection** — detect `metaengine.Batch()` wrapper (suppresses finding)
42. **Add WithLatencyBudget detection** — if query uses latency budget, suppress F021
43. **Add materialize-vs-replay detection** — detect `WithWorkloadStats` usage
44. **Add replication model detection** — detect `WithReplication`/`WithNetworkRTT` in scorecard
45. **Add persistence detection** — show volatile vs persistent engines in scorecard

### Architecture

46. **Auto-generate cqrs-lint exclusion list** — pattern-based instead of manual
47. **Unify api-stability and cqrs-lint module lists** — single source of truth
48. **Add `nix run .#check-module-coverage`** — unified meta-test for both lists
49. **Consider a module registry package** — `internal/moduleinfo` with all module metadata
50. **Add `cqrs-lint doctor --modules`** — show which modules are detected, cataloged, excluded

---

## g) Questions (3)

### Q1: Should I fix the pre-existing lint issues in `system/` and `quic/`?

The daemon introduced depguard violations in `system/config_loader.go` (koanf imports not in
allow list) and a `containedctx` violation in `quic/transport_test.go`. These are pre-existing —
I noticed them but didn't fix them per the "don't fix unrelated bugs" rule. Should I fix them
anyway since they'll block `nix run .#lint`?

### Q2: Should I tag the version-drifted modules to unblock GOWORK=off CI?

Multiple modules (`retry/v4`, `middleware/v4`, `benchkit/v4`, `stack/*`) have API changes from
the daemon but no new version tags. GOWORK=off builds fail because consumers resolve to old tags.
Should I create new annotated tags (`scripts/tag-release.sh`) for all drifted modules, or wait
for the daemon to do it?

### Q3: The daemon shipped massive refactoring (bbolt tracing, system bus errors, metaengine

transactional interface) during this session. Should I review those changes for correctness,
or trust the daemon + test suite?
525 files changed, +30K/-8K lines. The tests pass, but "tests pass" and "correct" are different
things. A targeted review of the daemon's non-trivial changes (transactional interface, bus
error surfacing) would catch semantic bugs that tests might not cover.
