# W1 Session Report — M39 Residue Cleared, M40 Two-Thirds, Chain #5 Storm-Riding (2026-11-15, 07:30–08:17 CET)

> Standing order: "GET SHIT DONE! The WHOLE TODO LIST!" over plan
> `docs/planning/2026-11-13_21-12_SUPERB-truth-green-release-train-v5-runway-pareto-plan.md`.
> This session resumed the W0 continuation after the 18-46 report, fixed the
> coverage DRIFT that killed chain #4, armed chain #5, and cleared the whole
> M39 residue plus 2 of 5 M40 items. Interrupted mid-:839 for this report.
> **Status at write: WAITING FOR INSTRUCTIONS** (chain #5 storm-rides in background).

## Environment

- Load1 **847–1115** the whole session (foreign fleets; 32-core host). All
  work was load-independent by design; the chain storm-rides by design.
- **A parallel session is ALSO active**: git log shows
  `ddcdfcce5 chore(release): record metaengine/v4 v4.25.0 (PlannedTableSweeper
  + Store.TrimPlannedBefore, MF3.2)` — someone is cutting a metaengine release
  tag mid-flight. Chain #5's tree-stability refusals are partly theirs.
- Auto-commit daemon absorbed every edit promptly (tree clean at write).

---

## a) FULLY DONE (verified this session)

### 1. Coverage DRIFT fixed — `pkg/rules/version` 84.3 → **89.5** (floor 88.0)

The debt that killed chain #4 at 08:38 yesterday (F232's ~80 lines
suite-tested but never coverage-tested):

- **Seam refactor** `cmd/cqrs-lint/v4/pkg/rules/version/v003_latest.go`:
  `latestMinorFromProxy` split into guard wrapper + `latestMinorFromProxyLive`
  (the `testing.Testing()` guard made every line below it unreachable under
  `go test` — the same injectable-seam pattern as `latestMinorFetch`).
- **NEW** `v003_latest_proxy_test.go` — 12 tests: wrapper guard;
  bypass-never-reaches-proxy (counter-proven, zero proxy hits); cache
  hit/poisoned-entry/fetch-success + cache-absorbs-second-call (1 hit);
  fetch-failure-not-cached; `firstProxyBase` 6 env cases;
  `pathHasTrailingGlob` 4; `globPrefixMatches` 7 (incl. pattern-longer,
  malformed-glob); `moduleBypassesProxy` malformed pattern;
  `fetchLatestMinorFromBase` error branches (non-200, garbage body, empty
  version, truncated body, malformed URL); semaphore cancellation (4 slots
  filled + pre-canceled ctx — deterministic, no 1500ms wait).
- **Verified**: scoped `-cover` = **89.5%** (floor 88.0, +1.5pp margin);
  whole `pkg/rules` tree green (9 packages). Semaphore test is safe under
  `-shuffle`: sequential-phase vs parallel-phase semantics hold regardless
  of order.
- **Q3 policy resolved operationally**: restored the floor with tests, did
  NOT use the new `--issue` acceptance path. The owner-policy question
  stays open only for FUTURE debt (see questions).

### 2. Chain #5 ARMED (07:36) — storm-riding

`build/verify-chain-20261115-073615.log`, pid 1089341. At write: attempt 13
of the wait-loop, refusals = load ceiling (847 ≥ 10) + tree stability
(parallel session + daemon). 4h max-wait from first GREEN attempt. The
STALE improvements (analyzer 68.8 / doc-check 74.0 / sqliteengine 74.3) are
**green-but-STALE** per the compare logic (verified in
`check-coverage.sh:381-393` — only DRIFT/MISSING/DANGLING fail), so the
chain will NOT die on them.

### 3. M39 residue — ALL 5 items closed

