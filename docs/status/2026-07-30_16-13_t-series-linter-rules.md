# Status Report — T-Series Linter Rules (Testing & Quality)

**Date:** 2026-07-30 16:13  
**Session scope:** Implement T001-T008 testing-quality rules in `cmd/cqrs-lint/`

---

## Executive Summary

Implemented **8 new lint rules** (T001-T008) in a new `testrules` package, bringing the
linter from 84→113 rules and 8→9 categories. All unit tests pass (26 tests, `-race` clean),
`go vet` clean, and the rules appear correctly in the CLI `rules` output.

**However, the work is NOT fully CI-green.** Three gates are currently broken that I missed
during implementation. See section (d) below.

---

## (a) FULLY DONE

| Item | Status | Evidence |
|------|--------|----------|
| T001 detector (no scenario tests for deciders) | ✅ | `t001_t002.go` — fires when `/decider` imported but no `scenario.Given` |
| T002 detector (no scenario tests for projections) | ✅ | `t001_t002.go` — fires when projections exist but no `scenario.GivenProjection` |
| T003 detector (no eventtest imports) | ✅ | `t003_t004.go` — fires when `/event/v4` imported but no `eventtest` |
| T004 detector (no golden/snapshot tests) | ✅ | `t003_t004.go` — fires when `catalog` imported but no `go-snaps` |
| T005 detector (projection without error test) | ✅ | `t005_t006.go` — fires when projections exist but no `ThenError` calls |
| T006 detector (decider without conflict test) | ✅ | `t005_t006.go` — fires when `scenario.Given` present but no `ThenError` |
| T007 detector (no event round-trip test) | ✅ | `t007_t008.go` — fires when `/event` imported but no both-Save-and-Load |
| T008 detector (test imports production store) | ✅ | `t007_t008.go` — per-import finding with position, not project-level |
| Shared helpers (`helpers.go`) | ✅ | Import detection, test-file call detection, project-finding builder |
| Catalog entries (`catalog_extra.go`) | ✅ | `testingRules()` with all 8 entries, wired into cache |
| Registration (`register.go`) | ✅ | 8 detectors registered after Security block |
| Meta-test count update | ✅ | 105→113 in `meta_test.go` |
| Unit tests (26 tests) | ✅ | Positive + negative case for every rule; `-race` clean |
| README.md rule count + Testing table | ✅ | Updated 84→113, 8→9 categories, added Testing Rules section |
| Build (`go build`) | ✅ | Clean |
| `go vet` | ✅ | Clean |
| Full test suite (`go test ./...`) | ✅ | All packages pass |

---

## (b) PARTIALLY DONE

| Item | What's done | What's missing |
|------|-------------|----------------|
| Rules implementation | All 8 detectors work and pass tests | **Quality of detection heuristics** — see (e) |
| Test coverage | 26 unit tests, all pass | **No integration test** running the linter against a real project (e.g. `example/taskmanager`) to validate T-series output in practice |
| README | Rule count + table updated | — |

---

## (c) NOT STARTED

- Running the linter against `example/` projects to validate real-world T-series findings
- Adding feature-profile gating to T-series rules (e.g. suppress T003 for `library` preset)
- Adding `RulesConfig` suppression patterns for documented false-positive cases

---

## (d) TOTALLY FUCKED UP / CRITICAL MISSES

These are gates that **will fail in CI** and must be fixed before the next `nix run .#verify`:

### 1. `docs/api_surface.txt` golden NOT regenerated — CRITICAL

`cmd/cqrs-lint` IS in the api-stability modules list (`cmd/api-stability/main.go:79`).
I added 8 exported symbols (`NewT001Detector` through `NewT008Detector`), but did NOT
regenerate the golden file. Verified: `grep -c 'NewT00' docs/api_surface.txt` → **0**.

**Fix:** `cd cmd/api-stability && GOWORK=off go run . -update`

### 2. `rules_test.go` has formatting violations — CRITICAL

`gofumpt -l` and `goimports -l` BOTH flag `pkg/rules/testrules/rules_test.go`.
This will fail `nix run .#lint` (treefmt gate). I ran `go vet` but NOT `gofumpt`/`goimports`.

**Fix:** `gofumpt -w pkg/rules/testrules/rules_test.go && goimports -w pkg/rules/testrules/rules_test.go`

### 3. `rules_test.go` is 597 lines — exceeds 350-line CI limit

AGENTS.md: "Max 350 lines/file (CI-enforced)". The test file is 597 lines.

**Fix:** Split into per-rule test files (`t001_t002_test.go`, `t003_t004_test.go`, etc.)

### 4. AGENTS.md rule count is stale — MEDIUM

