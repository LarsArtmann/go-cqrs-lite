# Status Report: cqrs-lint Implementation

> **Date**: 2026-07-16 05:07
> **Session scope**: Building the cqrs-lint domain-aware linter from the execution plan
> **Companion docs**: [Execution Plan](../planning/2026-07-16_03-54_cqrs-lint-execution-plan.md), [Linter Research](../research/domain-linter-research.md)

---

## 1. Executive Summary

Built a working domain-aware linter for go-cqrs-lite consumers in one session. The linter compiles, runs against real consumer projects, detects real issues, produces all four output formats, and has 30 unit tests passing. However, the execution plan defined 50 Level-1 tasks and 117 Level-2 tasks; this session completed approximately 40% of them. The remaining work is primarily additional rules (21 of 47 remain unimplemented), file-length CI violations, deeper test coverage, and integration tests.

---

## 2. What Is FULLY DONE

### Working binary

- `cqrs-lint` binary builds and runs: `go build -tags "goexperiment.jsonv2" -o /tmp/cqrs-lint ./cmd/cqrs-lint/`
- Verified against 5 real consumer projects: bank-sync (88/100), DiscordSync (68/100), crush-daily (50/100), storbi (84/100), taskmanager (72/100)

### Rules implemented (26 of 47)

| Category                 | Implemented                   | Remaining      |
| ------------------------ | ----------------------------- | -------------- |
| Correctness (C001-C012)  | 10 of 12 (C004, C011 missing) | 2              |
| API misuse (A001-A008)   | 8 of 8 planned for Phases 1-3 | 12 (A009-A020) |
| Boilerplate (B001-B003)  | 3 of 15                       | 12             |
| Architecture (E004-E005) | 2 of 7                        | 5              |
| Consistency (D001-D002)  | 2 of 5                        | 3              |
| Security (S001)          | 1 of 3                        | 2              |

### CLI features

- [x] Text output (human-readable with severity badges)
- [x] JSON output (full report with all finding fields)
- [x] SARIF output (GitHub Code Scanning compatible)
- [x] Markdown output (table format)
- [x] `--health-score` (0-100 score with breakdown)
- [x] `--fast` (Critical/High correctness rules only)
- [x] `--fix` / `--dry-run` (auto-fix with CQRSFixProvider)
- [x] `--min-severity` (filter by severity)
- [x] `--only` (filter by category)
- [x] `--quiet` / `--verbose`
- [x] `rules` subcommand (list all rules with descriptions)
- [x] `version` subcommand
- [x] `--help` with examples

### Library functions

- [x] `event.Single()` — creates single-event slice (3 tests passing)
- [x] `decider.StrictApply[T]()` — wraps fold to error on unknown events (3 tests passing)

### Infrastructure

- [x] `go.mod` with replace directives for go-finding and go-finding/pipeline
- [x] Added to `go.work`
- [x] CQRSRegistry builder (AST scanning for commands, events, folds, deciders, projections)
- [x] AnalysisContext (shared state for all detectors)
- [x] CQRSFixProvider (BeforeCode/AfterCode substring matching)
- [x] Suppression comment parser (`//cqrs-lint:ignore(rule-id) reason`)
- [x] Health score computation with severity-weighted deductions
- [x] Rule registration system (AllRules catalog + RegisterAll + RegisterCritical)
- [x] GitHub Actions workflow example
- [x] README.md with quickstart, rule reference, architecture, CI integration

### Tests (30 test functions across 10 packages)

| Package               | Tests | Status   |
| --------------------- | ----- | -------- |
| event (Single)        | 3     | All pass |
| decider (StrictApply) | 3     | All pass |
| correctness rules     | 13    | All pass |
| api rules             | 4     | All pass |
| boilerplate rules     | 1     | All pass |
| architecture rules    | 1     | All pass |
| consistency rules     | 2     | All pass |
| security rules        | 2     | All pass |
| fix provider          | 3     | All pass |
| suppression           | 3     | All pass |

### Documentation

- [x] README.md with quickstart, full rule reference table, architecture, CI integration
- [x] AGENTS.md updated with cqrs-lint module entry + test command
- [x] GitHub Actions workflow example

---

## 3. What Is PARTIALLY DONE

### Rule coverage: 26 of 47 rules (55%)

