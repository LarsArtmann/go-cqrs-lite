# Session Review: Scorecard Ratchet Execution (composition credit, waivers, modernity)

**Date:** 2026-10-09 05:23 · **Repo:** go-cqrs-lite · **Session type:** implementation — the execution phase of `2026-10-09_02-41_cqrs-lint-purpose-recalibration-session-review.md` (consultation → user correction → "execute" instruction)
**Scope:** This session only. Self-review + status report combined; `.md` per explicit user instruction (HTML-canonical overridden, same as the prior report).

## What this session was

Executed groups A/C/D/E/G of the prior report's plan inside `cmd/cqrs-lint`: diagnosed and fixed the scorecard false negatives (import-only usage detection vs composition wiring), built the waiver mechanism, added the Modernity headline, wrote the mission into four doc surfaces, ran the full gate set, verified end-to-end against the real `~/projects/journal`, and harvested the remainders into TODO_LIST. Groups B (research) became C's foundation; F (journal-side edits) stayed owner-gated.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Root cause proven, not guessed: `DetectUsedModules` is import-only (module_detect.go:54-93); journal's go.mod shows `system/v4` + `metaengine/sqliteengine` + `modernc.org/sqlite` direct requires; `system/go.mod` wires `storage/memory` | session tool output; doctor's "store: sqlite" comes from `fp.Stores`, which the scorecard ignored |
| 2 | **Composition credit** — `scorecard_credit.go` (+ `FeatureProfile.HasSystemComposition` with `systemtest` path-boundary exclusion); credit never overrides direct imports, never pulls irrelevant rows | `scorecard_credit_test.go` (4 tests incl. journal-shape); live journal run |
| 3 | **Waivers** — `analyzer.ScorecardWaiver`/`ScorecardSettings` + `ValidateScorecardWaivers` (load-time, CLI + embedded paths) + `ComputeScorecardWithWaivers` + `resolveScorecardWaivers` (path-config wins per key, cwd fills); WAIVED partition in text/markdown/JSON/SARIF; waive-used/irrelevant/unknown/duplicate = hard errors; trigger-less shamed | `scorecard_waivers_test.go` (9 tests), `project_config_scorecard_test.go`; binary probes both directions (bogus key exits loudly; valid waiver renders WAIVED) |
| 4 | **Modernity headline** — `ModernityGrade`/`ModernityHint` (Legacy/Partial/Modern) wired into runScorecard + all four render formats (`modernity_grade`, `waived_count` props) | `scorecard_modernity_test.go` (7 cases); journal run shows `Modernity: Modern` |
| 5 | **Mission docs** — README purpose banner + § Scorecard; IMPROVEMENT_IDEAS mission note; AGENTS.md Internal Contract #28; skill `advanced.md` scorecard block | doc-check ✓ (1175 refs) after each edit |
| 6 | **Gates** — build+vet; main+analyzer suites green; api golden regen (+6 additive exports) + TestEvery; changelog-symbols (6 citations honest); doc-check ×2; duplication gate 0 new clones; doctor golden updated (only `hasSystemComposition`); gofmt clean; file-size — zero flags in my files | session logs; `git diff` of goldens |
| 7 | **Live e2e vs journal** — 2/28 → **5/29 (17%)**; the three false-MISSING persistence rows (SQLite Stack, SQL Storage, Memory Stack) now USED with wiring-path evidence; re-verified reproducible from a NEUTRAL cwd (/tmp) — the number is not a cwd-preset artifact | binary runs at 04:4x and 05:23 |
| 8 | **Harvest + report closure** — 6 deliberate remainders in TODO_LIST cqrs-lint section; execution addendum appended to the prior report | TODO_LIST.md, prior report |

Bonus refactors: `writeModuleSection` dedup in render (3 duplicated table blocks → 1 helper); SARIF renderer split into `scorecard_render_sarif.go`, keeping `scorecard_render.go` under its 444-line baseline.

## b) PARTIALLY DONE

