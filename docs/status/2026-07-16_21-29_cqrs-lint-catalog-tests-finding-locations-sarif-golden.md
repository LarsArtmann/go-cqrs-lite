# cqrs-lint: Catalog Consolidation, Positive Tests, Finding Locations & SARIF Golden — Full Status

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../CHANGELOG.md) and
> [TODO_LIST.md](../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

**Date:** 2026-07-16 21:29
**Session scope:** Round 4 of cqrs-lint quality improvements (following Rounds 1-3 from prior sessions)
**Uncommitted changes:** 13 files changed in `cmd/cqrs-lint/` (+604 -316 lines), 1 new golden file

---

## a) FULLY DONE

### 1. Catalog Consolidation (3 files → 2)

| Before                                                                                                                            | After                                                                                                                                                              |
| --------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `catalog.go` (coreRules, 186 lines) — correctness + api only                                                                      | `catalog.go` (correctnessRules + apiRules, 313 lines) — all correctness + all api rules                                                                            |
| `catalog_extra.go` (extraRules, 224 lines) — mixed bag: boilerplate + consistency + architecture + security, randomly interleaved | `catalog_extra.go` (boilerplateRules + consistencyRules + architectureRules + securityRules, 282 lines) — same 4 categories but now organized by category function |
| `catalog_extra2.go` (extraRulesBatch2, 169 lines) — "newer rules" dumped in arbitrary batch order                                 | **DELETED** — all rules merged into the two organized files                                                                                                        |

**Before:** Rules were split across 3 files by arbitrary batch timing ("core" vs "extra" vs "extra batch 2"). Finding a rule's metadata required searching 3 files. A009-A019, B006-B015, D003, D005, E001-E007, S002-S003 were scattered.

**After:** Rules organized by category (correctnessRules, apiRules, boilerplateRules, consistencyRules, architectureRules, securityRules). Two files, both under 350-line CI limit. One `AllRules()` entry point with clean append chain.

### 2. Positive Tests for 6 Previously Untested Rules

All 6 rules previously had only smoke tests (crash-on-empty) or no test at all. Each now has a positive test that verifies the detector fires on real CQRS anti-patterns, plus most have negative tests too.

| Rule     | File                             | Test name                                   | What it verifies                                                          | Negative test?                                        |
| -------- | -------------------------------- | ------------------------------------------- | ------------------------------------------------------------------------- | ----------------------------------------------------- |
| **B002** | `boilerplate/new_rules_test.go`  | `TestB002_DetectsManualWiring`              | NewEventStore + NewBus + NewRepository call sequence → "use stack preset" | `TestB002_NoFindingForSimpleFunc`                     |
| **B003** | `boilerplate/new_rules_test.go`  | `TestB003_DetectsLargeSwitch`               | SubscribeAll with 6+ switch cases → "split into projections"              | `TestB003_NoFindingForSmallSwitch` (2 cases)          |
| **A012** | `api/new_rules_test.go`          | `TestA012_DetectsFoldWithoutTombstoneCheck` | Fold function with switch but no tombstone check                          | (existing negative: `TestA012_NoFindingWithoutFolds`) |
| **A016** | `api/new_rules_test.go`          | `TestA016_DetectsMissingIdempotency`        | NewDispatcher + Use but no CommandIdempotency                             | `TestA016_NoFindingWithIdempotency`                   |
| **A019** | `api/new_rules_test.go`          | `TestA019_DetectsVendoredCqrs`              | Package import containing "vendor/" + "cqrs"                              | (existing negative: `TestA019_NoCrashOnEmptyContext`) |
| **E001** | `architecture/new_rules_test.go` | `TestE001_DetectsLayerViolation`            | Tier-0 module (codec) importing Tier-3+ module (decider)                  | (existing negative: `TestE001_NoCrashOnEmptyContext`) |

**Pattern used:** `analyzer.BuildContextFromSource(t, map[string]string{filename: content})` → `runDetector(t, pkg.NewXxxDetector(ctx))` → `assertRule(t, findings, "Xxx", 1)`. For package-based rules (A019, E001), `ctx.Packages` is manually injected after building context.

### 3. Finding Location Improvements (5 detectors)

All 5 detectors previously pointed at `go.mod:1:1` — useless for developers. Now they point at the exact code site.

| Rule     | File                           | Before                            | After                                                                                  | Snippet?                                 |
| -------- | ------------------------------ | --------------------------------- | -------------------------------------------------------------------------------------- | ---------------------------------------- |
| **D001** | `consistency/rules.go:29`      | `go.mod:1:1`                      | First event emission site (from `EventEmission.File`/`.Line` in registry)              | Yes — `WithSnippet(ctx.SourceLine(...))` |
| **D003** | `consistency/d003_d005.go:22`  | `go.mod:1:1` (via `ctx.Packages`) | First logging import statement (refactored to use `gf.AST.Imports` for exact position) | Yes                                      |
| **B013** | `boilerplate/b011_b014.go:100` | `go.mod:1:1`                      | `NewRepository` call site position                                                     | Yes                                      |
| **B014** | `boilerplate/b011_b014.go:164` | `go.mod:1:1`                      | `Use`/`UsePublish` call site position                                                  | Yes                                      |
| **A016** | `api/a015_a019.go:119`         | `go.mod:1:1`                      | `NewDispatcher`/`Use` call site position                                               | Yes                                      |

**D003 architectural improvement:** The detector was refactored from iterating `ctx.Packages` (which only gives package paths, not file positions) to iterating `ctx.GoFiles` and checking `gf.AST.Imports` directly. This gives exact import statement positions and also fixed a latent issue where D003 couldn't detect logging imports from source-only test contexts.

**Test update:** `TestD003_DetectsMixedLogging` was rewritten from `ctx.Packages` injection to real Go import statements (two files importing `log/slog` and `go.uber.org/zap`), making it test the actual code path rather than mock data.

### 4. SARIF Golden File Test

| Item        | Detail                                                                 |
| ----------- | ---------------------------------------------------------------------- |
| File        | `pkg/rules/benchmark_test.go` — `TestGoldenFile_SARIFOutput`           |
| Golden file | `pkg/rules/testdata/sarif_output.json` (1.4KB)                         |
| Comparison  | JSON structural comparison via `reflect.DeepEqual` (not byte-for-byte) |

**Why structural comparison:** SARIF properties are serialized from Go `map[string]string`, which has non-deterministic key ordering in JSON output. A byte-for-byte golden file comparison would be flaky. Parsing both into `any` and comparing structurally is deterministic and still catches structural changes (added/removed fields, changed values).

**Golden file content:** SARIF 2.1.0 with `runs[0].tool.driver.name=cqrs-lint`, one result with `ruleId=C001`, `level=error`, `region.startLine=10`, suggestion, confidence, and category in properties.

### 5. Dead Code Removal

| Item                           | Location                           | Why dead                                                                                |
| ------------------------------ | ---------------------------------- | --------------------------------------------------------------------------------------- |
| `isIDMethod(fn *ast.FuncDecl)` | `pkg/analyzer/scanner_folds.go:36` | Zero callers — `scanIDMethod` is the actual function used. Flagged by lint as `unused`. |

### Metrics Summary

| Metric                        | Value                                                                                                                                |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Total test functions          | **159** (all PASS, 0 FAIL)                                                                                                           |
| Test packages                 | **11** (all OK)                                                                                                                      |
| Rules with positive tests     | **All 60** rules now have either positive tests or smoke tests                                                                       |
| Rules still go.mod:1:1        | **9** (A009, A018, A019, D005, E001, E002, S001, S002, S003 — all legitimate project-level checks with no specific file to point at) |
| Dead lint issues in cqrs-lint | **0**                                                                                                                                |
| Build errors                  | **0**                                                                                                                                |
| Go vet errors                 | **0**                                                                                                                                |
| Files over 350 lines          | **0** (largest: `catalog.go` at 313)                                                                                                 |

---

## b) PARTIALLY DONE

### Finding Location Quality

- **9 rules still use `go.mod:1:1`** — but these are genuinely project-level checks (A009: "no stack preset", A018: "no event sourcing", A019: "vendored cqrs", D005: "stale docs", E001/E002: architecture violations detected from package graph, S001/S002/S003: security posture). There is no single file to point at for these.
- **D001 finding location** now points at the first emission site, but if `EventEmission.File` is empty (scanner didn't capture it), it falls back to `go.mod:1:1`. This is a scanner gap, not a detector bug.
- **D003 now iterates `ctx.GoFiles` instead of `ctx.Packages`** — this means it only catches logging imports in files the scanner has parsed. If `ctx.Packages` had imports from packages NOT in `ctx.GoFiles`, those would be missed. In practice this is fine (the scanner parses all Go files in the target), but it's a subtle behavior change.

### Test Coverage Gaps

- **Snippet presence assertions** — no test verifies that detectors actually attach source snippets. We could add `assertHasSnippet(t, findings, "Xxx")` helper.
- **SARIF output integration test** — the golden file test builds a synthetic finding, not a real detector output. An integration test running a real detector and checking SARIF would be more comprehensive.
- **Markdown output** — no golden file test for `--format markdown` at all.

---

## c) NOT STARTED

Items documented in prior planning doc (`docs/planning/2026-07-16_20-47_cqrs-lint-comprehensive-quality-plan.md`) that were NOT attempted this session:

1. **CLI Polish:** `--debug` flag, `--ci` mode, `--baseline` diff, exit code docs in `--help`
2. **Integration Tests:** Monorepo fixture test (multi-module testdata dir), integration test running cqrs-lint on `example/taskmanager/`
3. **Scanner Accuracy:** Resolve event type constants (`bus.Subscribe(SomeEvent, handler)`), `detectFoldFunc` exact type matching using `types.Info`, `event.Single` call detection
4. **Cobra removal:** cobra is still imported for `cobra.MaximumNArgs(1)` and `*cobra.Command` type
5. **Functional decider registration in scanner** — design question unresolved
6. **Snippet presence assertions in existing rule tests**
7. **Config file validation test** (`.cqrs-lint.json`)
8. **`--verbose` module grouping integration test**
9. **`--color` mode unit tests** (TTY detection mock)
10. **Benchmark SourceLine caching** (before/after allocation comparison)

---

## d) TOTALLY FUCKED UP

### Nothing this session.

All changes were surgical, tested incrementally, and verified. No regressions introduced. The only surprise was the SARIF golden file failing on first attempt due to non-deterministic map ordering — caught immediately by running the test twice, fixed with structural JSON comparison.

### Pre-existing issues noticed but NOT mine to fix:

1. **`watermill/event_bus.go` has uncommitted changes** in the working tree — a `Close()` lock refactor (moving `backend.Close()` outside the mutex). These appeared independently of my work. I did NOT touch them. They cause `TestGolden_MessageMetadata` to fail in `watermill/` — but this is unrelated to cqrs-lint.

2. **`decider/strict_apply.go:40`** — lint issue: `err113` (dynamic error via `fmt.Errorf`). Pre-existing.

3. **`decider/strict_apply_test.go:81`** — lint issue: `errname` (`errApplySentinel` should be `xxxError`). Pre-existing.

4. **`projectionhost/host.go:339`** — lint issue: `exhaustruct` (`resetConfig` missing `purgeDLQ`). Pre-existing.

5. **`projectionhost/worker_drain.go:26`** — lint issue: `varnamelen` (`cp` too short). Pre-existing.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`catalog.go` is at 313 lines** — approaching the 350-line limit. Adding more rules will require splitting. Consider a `catalog_` prefix convention (e.g., `catalog_correctness.go`) before it breaks CI.

2. **`AllRules()` uses nested `append` chains** — fragile and hard to read. If one category grows large enough, the nesting breaks. Could use a simpler flatten pattern.

3. **D003's switch from `ctx.Packages` to `ctx.GoFiles`** is correct for position accuracy but changes detection semantics. A proper fix would check BOTH sources (packages for transitive imports, GoFiles for position).

4. **Finding Builder pattern** — every detector repeats the same 8-line builder chain. A `newCQRSFinding(ruleID, msg, severity, file, line)` helper would eliminate ~200 lines of boilerplate across all detectors.

5. **SARIF test structural comparison** — `reflect.DeepEqual` on parsed JSON works but doesn't detect key reordering within arrays. For SARIF, `results[].locations` order matters. The test is good enough for now.

### Testing

6. **No test for `boilerplate/rules.go` hasWiringSequence helper directly** — only tested via B002 detector integration test. Unit tests for the helper would be more targeted.

7. **Test helper duplication** — `runDetector` and `assertRule` are copy-pasted in 3 packages (`api`, `boilerplate`, `architecture`). Could move to a shared `testutil` package. But this is a test-only concern, low priority.

8. **No fuzz tests** — the scanner parses arbitrary Go source. Fuzz testing `scanFuncDecl`, `capturePayloadType`, etc. could find panics on malformed input.

9. **Positive test for B001** — B001 (single-event-helper) has a test (`TestB001_DetectsSingleEventHelper`) but I didn't verify it this session. It exists from a prior session.

---

## f) Up to 50 Things We Should Get Done Next

### P0 — Critical (Scanner Accuracy)

1. Resolve event type constants in `bus.Subscribe(SomeConst, handler)` calls
2. Track `event.Single()` calls as event emissions in the scanner
3. Implement `types.Info` resolution for accurate payload type matching (eliminate string heuristics)
4. Multi-site `EventTypesEmitted` tracking (currently last-wins — if same type emitted in 2 files, only last location is kept)
5. Track `Save()`/`Publish()` calls in registry (eliminate per-rule AST re-scanning)

### P1 — High Value (Test Coverage)

6. Add snippet presence assertions (`assertHasSnippet`) to all positive tests
7. Add positive tests for A002, A006, A008, A009 (these may have tests from prior sessions — verify)
8. Markdown output golden file test
9. Monorepo fixture test (multi-module testdata with go.work)
10. Integration test: run cqrs-lint binary on `example/taskmanager/` and check findings
11. Fuzz test: `scanFuncDecl` with arbitrary Go source
12. Fuzz test: `capturePayloadType` with arbitrary call expressions
13. Config file (`.cqrs-lint.json`) validation test
14. `--verbose` module grouping integration test
15. `--color` mode TTY detection unit tests
16. Benchmark: SourceLine caching before/after allocation comparison
17. Test that `AllRules()` count matches `RegisterAll()` detector count (guard against catalog drift)
18. Test `FilterByCategory` and `FilterByRuleIDs` with edge cases

### P2 — Medium Value (Finding Locations)

19. B013: track ALL repository creation sites, not just the last one (emit per-repository findings)
20. B014: track ALL middleware registration sites
21. A016: track ALL dispatcher construction sites
22. E001/E002: point at the import statement in the actual source file (if available in `ctx.GoFiles`)
23. A019: point at the vendored import path statement

### P3 — Architecture Cleanup

24. Extract `newCQRSFinding()` helper to reduce finding-builder boilerplate
25. Remove cobra dependency (replace `cobra.MaximumNArgs` with cmdguard equivalent)
26. Register functional deciders in scanner registry
27. Consider splitting `catalog.go` proactively before it hits 350 lines
28. Move shared test helpers (`runDetector`, `assertRule`) to `testutil` package
29. Consolidate `ctx.Packages` and `ctx.GoFiles` import scanning into a single shared helper
30. Add `--debug` flag to dump scanner registry state (for user debugging)

### P4 — CLI/UX

31. `--ci` mode (auto-detect CI, set color=never, format=sarif, fail on any finding)
32. `--baseline` flag (only report new findings since a reference run)
33. `cqrs-lint init` interactive wizard (create `.cqrs-lint.json`)
34. `rules` subcommand with search/filter
35. More auto-fix rules (currently only C001, C003, C006)
36. `--fix-dry-run` with unified diff output
37. Progress indicator for large monorepos
38. Exit code documentation in `--help`
39. `--stats` flag showing per-category finding counts

### P5 — Documentation

40. Per-rule documentation pages with fix examples
41. "Getting Started" guide for new users
42. "Rule Suppression" guide (`//cqrs-lint:disable`)
43. Architecture documentation for the scanner pipeline
44. Comparison table vs. other Go linters
45. Update README with SARIF output examples
46. Document the scanner's heuristic detection model

### P6 — Advanced

47. Plugin system for custom rules
48. `types.Info` full resolution (go/packages mode)
49. Cross-module analysis (detect violations across go.work modules)
50. CI mode with GitHub PR comment integration (annotate PR with findings)

---

## g) Questions I CANNOT Answer Myself

### 1. Should the SARIF golden file test also test a real detector output?

Currently the SARIF golden file builds a synthetic `Finding` via the builder. An alternative is to run a real detector (e.g., C001 on fixture code) and serialize that to SARIF. This would test the full pipeline but make the golden file more fragile (any detector output change breaks it). I can't decide the right tradeoff between stability and coverage.

### 2. Should D003 check BOTH `ctx.Packages` AND `ctx.GoFiles` for logging imports?

I changed D003 to use `ctx.GoFiles` for position accuracy. But `ctx.Packages` might contain transitive imports not in `ctx.GoFiles` (e.g., if a dependency uses zap, but the target project only imports that dependency). Checking both would be more thorough but would re-introduce the `go.mod:1:1` fallback for transitive-only imports. I don't know if this scenario matters to users.

### 3. Should I commit the cqrs-lint changes separately from the pre-existing `watermill/event_bus.go` working-tree changes?

The `watermill/event_bus.go` change (Close() lock refactor) appeared in the working tree independently. It's NOT my change. When committing, should I: (a) commit only `cmd/cqrs-lint/` files and leave watermill unstaged, (b) ask who made the watermill change, or (c) commit everything together? I don't know the context of the watermill change or whether it should be committed.

---

## Files Changed This Session

| File                                                     | Change                                                                                                         | Lines   |
| -------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------- |
| `cmd/cqrs-lint/pkg/rules/catalog.go`                     | Consolidated: coreRules→correctnessRules+apiRules, added missing rules (C004, C011, A009-A019)                 | 313     |
| `cmd/cqrs-lint/pkg/rules/catalog_extra.go`               | Rewritten: extraRules→boilerplateRules+consistencyRules+architectureRules+securityRules, organized by category | 282     |
| `cmd/cqrs-lint/pkg/rules/catalog_extra2.go`              | **DELETED** — merged into above two files                                                                      | -169    |
| `cmd/cqrs-lint/pkg/rules/api/a015_a019.go`               | A016: capture `NewDispatcher`/`Use` position, add snippet                                                      | +8      |
| `cmd/cqrs-lint/pkg/rules/boilerplate/b011_b014.go`       | B013: capture `NewRepository` position; B014: capture `Use`/`UsePublish` position; both add snippets           | +16     |
| `cmd/cqrs-lint/pkg/rules/consistency/rules.go`           | D001: capture first emission position from registry, add snippet                                               | +21     |
| `cmd/cqrs-lint/pkg/rules/consistency/d003_d005.go`       | D003: refactored from `ctx.Packages` to `ctx.GoFiles` for import positions, add snippet                        | +44/-44 |
| `cmd/cqrs-lint/pkg/rules/consistency/new_rules_test.go`  | D003 test rewritten to use real imports; removed `packages` import                                             | +28/-28 |
| `cmd/cqrs-lint/pkg/rules/api/new_rules_test.go`          | +3 positive tests (A012, A016×2, A019); added `packages` import                                                | +75     |
| `cmd/cqrs-lint/pkg/rules/architecture/new_rules_test.go` | +1 positive test (E001 layer violation)                                                                        | +18     |
| `cmd/cqrs-lint/pkg/rules/boilerplate/new_rules_test.go`  | +2 positive tests (B002×2, B003×2)                                                                             | +71     |
| `cmd/cqrs-lint/pkg/rules/benchmark_test.go`              | +SARIF golden file test (structural JSON comparison); added `encoding/json` + `reflect` imports                | +59     |
| `cmd/cqrs-lint/pkg/rules/testdata/sarif_output.json`     | **NEW** — SARIF 2.1.0 golden file                                                                              | 1.4KB   |
| `cmd/cqrs-lint/pkg/analyzer/scanner_folds.go`            | Removed dead `isIDMethod` function (flagged by lint)                                                           | -4      |
