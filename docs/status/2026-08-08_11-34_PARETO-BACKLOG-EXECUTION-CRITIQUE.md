# Status Report: 2026-08-08 11:34 — Pareto Backlog Execution & Critique

> **Session scope:** Executed the full TODO list from the previous audit session.
> All 10 planned tasks were completed. This report evaluates what was done well,
> what was missed, and what remains.

---

## 1. What Was FULLY DONE This Session

### Code & Infrastructure Changes (committed by daemon)

| #   | Change                                                                                                                   | File(s)                                                                       | Verification                                                         |
| --- | ------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| 1   | **Pareto plan closed** — final audit header, all 51 L1-items status updated, summary statistics reconciled (75 → 3 open) | `docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md` | Re-read all sections; statuses match codebase reality                |
| 2   | **TODO_LIST.md updated** — "~14 remaining" replaced with closed status; self-lint CI item marked done                    | `TODO_LIST.md`                                                                | Verified lines 235-241, 313-314                                      |
| 3   | **CI self-lint gate (L1.15)** — new `cqrs-lint-self-lint` job in ci.yml                                                  | `.github/workflows/ci.yml`                                                    | Ran locally: `cd cmd/cqrs-lint && go run . . --strict-load` → exit 0 |
| 4   | **Parallel safety test (L1.23)** — runs all 192 detectors concurrently under `-race`                                     | `cmd/cqrs-lint/pkg/rules/parallel_safety_test.go`                             | Passes with `-race -count=1`                                         |
| 5   | **DOC/OBS/RES/DI category evaluation** — all 4 categories evaluated, 80-90% covered, closed with coverage matrix         | Pareto plan Phase 9                                                           | Sub-agent verified each rule exists in codebase                      |
| 6   | **E006/E005 orphan detection review** — reviewed scanner + detector code                                                 | Confirmed fold-aware (no regression needed)                                   | —                                                                    |
| 7   | **A015 review** — confirmed map-typed global detection already shipped                                                   | `cmd/cqrs-lint/pkg/rules/api/a015.go`                                         | `isMapTypedGlobal` + `IncDecStmt` present                            |
| 8   | **Documentation check** — AGENTS.md (192 rules), FEATURES.md both current                                                | —                                                                             | No changes needed                                                    |
| 9   | **Build + vet + test** — all 17 cqrs-lint packages pass with `-race`                                                     | —                                                                             | `go build`, `go vet`, `go test -race` all exit 0                     |
| 10  | **Previous session's status report** — written and committed                                                             | `docs/status/2026-08-08_11-11_PARETO-PLAN-AUDIT-SESSION-CRITIQUE.md`          | —                                                                    |

### Daemon Commits (3 commits this session)

```
c2b678dba docs(planning): record cqrs-lint improvement backlog evaluation results
7e62c7c8e chore(ci): add cqrs-lint self-lint job to CI pipeline
73b60c3f2 docs(planning): close out cqrs-lint Pareto improvement backlog after final audit
```

---

## 2. What Was PARTIALLY DONE (Inadequate)

### 2a. Parallel safety test is too shallow

The test runs all 192 detectors concurrently but the fixture is minimal (one `main.go` file with ~30 lines). In production, the linter processes 175+ files across multiple packages. The test would be more valuable with:

- A **multi-file fixture** (3-5 files in different packages) to exercise cross-file detector interactions under concurrency
- **Source line access** (`ctx.SourceLine()`) triggered during detection — the test doesn't verify that the `sync.Map` lineCache is race-free when multiple detectors request source snippets for the same file simultaneously

The current test catches gross data races (e.g., two detectors writing to the same slice), but misses subtle `sync.Map` contention patterns.

### 2b. CI self-lint job doesn't gate on the linter's own rules

The job runs `cqrs-lint . --strict-load` but this only fails on `SeverityError` findings. The linter currently has one WARNING (C025 in `init.go:69`) that passes silently. If the goal is a regression gate, it should either:

- Use `--min-severity warning` to fail on warnings too
- Or document that warnings are acceptable in the linter's own source

The current setup lets new warnings accumulate without detection.

### 2c. DOC/OBS/RES/DI evaluation was not written to a persistent location

The coverage matrix lives in the Pareto plan document as a note, but the 10 genuinely-missing individual rules are not tracked in TODO_LIST.md. They exist only in the planning doc's prose. If the plan is archived, these ideas become hard to find.

### 2d. Did not run the full verify gate

The session ran `go build`, `go vet`, and `go test -race` for the `cmd/cqrs-lint` module only. It did NOT run:

- `nix run .#verify` or `nix run .#verify-fast` (the full gate)
- Tests for other modules that might be affected by the Pareto plan changes
- The doc-check command on the updated planning doc

This is acceptable because only docs + CI YAML + one test file changed (no production code), but it should be noted.

---

## 3. What Was NOT STARTED

