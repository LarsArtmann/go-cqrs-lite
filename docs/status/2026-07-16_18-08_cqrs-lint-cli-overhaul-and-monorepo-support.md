# cqrs-lint: CLI Overhaul + Monorepo Support Status

**Date:** 2026-07-16 18:08
**Session focus:** Fix broken CLI output, enable fang/go-output, add monorepo support
**Previous report:** `2026-07-16_17-16_cqrs-lint-p2-p3-brutal-self-review.md`

---

## What triggered this session

1. **Compile error**: `slices.Contains()` called with no arguments in `a011_a014_a017.go:36` — the previous session's snippet population work accidentally deleted `looksLikeEventPayload(name)` and replaced it with a broken stub
2. **No colors, no fang**: `WithFang(false)` explicitly disabled styled help. Hand-rolled ANSI in `color.go` instead of using the `go-output` library already available as a transitive dep
3. **Health-score replaced findings**: `--health-score` was an early return that threw away all finding output — you'd see a score but not WHAT the problems were or WHERE they were
4. **No monorepo support**: `BuildContext` loaded one module via `packages.Load(cfg, "./...")`. Monorepos with multiple `go.mod` files (SwettySwipperWeb: 10 modules) were invisible — pointing at root gave "No Go files found"

---

## a) FULLY DONE

| Area                       | What was done                                                                                                                                                                                                           | Verification                                            |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| **Compile fix (A011)**     | Restored `looksLikeEventPayload()` helper, removed unused `"slices"` import, fixed `slices.Contains()` → `looksLikeEventPayload(name)`                                                                                  | Build passes, A011 detector now runs                    |
| **Fang enabled**           | Removed `WithFang(false)` from cmdguard setup. Default is `true` — now gets styled help, `--no-color`, `--version`, clean error display                                                                                 | `cqrs-lint --help` shows styled output                  |
| **go-output integrated**   | Promoted `go-output` + `go-output/table` from indirect to direct deps. `rules` command renders as a proper colored table (ID, Rule, Severity, Category, Description, footer with count)                                 | `cqrs-lint rules --color always` renders lipgloss table |
| **Health-score appends**   | Removed early `return nil`. Now findings print first, then health score tables appear below. Score renders as two go-output tables: score/grade + deduction breakdown sorted by impact                                  | `cqrs-lint --health-score` shows findings THEN score    |
| **Init config keys fixed** | `min_severity` → `min-severity`, `min_confidence` → `min-confidence` — now matches `AppConfig` struct tags (`flag:"min-severity"`)                                                                                      | `cqrs-lint init` creates valid config                   |
| **Monorepo support**       | `BuildContext` now walks directory tree via `filepath.WalkDir`, discovers all `go.mod` files (skipping vendor/.git/node_modules), loads each module separately with shared `FileSet`, merges into one `AnalysisContext` | SwettySwipperWeb: 0 → 130 files across 10 modules       |
| **color.go deleted**       | Replaced 101-line hand-rolled ANSI file with `output.go` using `go-output.ColorMode` for color detection. ANSI codes retained for finding-level output (severity colors)                                                | All tests pass                                          |
| **Test cleanup**           | Removed duplicate `healthGrade()` and `deduplicate()` test-only re-implementations. Updated health tests to test real `ComputeHealthScore` + `renderHealthScore`                                                        | 114 tests pass, 0 failures                              |
| **Golden file updated**    | `json_output.json` updated (A011 now emits findings that were previously lost)                                                                                                                                          | Golden test passes                                      |
| **Lint clean**             | Fixed exhaustive switch linter complaint on `output.ColorMode`                                                                                                                                                          | `nix run .#lint` — cqrs-lint shows 0 issues             |

### Files changed (11 files, +175/-206 lines net reduction)

