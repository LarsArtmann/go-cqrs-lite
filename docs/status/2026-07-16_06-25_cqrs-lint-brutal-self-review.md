# cqrs-lint Implementation Status — 2026-07-16

## Executive Summary

This session executed the cqrs-lint execution plan: migrated the CLI from hand-rolled flags to cmdguard, split all oversized files to meet CI's 350-line limit, fixed 3 bugs, implemented 26 new rules (26→52), added integration tests, config file support, `--min-confidence` flag, and updated documentation.

**However, a brutally honest self-review reveals significant problems that must be addressed before this is production-ready.**

---

## a) FULLY DONE (Working & Verified)

| #   | Item                              | Details                                                                                                                                | Verification                                     |
| --- | --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| 1   | C010 method-name matching bug fix | Fold name receiver prefix stripped before matching                                                                                     | Build + tests pass                               |
| 2   | Suppression filter rewrite        | Now reads actual source files instead of empty Snippet field; file-cache with fallback                                                 | 4 tests pass, including real-file test           |
| 3   | Dead code removal                 | Removed `_ = sel` (correctness + boilerplate), dead `prefix` variable in `detectorCategory`                                            | Build passes                                     |
| 4   | correctness/rules.go split        | 953 lines → 12 per-rule files (c001.go through c012.go) + helpers.go + doc.go                                                          | All <350 lines, tests pass                       |
| 5   | api/rules.go split                | 580 lines → 8 per-rule files (a001.go through a008.go) + helpers.go + doc.go                                                           | All <350 lines, tests pass                       |
| 6   | analyzer/builder.go split         | 452 lines → scanner.go (346) + ast_helpers.go (114)                                                                                    | All <350 lines, tests pass                       |
| 7   | C004 + C011 correctness rules     | C004: checkpoint-before-async-complete; C011: nondeterministic-decider (rand.* in decider)                                             | Build passes, integration test finds 57 findings |
| 8   | 19 new rules implemented          | A009-A019, B004-B014, D003-D005, E001-E007, S002-S003                                                                                  | Build passes, registered in pipeline             |
| 9   | `--min-confidence` flag           | Filters findings by confidence level (low/medium/high)                                                                                 | CLI test passes                                  |
| 10  | Config file support               | `.cqrs-lint.json` auto-loaded via cmdguard's `WithConfigFile`                                                                          | CLI build passes                                 |
| 11  | Integration tests                 | 3 tests: full pipeline run against taskmanager (52 detectors, 57 findings), critical subset verification, category filter verification | All pass with workspace enabled                  |
| 12  | README rewrite                    | Complete rule reference table (52 rules), CLI flags table, config file docs, architecture description                                  | Written                                          |
| 13  | AGENTS.md updated                 | Module description updated: "52 rules across 6 categories"                                                                             | Written                                          |
| 14  | nix fmt                           | All files formatted                                                                                                                    | 6 files changed on final run                     |
| 15  | All workspace tests pass          | cqrs-lint (33 test functions) + event/command/decider/codec/dispatcher/metadata/schema/snapshot etc.                                   | Only pre-existing `id/v4` fuzz failure           |
| 16  | No files exceed 350 lines         | All 56 .go files verified                                                                                                              | `find` check passes                              |

---

## b) PARTIALLY DONE

| #   | Item                      | What's Done                                                                                                                                                                 | What's Missing                                                                                                                                                                                              |
| --- | ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Rule implementation       | 52 rules registered with detectors                                                                                                                                          | **9 rules from research doc NOT implemented**: A011, A014, A017, B006, B007, B009, B010, B012, B015. These are catalog-listed but have no detector — they silently produce zero findings                    |
| 2   | Stub detectors            | 4 detectors return `nil, nil` unconditionally: E002 (circular-dependency), E003 (missing-module-boundary), E007 (query-without-handler), D005 (stale-documentation-version) | Need actual detection logic or explicit "not-yet-implemented" documentation                                                                                                                                 |
| 3   | Test coverage             | 33 test functions, integration tests work                                                                                                                                   | **Most new rules have ZERO unit tests**: Only C001-C009, A002, A006, A008, B001, D002, E004, S001 have tests. All new rules (C004, C011, A009-A019, B004-B014, D003-D004, E001, E006, S002-S003) lack tests |
| 4   | CLI migration to cmdguard | Replaces hand-rolled flags, subcommands work, config file loads                                                                                                             | `os.Exit(1)` called inside `run()` function (line 211) instead of returning error — untestable, bypasses cmdguard's exit code handling                                                                      |

