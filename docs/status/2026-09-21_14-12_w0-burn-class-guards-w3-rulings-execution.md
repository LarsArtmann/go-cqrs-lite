# Status Report: W0 Burn-Class Guards + W3 Rulings Execution (SUPERB plan M01–M10 wave)

**Date:** 2026-09-21 14:12 CEST
**Session scope:** execute the 2026-09-20 17:40 SUPERB owner-unblock/trust plan
(M01–M27) from the top: M02–M05 guard build, M01 W3 ruling application (owner
answered live), M07/M09 CI wiring, M10 VM hardening — with M06/M21/M24
discovered done by concurrent sessions mid-flight and integrated instead of
duplicated.

---

## a) FULLY DONE

| # | What | Evidence |
|---|------|----------|
| A1 | **M02 — `.golangci.yml` hash-golden tripwire** — `scripts/check-golangci-hash.sh` (sha256 vs `scripts/golangci-config-hash.golden.txt`; `--update` re-pin REFUSES to run until the shape gates pass, so corruption can never be pinned as baseline); wired into `#check-lint-config` (→ `#verify`), the pre-commit `.golangci.yml` trigger, and `#check-release-scripts`; 4-leg self-test + mutation-proof (mangled depguard entry → gate fails → restored) | `nix run .#check-lint-config` green end-to-end; `#check-release-scripts` 86/86; gotcha "Incidents 8-11" in docs/agents/gotchas-tooling-build.md |
| A2 | **Incident #11 caught LIVE during M02 rollout** — the golden was first pinned over a config that was ALREADY corrupted at HEAD: auto-commit `96dc20986` (09-20 17:00, 111 files) carried go downgrade 1.27.1→1.26.7 + `goexperiment.jsonv2` tag resurrection + depguard block deletion + exhaustruct rationale-comment stripping + gci re-enable, and sat in HEAD ~4h because no gate ran. Repaired from last-good `9212c8408` (zero legitimate hunks in the diff — verified hunks are all five corruption shapes), depguard golden regenerated with the correct 4-space indent (the checked-in golden was 0-indent, which makes the splice invisible to `extract_block` — a latent FICTION loop) | `git diff 9212c8408 96dc20986 -- .golangci.yml` (5 hunks, all corruption); post-repair `check-formatters`/`restore-depguard`/`check-depguard` (137 deps) all green untouched |
| A3 | **M03 — `can-run-composed-gate.sh --wait-loop`** — `--max-wait` (default 3600s) / `--retry-interval` (default 120s); full assert (procs + tree stability + load) retried until GREEN; self-test grew rebound-recovery and timeout legs | 5/5 legs green locally AND under `CI=true` |
| A4 | **M04 — `scripts/preflight-composed.sh`** — 6 cheap phases (lint-config, templ, bench-gate, coverage, api-stability, duplication), stop-at-first-red with printed remedies, `PREFLIGHT_ONLY/SKIP` env hooks, hermetic self-test via `PREFLIGHT_FAKE_RC_*`; launch recipe documented in gotchas | self-test 5/5; **caught a REAL templ FileName drift on its first live run** (committed codegen carried cwd-corrupted `catalog/docserver/*.templ` FileNames — the known root-run-templ class); regenerated from `catalog/docserver` → both `#check-templ` legs green |
| A5 | **M05 — in-`#verify` load guard** — `scripts/verify-load-guard.sh` at the top of `#verify`: refuses > ceiling (~0.1s) with the retry recipe (`--wait-loop`), the preflight pointer, and the `VERIFY_FORCE=1` escape hatch; delegates the probe to calibration-gate (one load semantics repo-wide); CI passes through | 4 behaviors fixture-tested (quiet pass / loud refuse+recipe / force / CI) |
| A6 | **M01 — all six W3 rulings applied** — Q4 **yes**: `scripts/check-go-version.sh` (contract read from go.work, stub-binary self-test 4/4) + flake app `#check-go-version` + `#verify` head + nightly `Go version contract` step + contract note in gowork-modes.md. Q5 **idk→advisory default**: `scripts/lib/verify-lock.sh` flock (machine-level lock path outside the worktree, holder-PID diagnosis, `VERIFY_LOCK=0` opt-out) taken by `#verify`, pre-commit (60s wait, warn-only), `tag-release.sh`/`batch-release.sh` on WRITE windows only. Q7: `example/taskmanager/v4.*` remote tags **deleted** (`v4.0.0/v4.0.1/v4.1.0` + local; baseline lines trimmed). Q1: readme_claims **kept** (ownership note in module-map). Q3: 10:31 repair **ratified** (row struck). Vehicle: **VM** — ephemeral-mysql productization CLOSED, codification row rewritten as VM-hardening | all committed `66f88cbea` + daemon waves; `#check-release-scripts` green after the tag deletion (86 assertions) |
| A7 | **M07 — both red CI legs root-caused and fixed** — (1) benchkit/actionlint+shellcheck leg: GH runners set `CI=true`, and all three load-gate scripts treat CI as informational → every self-test failure leg flipped green on the runner (12/12 local vs red remote EXPLAINED). Fixed with per-leg `CI=false` isolation + explicit `SELFTEST_CI_LEG=1` opt-in for the CI leg; verified 86/86 under `CI=true`. (2) coverage-gate job was **vacuous fiction**: the root module has NO packages, so `go list ./...` iterated an empty set — the 80% loop never ran anything; the leg only ever exercised a failing toolchain download. Rebuilt: plain `setup-go` (`go-version-file: go.mod`, mirroring go-work-sync/modsums) + a REAL core-tier floor (event/command/query/metadata/scheduling, measured 91.0/92.3/90.1/94.4/96.0 today) | `CI=true nix run .#check-release-scripts` → rc 0, 86 passes, 0 assertion failures; actionlint clean |
| A8 | **M09 — README truth gates onto the push leg** — `check-readme-links` + `check-readme-deprecated` (self-test-then-gate) added to the `lint-scripts` job in ci.yml; nightly stays as backstop; GracefulClose select-race gotcha appended to gotchas-testing.md (fix verified at `system/system.go:322`); QEMU-33070/slirp gotcha verified already recorded; both TODO rows struck with evidence | gates green (rc 0), actionlint clean |
| A9 | **Concurrent-session integration (no duplication)** — M06/T18b (load-sweep PASS + go1.27.1 baseline re-pin + noise-gate `--force-save` hardening), M21 (queue M4: deadlock backoff+jitter, pool options, `testutil/mysqltestcontainer` = module #96, PG `-race -count=2`, T20 PapDashboard ADOPT verdict, dep-validation memo), M24 (ADR-0144 DeferClose sweep, `SortPaginate`/`ScanScoredVector` consolidation, ADR-0145 retry idioms) — all read, verified against gates, TODO rows closed/repointed, NOT re-executed | reports `docs/status/2026-09-20_22-01/22-02`, `2026-09-20_dogfooding-followups-execution.md` |
| A10 | **Repo-count sync 95→96** — AGENTS (×2), ROADMAP Unreleased cell, module-map census line; dogfooding reports' broken `../../adr/` links repointed to `../adr/` | `check-canonical-facts` 0 drift; `check-doc-links` 0 broken |
| A10b | **Pushed** — `547be142a..26d087686` (+ follow-on daemon waves); one authored commit `66f88cbea` (W3 rulings) survived the daemon | `git push` output |

## b) PARTIALLY DONE

1. **M10 VM hardening (F51 core landed, leg run + ceremony missing)** —
   `vm-mysql.sh` now has the stale-port pre-flight (`ss` with `/dev/tcp`
   fallback, orphan-QEMU diagnosis + remedy message) and the process-group
   trap (`set -m` + group TERM→KILL, closing the crashed-driver-orphans-QEMU
   class). Syntax + shellcheck clean; positive + negative probe tests passed
   (planted listener on a scratch port detected). NOT done: an actual
   `#integration-mysql-vm` run through the new script (heavy leg), the F52
   AGENTS integration-rows update, and the TODO row's evidence line.
2. **Wave-end TODO bookkeeping** — M02/M03-M05/M09/W3 rows struck with
   evidence; M07/M10 rows still open in TODO_LIST (M07's disposition text and
   M10's evidence not yet written by me).
3. **Composed `#verify` with the new guards** — the new verify-head guards
   (load, go-version, flock) have fixture-level evidence only; one real
   composed run has not been recorded this session (load was 56-75 during the
   window I checked).

## c) NOT STARTED (from the plan, in plan order)

- **M08** — CI retry-once, Module-Isolation leg-set diff, nightly 04:12 triage, TagContent clean-confirm.
- **M11/M12** — upstream filings (owner APPROVED all three in this session's W3 batch: exhaustruct_v5 panic, go/types+x/tools race, turso-go native-lib family). Nothing filed yet.
- **M13** — FEATURES maturity census (95/96 modules rowed, last-verified stamps).
- **M14/M15** — TODO 634b (READMEs into doc-check) and 634d (quick-start drift guards).
- **M16** — docs-truth bundle: recipes §2.11 sync, the 3 ambiguous-alias advisories (seen live in this session's doc-check output: core.md/recipes.md:123/faq.md:233), per-file archived-waves index, the 11-live-reports NOTE.
- **M17/M18** — goal-shaped-app adoption + demos.
- **M19/M20** — FilterOp `contains`/`prefix`; lease/AggregateOn/Scan-default one-pagers. ⚠ M19 touches metaengine while a pin/tag wave is in flight (`metaengine@v4.14.0` references `record.DeferClose`; pinned `record@v4.5.1` lacks it — noted by the queue session).
- **M22/M23** — benchkit polish tail (Min column, `--strict`, list-phases map, CSV/CoV, RunSuite variant); md-go-validator gate + P2 skips.
- **M25/M26/M27** — temporal property/soak/bigtable decisions; watermill NATS leg (note: `scripts/ephemeral-nats.sh` already exists); the ~14-item polish wave.

## d) TOTALLY FUCKED UP

1. **Pinned the first hash golden OVER the live corruption** — incident #11 was sitting in HEAD and I ran `--update` before running the shape gates; only the follow-up `check-formatters`/`restore-depguard` run exposed it. The `--update` health precheck exists BECAUSE of this, but v1 of the gate shipped with the pin-the-corruption hole open. The gate's own docs now say "pin after eyeballing" — the code should have enforced that from the start.
2. **Mutation test #1 was a no-op and I nearly reported the gate as broken** — my sed pattern didn't match anything (wrong indentation guess), `diff -q` was silent, and the "gate passed" output was the gate correctly hashing an UNCHANGED file. One extra assertion (mangle must produce a diff) would have saved the false alarm.
3. **verify-lock v1 polluted the tree** — wrote `.verify.lock` into the repo root, which broke the tag-release smoke "tree fully restored" assertion (fixtures have no .gitignore). v2 moved the lock to a machine-level path — which then exposed the batch→tag `exec` re-acquire self-refusal (flock is per open-file-description). Two design mistakes in one small feature; both fixed, both now fixture-covered, but the smoke suite caught them — not me.
4. **`SELFTEST_CI_LEG=1 CI=true` without `export`** — assignments without a command don't export, so the calibration CI leg ran unexported and failed; took one debug cycle.
5. **Edit-tool races with the auto-commit daemon** — one pre-commit edit (canary-leg `CI=false`) silently failed to land on the first attempt ("file modified since read" class); re-applied via sed. Same daemon absorbed FOUR of my waves into `chore:` commits — only `66f88cbea` carries an authored message. I verified and committed too slowly, repeatedly.
6. **Ran `templ generate` from the repo root once** — the exact cwd trap gotchas-tooling-build warns about; caught it via `git status` (zero diff) before any damage, regenerated from `catalog/docserver`.
7. **Instrumented test copy broke its own harness** — my `/tmp` copy of test-batch-release.sh resolved `$SCRIPT` to `/tmp/batch-release.sh` (missing), producing a garbage failure I had to debug past before seeing the real one.
8. **Left a planted `http.server 33071` listener running** when interrupted mid-mutation-test; killed it at report time (verified `pgrep` empty). Small, but it is exactly the leaked-process class the gotchas warn about.

## e) WHAT WE SHOULD IMPROVE

1. **Commit within seconds of verification** — the daemon converts uncommitted verified work into `chore:` history; authored messages are the audit trail.
2. **Health-check before pinning ANY golden** — every `--pin/--save/--update` path in this repo should refuse on known-bad inputs; the benchmark-gate got the same treatment (noise-gate refuse) from another session the same day. Generalize the pattern.
3. **Mutation tests must assert the mangle landed** (diff/grep before running the gate) — a no-op mangle validates nothing and can falsely exonerate the gate.
4. **New gates need a `CI=true` leg from day one** — every script with a CI pass-through will behave differently on runners; self-tests must pin BOTH modes (this session found the class in three scripts at once).
5. **Read the smoke suite before wiring new behavior into its target scripts** — the tree-restore assertion and the audit delegation both encode contracts I violated before reading them.
6. **`go list`-based CI loops need a zero-packages guard** — an empty iteration is fiction, not a pass; the coverage-gate job survived weeks as fiction.
7. **FK: the pre-commit lint-config trigger fired for every `.golangci.yml`-staged commit but the corruption still landed via auto-commit** — worth checking whether the daemon commits with `--no-verify` (candidate root cause for the whole incident class; not investigated this session).

## f) NEXT TASKS (sorted by impact)

1. ★ Record M07 leg dispositions + M10 evidence in TODO_LIST (both rows still open).
2. ★ Run one real `#integration-mysql-vm` leg through the hardened `vm-mysql.sh` (proves pre-flight + group-kill under fire); then F52 (AGENTS integration rows).
3. **M11 exhaustruct_v5 `skippedNamed` panic filing** — repro fresh on latest v5, draft in github-voice, file (owner approved).
4. **M11 go/types+x/tools race filing** — fresh repro, confirm not-yet-fixed, file.
5. **M12 turso-go native-lib family filing** — two repros (hash-mismatch, lazy-init), pin-bump probe, file + bump-or-hold decision.
6. **M08 CI green 2** — retry-once wrapper, Module-Isolation leg-set diff across last 5 runs, nightly 04:12 log triage, `TestTagContentMatchesChangelog` clean-run confirm.
7. **M16 docs-truth bundle** — the 3 alias advisories (import-scope the blocks), recipes §2.11 live-latency symbol-diff, archived-waves index, docs/status >10-reports note.
8. **M13 FEATURES census** — all 96 modules rowed + last-verified stamps + script-derived counts (F69) so the matrix can't rot.
9. **M15 quick-start drift guards** — 5 fences → compile tests (getting-started pattern).
10. **M14 READMEs into doc-check** — multi-README spike → flake app → nightly leg → mutation test.
11. **M17 goal-shaped-app pin bump to system v4.8.0** + `DomainConfig.Events` + `.On` (then M18 demos).
12. **M19 FilterContains/FilterPrefix** — coordinate with the in-flight metaengine/record pin-tag wave FIRST (M19 grows metaengine's API surface mid-flight).
13. **M20 one-pagers** — lease story, `AggregateOn` seam, Scan-default v5 survey (pure docs, no flight risk — do while M19 waits).
14. **M22 benchkit polish tail** — Min column, `--strict` NOISY fail, list-phases map, CSV/CoV columns, `RunSuite` variant, stale-baseline protocol.
15. **M23 md-go-validator gate** — commit baseline + flake app + CI leg + the 9 P2 skips.
16. **M25 temporal tails** — memory version-chain property tests, sqlite restart soak, bigtable MapUpdateAt/MaxAge decisions.
17. **M26 watermill NATS leg** — `ephemeral-nats.sh` exists; add the roundtrip leg + optional `#integration-nats` app + sibling-skill tail.
18. **M27 polish wave** — ~14 XS items (smoke-all resume/timing, `--from-manifest`, TagContent train threshold, exclusion-map unification, new-module scaffold, verification-ladder doc, …).
19. **Verify the 2 fixed CI legs on the remote** (M07/M09 changed ci.yml; the next push's run should show actionlint+shellcheck, README gates, and coverage-gate green) — F40's watch step.
20. **Composed `#verify` re-record** through the new guard chain in the next quiet window (`preflight-composed.sh && can-run-composed-gate --wait-loop && #verify`) — S-rule: every wave ends composed-green.
21. **Investigate whether the auto-commit daemon bypasses pre-commit** (`--no-verify` or hooksPath absent in its env) — candidate root cause for the entire config-corruption class (e §7).
22. **Add "assert the mangle landed" to the two existing mutation fixtures** (check-golangci-hash self-test, restore-depguard self-test) per e3.
23. **Check-canonical-facts extension**: derive the module count into FEATURES too (M13 F69 overlap).
24. **Add a `CI=true` self-test leg to every gate script lacking one** (calibration/can-run/golangci-hash/go-version/load-guard now covered; audit the rest).
25. **vm-mysql-nspawn.sh**: consider the same stale-port pre-flight + trap (nspawn has no QEMU orphan class, but the port check is cheap insurance).
26. **Archive the finished 2026-09-20/21 status reports** (docs/status holds >10 live — gate NOTE) once the concurrent sessions settle.
27. **`benchmark-regression.sh --save` + hash-golden symmetry**: consider a provenance precheck pattern audit across `--save`-class tools (post-D1 hardening, one more sweep).
28. **README push leg watch**: confirm the two new steps don't push lint-scripts past its timeout on runners.
29. **M16 alias advisories**: import-scope core.md:448, recipes.md:123, faq.md:233 → golden 0 advisories.
30. **api-stability golden**: confirm `WithMaxOpenConns`/`WithMaxIdleConns` (queue session's exports) made it into the golden + `#check-api-stability` green (they reported it; spot-verify).
31. **Nightly `Go version contract` step**: verify it passes on the runner (nix go = 1.27.1 locally; runner nixpkgs may differ — the gate is DESIGNED to fail loud there if so; watch first run).
32. **Todo tool ledger**: my in-session todo list has M07/M09/M10 completed but TODO_LIST rows for M07/M10 open — close the loop (task 1) before the next docs-health pass.
33. **M19 prerequisite**: pin-sweep `--check --remote` state snapshot before touching metaengine.
34. **Goal-shaped T20 follow-up**: the PapDashboard ADOPT verdict needs its pending queue-family tag wave — tracked in the queue section; keep un-tagged work out of v4 promises.
35. **Flywheel: preflight-composed phases list** should grow `check-turso-version` + `check-error-taxonomy` once runtime stays <5min each (they're in #verify).
36. **scripts/lib/verify-lock.sh consumers audit**: any OTHER long-window scripts (smoke-all, load-sweep chains) that should take the advisory lock.
37. **Coverage-gate core set**: metadata/scheduling joined the floor; consider Tier-2 (schema, snapshot, projection) after two green weeks.
38. **Document the lock in AGENTS** (verify windows section) — gotcha line exists for the recipe, not for the lock's semantics.
39. **`can-run-composed-gate --wait-loop` + flock interplay**: the wait-loop's retry loop holds no lock; verify the composed recipe acquires AFTER GREEN (it does — verify head), document it.
40. **Renew the W3 consolidation sheet** (`docs/status/2026-09-20_11-36_owner-bundle-w3.md` is stale-pending; all six now answered — it can be archived with a RESOLVED banner at the next docs pass).
41. **Dgraph constants re-anchor campaign** (row 306d, pending quiet window) — candidate for `quiet-window-run.sh` (promoted by the concurrent session).
42. **SearchQuery count=5 quiet re-run** (row 306c remainder) — same vehicle.
43. **Sweep other CI jobs for the empty-`go list` fiction class** (modsums/go-work-sync verified real; audit the remaining plain-go jobs).
44. **Add `check-go-version` to `verify-parallel.sh`/`verify-ci` heads** for parity with `#verify`.
45. **scripts/README or scripts/INDEX.md** — 60+ scripts, several new this week; a one-line-per-script index would cut discovery cost (candidate for M27).
46. **Tag the guard wave** when the queue-family tag wave lands (batch-release bar) — the new scripts are unreleased tooling; no consumer tag needed, but the CHANGELOG [Unreleased] should cite them (check-changelog-symbols will demand real symbols — cite the flake apps).
47. **CHANGELOG [Unreleased] entry for the guard wave** (hash-golden tripwire, wait-loop, preflight, load guard, go-version gate, flock, coverage floor, README push gates) — not yet written by me.
48. **Retire `/tmp` evidence**: this session's decisive outputs (crs2.log, ccf.log, preflight logs) are ephemeral; copy excerpts into the next status report or docs/benchmarks as needed.
49. **verify-load-guard ceiling policy**: default 10 matches wait-for-quiet; calibration uses 5 — document the two-tier intent (verify vs bench) in gowork-modes.
50. **After M01–M26 close**: run the full gates + composed verify + write the wave-close status report + push — the S03 end-of-wave rule.

## g) QUESTIONS I CANNOT ANSWER MYSELF

Tried: (1) read the queue session's pin-skew note (metaengine v4.14.0 vs record v4.5.1) — cannot determine from the tree whether the wave is mid-tag RIGHT NOW or already landed; (2) checked `gh` availability for remote CI evidence — did not attempt authenticated API calls without a ruling; (3) filing targets need repo confirmations I can find myself, but process preferences I cannot.

**Q1 (M19 timing):** The plan's FilterContains/FilterPrefix work grows metaengine's exported surface while the queue-family tag wave and the metaengine/record pin-skew are in flight. Push M19 through NOW (same-edit golden regen, rebase risk on the wave), or hold M19 until the wave's tags land and execute M20/M13–M18/M22–M27 first?

**Q2 (M08 remote evidence):** Am I cleared to use `gh` against this repo's Actions (read the 04:12 nightly logs + re-run/retry failed legs, which consumes Actions minutes), or is remote CI interaction still billing-gated like the triage row says — in which case M08 shrinks to local-fix-only (retry-once wrapper + leg-set diff from artifacts I already have)?

**Q3 (M11/M12 filings process):** The three upstream issues are approved — file direct from the verified repros in your voice, or draft all three and show you before anything goes public? (And confirm filing identity = your `gh` auth, which the filings skill assumes.)

---

*Session artifacts (ephemeral /tmp): `/tmp/crs.log`, `/tmp/crs2.log` (release-scripts 86/86 normal + CI=true), `/tmp/ccf.log` (canonical-facts), `/tmp/preflight-*.log` (first live preflight incl. the templ catch), `/tmp/tbr.sh` (instrumented batch-release test copy). Key commits: `66f88cbea` (W3 rulings, authored), guard-wave daemon commits `a39fa95b3`/`d3a241739`/`69b4b1ebd`/`f143871cf`/`58fefca6d`/`80eaa7f9c`/`26d087686`; incident-#11 repair rode `58fefca6d`-class daemon waves. Foreign in-flight files left untouched: `TODO_LIST.md` (2026-09-21 midday bench rulings), `docs/status/2026-09-21_12-38_t18b-promotion-gate-fix-proven-matview-lottery.md`.*