| # | Item | Done | Open | Effort |
|---|------|------|------|--------|
| 1 | Waiver e2e at the binary level | manual /tmp probes (both directions) | not pinned in `scripts/check-cqrs-lint-cli.sh`'s probe set — a future render change could silently break the config-file→WAIVED chain | S |
| 2 | Config-surface documentation | README § Scorecard documents waivers fully | `cqrs-lint explain` does NOT list `scorecard.waivers` (verified: zero hits in explain*.go) while README line 52 claims "full documentation of every config key" — either explain grows the key or the README copy overpromises (also true for `health`, pre-existing) | S/M |
| 3 | Threshold-gate semantics | gate unchanged, math now waiver-adjusted | README does not state that `--scorecard-threshold` applies to the waiver-adjusted coverage | S |

## c) NOT STARTED

- `cqrs-lint init` skeleton does not include a commented `scorecard.waivers` example (considered post-hoc; not raised during implementation).
- Releasing/tagging cqrs-lint so the journal can adopt (correctly not started — release waves are owner-gated).
- Doctor cross-render hint for composition-credited rows (harvested).
- Fixing the cwd-based preset resolution (harvested; behavior change needs a decision — see g/1).

## d) TOTALLY FUCKED UP

1. **First waiver-plumbing cut shipped unverified and was wrong.** I wired waivers only through cmdguard's cwd config (`cfg.ScorecardSettings`), believed config flowed from the scored project, and my unit tests stayed green because they called `ComputeScorecardWithWaivers` directly — bypassing the CLI resolution layer entirely. The manual /tmp probe exposed it: an invalid waiver key was SILENTLY IGNORED, exit 0. Root cause: I tested the pure functions hard and the 3-line glue not at all. Mitigated same session (`resolveScorecardWaivers` + merge test + binary probes), but the check-cqrs-lint-cli.sh probe pattern existed precisely for this class and I initially left it as a TODO instead of doing it.
2. **Mid-session misdiagnosis of "preset contamination."** During the plumbing probe I concluded the journal run's numbers were computed under the wrong preset (cmd/cqrs-lint's self-lint local-cli) and reported the concern as near-fact. The 05:23 neutral-cwd re-run reproduces 5/29 exactly — auto-detected profile relevance, not cwd preset, drives the partition. The concern was worth checking; asserting it without re-running was sloppy. (The underlying cwd-preset quirk is real and harvested — but it did NOT skew my numbers.)
3. **Burned 10 minutes on self-inflicted test contention.** I ran two full cqrs-lint suites concurrently; the first hit the 10m timeout (600.114s) and I briefly chased "a hanging test" before the sequential rerun finished in 20-35s. #verify-exclusivity discipline exists in AGENTS.md for exactly this; I applied it to nix gates but not to my own background `go test` runs.
4. **The explain claim was pattern-matched, not verified, during implementation.** I grepped one file (`explain_features.go`), saw `health` absent, and moved on — only nailing the actual gap (explain carries zero scorecard keys) during THIS review. Same failure class as the consultation round-1: confidence ahead of evidence.

## e) WHAT WE SHOULD IMPROVE

1. **Binary-level probe FIRST for any CLI-plumbed feature** — the false-silence failure (d/1) is invisible to unit tests by construction. New rule for cqrs-lint work: if it touches AppConfig→command flow, add the check-cqrs-lint-cli.sh probe in the same change.
2. **Report metrics under the conditions you claim, or flag the conditions.** I reported 5/29 as clean while privately suspecting skew (d/2); the 30-second neutral-cwd rerun should have happened before the number went in the addendum.
3. **Verify doc-claims before extending the surface under them** (d/4): README's "every config key" sentence was checkable in seconds.
4. **Serialize heavy suites** (d/3); background convenience cost 10 minutes and a wrong hypothesis.
5. **The tool now needs its own dogfood**: cqrs-lint's self-lint config could carry a waiver example once explain/init catch up — eat the ratchet's own cooking.

## f) Next tasks (18 — supersedes/extends the 6 already in TODO_LIST; those are referenced, not duplicated)