---

## c) NOT STARTED

| #   | Item                                      | Impact       | Notes                                                                                                                                                                                                                                                      |
| --- | ----------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Depguard allow list update                | **BLOCKING** | `.golangci.yml` depguard does NOT allow `go-finding`, `cmdguard/v3`, `spf13/cobra`, or `charm.land/*` — lint will FAIL on cqrs-lint module                                                                                                                 |
| 2   | Dependency budget review                  | **BLOCKING** | cmdguard pulls 52 transitive deps (charm.land/fang, lipgloss, go-output, samber/do, samber/do-auditlog, etc.). The project enforces per-module dep budgets via `check-layers`. cqrs-lint went from 0 transitive to 52 — this is a massive budget violation |
| 3   | `check-layers` script update              | High         | `scripts/check-module-layers.sh` likely doesn't know about cqrs-lint's dependency graph                                                                                                                                                                    |
| 4   | Snippet field population                  | Medium       | Detectors never set `Finding.Snippet` — the suppression fallback works, but SARIF/JSON output lacks code context                                                                                                                                           |
| 5   | `--only` flag for individual rule IDs     | Medium       | Currently only filters by category, not individual rule IDs (e.g., `--only C001,C002`)                                                                                                                                                                     |
| 6   | Doctor command                            | Low          | cmdguard provides `DoctorCommand` infrastructure but it's not wired up                                                                                                                                                                                     |
| 7   | Version subcommand using cmdguard version | Low          | Currently uses custom `version` string instead of `cmdguard.WithCLIVersion` properly integrated                                                                                                                                                            |

---

## d) TOTALLY FUCKED UP

| #   | Issue                                              | Severity     | Details                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| --- | -------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Dependency explosion from cmdguard**             | **CRITICAL** | Adding cmdguard pulled in **52 indirect dependencies** including the entire charm.land ecosystem (fang, lipgloss, colorprofile, ultraviolet, x/ansi, x/term, x/termios, x/windows), go-output (14 sub-modules), samber/do + samber/do-auditlog, spf13/cobra + pflag, mousetrap, and more. For a LINTER tool, this is absurd. The previous hand-rolled CLI had ZERO external deps beyond go-finding and golang.org/x/tools. This needs immediate attention. |
| 2   | **`os.Exit(1)` inside `run()` function**           | **High**     | The function `run()` calls `os.Exit(1)` directly in a for-loop at line 211 when it finds error-severity findings. This makes the function untestable (any test calling `run()` with error findings kills the test process), bypasses cmdguard's `ExecuteAndExit` error handling, and prevents proper cleanup/defer execution. Should return a sentinel error and let the caller decide the exit code.                                                      |
| 3   | **`AConfig` struct tag formatting**                | Medium       | The `MinConfidence` field in `AppConfig` has inconsistent struct tag alignment vs other fields — gofmt should fix but it survived `nix fmt`, suggesting the tag format may confuse formatters.                                                                                                                                                                                                                                                             |
| 4   | **`scanner.go` at 346 lines — barely under limit** | Medium       | scanner.go is 346/350 lines. Adding any more scanning logic will breach CI. Should proactively split into scanner_commands.go + scanner_folds.go.                                                                                                                                                                                                                                                                                                          |
| 5   | **Broken git index warnings**                      | Low          | `nix fmt` reports warnings about deleted files still in the git index (`builder.go`, `rules.go`). Need `git add` to stage the deletions.                                                                                                                                                                                                                                                                                                                   |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **Reconsider cmdguard dependency** — 52 transitive deps for a linter CLI is unacceptable. Options: (a) slim cmdguard without fang/charm.land, (b) keep hand-rolled CLI, (c) extract a minimal flag-parsing subset
2. **Return errors, not `os.Exit`** — The `run()` function must return a typed error for the exit-code logic. Use `finding.SeverityError` as a sentinel
3. **Proactive file splitting** — scanner.go (346 lines) and helpers.go (332 lines) are dangerously close to the 350-line limit
4. **Snippet field population** — Detectors should populate `Finding.Snippet` with the source line for better SARIF/IDE integration
5. **Rule interface simplification** — Each detector follows the same pattern (iterate GoFiles, ast.Inspect, build finding). Consider a higher-level helper that reduces boilerplate per rule