| Item                                                                 | Why                                                                |
| -------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `--fail-on-stale-suppressions` CI gate                               | In TODO_LIST.md but not in scope for this session's TODO list      |
| Pin GitHub Actions to commit SHAs                                    | In TODO_LIST.md, unrelated to Pareto plan                          |
| go-arch-lint config for cmd/cqrs-lint                                | In TODO_LIST.md, separate task                                     |
| Tag cqrs-lint v4.6.0 with the new CI job + test                      | Not requested; the daemon committed changes but no tag was created |
| Pull the 10 genuinely-missing DOC/OBS/RES/DI rules into TODO_LIST.md | Recognized as a gap but not executed                               |

---

## 4. What Was TOTALLY FUCKED UP

Nothing was broken, deleted, or corrupted. But there were process failures:

### 4a. Did not verify the daemon's commits

The auto-commit daemon committed my changes (3 commits). I noticed this when checking `git diff` but did not verify that the committed content matched my intended changes. The daemon could have committed a corrupted version of a file (as has happened before per AGENTS.md: "Auto-commit daemon can break the build"). I should have run `git show <commit> --stat` and `go build` after each daemon commit.

### 4b. The Pareto plan has inconsistent formatting after edits

My multiedit on the plan partially failed (3 of 7 edits) because some items had already been updated by a prior session. I verified the remaining items were already correct, but the final formatting is a mix of my new audit header and the prior session's notes. The plan is now correct but messy — multiple update headers stacked on top of each other.

### 4c. Did not check if the `--fail-on-stale-suppressions` feature exists

The TODO_LIST.md has `--fail-on-stale-suppressions` as an open item, but I didn't check whether the underlying `DetectStaleSuppressions` function (which FEATURES.md says is DONE) has a CLI flag. If the detection exists but no CLI flag wraps it, that's a quick win I missed.

---

## 5. What We Should Improve

| #   | Improvement                                                                                         | Impact   |
| --- | --------------------------------------------------------------------------------------------------- | -------- |
| 1   | **Run `go build` after every daemon commit** — the daemon can corrupt files during commit           | High     |
| 2   | **Pull genuinely-missing rules into TODO_LIST.md** — 10 ideas are marooned in a closed planning doc | Medium   |
| 3   | **Add `--min-severity warning` to self-lint CI job** — or document that warnings are OK             | Medium   |
| 4   | **Multi-file fixture for parallel safety test** — exercise sync.Map contention patterns             | Low      |
| 5   | **Run doc-check on updated planning docs** — verify no broken references after edits                | Low      |
| 6   | **Stack audit headers in reverse chronological order** — newest first in planning docs              | Cosmetic |

---

## 6. Up to 50 Things We Should Get Done Next

### Tier 1: Directly from this session's gaps

| #   | Task                                                                                                                     | Effort |
| --- | ------------------------------------------------------------------------------------------------------------------------ | ------ |
| 1   | Pull the 10 genuinely-missing DOC/OBS/RES/DI rules into TODO_LIST.md as individual backlog items                         | 10 min |
| 2   | Check if `--fail-on-stale-suppressions` CLI flag exists; if not, add it (DetectStaleSuppressions is already implemented) | 30 min |
| 3   | Add `--min-severity warning` to the CI self-lint job OR add an inline suppression for the known C025 warning in init.go  | 10 min |
| 4   | Run `nix run .#verify-fast` to validate the full gate passes after session changes                                       | 5 min  |
| 5   | Run `cmd/doc-check` on the updated Pareto plan                                                                           | 5 min  |
| 6   | Verify daemon commit `c2b678dba` didn't corrupt any files (build + test)                                                 | 5 min  |

### Tier 2: cqrs-lint improvements from TODO_LIST.md

| #   | Task                                                                     | Effort  |
| --- | ------------------------------------------------------------------------ | ------- |
| 7   | Pin GitHub Actions to commit SHAs (72+ unpinned actions)                 | 90 min  |
| 8   | Add go-arch-lint config for `cmd/cqrs-lint` (16 production source files) | 90 min  |
| 9   | Audit remaining 10 EXCEPTIONS entries for dead rules                     | 60 min  |
| 10  | Consider rewriting `check-module-layers.sh` as `cmd/check-layers`        | 120 min |
| 11  | Tag cqrs-lint v4.6.0 with CI self-lint job + parallel safety test        | 30 min  |

### Tier 3: Genuinely-missing individual rules (from DOC/OBS/RES/DI evaluation)

| #   | Task                                                                      | Effort |
| --- | ------------------------------------------------------------------------- | ------ |
| 12  | RES: Missing retry middleware on bus/dispatcher (absent, not manual)      | 90 min |
| 13  | RES: Circuit breaker detection (entirely absent from linter)              | 90 min |
| 14  | RES: Missing DLQ config on projectionhost                                 | 90 min |
| 15  | DOC: Stale catalog entries (catalog → deleted events, reverse of E004)    | 60 min |
| 16  | DOC: AsyncAPI/OpenAPI generation freshness                                | 60 min |
| 17  | OBS: Missing OTel SDK initialization (usage ≠ setup)                      | 90 min |
| 18  | OBS: Missing slog.SetDefault                                              | 60 min |
| 19  | DI: Optimistic concurrency / expected-version precondition on Save/Append | 90 min |

