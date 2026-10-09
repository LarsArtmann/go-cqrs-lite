# Session Review: Probe Validation, Multi-Module Blindspot Fix, Gate Closure

**Date:** 2026-10-09 20:45 · **Repo:** go-cqrs-lite · **Session type:** continuation of the scorecard-ratchet arc (4th session) — executed the 17:32 report's "Exact Next Steps" under the standing execute-and-verify instruction
**Scope:** this session only. `.md` per explicit user instruction (HTML-canonical overridden — same standing convention as every report in this arc). Concurrent sessions were live in-tree the whole time (schema/system/systemscenario + cqrs-htmx itself got commits at 18:05/18:10 mid-probe).

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | Evidence                                                                                                                                                                |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Baseline re-test after tree movement (concurrent sessions had landed since handoff)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | main+analyzer suites green (35.9s + 0.4s)                                                                                                                               |
| 2  | **Gate: check-file-size — my 2 violations FIXED.** `doctor.go` 447→359 (profile renderers → new `doctor_render.go`, 96 lines); `feature_detect.go` 357→339 (`stackPresetFromImport` → new `feature_detect_stack.go`, 21 lines)                                                                                                                                                                                                                                                                                                                                                  | build+vet+full suites green after split; gate re-run shows only foreign violations                                                                                      |
| 3  | Gates: check-md-go ✓ (105 baselined), check-lint-config ✓                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | session logs                                                                                                                                                            |
| 4  | **Validation probes ×4** (the handoff's step 2) — journal 5/30 (16%) Minimal, **Modernity: Modern** (regression PASS; denominator 29→30 attributed: foreign `systemscenario` catalog entry, registered 06:10); cqrs-htmx 14/32 (43%) Fair, **Modernity: Legacy**; go-appkit 8/30 (26%), **Modernity: Modern**; self-lint exit 0 (C025 warning pre-existing in doctor_audit.go, 10-04)                                                                                                                                                                                           | binary runs with rebuilt /tmp/cqrs-lint; two consecutive runs stable                                                                                                    |
| 5  | **REAL BUG found by the probes + fixed: multi-module blindspot.** `BuildContext` exposes only the PRIMARY module's profile, so submodule signals were invisible to the scorecard while v007 counts were project-wide (inconsistent inside one panel). cqrs-htmx's stack imports live in `usermgmt/`; go-appkit's system wiring in `cqrs/`. Fix: `stackPresetUseCount` (distinct stack surfaces across ALL per-module profiles) + `compositionWideProfile` (system.New/pushdown signals unioned for composition credit AND Modernity) in `scorecard.go`, wired in `runScorecard` | 3 regression tests; probes re-verified end-to-end; the cqrs-htmx "bundle=1 not 2" delta traced to `//go:build ignore` on sqlite_setup.go — honest exclusion, documented |
| 6  | ADR-0152 ratchet-evidence addendum (cqrs-htmx Legacy + go-appkit Modern = the ADR's migration ordering made measurable)                                                                                                                                                                                                                                                                                                                                                                                                                                                         | docs/adr/0152 §Evidence                                                                                                                                                 |
| 7  | Docs: advanced.md (multi-module union bullet, scorecard JSON schema table, doctor `features` row now names `stackPresets`); README (union note + "Config resolution order (path vs cwd)" section); CHANGELOG bullet extended with clause (c2) — survived an mtime-guard rejection by fresh-viewing first                                                                                                                                                                                                                                                                        | doc-check 1184 refs ✓ after edits                                                                                                                                       |
| 8  | TODO_LIST harvest: 5 of 6 scorecard-ratchet rows → `[x]` DONE with dated receipts; journal row stays owner-gated, annotated with the 30-denominator note                                                                                                                                                                                                                                                                                                                                                                                                                        | TODO_LIST.md §cqrs-lint                                                                                                                                                 |
| 9  | Status-report addenda: 05:23 report (points at 17:32 as current truth) + 17:32 continuation addendum (full verified inventory of this session)                                                                                                                                                                                                                                                                                                                                                                                                                                  | both files                                                                                                                                                              |
| 10 | **Mutation-tested the reflect explain pin** per repo convention: deleted the `scorecard` topLevelKeys row → `TestRenderExplain_DocumentsEveryConfigFileKey` FAILED → restored → green                                                                                                                                                                                                                                                                                                                                                                                           | session logs                                                                                                                                                            |
| 11 | Final battery: gofmt (touched files) ✓; file-size mine-clean ✓; doc-check 1184 ✓; changelog-symbols 37 honest ✓; api-stability TestEvery ✓; full suites ✓; check-cqrs-lint-cli.sh probes + `--self-test` ✓; dup gate — 6 foreign groups (all systemscenario/* + pre-existing output.go idiom), ZERO mine (my splits are moves, not copies)                                                                                                                                                                                                                                      | session logs                                                                                                                                                            |

## b) PARTIALLY DONE

| # | Item                                  | Done                                                                                                                                                                          | Open                                                                                                                                                                                                                                                          |
| - | ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | 17:32 report f-list (24 next-tasks)   | 4 closed by this session (mutation test, doctor-schema stackPresets row, scorecard JSON schema table, CONFIG RESOLUTION ORDER note) — documented in the continuation addendum | the LIST itself was not ticked inline; a docs-health ANNOTATE pass should mark the 4                                                                                                                                                                          |
| 2 | Formatting gate for new files         | `gofmt -l` clean on all 7 touched Go files                                                                                                                                    | `nix fmt`/treefmt (import-grouping) NOT run — deliberately: treefmt walks the whole shared tree and would reformat the concurrent sessions' files. Likely clean by inspection (stdlib→local grouping followed); CI `--fail-on-change` risk remains unverified |
| 3 | Multi-module union coverage           | unit tests ×3 + real-project probes (cqrs-htmx, go-appkit)                                                                                                                    | no BINARY-level probe fixture in check-cqrs-lint-cli.sh (a two-module temp project pinning union behavior)                                                                                                                                                    |
| 4 | Doctor JSON schema for `stackPresets` | advanced.md row documents it; omitempty keeps clean runs unchanged                                                                                                            | no golden test verified this session that exercises a stack-carrying profile through `doctor --format json`                                                                                                                                                   |

## c) NOT STARTED (owner-gated or other sessions' — deliberately untouched)

1. **Tag decision** for cqrs-lint (dup gate still red on the concurrent session's systemscenario groups).
2. **lint/doctor path-preset unification** (CI behavior change; scorecard-only by design so far).
3. **systemscenario clone cleanup ownership** + the `stale.go` 489→533 growth (authored 09-06 "fix(lint)" commit — predates both live sessions; keeps check-file-size red).
4. **Journal-side adoption** (waivers replacing the moat; needs the tag first).
5. **cqrs-htmx stack migration** — their session, actively committing during my probes.
6. rules/* file-size NEW offenders (d005_version 406, v007_tables 357, resilience/helpers 507) — concurrent-branch territory per handoff.

## d) TOTALLY FUCKED UP (all self-caught, zero damage escaped the session)

1. **Pipe-exit misread (the worst one).** `/tmp/cqrs-lint lint … | tail -5; echo "exit: $?"` — I printed **tail's** exit code as the linter's and briefly believed self-lint was green. The repo's own gotchas doc (exit-code traps) warns exactly this. Caught one step later by re-running unpiped (real exit 1 → investigated properly). A wrong green in a report would have been a lie with a lifecycle.
2. **Phantom subcommand.** Ran `cqrs-lint lint` — "lint" was parsed as a PATH arg, producing a confusing phantom-module load error. The bare root invocation is the lint. Wasted a diagnostic cycle.
3. **Trusted the handoff's mechanism, not just its conclusion.** Expected cqrs-htmx Legacy "via stack presets"; the initial `stack-surface: 0` did not trigger immediate suspicion — I only dug in after a doctor-vs-scorecard contradiction. Then predicted stack=2 vs actual 1 and hypothesized detector bugs BEFORE checking build tags (`//go:build ignore` was the answer). Order of elimination was backwards twice.
4. **The bug I fixed was mine from the 17:32 wave.** StackPresets detection + panel counting shipped green on unit tests and a single-module e2e (journal); the multi-module blindspot lived in that shipment. Session-3's own report predicted the cqrs-htmx Legacy flip without proving the mechanism.
5. **Mutation-cycle daemon window.** corrupt→test→restore on explain.go ran across three tool calls while the auto-commit daemon is live — the mutated state could have been absorbed into a `chore:` commit. It wasn't (verified clean after restore). Next time: one batched shell command.
6. **No pre-written expectation for go-appkit.** journal and cqrs-htmx had expected values in the handoff; go-appkit was probed open-ended. Its initial "Partial" was only challenged because the ADR calls it a system wrapper — the tension was luck-adjacent, not discipline.

## e) WHAT WE SHOULD IMPROVE

1. **Probe-first shipping policy for detection/scoring features**: every feature that reads the feature profile ships WITH written probe expectations for ALL companion projects (journal, cqrs-htmx, go-appkit) — written BEFORE the run, so a mismatch is a bug report, not a puzzle.
2. **Union-semantics unit tests from day one** for anything aggregating per-module profiles — the loader's primary-only contract is documented, and I still tripped on it.
3. **Exit-code discipline**: `pipefail` or capture the tool's exit directly; never echo `$?` after a pipe.
4. **Batch mutation tests** (corrupt + run + restore in one command) to shrink the daemon-absorption window to ~seconds.
5. **Check build tags / file exclusion BEFORE suspecting detectors** when a count is lower than expected.
6. **Tick source lists when closing their items** (17:32 f-list is prose-closed, not marker-closed).
7. **Scoped formatter verification** for new files even when gofmt passes — treefmt grouping is a separate gate (CI risk carried in b/2).
8. **Doctor's composition hint has the same primary-only blindspot** I just fixed in the scorecard: the hint condition reads the primary profile, so submodule-only composition (go-appkit) does not get the hint in doctor's primary section (per-module section still shows the data). Same class, one surface over.

## f) Next things (up to 50; honest count 28 + pointer to the 17:32 remainder)

**This session's direct fallout (do first):**

1. docs-health ANNOTATE pass on the 17:32 report f-list: mark 4 items DONE with receipts.
2. Binary probe for the multi-module union in `check-cqrs-lint-cli.sh` (two-module fixture: stack-in-submodule → `stack_preset_uses ≥ 1`; system-in-submodule → `modernity_grade` Modern).
3. Union-semantics note in `explain`'s SCORECARD section (README + advanced.md already document it; explain is the third surface and currently silent).
4. Doctor composition hint: union the condition across per-module profiles (mirror `compositionWideProfile`).
5. Verify/pin `doctor --format json` golden with a stack-carrying fixture (`stackPresets` field).
6. Root `--scorecard` flag parity probe (subcommand is probed; the root path shares runScorecard but is unprobed).
7. Scoped treefmt check on the 7 touched files (close the CI `--fail-on-change` risk).
8. Mutation-test the other new pins per repo golden policy: trigger-signal tests, union tests (only the explain pin was mutation-tested).
9. SARIF probe on a multi-module stack-carrying project (props flow from the same field; cheap certainty).
10. FAQ.md entry: "why does my submodule's system.New count?" (multi-module scoring semantics, one paragraph).
11. Grade-band thresholds (Minimal/Sparse/Fair/…) — document the actual cutoffs next to the band names (advanced.md table lists names only).

**Owner-gated (unchanged, questions in g):**
12. Tag cqrs-lint (or hold).
13. lint/doctor path-preset unification.
14. systemscenario clone ownership + cleanup.
15. stale.go 489→533: attribute (09-06 authored commit) and shrink.
16. Journal-side adoption after tag.

**Foreign/coordination (not mine to start):**
17. rules/* three NEW file-size offenders — confirm concurrent-branch ownership, then split.
18. cqrs-htmx stack-surface migration coaching (their live session; my Legacy grade is the input).
19. `systemscenario` catalog entry relevance policy — it now renders as MISSING for system-composed apps (journal); decide whether that row should be profile-restricted or is a fair adoption prompt.
20. systemscenario/equivalence.go pairing with `output.go:174` (the 5-line make/append idiom) — whoever owns the cleanup may prefer an `//art-dupl:accept` on the systemscenario side.

**Quality-of-life from this session's stumbles:**
21. Add a `--self-test`-style pinned expectation table (project → expected score/grade) to the probe script so regression probes diff against recorded values automatically.
22. Write the pipe-exit lesson into `docs/agents/gotchas-tooling-build.md` if not already there (it likely is — re-read before duplicating).
23. Pre-probe checklist doc for future waves: expectations first, build tags second, exit codes unpiped.
24. Consider a `cqrs-lint scorecard --explain-grades` flag rendering band thresholds + modernity rules in-band (kills item 11's doc drift risk).
25. Consider pinning `ModernityGrade` band wording in a golden (text drift across renders is currently test-asserted by substring only).
26. Re-run the whole probe set after the concurrent sessions land (numbers are snapshots; cqrs-htmx moved mid-session).
27. Batch re-verify (`#verify`) once the tree is quiet and gates can run exclusively — this session ran scoped gates only, by design under concurrency.
28. Harvest this report's f-list into TODO_LIST/ROADMAP via docs-health (NOT done now, per the user's report-only instruction).

**Plus:** the 17:32 report's f-list still carries ~20 open items not repeated here (scorecard schema tooling, journal C033 routing, E018 parity, etc.) — see its continuation addendum for the 4 closed.

## g) Questions I can NOT figure out myself

1. **Tag cqrs-lint now or hold?** The dup gate is red solely on the concurrent session's `systemscenario/` groups (zero mine); tagging over a red gate repeats the 2026-10-06 red-suite lesson, but holding blocks journal adoption of waivers/modernity. Your arbitration between the two sessions' cadence.
2. **Should lint/doctor also resolve `<path>/.cqrs-lint.json` presets** (scorecard now does; lint/doctor still read cwd config)? It is a CI behavior change for everyone who runs with `--path` from a config-carrying cwd — policy, not code.
3. **Who owns the red-gate residue:** the systemscenario clone groups (concurrent session, presumably theirs) AND `stale.go`'s 489→533 growth (an authored 09-06 commit that predates both live sessions — I could shrink it, but "never revert changes you didn't author" applies until you say it's mine)?

---

**Standing notes:** no manual commits (harness contract; daemon absorbs). Concurrent sessions' files untouched throughout (schema/_, systemscenario/_, system/_, rules/_, stash). Did I lie? Once, briefly, to myself: the piped exit code reported self-lint green mid-session — corrected within the same step before any decision consumed it. Everything in section a is re-runnable from the logged commands.