### Testing

6. **Unit tests for ALL new rules** — 26 new rules have zero tests. Each rule needs at minimum: a positive test (detects the pattern) and a negative test (doesn't false-positive)
7. **Test for stub detectors** — Verify E002/E003/E007/D005 return empty (or implement them)
8. **Golden file tests** — Verify JSON/SARIF output format stability across versions
9. **Property-based testing** — Use rapid (already a dependency) for AST edge cases
10. **Benchmark tests** — Measure per-rule overhead on large codebases

### Rule Quality

11. **A015 (global-mutable-state) false positive risk** — Detects any var with "cache"/"registry"/"instance" in name, which catches legitimate uses like `var ErrCacheMiss`
12. **A009 (missing-stack-preset) positioning** — Reports at `go.mod:1:1` which is imprecise for SARIF consumers
13. **D003 (inconsistent-logging) overcounting** — Transitive imports from libraries may trigger false positives
14. **B008 (manual-retry) heuristic** — Requires both a "retry"-named variable AND `time.Sleep` — too narrow, misses backoff patterns without Sleep
15. **S002 (missing-encryption) trigger logic** — Detects PII by event payload TYPE NAME containing "email"/"phone" etc., not by actual field analysis

### CI & Infrastructure

16. **Update depguard allow list** — Add go-finding, cmdguard, cobra to `.golangci.yml`
17. **Update `check-layers` budget** — Account for cmdguard's transitive dep tree or find a slimmer alternative
18. **Wire cqrs-lint into CI** — The module should be in the `nix run .#build` and `nix run .#lint` pipelines
19. **Pin go-finding and cmdguard versions** — Currently using `00010101000000-000000000000` pseudo-versions with replace directives. Need proper version tags
20. **`go.work` entry verification** — Ensure the workspace correctly resolves cqrs-lint's replace directives

---

## f) Next 50 Tasks (Sorted by Impact × Urgency)

### P0 — Blocking (must fix before merge)

| #   | Task                                                                                     | Est. Time |
| --- | ---------------------------------------------------------------------------------------- | --------- |
| 1   | Fix depguard allow list — add go-finding, cmdguard, cobra, charm.land to `.golangci.yml` | 5 min     |
| 2   | Replace `os.Exit(1)` in `run()` with returned sentinel error                             | 10 min    |
| 3   | Run `nix run .#lint` on cqrs-lint module and fix ALL linter errors                       | 30 min    |
| 4   | Run `nix run .#build` and verify it compiles cqrs-lint                                   | 5 min     |
| 5   | Stage deleted files with `git add` to fix git index warnings                             | 2 min     |
| 6   | Audit dependency budget — decide if cmdguard's 52 transitive deps are acceptable         | 15 min    |

### P1 — High Impact

| #   | Task                                                                   | Est. Time |
| --- | ---------------------------------------------------------------------- | --------- |
| 7   | Implement A011 (inconsistent JSON key casing — event payload specific) | 10 min    |
| 8   | Implement A014 (deprecated API usage detection)                        | 15 min    |
| 9   | Implement A017 (missing snapshot strategy for large aggregates)        | 15 min    |
| 10  | Implement B006 (duplicate FK stub SQL)                                 | 10 min    |
| 11  | Implement B007 (repeated handler registration — suggest table-driven)  | 10 min    |
| 12  | Implement B009 (emit function boilerplate)                             | 10 min    |
| 13  | Implement B010 (catalog event list boilerplate)                        | 10 min    |
| 14  | Implement B012 (make-event helper)                                     | 10 min    |
| 15  | Implement B015 (missing test utilities)                                | 10 min    |
| 16  | Implement E002 (circular dependency detection) — replace stub          | 20 min    |
| 17  | Implement E003 (missing module boundary) — replace stub                | 20 min    |
| 18  | Implement E007 (query without handler) — replace stub                  | 15 min    |
| 19  | Implement D005 (stale documentation version) — replace stub            | 15 min    |
| 20  | Write unit tests for C004 (checkpoint async)                           | 10 min    |
| 21  | Write unit tests for C011 (nondeterministic decider)                   | 10 min    |
| 22  | Write unit tests for A009-A013                                         | 30 min    |
| 23  | Write unit tests for A015-A019                                         | 30 min    |
| 24  | Write unit tests for B004-B008                                         | 20 min    |
| 25  | Write unit tests for B011-B014                                         | 20 min    |
| 26  | Write unit tests for D003-D004                                         | 15 min    |
| 27  | Write unit tests for E001 (layer violation)                            | 10 min    |
| 28  | Write unit tests for E006 (event without projection)                   | 10 min    |
| 29  | Write unit tests for S002-S003                                         | 15 min    |

### P2 — Quality Improvements

| #   | Task                                                                                   | Est. Time |
| --- | -------------------------------------------------------------------------------------- | --------- |
| 30  | Populate `Finding.Snippet` field in all detectors                                      | 45 min    |
| 31  | Add `--only C001,C002` flag for individual rule selection (not just categories)        | 15 min    |
| 32  | Split `scanner.go` (346 lines) proactively into scanner_commands.go + scanner_folds.go | 20 min    |
| 33  | Split `helpers.go` (332 lines) proactively into helpers_tx.go + helpers_ast.go         | 15 min    |
| 34  | Fix A015 false positive risk (exclude `Err*` prefixed vars)                            | 10 min    |
| 35  | Add golden file tests for JSON/SARIF output stability                                  | 30 min    |
| 36  | Add benchmark tests for pipeline performance                                           | 30 min    |
| 37  | Wire up cmdguard `DoctorCommand` for environment diagnostics                           | 15 min    |
| 38  | Pin go-finding and cmdguard with proper version tags                                   | 10 min    |
| 39  | Update `scripts/check-module-layers.sh` for cqrs-lint dependency graph                 | 15 min    |
| 40  | Add `--rules-config` flag for custom rule severity/confidence overrides per project    | 30 min    |

### P3 — Polish

| #   | Task                                                                    | Est. Time |
| --- | ----------------------------------------------------------------------- | --------- |
| 41  | Improve S002 PII detection (analyze struct fields, not just type names) | 30 min    |
| 42  | Improve B008 retry detection (detect backoff patterns without Sleep)    | 20 min    |
| 43  | Add `--exclude` flag for path exclusion patterns                        | 15 min    |
| 44  | Add colored text output for terminal (red/yellow/green by severity)     | 20 min    |
| 45  | Add `--watch` mode for continuous linting on file change                | 45 min    |
| 46  | Add `cqrs-lint init` command to generate `.cqrs-lint.json` template     | 15 min    |
| 47  | Add SARIF rule metadata (help URLs, CWE mapping for security rules)     | 30 min    |
| 48  | Add `.cqrs-lintignore` file support (path-based exclusions)             | 20 min    |
| 49  | Create CONTRIBUTING.md with rule development guide                      | 30 min    |
| 50  | Add pre-commit hook installation (`cqrs-lint install-hooks`)            | 20 min    |

---

## g) Top 2 Questions I Cannot Answer Myself

### 1. Is the cmdguard dependency acceptable despite pulling 52 transitive dependencies?

cmdguard brings the entire charm.land ecosystem (fang, lipgloss, go-output with 14 sub-modules), samber/do DI container, and spf13/cobra. For a linter that previously had ZERO non-stdlib deps, this is a massive change. The project enforces per-module dependency budgets. **Should we:**

- (a) Accept it — the dependency budget applies to library modules, not tooling?
- (b) Slim it down — ask for a cmdguard-lite without fang/charm.land?
- (c) Revert to hand-rolled CLI and lose struct-tag flags + config file loading?

### 2. Should the 9 unimplemented rules (A011, A014, A017, B006, B007, B009, B010, B012, B015) be in the catalog now or removed until implemented?

They appear in `AllRules()` metadata (the `rules` subcommand lists them), giving users the impression they're active detectors. But they have no detector registered in `RegisterAll()`, so they silently produce zero findings. **Should we:**

- (a) Remove them from the catalog until implemented (honest)?
- (b) Mark them as "planned" in the listing?
- (c) Implement them as quick stubs that at least attempt detection?