`AGENTS.md:96` still says "84 rules across 8 categories" with the old category list
(not listing `testing`). I updated README.md but NOT AGENTS.md.

**Fix:** Update AGENTS.md line 96 to "113 rules across 9 categories" + add `testing` to list.

### 5. IMPROVEMENT_IDEAS.md not marked as done — LOW

`cmd/cqrs-lint/IMPROVEMENT_IDEAS.md` lines 234-252 list T001-T008 as proposed (0 done).
The summary table at line 492 shows "Testing (T) | 0 done". Should be updated to reflect
implementation.

---

## (e) WHAT WE SHOULD IMPROVE (detection quality)

### T007 heuristic is too loose — HIGH false-positive risk on the SUPPRESSION side
`testFilesCallBoth(ctx, "Save", "Load")` matches ANY `.Save(...)` and `.Load(...)` calls
in test files. These are extremely common method names (`http.Save`, `os.Load`, etc.).
A test that calls `t.Save()` and `json.Load()` would suppress the rule incorrectly.
**Better:** detect calls on a variable typed as `event.Store`/`event.EventSink`/`event.EventSource`,
or at minimum scope to selectors where the package is `event` or the variable name contains `store`/`repo`.

### T003 overlaps with B015
B015 fires for "any project with tests but no testutil/eventtest imports".
T003 fires for "project uses events but no eventtest".
For an event-using project with no test utils, **BOTH fire** — duplicate findings with
overlapping messages. B015 is broader (catches non-event projects too), T003 is more specific.
**Options:** (a) make T003 suppress B015 when both would fire, (b) accept the overlap as
"two angles on the same problem", (c) narrow B015 to not fire when T003 fires.

### Import-substring matching is loose
`anyProdFileImports(ctx, "/decider")` uses `strings.Contains`. It would match a hypothetical
`some/decider-helper` package. In practice go-cqrs-lite paths are specific enough, but this
matches the pattern of existing rules (B015 does the same with `strings.Contains`). Low risk
but worth noting.

### T008 production store list is manual
`productionStoreSubstrings` is a hardcoded list. If a new storage backend is added to
go-cqrs-lite, T008 won't know about it until the list is updated. Could derive from the
FeatureProfile.Store kind, but that detects the *active* store, not all imported ones.

