# Status Report — Issue #67/#69 Fix Wave (2026-10-13 18:14)

Session scope: "Review and fix/resolve ALL open GitHub issues" — exactly two were open (#67 `system.GracefulClose` skips Close on drain error; #69 scorecard blind to test-only imports). Both fixed, tested, closed. Report covers this session only.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | **#67 fixed**: drain error no longer skips Close — `system/lifecycle.go:137` runs `Close()` unconditionally, returns `errors.Join(drainErr, closeErr)`; godoc strengthened; Phase 2 pre-expired-context guard untouched | `TestSystem_GracefulClose_DrainError_StillCloses` (old `NoClose` test inverted), `DrainTimeout` now asserts engine closed once |
| 2 | **#67 verified**: system + systemtest + systemscenario suites green, `-race` included | `ok` all three modules, 2.8s race run |
| 3 | **#69 fixed**: `analyzer.CollectTestImportPaths` (`test_imports.go:27`, go/parser `ImportsOnly` walk, vendor/testdata/.git skipped) + Pass 2 in `DetectUsedModules` via same `matchModule`/`pathBoundaryMatch`; typed load stays `Tests: false` (the issue's suggested shape verbatim) | 4 new tests: walk (dedupe/skip-dirs/broken-file), no-go.mod fallback, test-only detection, composed pipeline integration |
| 4 | **#69 verified against the reporting consumer**: scorecard over go-credentials-store @ master → `System Scenario Testing: used` (was `missing`); `Scenario Testing` correctly still missing (that consumer doesn't import `scenario`) | live CLI run this session |
| 5 | Both issues closed with voice-checked evidence comments (github-voice checker: 0 FAIL each) | `gh issue list --state open` → `[]` |
| 6 | API surface: golden regenerated (+1 export `CollectTestImportPaths`, exactly the one line), `TestEvery` green | `docs/api_surface.txt` diff = 1 insertion |
| 7 | Fix-on-sight (pre-existing gate failures from 10-12 waves): `types.go` 353→290 lines (`GoFile`+`ProfileForFile` → new `gofile.go`, pure move); `docs/drafts/issue-62-close.md` fence `// skip-validate`; suppression clone pair → shared `readGoFileLines` + one parity-contract `art-dupl:accept` | file-size gate ✓, md-go gate ✓, duplication gate ✓ |
| 8 | Gates green: file-size ratchet, changelog-symbols (13 citations verified), doc-check (1284 refs), md-go (1511 blocks), duplication (187 baseline, 0 new), full `#lint` → **0 findings in every touched module** | outputs in session log |
| 9 | Paper trail: CHANGELOG `[Unreleased]` Fixed ×2 (additive-only diff verified), FEATURES 2 rows updated | `git diff --stat` = 2 insertions |

## b) PARTIALLY DONE

| # | Item | What's missing |
|---|------|----------------|
| 1 | Verification | Module suites + all script gates green, but **full `nix run .#verify` never ran** (would fail on pre-existing lint debt regardless); `#verify-fast` not run either |
| 2 | Formatting | Only `test_imports_test.go` explicitly treefmt'ed; repo-wide `nix fmt --fail-on-change` (the actual CI gate) never executed — other new/edited files rely on careful editing, not the formatter |
| 3 | Consumer impact check | Scorecard behavior verified on go-credentials-store, but did NOT inspect their `.cqrs-lint.json` for a stale scenario waiver — if present, their next run hits the waive-used HARD error |
| 4 | Issue closure | Close confirmed + empty open list; comment bodies not re-verified via `gh issue view` (skill step 8 says verify landing) |

## c) NOT STARTED

1. go-appkit migration: its hand-rolled `Drain` + always-`Close` + `Join` wrapper and `TestEventService_Shutdown_DrainTimeoutStillCloses` are now redundant — flagged in the closing comment, no code touched (sibling repo).
2. `stack.Bundle.GracefulClose` analogous semantics: stack still **aborts Close on drain error** (`TestBundle_GracefulClose_DrainErrorAbortsClose`) — now intentionally divergent from `system`; no decision/ADR recorded. Deliberately out of the issue's scope.
3. Performance measurement of the new scorecard walk (felt instant on consumer; the 103-module self repo run was inside a 21s suite — never isolated/benchmarked).
4. cqrs-lint README/RULES.md one-liner that `_test.go` imports count as adoption (docs stayed truthful, just not extended).
5. AGENTS.md memory updates from session learnings (see e/6).
6. `#check-coverage` gate.
7. Nothing pushed — CI never observed (daemon commits locally only).

## d) TOTALLY FUCKED UP

1. **CHANGELOG entry merge (caught + repaired same minute)**: my first edit's `old_string` ended mid-line on a ~2,400-char bullet, gluing the doctor entry's body onto my new #69 bullet. Caught by immediate diff read, repaired, final diff verified purely additive (2 insertions, 0 deletions). Lesson: never anchor edits mid-line in long-bullet files.
2. **Naive first clone fix**: consolidating the suppression preamble shrank the clone instead of clearing it (the AGENTS-documented unmasking behavior — I predicted the original flag but not the second round). Two gate runs burned; final state correct. Also the `art-dupl:accept` directive sits 2 lines above the region start — it suppressed, but the docs say "directly on/above"; lucky, not rigorous.
3. **Stale gopls diagnostics flashed `WrongArgCount` all session** (contradicted by green builds/tests); never ran `lsp_restart` — tolerated noise instead of fixing it.

## e) WHAT WE SHOULD IMPROVE

1. **Definition-of-done for fix waves**: include repo-wide `nix fmt --fail-on-change` and at least `#verify-fast`; module-green + script-gates is strong but not the CI contract.
2. **Long-line Markdown editing**: anchor at bullet starts; verify with `git diff --stat` immediately after every md edit (this caught d/1 — keep doing it).
3. **Consumer-blast-radius habit**: when a fix changes scorecard/waiver semantics, grep the fleet's `.cqrs-lint.json` files in the same session — the first-party fleet is small and checkable.
4. **Measure before shipping a new walk**: a tree walk now rides every `cqrs-lint scorecard` run; should have timed it on the 103-module self repo before declaring done.
5. **Waiver transition communication**: consumers with stale waivers get a hard error on first run post-upgrade. The error message is self-explanatory, but a CHANGELOG/release-note callout in the cqrs-lint changelog entry specifically (not just root) would soften the upgrade cliff.
6. **Memory protocol**: two durable learnings (art-dupl directive placement tolerance; `docs/drafts/` fences are LIVE for the md-go gate — bit me via issue-62's draft) belong in project AGENTS.md gotchas; not written yet.
7. **LSP hygiene**: `lsp_restart` when diagnostics contradict a green build; don't read stale errors for 2 hours.

## f) Things to get done next

| # | Task | Impact |
|---|------|--------|
| 1 | Run repo-wide `nix fmt --fail-on-change` (or `nix fmt` + commit) — CI parity for the 12 files this session touched/created | High (CI gate) |
| 2 | Run `nix run .#verify` once; expected outcome: fails ONLY on the 78 pre-existing lint findings — proves nothing else regressed | High |
| 3 | Triage the 78 pre-existing lint findings across 11 modules (schema, watermill, cmd/cqrs-upgrade, metaengine, duckdb/turso/pg/mysql engines, core/v5, systemscenario, systemtest); start with `systemtest/time_fidelity_test.go:81` gocognit 79 | High (CI red) |
| 4 | Sweep fleet `.cqrs-lint.json` (go-credentials-store, cqrs-htmx, go-appkit, browser-history…) for scenario/systemscenario waivers that now trip waive-used; remove before the next cqrs-lint tag | High |
| 5 | go-appkit: migrate `EventService.Shutdown` → `sys.GracefulClose`; drop the hand-rolled wrapper + now-redundant drain-timeout test | Medium |
| 6 | Decide `stack.Bundle.GracefulClose` drain-error semantics (align with system vs document divergence as v5-frozen); if aligned, invert its `DrainErrorAbortsClose` test too | Medium |
| 7 | Benchmark `CollectTestImportPaths` on this 103-module repo (self scorecard); if >~100ms, short-circuit dirs without `.go` files or reuse module-dir package hints | Medium |
| 8 | cqrs-lint README scorecard section + RULES.md: one line "imports in `_test.go` count as adoption (#69)" | Low |
| 9 | AGENTS.md gotchas: (a) art-dupl accept-directive placement tolerance observed; (b) `docs/drafts/` is LIVE for md-go — new draft fences need `// skip-validate` | Medium (recurring traps) |
| 10 | `gh issue view 67` / `69` — confirm comment bodies rendered as intended | Low |
| 11 | Run `nix run .#check-coverage` for drift after the new code | Low |
| 12 | Scorecard e2e: multi-module monorepo fixture exercising Pass 2 across nested go.mods | Low |
| 13 | Consider evidence provenance: test-only rows could render "test-only import" instead of a bare path (additive JSON change → doctor/scorecard goldens) | Low |
| 14 | HARVEST this report's f-list into TODO_LIST.md (docs-health HARVEST) — timestamped reports are not backlog homes | Medium |
| 15 | Watch the first CI run after the next push: `api_surface.txt` + doctor golden must match (goldens regenerated locally, but CI is the referee) | Medium |
| 16 | Optional hardening: unit test pinning that `CollectTestImportPaths` skips symlinked dirs (WalkDir behavior — document or test the contract) | Low |

## g) Questions I cannot answer myself

1. **Waive-used transition policy**: a consumer with a stale scenario waiver now gets a hard error on first run after the next cqrs-lint tag. Keep as-is (hard error is the designed ratchet, fleet is first-party and small) — or ship a one-release "stale waiver" warning that names the exact key before it turns hard?
2. **stack semantics**: should `stack.Bundle.GracefulClose` be aligned with `system`'s new always-close-on-drain-error contract (fix + test inversion + CHANGELOG row), or is stack frozen as deprecated-until-v5 and the divergence gets a one-line doc note only?
3. **go-appkit timing**: migrate `EventService.Shutdown` to `sys.GracefulClose` now as its own mini-change, or hold it to ride the next tag wave (it depends on an untagged `system` behavior change — the fix sits in `[Unreleased]`)?

---

*Report format: `.md` at explicit user path (overrides status-report skill's HTML canonical; flagged per skill policy). Snapshot only — HARVEST f-list before this file goes stale.*