| File                                                | Change                                                                                                       |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `cmd/cqrs-lint/color.go`                            | **DELETED** (replaced by output.go)                                                                          |
| `cmd/cqrs-lint/output.go`                           | **NEW** — go-output integration, table rendering, ANSI finding output                                        |
| `cmd/cqrs-lint/main.go`                             | Removed `WithFang(false)`, health-score moved after findings, `formatFindingsText()` replaces old color path |
| `cmd/cqrs-lint/commands.go`                         | `rules` command uses `renderRulesTable()`, init command kept here (was duplicated)                           |
| `cmd/cqrs-lint/health.go`                           | Removed `FormatHealthScore()`, added `sortedBreakdown()` + `breakdownEntry` for table rendering              |
| `cmd/cqrs-lint/health_test.go`                      | Removed `healthGrade()` duplicate, tests `renderHealthScore` + `ComputeHealthScore` directly                 |
| `cmd/cqrs-lint/init.go`                             | Config template keys fixed to kebab-case                                                                     |
| `cmd/cqrs-lint/main_test.go`                        | Removed `deduplicate()` duplicate, inline test logic                                                         |
| `cmd/cqrs-lint/go.mod`                              | `go-output` + `go-output/table` promoted to direct deps                                                      |
| `cmd/cqrs-lint/pkg/analyzer/loader.go`              | **Monorepo support**: `findGoModDirs()` + per-module loading                                                 |
| `cmd/cqrs-lint/pkg/rules/testdata/json_output.json` | Golden updated (A011 findings now present)                                                                   |

### Current state

- **61 rules**, **114 tests** passing, **0 lint issues**
- **7 Go files** in cmd/cqrs-lint/ root, all under 350 lines (largest: main.go at 250)
- **Build**: `cd cmd/cqrs-lint && GOWORK=off go build -tags "goexperiment.jsonv2" ./...`

---

## b) PARTIALLY DONE

| Area                   | Status                                                                                                    | What's missing                                                                                                                                                                                         |
| ---------------------- | --------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Snippet coverage**   | ~14 detector files have `.WithSnippet()`                                                                  | 47 detector files still have no snippets — API (A001-A019), boilerplate (B001-B015), architecture (E001-E007), consistency (D001-D005), security (S002-S003) detectors still lack `SourceLine()` calls |
| **go-output adoption** | Rules table + health-score table use go-output                                                            | Finding output still uses hand-rolled ANSI (`formatFindingsText`) instead of go-output tables. The finding-level color is fine for now (one finding per block), but could be a table for compact view  |
| **Verbose mode**       | Flag exists (`--verbose`)                                                                                 | Does nothing — no additional output beyond the default. Should show rule execution timing, per-module breakdown, skipped files, etc.                                                                   |
| **GOWORK sensitivity** | Works when `go.work` is present (normal `go run`) and when `GOWORK=off` is set for cqrs-lint's own module | If a consumer project has a `go.work` that conflicts with its `go.mod` replace directives, `packages.Load` may fail. Not handled gracefully                                                            |

---

## c) NOT STARTED

From the prior 34-item plan (P0-P4), these remain:

### P0 (doc drift / cleanup)

- [ ] Update `README.md`: still says "52 rules", needs 61 + new CLI features documented
- [ ] Update `AGENTS.md`: still says "52 rules" in cqrs-lint module entry

### P1 (snippets/performance)

- [ ] Extend snippets to remaining 47 detector files
- [ ] Add file caching to `SourceLine()` (reads file from disk every call)
- [ ] Fix A009 positioning (points at `go.mod:1:1` instead of actual code)

### P2-P4 (features)

- [ ] SARIF golden test
- [ ] Property-based tests (rapid)
- [ ] Per-rule integration test (one test file per rule with real source)
- [ ] `.cqrs-lintignore` support
- [ ] `cqrs-lint doctor` command (diagnose config/env)
- [ ] SARIF metadata enrichment
- [ ] `--rules-config` flag (custom rule config file)
- [ ] Pre-commit hook script
- [ ] `--watch` mode (re-lint on file change)
- [ ] Split `filters.go` helpers
- [ ] Pin dependency versions in go.mod
- [ ] `check-module-layers.sh` update
- [ ] CI badge in README