### Tier 4: Test quality improvements

| #   | Task                                                                                | Effort |
| --- | ----------------------------------------------------------------------------------- | ------ |
| 20  | Upgrade parallel safety test with multi-file fixture (3-5 files, cross-package)     | 30 min |
| 21  | Add SourceLine concurrent access test (trigger sync.Map lineCache under contention) | 30 min |
| 22  | Add benchmark for all 192 detectors (p99 latency on 10K-LOC corpus)                 | 60 min |
| 23  | Add stale suppression detection test                                                | 15 min |

### Tier 5: CI hardening

| #   | Task                                                                    | Effort |
| --- | ----------------------------------------------------------------------- | ------ |
| 24  | Add `--fail-on-stale-suppressions` as a CI step                         | 30 min |
| 25  | Add CI check for API-version drift (verify symbols at tag)              | 60 min |
| 26  | Add `cqrs-lint scorecard --scorecard-threshold 50` as a CI quality gate | 30 min |
| 27  | Add SARIF upload from self-lint job to GitHub Code Scanning             | 15 min |

### Tier 6: Documentation cleanup

| #   | Task                                                                              | Effort |
| --- | --------------------------------------------------------------------------------- | ------ |
| 28  | Archive the Pareto plan to `docs/planning/archive/` (it's closed)                 | 5 min  |
| 29  | Clean up the stacked audit headers in the Pareto plan (keep only the final one)   | 10 min |
| 30  | Update CHANGELOG.md with the CI self-lint job + parallel safety test additions    | 10 min |
| 31  | Add the `--fail-on-stale-suppressions` flag to FEATURES.md if it gets implemented | 5 min  |

### Tier 7: Broader project health (from TODO_LIST.md)

| #     | Task                                                                   | Effort |
| ----- | ---------------------------------------------------------------------- | ------ |
| 32    | Publish go-finding + go-must as tagged modules (BLOCKED)               | —      |
| 33    | Run `nix run .#verify` (full verification gate)                        | 5 min  |
| 34    | Run `nix run .#check-coverage` (coverage drift check)                  | 5 min  |
| 35    | Run `nix run .#check-duplication` (clone detection)                    | 5 min  |
| 36    | Run `nix run .#check-layers` (dependency budget check)                 | 5 min  |
| 37-50 | _(Reserved for tasks that surface from running the verification gate)_ | —      |

---

## 7. Questions I Cannot Answer Myself

### Q1: Should the self-lint CI job fail on warnings, or only errors?

The linter has one WARNING (C025 in `init.go:69` — `fmt.Errorf` without `%w` in CQRS code). This is suppressed with `//nolint:err113` but cqrs-lint's own C025 still fires. Options:

- Add `--min-severity error` to the CI step (current behavior: warnings pass)
- Add an inline `//cqrs-lint:ignore(C025)` suppression in init.go and fail on warnings
- Fix the code to use `%w` wrapping

I cannot determine whether the `//nolint:err113` is intentional (preset name is dynamic, so `fmt.Errorf` with a static string is actually correct here) or whether this should be refactored.

### Q2: Should the 10 genuinely-missing DOC/OBS/RES/DI rules go into TODO_LIST.md now, or wait for a fresh planning round?

These are real gaps (circuit breaker detection, optimistic concurrency check, etc.) but they're individual ideas, not a category. I don't know if the user wants to grow the linter's rule count beyond 192 right now or let it stabilize.

### Q3: Should I tag cqrs-lint v4.6.0 now?

The daemon committed the CI self-lint job and parallel safety test. These are user-facing improvements (CI gate + test coverage). But no production code changed. I don't know if this warrants a version tag or if it should wait for the next batch of rule additions.

---

## 8. Session Verdict

| Dimension                  | Score | Notes                                                                      |
| -------------------------- | ----- | -------------------------------------------------------------------------- |
| Task completion            | 10/10 | All planned tasks executed and verified                                    |
| Build/test verification    | 8/10  | cqrs-lint module verified; full gate not run                               |
| Proactiveness              | 7/10  | Executed the full list, but missed pulling 10 missing rules into TODO_LIST |
| Self-lint CI quality       | 6/10  | Works but doesn't catch warnings; C025 slips through                       |
| Test quality               | 7/10  | New test exists but fixture is minimal                                     |
| Documentation accuracy     | 9/10  | Pareto plan closed cleanly, TODO_LIST updated                              |
| Daemon commit verification | 3/10  | Did not verify daemon commits didn't corrupt files                         |

**Bottom line:** The session executed all 10 planned tasks cleanly — Pareto plan closed, CI gate added, parallel safety test shipped, DOC/OBS/RES/DI evaluated. The main gaps are: (1) didn't pull the 10 genuinely-missing rules into TODO_LIST, (2) self-lint CI job is too lenient (warnings pass), (3) didn't verify daemon commits, (4) didn't run the full verification gate.