The research document defines 47 rules. 26 are implemented. The missing 21 are primarily:

- **A009-A020** (12 API misuse rules): lower frequency patterns (e.g., `fmt.Sprintf` in event type, missing `WithCorrelationID`, wrong error wrapping)
- **B004-B015** (12 boilerplate rules): nice-to-have patterns (e.g., manual metadata handling, manual checkpoint logic)
- **D003-D005** (3 consistency rules): minor style checks
- **E001-E003, E006-E007** (5 architecture rules): require cross-module analysis (e.g., layer violations, cyclic dependencies)
- **S002-S003** (2 security rules): missing encryption/signing detection
- **C004** (checkpoint-before-async): partial detection logic exists in registry but no detector
- **C011** (non-deterministic decider): requires deeper analysis

### Auto-fix: 3 of 18 auto-fixable rules wired

The CQRSFixProvider works and handles C006 and C003 via BeforeCode/AfterCode matching. But 15 more rules are marked auto-fixable in the research doc and have no fix data attached:

- C001 (missing tx commit): has BeforeCode/AfterCode in the detector but not tested with the fix provider end-to-end
- C002 (broken command ID): marked as suggest-only
- C005 (raw json.Unmarshal): suggest-only
- C010 (swallowed error): no fix data

### Suppression: works but limited

The suppression filter only checks the finding's `Snippet` field for `//cqrs-lint:ignore` comments. It does NOT parse AST comments from the actual source file at the finding's line. This means suppression only works when the Snippet happens to contain the comment text, which is unreliable.

### Integration tests: none written

The execution plan (L14, S41-S44) specified integration tests that run the linter against real consumer projects (Kernovia, DiscordSync, bank-sync, storbi, crush-daily) and verify specific rule findings. These were not written. The linter was verified manually against these projects, but there are no automated integration tests.

---

## 4. What Is NOT STARTED

### From the execution plan (Phase 3-4 items)

- [ ] **L30: Rule A008 test** (rule is implemented, no test for the OO-detection variant)
- [ ] **L31-L34: Rules C004, C008** (C008 implemented, C004 not started)
- [ ] **L35-L36: Rules E004-E005** (implemented, E001-E003/E006-E007 not started)
- [ ] **L38: Config file** (`.cqrs-lint.json` schema and loader — not started)
- [ ] **L39: `--fast` mode** (implemented as flag but no benchmark proving <2s)
- [ ] **L40-L41: Rules D001-D005** (D001-D002 implemented, D003-D005 not started)
- [ ] **L42-L43: Rules E001-E007** (only E004-E005 implemented)
- [ ] **L44-L45: Rules B004-B015, A009-A020** (not started)
- [ ] **L46: golangci-lint plugin** (go/analysis wrapper — not started)
- [ ] **L47: LSP mode** (`cqrs-lint lsp` — not started)
- [ ] **L48: README** (done but could be more comprehensive)
- [ ] **L49: Recommendation engine** (pattern triggers + migration suggestions — not started)
- [ ] **L50: Doctor command** (verify go.mod, verify package loading — not started)

### From the research document

