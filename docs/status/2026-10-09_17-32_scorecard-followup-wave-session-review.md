# Session Review: Scorecard Follow-Up Wave (path-preset, stack Legacy, trigger expiry, config-docs truth)

**Date:** 2026-10-09 17:32 · **Repo:** go-cqrs-lite · **Session type:** implementation — execution of the 05:23 report's backlog (f/1–f/8 + f/13–f/17 subset) under the standing "execute until done" instruction
**Scope:** This session only. `.md` per explicit standing user override of the skill's HTML default.

## What this session was

Continued the scorecard-ratchet work: fixed the cwd-preset quirk (`scorecard --path X` now honors the scored project's own config), made stack-surface imports a first-class v5-removed signal (Legacy grade + DEPRECATED panel counter per ADR-0123), gave waiver triggers detectable expiry, made `explain`'s "every config key" claim mechanically true, pinned the waiver chain at the binary level, and updated all doc surfaces. Ran the gate set until the user's report request arrived mid-gates.

**Context factor that shaped the session:** a concurrent session is LIVE in this same working tree (schema/* edits at 15:41–17:03, systemscenario/ landed in the same auto-commits). All my work was scoped to `cmd/cqrs-lint` + `scripts/` + docs; nothing of theirs was touched.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                                                                                                                     | Evidence                                                                                                                                                                               |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **Preset resolution fix** — `resolveScorecardPreset` (scorecard_command.go): `<path>/.cqrs-lint.json` preset wins, cwd config fills; wired at BOTH entry points (subcommand + root `--scorecard`) before `applyConfigOverrides` so feature-profile resolution sees the right preset                                                                                                                      | `TestResolveScorecardPreset` (project-wins / passthrough / unknown-errors); binary probe (row 7)                                                                                       |
| 2  | **Stack-surface Legacy signal** — `stackPresetFromImport` (engine presets named, root module + subpackages → `bundle`), `FeatureProfile.StackPresets` (+ String line "removed at v5 — ADR-0123"), `ScorecardDeprecated.StackPresetUses` + system.New migration suggestion, `ModernityGrade` Legacy when stack imports exist (beats system.New/pushdown Modern signals), text+markdown panel lines        | `TestStackPresetFromImport` (8 paths incl. systemtest boundary), `TestDetectFeatures_StackPresetImport`, 2 new `TestModernityGrade` cases (stack+pushdown→Legacy, stack+system→Legacy) |
| 3  | **SARIF panel counts** — `removedApiUses`/`deprecatedTransportUses`/`stackPresetUses` props, all `omitempty` (clean runs byte-identical)                                                                                                                                                                                                                                                                 | scorecard_render_sarif.go; module suite green                                                                                                                                          |
| 4  | **Waiver trigger expiry** — 3 detectable signal families (server / async-bus / transport), word-exact token matching (`business` ≠ `bus`), `TRIGGER LIKELY FIRED` evidence suffix + per-waiver re-litigation recommendation; advisory wording, never silent removal                                                                                                                                      | `TestComputeScorecardWithWaivers_FiredTriggerIsSurfaced`, `..._UnfiredTriggerStaysQuiet` (3 negative cases)                                                                            |
| 5  | **Doctor composition hint** — system.New detection prints a cross-render hint pointing at the scorecard's composition credit                                                                                                                                                                                                                                                                             | `TestRenderDoctorFeatureProfile_CompositionHint` (presence + absence)                                                                                                                  |
| 6  | **explain made true** — new `scorecard` top-level key + full SCORECARD section (waiver semantics, path-vs-cwd resolution, trigger firing); ALSO discovered + documented `typed-info` (undocumented before — the README claim was false for it too, pre-existing); NEW mechanical enforcement: `TestRenderExplain_DocumentsEveryConfigFileKey` reflect-walks AppConfig's json keys against `topLevelKeys` | explain.go, explain_test.go                                                                                                                                                            |
| 7  | **init skeleton** — commented `scorecard.waivers` example (key/reason/trigger) in the default template                                                                                                                                                                                                                                                                                                   | init.go + assertion in `TestGenerateInitConfigDefaultProducesValidJSON`                                                                                                                |
| 8  | **Binary probes ×3** in `scripts/check-cqrs-lint-cli.sh`: WAIVED renders from project config (graph row, exit 0), bogus waiver key exits nonzero, path-preset widens relevance past cwd config (relevant_total delta) — each with a fault-injected `--self-test` leg + waiver positive control                                                                                                           | `--self-test` catches all planted faults; full real-binary run "all contract probes passed"                                                                                            |
| 9  | **Docs** — README (threshold gates waiver-adjusted coverage; trigger firing; Legacy redefined incl. stack surfaces), skill `advanced.md` (waiver resolution + expiry, stack panel, SARIF props), `modules.md` checked (carries no scorecard flags — nothing to do, TODO f/17 closed)                                                                                                                     | diff                                                                                                                                                                                   |
| 10 | **CHANGELOG follow-up entry** (path-preset, stack modernity, trigger expiry, config-docs)                                                                                                                                                                                                                                                                                                                | `check-changelog-symbols.sh`: "15 pkg.Symbol citation(s)… honest"                                                                                                                      |
| 11 | **Gates green** — module suites `go test . ./pkg/analyzer` 33s+0.2s ok; doc-check 1181 refs ✓; api golden: verified FIELDS are not tracked (golden lists top-level symbols only) so no regen needed, `TestEvery` green; changelog-symbols ✓; gofmt clean; probe script green twice                                                                                                                       | session log                                                                                                                                                                            |
| 12 | **Decisions on the 3 open questions** (documented here in g/answers): preset fix scoped to scorecard; explain claim made binding + enforced; release remains owner-gated                                                                                                                                                                                                                                 | this report                                                                                                                                                                            |

## b) PARTIALLY DONE

| # | Item             | Done                                                                                                                                                                                                                                                              | Open                                                                                                                                          | Effort |
| - | ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1 | Gate sweep       | module tests, doc-check, api golden, changelog-symbols, dup gate RUN + verdict classified                                                                                                                                                                         | `check-file-size`, `check-md-go`, `check-lint-config` not yet run (user report arrived mid-gates)                                             | S      |
| 2 | Dup gate verdict | RED diagnosed and fully classified: **all 6 new clone groups live in `systemscenario/`** (concurrent session's module); the one cqrs-lint member (`output.go:174-178`, a 5-line `make/append` idiom) pairs with THEIR `equivalence.go` — **zero groups are mine** | re-pin/annotation deliberately left to the owning session                                                                                     | —      |
| 3 | Fleet validation | nothing run yet                                                                                                                                                                                                                                                   | journal re-run (expect 5/29 Modern, unchanged), cqrs-htmx (expect **Legacy** now — the ADR-0152 first-migration target), go-appkit, self-lint | S      |

## c) NOT STARTED

- TODO_LIST receipt updates (5 of the 6 harvested rows are now DONE — receipts not yet written).
- Execution addendum on the 05:23 report pointing here.
- Release tagging (owner-gated, unchanged).
- Journal-side adoption (owner-gated, unchanged).
- lint/doctor `--path` config unification (deliberately deferred — see g/Q2).

## d) TOTALLY FUCKED UP

1. **Wrote the same test bug twice.** `TestResolveScorecardPreset` first shipped with `cfg.Path` unset on BOTH cases — `LoadProjectConfig("")` reads the CWD config (here: cmd/cqrs-lint's own self-lint config). First case failed loudly; the second was a latent cwd-dependence I caught by inspection. The entire session is about cwd-config leakage; tripping on it in the test itself is the most on-brand mistake available.
2. **Four edit-tool failures through haste:** (i) SARIF multiedit mangled brace placement (caught by build); (ii) a no-op edit fused `)` and `if` into invalid syntax (caught by compile); (iii)+(iv) two stale-mtime guard rejections on files the concurrent session touched — I re-read and recovered each time, but the guard should have prompted the fresh read FIRST.
3. **Ran the api golden regen before checking what the golden tracks.** Fields aren't tracked (top-level symbols only) — the regen was unnecessary. "Verify before running" is a rule I enforced on others in this very repo (explain claim) and skipped myself.
4. **Ran a repo-wide dup gate while knowing a concurrent session was live** (schema mtimes at 15:41) without pre-scoping or pre-classifying authorship — cost three gate runs (~90 s) to sort whose clones they were.
5. One nonsense command (`sed -i … /dev/null`) — harmless, pure slop.

## e) WHAT WE SHOULD IMPROVE

1. **Shared-tree protocol:** when a concurrent session is live, classify gate failures by authorship before concluding anything; prefer scoped runs where the gate allows.
2. **Edit hygiene after mtime-guard rejection:** fresh `view` of the exact region, always, before re-editing.
3. **Test-fixture rule:** every `AppConfig` constructed in tests MUST set `Path` explicitly — the cwd-leak class is now load-bearing in this tool; a helper or lint could enforce it.
4. **The mechanical explain test should be mutation-tested** (delete a `topLevelKeys` row → must fail) before it's trusted as a pin — same repo policy as the probe self-tests.
5. **Verify what a golden/artifact tracks before regenerating it** — apply my own explain lesson to my own tooling use.

## f) Next tasks (24, priority-ordered)

1. Run remaining gates: `check-file-size`, `check-md-go`, `check-lint-config` — S · gates.
2. Journal scorecard re-run — expect 5/29 (17%), Modern, unchanged (no stack imports) — S · validation.
3. cqrs-htmx scorecard run — expect Legacy via stack presets; feeds ADR-0152 migration-order evidence — S · validation.
4. go-appkit scorecard run — S · validation.
5. Self-lint: scorecard against cmd/cqrs-lint itself — S · validation (dogfood).
6. Fleet census: scorecard across all `~/projects` cqrs consumers, Legacy/Modern split — M · validation (ADR-0152 evidence).
7. TODO_LIST receipts: mark 5 of the 6 harvested rows DONE with dates (waiver e2e probe, trigger expiry, preset resolution, stack-preset modernity signal, doctor hint) — S · harvest.
8. Execution addendum on the 05:23 report pointing at this file — S · harvest.
9. Mutation-test `TestRenderExplain_DocumentsEveryConfigFileKey` (remove a row → red → restore) — S · quality.
10. lint/doctor `--path` config unification (project config wins for preset/rules; same convention) — M · feature (needs g/Q2).
11. doctor JSON schema table in advanced.md: add `stackPresets` note to the `features` row — S · docs.
12. Scorecard JSON schema table in advanced.md (summary/rows/props — mirrors the doctor table style) — S/M · docs.
13. Add preset-resolution note to explain's CONFIG RESOLUTION ORDER section (path-vs-cwd for scorecard) — S · docs.
14. Sweep for stale copies of the old Legacy definition text ("v5-removed APIs or deprecated transports") in generated/docs surfaces — S · docs.
15. CHANGELOG "Changed" split if the release wave prefers Added/Changed separation (doctor profile field + panel line are arguably Changed) — S · docs.
16. Expand trigger-signal families where honestly detectable (e.g. metaengine/pushdown appearing, snapshot appearing) — M · feature.
17. Document the NON-detectable trigger families (domain growth, multi-writer, schema break) as explicitly human-revisit-only — S · docs (honesty).
18. Consider a V-series lint advisory for stack-preset imports (lint-level coaching, not just scorecard) — M · decision.
19. output_determinism test for SARIF props presence/absence paths (omitempty) — S · quality.
20. Waiving-irrelevant error message: point at `doctor` for why the row is irrelevant — S · DX.
21. cqrs-htmx reaction plan to Legacy grading (their repo; owner-gated) — M · coordination.
22. Tree-wide `#verify` once the concurrent session lands (currently red via their systemscenario clones) — M · gate.
23. Release/tag cqrs-lint (owner-gated; see g/Q1) — S · release.
24. Journal adoption: waivers with F007 revisit triggers (owner-gated) — M · adoption.

## g) Questions I cannot figure out myself

1. **Release sequencing (carried from 05:23 g/2, sharpened):** tag cqrs-lint now so the journal can adopt credit/waivers/modernity/path-preset — or hold until the concurrent systemscenario session lands (its 6 clone groups keep `check-duplication` red tree-wide, and tagging over a red gate repeats the 2026-10-06 lesson)?
   _Tried:_ verified my module gates are green and the red dup verdict is 100% theirs; the ordering depends on their landing ETA, which only you know.
2. **lint/doctor path-config unification:** scorecard now resolves preset+waivers from the scored project's config. Should `lint ./x` and `doctor --path X` do the same for preset/rules (project wins, cwd fills)? Correct for the tool's purpose, but a behavior change for any CI running from a config-carrying cwd against another path — your appetite call, same as 05:23 g/1 but now concrete.
3. **Dup-gate ownership:** all 6 new clone groups live in `systemscenario/` (concurrent session). I deliberately did not re-pin the baseline or annotate their code. Confirm they own that cleanup — or instruct me to annotate/re-pin on their behalf.
   _Tried:_ classified every group's membership; only ambiguous member is a pre-existing 5-line idiom in `cmd/cqrs-lint/output.go` (untouched this session) paired with their file.

---

**Answers given autonomously to the 05:23 questions (flag if any should be revisited):**

- **g/1 (preset resolution):** implemented, scorecard-scoped, waivers' convention (path wins, cwd fills); lint/doctor unification deferred to a decision (Q2 above).
- **g/2 (explain claim binding?):** treated as BINDING — `scorecard.waivers` documented, `typed-info` discovered missing and added, and the claim is now mechanically enforced by a reflect test so it cannot rot again.
- **g/3 (release):** remains owner-gated; my recommendation: hold until the concurrent session lands (red dup gate).

**Standing notes:** no manual commit (harness contract; auto-commit daemon absorbed this session's edits — all code was green at each absorbed point). Concurrent session live in-tree: schema/* and systemscenario/ are theirs, untouched.

---

## Continuation addendum (2026-10-09 evening — same arc, gates + probes + hardening)

Executed this report's "Exact Next Steps" under the standing execute-and-verify
instruction. Everything below is verified, not claimed:

1. **Remaining gates:** file-size (my files), md-go, lint-config — green. My
   file-size additions were caught and FIXED by splitting:
   `doctor.go` 447→359 (profile renderers → new `doctor_render.go`) and
   `feature_detect.go` 357→339 (`stackPresetFromImport` → new
   `feature_detect_stack.go`). The 12 remaining violations are foreign
   (rules/*, stale.go from a 09-06 commit, schema/system/metaengine from the
   concurrent sessions).
2. **Validation probes found a REAL BUG — multi-module blindspot.** cqrs-htmx
   graded stack-surface 0 despite `usermgmt/stack_repositories.go` importing
   `stack/v4`: BuildContext exposes only the PRIMARY module's profile, so
   submodule signals were invisible to the panel (while v007 counts are
   project-wide — inconsistent in one panel). Fixed project-wide:
   `stackPresetUseCount` (distinct surfaces across all per-module profiles)
   and `compositionWideProfile` (`system.New`/pushdown signals unioned for
   composition credit AND Modernity). Regression tests:
   `TestStackPresetUseCount_UnionsAllModuleProfiles`,
   `TestComputeDeprecatedPanel_CountsStackPresetsAcrossModules`,
   `TestCompositionWideProfile_UnionsSubmoduleSignals`.
3. **Probe results (rebuilt binary):** journal 5/30 (16%) Minimal,
   **Modernity: Modern** — regression PASS (denominator 29→30 is the foreign
   `systemscenario` catalog entry, registered 06:10). cqrs-htmx 14/32 (43%)
   Fair, **Modernity: Legacy** — 41 v5-removed uses + 1 stack-surface
   (`//go:build ignore` file correctly NOT counted); its modern side
   (systemadapter composition, metaengine + pushdown) now renders too.
   go-appkit 8/30 (26%), **Modernity: Modern** (the `cqrs/` submodule's
   system wiring finally counts). Self-lint exit 0 (C025 warning is
   pre-existing in doctor_audit.go, 10-04). Recorded as
   [ADR-0152](../../docs/adr/0152-fleet-first-module-topology-v5-dual-support.md) ratchet evidence.
4. **Mutation-tested the reflect explain pin** per repo convention:
   deleted the `scorecard` topLevelKeys row → test FAILED → restored →
   green. The pin has teeth.
5. **Docs:** advanced.md (multi-module union bullet, scorecard JSON schema
   table, doctor `features` row now names `stackPresets`); README (union
   note + "Config resolution order (path vs cwd)" section); CHANGELOG
   bullet gained (c2). TODO_LIST: 5 of 6 scorecard-ratchet rows DONE with
   dated receipts (journal-side adoption stays owner-gated, annotated with
   the 30-denominator note). 05:23 report addendum points here.
6. **Final gate set:** gofmt clean; file-size mine-clean; doc-check 1184
   refs ✓ (core/v5 ambiguity advisories are the concurrent session's, exit
   0); changelog-symbols 37 honest; api-stability TestEvery ✓; full
   main+analyzer suites ✓; check-cqrs-lint-cli.sh probes + `--self-test` ✓;
   dup gate unchanged — 6 foreign groups (systemscenario/* + pre-existing
   output.go idiom), ZERO mine (moves, not copies).

**Still owner-gated (unchanged):** (1) tag cqrs-lint now vs hold for the
concurrent session (dup gate still red on their groups); (2) lint/doctor
path-preset unification (CI behavior change); (3) confirm the concurrent
session owns the systemscenario clone cleanup. cqrs-htmx was under ACTIVE
concurrent development during probing (commits 18:05/18:10) — its 14/32 is a
snapshot, not a fixture.