---

## d) WHAT I FUCKED UP

1. **A011 compile error (previous session)**: The snippet-population commit (`2114eef0`) replaced `looksLikeEventPayload(name)` with `slices.Contains()` — no arguments. This made A011 non-compiling, which means **the entire cqrs-lint binary wouldn't build**. This was a P0 severity regression that should have been caught by running `go build` before committing. The user found it by trying to run the linter.

2. **GOWORK=off in packages.Load**: When adding monorepo support, I initially set `Env: append(os.Environ(), "GOWORK=off")` in the packages.Config. This **broke all consumer projects** that rely on `go.work` for replace directives (like SwettySwipperWeb pointing at local go-cqrs-lite checkouts). The `GOWORK=off` was needed for cqrs-lint's own tests, not for the linter's runtime. I removed it after testing showed services/api couldn't resolve deps. This should have been caught before committing.

3. **Fang was disabled for no reason**: `WithFang(false)` was set in the previous session with no documented reason. It disabled styled help, colored errors, `--version`, `--no-color`. The fix was trivial (delete one line) but it never should have been there.

4. **Health-score replaced findings**: The previous session implemented `--health-score` as an early `return nil` that threw away all finding output. The user's complaint ("Where are they? PROPER FUCKING LINTING") was about this exact design flaw. Health score should always have been an **addendum** to findings, not a replacement.

5. **Duplicate test functions**: `healthGrade()` and `deduplicate()` were test-only reimplementations of production logic, testing the copies instead of the real code. These were created in the previous session and should never have existed.

---

## e) WHAT WE SHOULD IMPROVE

1. **Always run `go build` before committing** — the A011 compile error was 100% preventable
2. **Test against real projects before committing** — the monorepo bug, the GOWORK=off bug, and the health-score-replaces-findings UX issue would all have been caught by a 10-second manual test
3. **Don't disable framework features without reason** — fang was disabled for no documented cause
4. **Use the libraries you already have** — go-output was a transitive dep the whole time, but hand-rolled ANSI was used instead
5. **Health-score should never replace findings** — it's supplementary information, not the primary output
6. **Monorepo support should have been day-one** — cqrs-lint itself IS a monorepo. The fact that it couldn't lint monorepos was a fundamental oversight
7. **The `--verbose` flag does nothing** — this is lying to users. Either implement it or remove it
8. **Finding output format** — each finding is a 4-line block (severity, location, rule, suggestion). For 30+ findings this is very tall. Consider a compact table mode or `--format compact`
9. **No progress indicator** — analyzing 130 files takes ~1s, but there's no spinner or progress feedback during analysis
10. **Module-level summary missing** — in a monorepo scan, findings aren't grouped by module. `services/api` findings mix with `services/battle` findings with no separation

---

## f) NEXT 50 THINGS TO DO

### Bugs & Fixes (P0)