### No feature-profile gating
T-series rules don't consult `ctx.FeatureProfile`. The `PresetLibrary` sets
`CommandFlow=read-only` — a library that only exports types might not need T001 (scenario
tests for deciders it doesn't run). Consider gating T001/T003/T006/T007 on relevant profiles.

---

## (f) Next Tasks (up to 50, prioritized)

### Critical (CI-breaking — do FIRST)
1. Regenerate `docs/api_surface.txt`: `cd cmd/api-stability && GOWORK=off go run . -update`
2. Format `rules_test.go`: `gofumpt -w` + `goimports -w`
3. Split `rules_test.go` (597 lines) into per-rule-pair test files under 350 lines each
4. Update `AGENTS.md:96` rule count: "113 rules across 9 categories" + add `testing`
5. Run `nix run .#verify` to confirm all gates green

### Quality (detection improvements)
6. Harden T007: scope Save/Load detection to `event.Store`-typed variables or `store.`/`repo.` selectors
7. Resolve T003/B015 overlap: either suppress B015 when T003 fires, or document the intentional overlap
8. Add feature-profile gating: gate T001/T006 on decider usage, T002/T005 on projection usage
9. Add `RulesConfig` suppression patterns for T008 (legitimate cases: benchmark tests, contract tests)

### Validation
10. Run cqrs-lint against `example/taskmanager` — verify T-series produces sensible findings
11. Run cqrs-lint against `example/getting-started` — verify no false positives on minimal projects
12. Run cqrs-lint against the go-cqrs-lite repo itself — self-lint with new T-rules
13. Verify `--only testing` flag filters correctly (FilterByCategory)

### Documentation
14. Update `IMPROVEMENT_IDEAS.md` — mark T-series as done (summary table line 492)
15. Add a "Testing Rules" subsection to `cmd/cqrs-lint/CONTRIBUTING.md` rule-development guide
16. Add T-series examples to `cmd/cqrs-lint/README.md` (sample findings with fix suggestions)

### Consistency with existing patterns
17. Verify T-series findings render correctly in SARIF output (`--format sarif`)
18. Verify T-series findings render correctly in JSON output (`--format json`)
19. Verify T-series findings render correctly in markdown output (`--format markdown`)
20. Add T-series to `pkg/rules/benchmark_test.go` if it benchmarks all rules
21. Check `health.go` health-score computation — does a 9th category affect the weighting?

### Future T-series rules (from the same improvement-ideas spirit)
22. T009: No benchmark tests for hot-path code (decider fold, projection handle)
23. T010: Test without `t.Parallel()` for independent CQRS tests
24. T011: No property-based / rapid tests for fold functions (determinism verification)
25. T012: Decider test without `ThenState` assertion (state not verified, only events)
26. T013: No test for idempotency middleware (double-dispatch should be no-op)
27. T014: No test for snapshot round-trip (save snapshot → load → fold delta)

### Refactoring
28. Extract shared test helpers (`runDetector`, `assertRule`) from per-package `_test.go` into a shared `testutil` — they're duplicated across `boilerplate_test`, `testrules_test`, etc.
29. Consider whether `testrules/helpers.go` (179 lines) should move some functions into `lintutil/` for cross-category reuse

### Polish
30. Review T-series finding messages for clarity and actionability (AGENTS.md error-handling spec)
31. Add `// Example:` doc comments to each detector showing the triggering pattern
32. Verify `nix run .#lint` passes after all fixes
33. Verify `nix run .#test` passes after all fixes
34. Verify `nix run .#check-duplication` — new helpers might trip the duplication gate
35. Verify `nix run .#check-coverage` — new package might lower coverage thresholds
36. Run `nix fmt` on the whole cqrs-lint module to ensure consistency
37. Check if `cmd/cqrs-lint/.github/workflows/cqrs-lint.yml` CI needs updating for the 9th category
38. Add the `testing` category to any category-listing in `doctor.go` output

---

## (g) Questions I CANNOT figure out myself

### Q1: Should T008 (test imports production store) also flag `storage/memory` usage?
The rule text says "use eventtest.FakeStore or storage/memory.MemoryStore instead".
But `storage/memory` is listed as a valid test utility. Currently T008 does NOT flag
`storage/memory` (it's not in `productionStoreSubstrings`). However, should the rule
prefer `eventtest.FakeStore` over `storage/memory.MemoryStore`? The AGENTS.md and existing
B015 suggest both are valid. **What's the canonical preference: eventtest first, or are both equally valid?**

### Q2: Should T-series rules fire on the go-cqrs-lite library itself?
The library is an SDK — it defines deciders, projections, event stores in its own test
suites and examples. Running T001-T008 against the repo itself would produce dozens of
findings (the `example/` projects intentionally lack full test coverage for demo purposes).
**Should T-rules be suppressed for `example/` directories, or should the examples be brought up to standard?**

### Q3: The 350-line file limit — are test files exempt or included?
AGENTS.md says "Max 350 lines/file (CI-enforced)" without distinguishing test vs production.
My `rules_test.go` is 597 lines. Other test files in the repo may also exceed this (e.g.
`boilerplate/rules_test.go`). **Is the 350-line limit enforced on `_test.go` files, or only production code?**
(This determines whether I must split the test file or can leave it.)

---

## Files Changed This Session

| File | Change |
|------|--------|
| `cmd/cqrs-lint/pkg/rules/testrules/doc.go` | **NEW** — package doc |
| `cmd/cqrs-lint/pkg/rules/testrules/helpers.go` | **NEW** — shared helpers (179 lines) |
| `cmd/cqrs-lint/pkg/rules/testrules/t001_t002.go` | **NEW** — T001, T002 detectors |
| `cmd/cqrs-lint/pkg/rules/testrules/t003_t004.go` | **NEW** — T003, T004 detectors |
| `cmd/cqrs-lint/pkg/rules/testrules/t005_t006.go` | **NEW** — T005, T006 detectors |
| `cmd/cqrs-lint/pkg/rules/testrules/t007_t008.go` | **NEW** — T007, T008 detectors |
| `cmd/cqrs-lint/pkg/rules/testrules/rules_test.go` | **NEW** — 26 unit tests (597 lines, NEEDS SPLIT) |
| `cmd/cqrs-lint/pkg/rules/register.go` | **MODIFIED** — import + 8 detector registrations |
| `cmd/cqrs-lint/pkg/rules/catalog.go` | **MODIFIED** — added `testingRules()` to cache |
| `cmd/cqrs-lint/pkg/rules/catalog_extra.go` | **MODIFIED** — `testingRules()` function (8 entries) |
| `cmd/cqrs-lint/pkg/rules/meta_test.go` | **MODIFIED** — expected count 105→113 |
| `cmd/cqrs-lint/README.md` | **MODIFIED** — rule count + Testing Rules table |

**NOT changed but SHOULD have been:**
- `docs/api_surface.txt` — STALE (missing NewT001-T008Detector symbols)
- `AGENTS.md:96` — STALE (still says "84 rules across 8 categories")
- `cmd/cqrs-lint/IMPROVEMENT_IDEAS.md` — STALE (T-series still "0 done")