1. Add the waiver e2e probe to `scripts/check-cqrs-lint-cli.sh` (+ self-test) — HIGH · S · Quality (TODO_LIST already).
2. Decide + fix `explain` scope: document `scorecard.waivers` (and `health`?) or correct README's "every config key" copy — HIGH · S · Documentation (needs g/3).
3. Decide preset resolution for `scorecard --path` (cwd vs `<path>` config) and implement — HIGH · M · Feature (needs g/1; TODO_LIST already).
4. Waiver trigger expiry surfacing (detectable conditions flag stale waivers loudly) — MEDIUM · M · Feature (TODO_LIST already).
5. Document `--scorecard-threshold` applies to waiver-adjusted coverage (README § Scorecard one-liner) — MEDIUM · S · Documentation.
6. `cqrs-lint init` skeleton: commented scorecard.waivers example — LOW · S · DX.
7. Legacy-persistence signal in ModernityGrade (stack-preset imports vs ADR-0123) — MEDIUM · M · Feature (TODO_LIST already).
8. Doctor cross-render hint for composition credit — LOW · S · Feature (TODO_LIST already).
9. Release decision: tag cqrs-lint so journal can adopt (needs g/2) — HIGH · S · Release.
10. Journal adoption: replace AGENTS.md moat with per-row waivers + F007 revisit triggers (owner-gated; TODO_LIST already).
11. Coordinate with the concurrent `cqrs-lint/a014-d013-scoped-fixes` session: 12 pre-existing file-size violations in metaengine/ + rules/* keep `check-file-size` red for everyone; its stash is mid-flight — HIGH · M · Coordination.
12. Reconcile the `stash@{0}` (WIP on the a014-d013 branch) — either land or drop it deliberately — MEDIUM · S · Cleanup.
13. `check-md-go` / lint-config pass over the new README/advanced.md blocks (not run this session; doc-check covered refs only) — LOW · S · Quality.
14. Consider `ModernityGrade` JSON schema note in advanced.md's doctor-schema table style for scorecard SARIF props (consumers diff reports) — LOW · S · Documentation.
15. Self-lint: run the new scorecard against cqrs-lint itself + first-party fleet (cqrs-htmx, go-appkit) to see credit/waiver behavior at scale — MEDIUM · S · Validation.
16. Mutation-test the waiver validation errors (corrupt → test fails → restore), matching the repo's golden-pin policy — MEDIUM · S · Quality.
17. Update `.agents/skills/go-cqrs-lite/references/modules.md` if it lists scorecard flags (unchecked this session) — LOW · S · Documentation.
18. CHANGELOG "Changed" entry for the doctor profile JSON field if the release wave prefers Added/Changed separation — LOW · S · Documentation.

## g) Questions I cannot figure out myself

1. **Preset resolution for `--path`:** should `cqrs-lint scorecard --path X` (and lint/doctor generally) resolve preset/features from `X/.cqrs-lint.json` instead of the operator's cwd config? Correct for the tool's purpose, but a behavior change for any CI that runs from a config-carrying cwd against another path — breaking-change appetite is yours.
   *Tried:* proved the cwd quirk live; proved my journal numbers were NOT skewed by it; the fix direction is clear, the compatibility call is not.
2. **Release sequencing:** tag cqrs-lint now so the journal can adopt credit/waivers/modernity, or hold until the concurrent a014-d013 branch lands (its work currently keeps `check-file-size` red and a stash mid-flight — a tag wave over a red gate repeats the 2026-10-06 red-suite lesson)?
   *Tried:* identified both constraints; the ordering depends on that session's state, which only you can arbitrate.
3. **Is `explain`'s "full documentation of every config key" (README:52) a binding contract?** If yes, `scorecard.waivers` (and pre-existing `health`) must be added; if it's aspirational copy, the README sentence gets narrowed. Verified the gap; the intent is not in the code.
   *Tried:* zero `scorecard` hits in explain.go/explain_features.go; `health` also absent — so the README already overpromised before this session.

---

**Standing notes:** no manual commit (harness contract; auto-commit daemon absorbs). Prior report's questions g/1-3 were answered autonomously and documented in its execution addendum — flag if any call should be revisited.