| Item | Deliverable | Verification |
| --- | --- | --- |
| :324 generated-artifact pre-push sweep | `scripts/check-generated-artifacts.sh` + `nix run .#check-generated-artifacts` — 5 legs (docs-ui.css `--check`, error-taxonomy `--check` via vendored erraudit cache, engine-capabilities gate, api-golden regen + `git diff --exit-code` **with drift-RESTORE**, architecture-export gate); EventCatalog npm leg deliberately excluded (network render-validation, not regen+diff) | self-test 3 fault classes (leg registration, planted api-golden drift detected AND restored, clean pass) green **through the nix app** |
| :326 erraudit wrapper | `nix run .#erraudit` flake app wrapping `scripts/erraudit-sweep.sh` (vendored-tarball build); go-env sourced per the script rule | app builds after shellcheck SC1091 fix (`# shellcheck disable=`); smoke = binary-override refusal path through nix; full 104-module sweep deferred to pre-push window (would burn CPU the chain needs) |
| :1343 `#verify-serial` tier | `scripts/verify-serial.sh` + app: `VERIFY_FORCE=1 VERIFY_TEST_P=1 nix run .#verify` under `timeout --kill-after=5m ${VERIFY_SERIAL_MAX:-6h}` | `--self-test` green through nix (VERIFY_FORCE still honored, VERIFY_TEST_P still wired, timeout present); same gate list BY CONSTRUCTION (no second copy to drift); honesty note in header re per-gate vs outer-bound |
| F250b flake.lock coherence | `scripts/check-flake-lock-coherence.sh` + app: refuses the incoherent commit shape — a HASH-PINNED input's rev (`md-go-validator`, `go-finding` — the only two feeding hash-bearing derivations, verified against mkPreparedSource topology) bumped with ZERO sha256-line movement; path-consumed inputs exempt; hash-without-rev = warn-only (re-pin class); `--baseline <rev>` range audits | self-test 5 fixtures; **caught a GENUINE historical incoherence at HEAD~30** (go-finding rev bump, zero hash movement, rc=1); live tree green |
| :1655 Sunday load-sweep watch | First real Sunday run FIRED 2026-11-15 02:25:50Z (timer + guard proven) but logged `scripts/load-sweep.sh: Permission denied` — file shipped without +x (masked because the flake app calls it via `bash`; the nightly's direct exec exposed it) | +x fixed (mode-only diff); `bash -n` clean; scanned ALL scripts for the class — 33 non-executable, but every other direct-exec invocation site verified executable (only load-sweep was bitten); same night's bench-leg noise-gate failures = known extreme-load class |

### 4. M40 — 2 of 5 done

- **:328 SSE drop counter** — DONE. `metaengine.SSEDroppedEvents()`
  (process-wide atomic on the ring's evict-oldest path). Pinned by 3 tests:
  `TestForwardWithDropOld_EvictionsCounted` (throttled-consumer repro: 2
  evictions counted, only newest survives — the taskset-throttled repro
  realized at logic level), `..._FilteredValuesNotCounted` (filter drops ≠
  backpressure), `TestServeSSE_FastConsumerNeverDrops` (healthy window must
  show delta 0 — the diagnostic contract). **Same-edit rule kept**: api
  golden regen (exactly +1 line), CHANGELOG Added row,
  `check-changelog-symbols` green (20 citations verified). Next CI
  recurrence of the systemscenario SSE flake now classifies itself in one
  query.
- **D9 watermill budget-tune** — DONE. Per-iteration **5s watchdog** in
  `TestCommandBusPublishRacingCloseNeverLeaksRawTransportError` (iteration
  goroutine + `select` done-vs-timeout on the test goroutine — legal
  FailNow; wedged goroutine dies with the process). **Mutation-verified**:
  planted 10s pre-Close sleep → FAIL at exactly 5.00s naming iteration 0 →
  restored → green single-shot AND repeat-shape (`-count=2 -race -cpu=1,4`).
  D-sheet D9 + watermill row closed with lineage preserved; scheduler
  diagnosis parked until the watchdog ever fires in nightly.

### 5. Docs/rows kept honest (same-edit)

- TODO_LIST.md: 8 rows closed with dated receipts (:324, :326, :1343, F250b(b),
  :1655, :328, watermill/D9 + D-sheet D9). One row edit initially DROPPED
  the :328 forensic lineage (CI run IDs, dates, local-repro attempts) —
  caught and re-restored into the receipt (memory rule: no silent concept
  loss).
- AGENTS.md Quick Reference: +4 rows (Verify serial, Gen artifacts, Lock
  cohere, Erraudit).
- CHANGELOG.md: [Unreleased] Added section created for the SSE counter.

---

## b) PARTIALLY DONE

- **M40 :839 checkpoint exact-count seam** — IN PROGRESS at interrupt:
  `projectionhost/v4/checkpoint_cadence_test.go` read in full (the flake:
  `saves == 3` exact-count over a 120ms wall-clock window; the `-race`
  sweep saw `saves = 4`; no mechanism established). **No edit made yet.**
  The row's own tension stands: exact-count guards cadence regressions,
  relaxing hides them; the commissioned fix is a clock seam (projectionhost
  has NO clock seam today — `WithClock` exists only in `scheduling/`, the
  ADR-0153 D4 sanctioned-additive pattern to copy).
- **M40 :1764 dgraph+redis shuffle watch** — row NOT YET READ; no action.
- **M39 receipts ride on chain #5** — all five closed in the TODO sheet, but
  the composed proof they will cite is still storm-riding.

## c) NOT STARTED

- **Baseline STALE pin** (todo 2): `check-coverage.sh --update` after chain
  green, in a quiet window (it RE-MEASURES ~1h; under load 847 a
  timeout-skipped package would measure 3-5pp low and the F250a guard would
  rightly refuse — running it now would waste the hour).
- **M34 receipts** (needs chain #5 green): dated composed-verify receipt,
  close the verify-debt rows, phase-boundary authored commit (2026-11-03
  ruling), `check-todo-list-hygiene.sh` + `check-todo-stale-premises.sh`,
  `#check-lint-config` re-run (I edited flake.nix — new apps parse and
  self-test, but the lint-config canary leg has not run).
- **M35 release wave** (needs M34): `build/f221-changed-set-tests.sh` re-run
  (tree moved — AND the parallel session just tagged metaengine v4.25.0,
  so the changed-set must be re-derived), `batch-release.sh --from-manifest`,
  `check-versions-manifest.sh --update`, `#check-release-scripts`, CHANGELOG
  release-train note, full `#check-generated-artifacts` + `#erraudit` runs
  (pre-push battery).
- **W2–W4 (M41–M57)**: untouched (v5 deletion program, v5 cut, metaengine
  promotion, keycodec-v2, testing-rigor programs, cqrs-lint pack, BDD pack,
  catalog truth, fleet census, docs tails, upstream matrix).

## d) TOTALLY FUCKED UP (nothing unrecovered)

- **Coherence-check false positive (caught in testing)**: the first draft
  red-flagged ANY lock-input rev bump with zero hash movement. Live
  plumbing proof against HEAD~3/10/30 immediately false-positived on real
  history (cmdguard bump — a path-consumed `use`-block sibling that
  legitimately never moves a hash). Fixed by scoping the red rule to
  `HASH_PINNED_INPUTS` = {md-go-validator, go-finding}, exempting the rest,
  + a 5th self-test fixture for the exempt class. Lesson: the "mapping
  design" F250b asked for was not optional bureaucracy — the coarse rule
  was wrong without it.
- **Shellcheck SC1091 build failure (one round-trip)**: writeShellApplication
  shells out to shellcheck; my `source go-env.sh` in the erraudit app text
  failed the build. Fixed with the disable directive; re-smoked green.
- **edit-tool tab miss (one round-trip)**: watermill test file uses tabs;
  first edit attempt failed on whitespace, second landed. No damage.
- **No chain interference**: all edits landed while chain #5 was in its
  wait-loop (tree-stability refusals are the concurrency mechanism — by
  design). The moment the log shows the gate STARTED, tracked-file edits
  must stop; not yet happened.

## e) WHAT WE SHOULD IMPROVE

1. **Coverage-test new prod code in the SAME session** — F232's miss
   (suite-tested, coverage-untested) cost a 10h-dead chain and this
   session's first hour. Rule candidate: any prod edit that adds a branchy
   helper gets its `-cover` delta checked before the session ends.
2. **Executable bits are invisible to the flake apps** — `bash script.sh`
   masking hides missing +x until a timer/service direct-execs it (the
   load-sweep incident). Candidate: a one-line leg in
   `check-lint-config` or canonical-facts that flags non-executable
   scripts referenced by direct exec.
3. **Parallel-session release tagging vs chain tree-stability** — the
   metaengine v4.25.0 tag landed mid-wait; the chain absorbs it (by
   design), but M35's changed-set derivation MUST re-read the tag state,
   not the staged manifest assumptions.
4. **The `#erraudit` full sweep has never run through the new app** —
   smoke only. Schedule it in the pre-push battery (it is 104 modules;
   under quiet load ~10 min).
5. **Status interrupts mid-test-read** — :839 was interrupted exactly at
   the point of choosing the seam shape; next session should re-read
   `checkpoint_cadence_test.go` + worker.go flush path from scratch.

## f) NEXT — up to 50, in execution order

1. Monitor chain #5 (`ls -t build/verify-chain-*.log | head -1` + tail) —
   DEAD-MAN CHECK FIRST on any resume.
2. When chain #5 GREEN: `#check-lint-config` re-run (flake.nix edited).
3. `check-todo-list-hygiene.sh` + `check-todo-stale-premises.sh`.
4. M34 receipt: dated composed-verify row, close verify-debt rows.
5. Phase-boundary authored commit (M39+M40+coverage fix) — 2026-11-03 ruling.
6. Quiet window: `bash scripts/check-coverage.sh --update` (STALE pins;
   F250a guards decreases).
7. `build/f221-changed-set-tests.sh` re-run (tree moved + parallel tag).
8. Verify the parallel session's metaengine v4.25.0 tag coheres (versions
   manifest, api golden).
9. `./scripts/batch-release.sh --from-manifest build/wave-r11-13.manifest`
   (collect refusals = pass B).
10. `bash scripts/check-versions-manifest.sh --update`.
11. `nix run .#check-release-scripts`.
12. CHANGELOG release-train note.
13. Pre-push battery: full `#check-generated-artifacts` run.
14. Pre-push battery: full `#erraudit` sweep.
15. `#check-flake-lock-coherence` against origin/master baseline pre-push.
16. M40 :839 — read worker.go flush path; add projectionhost clock seam
    (ADR-0153 D4 pattern) OR `>=3` + persisted-property assert; keep
    exact-count where deterministic.
17. M40 :1764 — read the dgraph+redis shuffle watch row; observe/dispose.
18. M40 :837 — keep observe-only (no recurrence since 2026-09-28).
19. Re-observe Sunday 2026-11-22: load-sweep leg must log CLEAN post-+x.
20. M41 v5 deletion program: `stack/*` census + deletion wave (T10 :205).
21. M41 read-tier deletions (Materialize, RelationalProjection, GraphProjection…).
22. M41 deriver ABSORB row (:1854).
23. M41 wire-vocabulary tail (:1909) + ghost-module tail (:1936).
24. M42 RawQuery rename + typed QueryDecl at cut (:894).
25. M42 scan-default flip on v5 branch (:2134).
26. M42 V5-MIGRATION-GUIDE (:208/:1959).
27. M42 mirror-lag eventtest fork (:1589).
28. M42 **cut v5.0.0** per cut-readiness checklist (:1966) — owner-gated.
29. M43 core/v5 23 copy-forwards post-split (:2380).
30. M43 `cqrs-upgrade --to-v5` codemod (:2375).
31. M44 metaengine promotion evidence table (:221/:231).
32. M44 memoryengine extraction (:207/:2371).
33. M45 keycodec-v2 dual-read migration (A9, :336).
34. M46 mutation-verify gate pilots (:2643/:210).
35. M46 synctest wave 1 (:2648/:211).
36. M47 differential engine harness (:2656/:213).
37. M48 cqrs-lint perf pack (:214/:2601).
38. M49 BDD consumer pack (:734/:737…).
39. M50 catalog truth (:625…:1460).
40. M51 metaengine substrate tails (21 rows).
41. M53 quiet-window calibration campaign (7 rows).
42. M54 fleet census + adoption truth (9 rows).
43. M55 docs/skill + code-quality tails (19 rows).
44. M57 upstream/blocked matrix (25 rows).
45. After M35: answer D16 (push?) — BLOCKED on owner.
46. D17 ratify 48-module manifest — BLOCKED on owner.
47. Q3 coverage-debt policy (restore vs --issue acceptance for internal
    tooling) — BLOCKED on owner (this session chose restore-the-floor).
48. `scripts/check-flake-lock-coherence.sh` per-attr mapping = documented
    next step (not rushed, per row).
49. Candidate: exec-bit leg for direct-exec'd scripts (see e-2).
50. Weekly: verify 2026-11-22 nightly log (bench + FIRST CLEAN load-sweep).

## g) QUESTIONS (cannot figure out myself)

1. **D16 — push master + the tag wave once chain #5 is green?** Local tags
   accumulate (now incl. the parallel session's metaengine v4.25.0); I do
   not push without an explicit ask. YES/NO?
2. **D17 — ratify the 48-module release manifest** (incl. the tursoengine +
   sqlstore policy holds) so M35's batch-release is the owner-blessed wave?
3. **Q3 — coverage-debt policy going forward**: always restore-the-floor
   with tests (what I did for pkg/rules/version), or is issue-ref'd floor
   acceptance (`--update --issue <ref>`) sanctioned for internal tooling
   (cmd/*) where 88% costs disproportionate test effort? F250a forces a
   structural answer.

— Recorded 2026-11-15 08:17 CET. Chain #5: attempt 13, storm-riding, healthy.
