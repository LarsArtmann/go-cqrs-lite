# Status Report: Pareto Continuation — T05/T09 Close + CI Red Triage Wave 2 (2026-11-04 02:18 CET)

**Session:** Commissioned continuation of `docs/planning/2026-11-03_18-52_SUPERB-full-backlog-pareto-plan.md` (resumed at the 22-15 report's Exact Next Steps). Owner questions asked at session start per 22-15 §e.4.

> **Scope — what this report VERIFIED:** every "fixed/green" claim below was verified this session by running the command (test runs, gate scripts, gosec/erraudit binaries, shfmt/shellcheck/actionlint). NOT verified: a green **CI run on master** carrying this session's fixes (the tree settles via daemon commits; the next full run is the T01.5 receipt), the gosec leg's full history (job-history query still running at report time), and the composed `#verify`/`#verify-fast` legs (host load1 >100 from parallel sessions all night; individual constituent gates were run instead).

---

## a) FULLY DONE (verified this session)

1. **T03 owner decisions ASKED and ANSWERED (3/3).** (i) metaengine promotion scope = **ALL 16 engine-family modules** (T27 commissioned; T13 carve unblocked); (ii) **authored commits authorized at phase boundaries** on commissioned plans (4th-consecutive-chore-wave pain); (iii) deployed-DB census = **TBD/parked** (sticky-key fallback stands). Recorded: TODO_LIST harvest section (ruling rows), AGENTS.md Ground Rule 4 amended, the old disk-truth-section owner-question row marked ANSWERED.
2. **T05 CLOSED (json/v2 determinism).** `adttest.AssertDeterministicMapEncoding` wired into 4 engine suites: pebble + sqlite = **real byte-level pins (PASS)**; bbolt + badger = documented skips (no `RawValueReader`; encoder pinned at keycodec level; pin auto-activates if raw reads land). Full LSM-family suites green: pebble 9.8s, badger 4.0s, bbolt 48.9s (`SOAK_SKIP_BOLT=1`), keycodec 0.007s. check-test-skips green (239 sites reasoned). CHANGELOG row written (Changed).
3. **CHANGELOG honesty gate green** — 21 pkg.Symbol citations verified; the gate-fix cluster got its consolidated Fixed row (pin-sweep regex+modsums, V007 exemption, calibration row, shfmt/actionlint, scratch rowing) per 22-15 §e.5 (rows written in-window, not deferred).
4. **CI-red triage wave 2 — every deterministic failing leg from the 21:27/21:52 runs root-caused locally and fixed green:**
   - **fmt**: 4 unformatted Go files (2 mine this wave, 2 from last session's daemon commits — v007_tables/drift_test were committed unformatted) formatted; `nix fmt` clean.
   - **shellcheck**: 15 findings across 9 gate scripts (SC1007, SC2012 ×2, SC2028, SC2034 ×5, SC2064, SC2086) — all fixed, zero-findings policy restored; the check-cycles patterns builder restructured to an array (SC2086) with the REAL gate re-run green (0 cycles / 510 edges); all 8 touched scripts' `--self-test` green.
   - **shfmt**: probe-proxy-tags.sh drifted again by a newer foreign commit — reformatted with the pinned 3.14.1.
   - **pin-sweep / Module-Layers leg**: root cause = 3 stale pins (cqrs-upgrade→cqrs-lint v4.18.0-v4.19.0-tagged, goal-shaped-app+systemcatalog→catalog v4.11.0); swept; `--check` green. Bonus: cqrs-upgrade now HAS the V007 exemption via the newer pin (22-15 §f item 48 closed).
   - **tidy / cold-cache leg**: ~15 modules carried stale transitive `/go.mod` hash lines from the day's tag waves; workspace-wide tidy; **103/103 modsums green**.
   - **package-surface / Verify-fast constituent**: correctness package 47→48 — reviewed the foreign c044→c044_display split (tests green, golden updated by them), re-pinned the baseline (8 offenders).
   - **erraudit**: TWO root causes. (i) My own last-session `id/utf8_validation_test.go` used `errors.As` — migrated to `errors.AsType[interface{ error; Code() string }]` (go-error-modernization skill loaded; id suite green; pinned v0.7.0 binary rc=0 on id). (ii) The local mirror ran a NEWER system erraudit that audits vendored JS (catalog/docserver/static) which CI's pinned v0.7.0 does not — mirror red, CI green, silent for days. Fixed structurally: `erraudit-sweep.sh` now builds from the SAME vendored tarball CI pins (cache + tarball-newness check); sweep result **104/104 CLEAN**. CI leg got `--disable-extensions` (verified v0.7.0 accepts the flag; future-proofs the bump).
   - **integration-tagged lint leg**: 12 of 16 modules failed on untidy sums breaking golangci loading (fixed by the tidy wave — re-run: 15/16 OK). The real findings in cmd/cqrs-lint: nilerr ×2 (f032.go, t009.go returned `nil, nil` on finding-build errors — now return the error, honest detector-defect surfacing), godox ×1 (reworded a TODO_LIST mention), dupl+nolintlint (the three catalog_* rule tables' per-file `//nolint:dupl` ping-pong replaced by a .golangci.yml path exclusion, contract-19 class; config hash re-pinned via the sanctioned `--update`). Final: `lint-module cmd/cqrs-lint integration` = **0 issues**.
5. **T09 HARVEST executed (docs-health skill, HARVEST mode).** Both 2026-11-03 plans reconciled: 18-52 (T-tasks, mine) was NEVER indexed in the TODO_LIST header — added; the 20-04 sibling (M-tasks, parallel session) noted as same-backlog re-cut. New harvest section (~28 rows): T01.5 close-out, cqrs-lint ≥v4.20.0 tag carrying T009 (foreign gap), T06→T27 routed with effort tags, foreign 22-24 rule-quality items, ActorID UTF-8 design row, scratch/ disposition, api-count stragglers. Ruling rows consolidated into the section (no disk-truth duplication).
6. **Dogfood receipt CLOSED by CI evidence** — the 21:27 run's "cqrs-upgrade dogfood (consumer dry-run)" job SUCCEEDED (carries the V007 exemption). The interrupted third local run is moot.

## b) PARTIALLY DONE

1. **T01.5 (CI green receipt).** All locally reproducible legs are green (above); remaining unknowns: **gosec** (45 findings reproduced locally: G115 ×14 `event/asrecord.go`, G201 ×21 + G202 ×8 `storage/timer_store.go`, G301+G304 `storage/turso/library_warmup.go`; run-history query for last-passed was still executing at report time — disposition not decided), **MySQL QEMU** (deferred, runner class), and the composed legs (Nix Flake Check / Verify-fast — ride the next run; their known constituents are fixed). The next master CI run on the settled tree is the receipt; QEMU is the long pole.
2. **Authored-commit phase boundary NOT exercised** — the ruling is live (AGENTS + TODO_LIST) but every edit so far was absorbed by the daemon before a natural boundary; mechanics (dirty-tree guard interplay) untested.
3. **cqrs-lint ≥ v4.20.0 (T009) release** — rowed with priority in the harvest section; not executed (release decision belongs to a fresh tree window).

## c) NOT STARTED (rowed in the harvest section, plan order)

- T06 doc-truth automation · T08 cqrs-upgrade `--to-v5` slice · T10/T11 v5 deletion waves · T12/T13 metaengine shape · T14 migration guide · T15 encryption ADR · T16–T19 testing rails · T20 lint perf · T21 FilterOps/AsyncAPI · T22 BDD tails · T23 Registry bridge · T24 housekeeping · T25 composed verify S04 · T26 untagged-API gate · T27 promotion wave (COMMISSIONED).

## d) TOTALLY FUCKED UP!

1. **The .golangci.yml edit split a rule's linter list mid-way.** I inserted my exclusion after `- forbidigo` having viewed only to line 854 — the `cmd/cqrs-lint/` rule's list continued (`- wsl_v5, - nlreturn, - gocyclo, - cyclop, - maintidx`) and the tail silently rebind to MY rule's `linters:` key. Symptom: 81 phantom findings (wsl_v5: 50, nlreturn: 30, maintidx: 1) that looked like MY exclusion broke linting. Caught by immediate re-lint; fixed by restructuring (complete list restored + my rule after it). The config-hash gate independently flagged the edit (re-pinned after the CORRECT version). Lesson: view through the block's END before inserting after a list — anchor on the following rule's `- path:` line instead.
2. **First bbolt suite run WITHOUT `SOAK_SKIP_BOLT=1`** — 613s of soak under load1>100, FAIL, and I initially treated it as a candidate real regression before remembering the documented policy (soaks are weekly; `#test` sets the skip for exactly this starvation reason). My omission, not a product bug.
3. **Log-path bug in my lint loop** (`/tmp/lint-$mod.log` with `$mod` containing slashes) — failure tails unreadable; re-ran with `tr / _`.
4. **`rg -rn` hook strike (4th on this host)** — reflexive `rg -rn ""` for a file listing; blocked, wasted a roundtrip. The exact trap the 22-24 foreign report documented as "third strike".
5. **False rc=0 through a pipe** — `cmd | head -3; echo rc=$?` measured head's exit in the erraudit probe and printed a misleading rc=0; re-ran unpiped. The pipeline-masking class the 22-24 report §d.3 warned about, committed by me in the same session I read that warning.

## e) WHAT WE SHOULD IMPROVE!

1. **Insert-after-list edits need the full block** — my one real bug this session came from editing against a truncated view. Rule of thumb: when inserting a sibling YAML rule, anchor the old_string on the NEXT rule's `- path:` line, never on the last seen entry of the previous list.
2. **Local-mirror/CI binary skew is a standing failure class** — erraudit drifted silently (mirror red, CI green, for days). The vendored-build default now in `erraudit-sweep.sh` is the pattern; audit other mirrored gates for the same skew (lint-module vs CI's nix lint versions, actionlint/shfmt channel pins are runner-vs-host too).
3. **rc hygiene** — never capture an exit code through a pipe without explicitly accounting for it (`PIPESTATUS[0]`, or no pipe). Two of my five fuckups were measurement bugs, not product bugs; measurement bugs are worse because they feel like receipts.
4. **Tag waves leave tidy-residue across NON-swept modules** (15 modules went untidy from waves cut by other sessions; no gate saw it until CI). The pin-sweep's terminal modsums assertion catches its OWN residue but not other waves'. Candidate: a cheap nightly tidy-diff gate (check-modsums already exists — wire it nightly if not already).
5. **Parallel-session cost accounting**: tonight's triage spent most of its budget re-deriving what parallel/foreign sessions left red mid-flight (fmt drift, phantom lint findings, untidy trees, a config-hash trip). The claim-rows-before-executing convention (22-15 §e.2) plus daemon staging delay remain the two highest-leverage fixes; both rowed.

## f) Next up to 50 (harvest input; full routing lives in the TODO_LIST harvest section)

1. gosec disposition: annotate-with-reasons / fix / legacy-exclude the 45 findings; confirm when the leg last passed (history query was mid-flight at report time)
2. Watch the next master CI run → close T01.5 (receipt: green run on a post-settle commit)
3. cqrs-lint ≥ v4.20.0 tag carrying T009; then fleet census for `< v4.19.0` pins (F032 invisible to them)
4. T06.1 `readme_compile_test.go`; T06.2 canonical-facts backtick-strip + `--self-test`; T06.3 backtick count sweep
5. T08.1–4 `--to-v5` import-rewrite design → tests → strict-gate → dry-runs (cqrs-htmx, go-appkit, bank-sync)
6. Exercise the authored-commit phase boundary once (policy mechanics: dirty-tree guard interplay)
7. MySQL QEMU leg: re-run on the settled tree (or accept Autoretry's classification)
8. T27 promotion wave: 16-module evidence table → stamp flips → FEATURES/SKILL/matrix → api golden regen (COMMISSIONED)
9. T12 memoryengine extraction; T13 five-package carve ADR (unblocked); T14 guide; T15 ADR-0139
10. T10 stack census/migration/delete; T11 read-tier deletions + ADR-0114/0126 shells
11. T16 mutation gate pilot; T17 synctest pilots; T18 planner MBT; T19 differential harness
12. T20 cqrs-lint perf profile/fix/bench-pin; T21 FilterOps + AsyncAPI; T22 BDD tails; T23 Registry bridge; T24 housekeeping; T25 verify-chain S04; T26 untagged-API gate
13. Foreign 22-24 §f: F032/T009 rule quality (anchor sort, TypesInfo qualifier, call-detection, suppression rows); taskmanager systemcatalog wiring (row exists)
14. ActorID invalid-UTF-8 design decision row; scratch/ owner disposition; api-count stragglers sweep
15. Audit other mirrored gates for binary skew (e.2)
16. Module-count predicate discrepancy: erraudit-sweep counts 104 modules, check-modsums counts 103 — align or document the differing find predicates
17. Harvest the 21-15 parallel report's §f if not fully covered (unverified — see §g.3)
18. TODO_LIST harvest-section formatting re-verify after daemon commit
19. Read 22-24 §b legs never run locally: doc-check on their symbol changes (their ritual debt, gate covers it)
20. gosec: if annotated, add `#nosec` reason style guide row to AGENTS error-handling section

## g) Questions I can NOT figure out myself (max 3)

1. **gosec posture** — the 45 findings concentrate in deprecated/legacy surfaces (`storage/timer_store.go` = the sqlstore-class store scheduled for v5 deletion; `event/asrecord.go` integer-width conversions; the turso test seam): annotate-with-reasons now, fix properly, or exclude legacy paths from the gosec leg until the v5 deletion waves remove the code? (Fixing deprecated code the v5 cut deletes is budget spent twice.)
2. **Authored-commit mechanics** — the daemon absorbs within ~2 min and tag-release's dirty-tree guard requires a clean tree: should phase-boundary commits happen even when the tree carries in-flight edits (risking the guard), or only at wave closes (risking another fully-absorbed wave)? A concrete rule ("commit when `git status --porcelain` shows only my phase's files") would settle it.
3. **21-15 parallel report** — I never opened its §f (worked from the 22-15 summary). Harvest it too, or is it fully subsumed by the 20-04 plan rows + the TODO_LIST harvest section?

---
*Receipts available in-session: gate outputs cited per item; erraudit v0.7.0 built from the vendored tarball and diffed against the system binary; gosec SARIF parsed (45 results across 5 rules); all touched scripts shfmt/shellcheck/self-test green.*