- [ ] **Version awareness** (v3 vs v4 API differences — rules don't disable for unavailable APIs)
- [ ] **Confidence-based filtering** (`--min-confidence high` flag — not implemented)
- [ ] **Configurable severity overrides** (per-rule severity via config — not implemented)
- [ ] **CorrelateFindings** pipeline feature (cross-rule correlation — not enabled)
- [ ] **VerifyAfterFix** pipeline feature (re-run detectors after fix — not enabled)
- [ ] **Metrics collection** (pipeline.Metrics for per-detector timing — not enabled)

---

## 5. What Is TOTALLY FUCKED UP

### CI violations: 3 files exceed 350-line limit

This is the biggest immediate problem. The repo's CI enforces a 350-line max per file:

| File                             | Lines   | Over by                         |
| -------------------------------- | ------- | ------------------------------- |
| `pkg/rules/correctness/rules.go` | **756** | **406 lines over (2.2x limit)** |
| `pkg/rules/api/rules.go`         | **465** | **115 lines over**              |
| `pkg/analyzer/builder.go`        | **412** | **62 lines over**               |
| `main.go`                        | 320     | OK (under limit)                |

`correctness/rules.go` is the worst offender — it crams all 10 correctness rules + helpers into one file. The CI WILL FAIL on this. Each rule should be in its own file (the execution plan specified `c006.go`, `c003.go`, `c002.go`, etc. — one file per rule).

### Suppression filter is fundamentally broken

The suppression parser only looks at the finding's `Snippet` field, which is populated from... nowhere. The detectors never set `Snippet`. So the suppression filter will NEVER match in practice. The filter needs to either:

1. Parse AST comments from the source file at the finding's line/position, or
2. Have detectors populate the Snippet field with surrounding source context

### No `_ = sel` dead code

In `api/rules.go` line 120, there's `_ = sel` — a dead variable from refactoring that was never cleaned up.

### C010 detector has a name-matching bug

The C010 detector tries to match `fold.FuncName` against `fn.Name.Name`, but `FuncName` for methods includes the receiver (e.g., `(*State).fold`), so the comparison `fn.Name.Name != fold.FuncName` will always fail for methods.

---

## 6. What We Should Improve

### Immediate (blocks CI / correctness)

1. **Split `correctness/rules.go` (756 lines) into one file per rule** — `c001.go`, `c002.go`, `c003.go`, etc. Move shared helpers to `helpers.go`.
2. **Split `api/rules.go` (465 lines) into one file per rule** — `a001.go`, `a002.go`, etc.
3. **Split `analyzer/builder.go` (412 lines)** — separate AST scanner from helper functions.
4. **Fix suppression filter** — parse AST comments at the finding's line instead of checking Snippet.
5. **Remove `_ = sel` dead code** in `api/rules.go`.
6. **Fix C010 method-name matching** — use `funcName(fn)` instead of `fn.Name.Name`.

### Near-term (improves quality)

7. **Add integration tests** that run the linter against real consumer projects (L14).
8. **Write C004 detector** (checkpoint-before-async) — partially analyzed in the registry.
9. **Add more test cases** — edge cases, false-positive fixtures, multi-file scenarios.
10. **Enable `VerifyAfterFix`** in the pipeline for real fix verification.
11. **Populate `Snippet` field** in detectors — improves output quality and enables suppression.
12. **Add `--min-confidence` flag** — useful for CI (only fail on high-confidence findings).
13. **Test the `--fix` flag end-to-end** — current fix provider tests are unit-level only.

### Future (expands capabilities)

14. **Implement remaining 21 rules** (A009-A020, B004-B015, D003-D005, E001-E003/E006-E007, S002-S003).
15. **Add `.cqrs-lint.json` config file** for per-project rule configuration.
16. **Add golangci-lint plugin** (go/analysis wrapper for ecosystem integration).
17. **Add LSP mode** for real-time editor feedback.
18. **Add version awareness** — detect v3 vs v4 and disable rules for unavailable APIs.
19. **Add doctor command** — verify setup health.
20. **Add recommendation engine** — migration suggestions (e.g., "you're on v3, consider migrating to v4").

---

## 7. Next 50 Things To Get Done

### Splitting & CI fixes (Priority 1)

1. Split `pkg/rules/correctness/rules.go` into `c001.go`, `c002.go`, `c003.go`, `c005.go`, `c006.go`, `c007.go`, `c008.go`, `c009.go`, `c010.go`, `c012.go`, `helpers.go`
2. Split `pkg/rules/api/rules.go` into `a001.go` through `a008.go`
3. Split `pkg/analyzer/builder.go` into `scanner.go` + `helpers.go`
4. Fix suppression filter to parse AST comments at finding position
5. Remove `_ = sel` dead code in `api/rules.go`
6. Fix C010 method-name matching logic
7. Run `nix fmt` on all new files
8. Verify CI passes (`nix run .#build`)

### Missing rules (Priority 2)

9. Implement C004 (checkpoint-before-async-complete)
10. Implement C011 (non-deterministic decider)
11. Implement D003 (inconsistent aggregate type naming)
12. Implement D004 (snake_case vs camelCase event payload fields)
13. Implement D005 (inconsistent command naming convention)
14. Implement E001 (layer violation — Tier 0 importing Tier 3+)
15. Implement E002 (cyclic module dependency)
16. Implement E003 (event type not in decider fold)
17. Implement E006 (projection not registered with host)
18. Implement E007 (command handler registered twice)
19. Implement S002 (missing event signing)
20. Implement S003 (missing event encryption for sensitive payloads)
21. Implement B004 (manual metadata handling — use metadata.CustomData)
22. Implement B005 (manual checkpoint logic — use projectionhost)
23. Implement B006 (manual event bus construction — use stack preset)
24. Implement B007 (manual handler registration — use RegisterAll)
25. Implement B008-B015 (remaining boilerplate patterns)
26. Implement A009 (fmt.Sprintf in event type)
27. Implement A010 (missing WithCorrelationID in decider)
28. Implement A011 (wrong error wrapping — not using error-family)
29. Implement A012-A020 (remaining API misuse patterns)

### Test improvements (Priority 3)

30. Write integration test: run linter on bank-sync, verify specific rules fire
31. Write integration test: run linter on DiscordSync, verify C009 fires
32. Write integration test: run linter on crush-daily, verify C007 fires
33. Write false-positive test: C007 with time.Now outside decider
34. Write false-positive test: C005 with non-event json.Unmarshal
35. Write test: `--fix` end-to-end on a fixture with C006
36. Write test: `--fix` end-to-end on a fixture with C003
37. Write test: suppression comment actually suppresses a finding
38. Write test: health score computation edge cases (0 findings, all-critical)
39. Write test: `--fast` mode only runs critical rules
40. Write test: `--only correctness` filter works
41. Write test: `--min-severity error` filter works

### Feature gaps (Priority 4)

42. Implement `.cqrs-lint.json` config file loading
43. Implement `--min-confidence` flag
44. Implement `doctor` command
45. Populate `Snippet` field in all detectors
46. Enable `VerifyAfterFix` in pipeline config
47. Enable pipeline `Metrics` for per-detector timing
48. Add version awareness (v3 vs v4 API detection)
49. Add golangci-lint plugin wrapper
50. Add LSP server mode

---

## 8. Top 2 Questions

### Q1: Should cqrs-lint be a separate Go module or a cmd/ subdirectory of go-cqrs-lite?

Currently it's `github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint` — a cmd subdirectory like cqrs-gen. But cqrs-lint is more than a code generator; it's a full analysis tool with its own package structure (`pkg/analyzer`, `pkg/rules/*`, `pkg/fix`, `pkg/suppression`). It also depends on go-finding and go-finding/pipeline, which none of the library modules depend on.

Should it be broken out into its own repository (`github.com/larsartmann/cqrs-lint`) for cleaner dependency isolation? Or stay as a cmd to keep the analysis tooling versioned alongside the library it lints?

### Q2: The research doc designed for cmdguard CLI scaffolding but we hand-rolled the CLI. Should we migrate?

The research doc (Section 4.4) specified using `cmdguard` for CLI scaffolding — struct-tag flags, config files, doctor command, output formats. The actual implementation hand-rolls CLI parsing in `main.go` (320 lines of flag parsing, switch statements, manual help text). This works but:

- Missing: config file support, struct-tag validation, typo suggestions, automatic help generation
- Risk: the hand-rolled CLI diverges from the research design intent
- Tradeoff: migrating to cmdguard adds a dependency but eliminates ~200 lines of boilerplate

Should we migrate now (before more features are built on the hand-rolled CLI) or keep the simple approach?

---

## 9. Raw Numbers

| Metric                             | Value                                                                          |
| ---------------------------------- | ------------------------------------------------------------------------------ |
| Go source files                    | 24                                                                             |
| Total lines of Go                  | ~1658                                                                          |
| Rules implemented                  | 26 of 47 (55%)                                                                 |
| Test functions                     | 30                                                                             |
| Test packages                      | 10                                                                             |
| All tests passing                  | Yes (319 total including existing tests)                                       |
| Full workspace build               | Passes                                                                         |
| Files over 350-line CI limit       | **3** (correctness/rules.go: 756, api/rules.go: 465, analyzer/builder.go: 412) |
| Consumer projects verified         | 5 (bank-sync, DiscordSync, crush-daily, storbi, taskmanager)                   |
| Execution plan tasks completed     | ~20 of 50 Level-1 tasks (40%)                                                  |
| Execution plan sub-tasks completed | ~50 of 117 Level-2 tasks (43%)                                                 |