1. Fix A009 positioning — points at `go.mod:1:1` instead of actual import site
2. Fix D005 false positive — flags README.md referencing "vendored-cqrs" when module path doesn't match
3. Fix E006 false positive — flags struct names like `HealthScore`, `AppConfig` as "event types" because they match the `*Event` suffix heuristic poorly (actually it's any type name, the heuristic is too broad)
4. Fix D002/D004 — counts ALL json tags in a file, including non-event structs (db models, API types)
5. Review B013 detection — flags repository creation without checking if correlation is already in context
6. Review A017 — fires on every `NewRepository` call even when aggregates are small

### CLI Polish (P1)

7. Implement `--verbose` properly: show per-module file counts, rule execution time, skipped files
8. Add `--format compact` — one-line-per-finding format for terminal overflow scenarios
9. Add `--format table` — go-output table with columns: Severity, Location, Rule, Message
10. Add progress indicator (spinner during package loading)
11. Group findings by module in monorepo scans (print module header before its findings)
12. Add `--sort` flag (by severity, by file, by rule)
13. Add `--stats` flag (show rule hit-count histogram)
14. Fix `--health-score` to work with `--format json` (currently only text)
15. Add exit code documentation to `--help` (0=clean, 1=errors found, 2=internal error)
16. Color the health-score grade (green=Excellent, yellow=Good, orange=Fair, red=Needs Improvement)
17. Add `cqrs-lint explain C001` command — show detailed rule docs with examples
18. Add `cqrs-lint doctor` — diagnose config, check go version, verify build tags

### Snippet & Finding Quality (P1-P2)

19. Extend snippets to all 47 remaining detector files
20. Add file caching to `SourceLine()` — currently reads entire file from disk per call
21. Add `Before`/`After` context lines to snippets (like ESLint)
22. Add `--context N` flag to control snippet context lines
23. Add fix diff preview to `--dry-run` output (show before/after)

### Rule Improvements (P2)

24. Add C013: event store `Save()` without optimistic concurrency check
25. Add C014: missing `context.Done()` check in long-running handlers
26. Add A020: using `event.Event` interface instead of concrete type alias
27. Add B016: repeated `fmt.Errorf` wrapping pattern — use `errorfamily.Wrap*`
28. Add D006: inconsistent error wrapping (mixed `fmt.Errorf` + `errors.Wrap`)
29. Add E008: module imports a deprecated CQRS module path
30. Improve S002: check for PII in event payloads structurally, not just field names
31. Improve S003: check for signing in middleware chain, not just publish
32. Add rule categories to JSON/SARIF output (currently only in rules table)
33. Add confidence scores to JSON output

### Testing & CI (P2-P3)

34. Add per-rule integration tests (one test per rule with realistic source)
35. Add property-based tests with `pgregory.net/rapid`
36. Add SARIF golden file test
37. Add monorepo test fixture (multi-module test directory)
38. Add benchmark for monorepo scanning (regression detection)
39. Add `cqrs-lint` self-lint to CI (dogfooding)
40. Add coverage report for analyzer package (currently 0% — no test files)
41. Add `.cqrs-lintignore` support (like `.gitignore`)
42. Add pre-commit hook installation script

### Architecture & DX (P3-P4)

43. Add `--rules-config` flag for custom rule configuration
44. Add `--watch` mode for re-linting on file change
45. Add `check-module-layers.sh` integration
46. Add CI badge to README
47. Add `cqrs-lint init --strict` for stricter defaults
48. Add shell completions (`cqrs-lint completion bash`)
49. Add `cqrs-lint baseline` command (generate baseline file for incremental adoption)
50. Add `cqrs-lint diff` command (only show new findings since last baseline)

---

## g) Top 2 Questions I Cannot Answer Myself

### 1. Should finding output use go-output tables or keep the current block format?

The current block format (severity + location on line 1, rule on line 2, suggestion on line 3, snippet on line 4) is readable for a few findings but becomes a wall of text at 30+ findings. A table format (`Severity | Location | Rule | Message`) would be more compact and scannable, but less detailed (no suggestion/snippet inline). Options:

- **A**: Keep blocks as default, add `--format table` for compact view
- **B**: Switch to table as default, add `--format detailed` for the current block format
- **C**: Auto-detect — use blocks for <10 findings, table for 10+

This is a UX/product decision, not a technical one.

### 2. Should cqrs-lint report findings from modules that have `go.mod` errors (e.g. `go mod tidy` needed)?

When scanning SwettySwipperWeb, several modules had `go.mod` issues (`go: updates to go.mod needed; to update it: go mod tidy`). Currently these modules are silently skipped (the `continue` on `pkg.Errors`). Options:

- **A**: Continue silently skipping (current behavior — may confuse users why a module has 0 findings)
- **B**: Emit a warning finding: "Module X has dependency errors — run `go mod tidy`. Analysis skipped."
- **C**: Try to load anyway with degraded analysis (syntax-only, no type info)

This affects user trust — silent skips look like the linter missed something.
