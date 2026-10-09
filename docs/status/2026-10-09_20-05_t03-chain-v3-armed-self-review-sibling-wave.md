# T03 Status — Chain v3 Armed on Sibling Wave; Self-Review; Two Robustness Mistakes Repeated

Session: 2026-10-09 14:53 → 20:05 (continuation of the halted 01:58–06:15 session; user issued the standing continue directive).
Task: T03/f045 — one GREEN composed `#verify` (v5-GOAL Pareto plan).
Author: resuming assistant session. Report written 20:05:39; then HALTED awaiting instructions.

## Context Recap (what this session inherited)

The prior session ended HALTED with 3 pending questions: (1) authorize mechanical systemscenario
repair? (2) verify scheduling? (3) T04/T05 priority? The user's continue directive resolved these
via plan-policy defaults: do the mechanical repair, relaunch the chain, fire T04/T05 after green.
**Evidence gathered this session made question (1) moot** — the sibling's 14:52:55 plan
(`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`) owns systemscenario
end-to-end (tag wave system v4.12.0 + systemscenario v4.0.0, "await symmetry, timeout errors,
quiet window" hardening = the 600s-hang fix, deriver bus-deadlock option). I touched nothing there.

## a) FULLY DONE (this session)

1. **Recon.** Load 19.69/20.28 at start; siblings active (last commit 40 s before recon). `/tmp`
   had been wiped: wrapper v2 + `/tmp/verify-t07f.log` + `/tmp/mysql-shuffle-seeds.txt` all gone.
   Tree clean; major sibling waves landed since halt: `core/v5` module (wave A, 8 Tier 0/1 trains),
   `--lockstep` release mode, BDD-harness adoption plan.
2. **Run-#3 blocker re-verification (all three).**
   - #1 untidy go.mod/go.sum: **FIXED by sibling** — `go mod tidy -diff` clean in systemscenario
     (read-only check, rc=0).
   - #2 api-golden drift on `systemscenario/method Await`: **FIXED by sibling** —
     `docs/api_surface.txt:8097` contains the method; standalone meta-test run
     (`TestEveryModuleGoSumIsTidy|TestAPISurface`) green in 2.965 s.
   - #3 600s test hang: **subsumed** — systemscenario no longer _compiles_ GOWORK=off:
     uses `system.Clock`/`WithClock`/4-arg `system.New` (seam landed 06:13, commit a2aec239f,
     post-v4.11.0) while go.mod pins `system/v4 v4.11.0` from the proxy with **no replace block**.
     Workspace-mode builds resolve local; verify's GOWORK=off phases cannot. Sibling's tag wave
     is the designed cure ("companions drop their pre-tag local replaces" — theirs to land).
3. **`--wait` parity gap closed** (`scripts/can-run-composed-gate.sh:257`): now passes
   `--max-load5 "$CEILING"` explicitly. Root cause: `wait-for-quiet.sh --max-load` silently
   defaults `MAX_LOAD5 = MAX_LOAD * 3 / 2` (line 39) — the same 1.5× load5 slack I removed from
   `check_load()` last session survived in the `--wait` path. Header §3 comment extended.
   Self-tests: can-run-composed-gate 6/6, wait-for-quiet 4/4.
4. **Gate registrations verified** for the sibling's new module: `core/v5` present in
   check-module-layers.sh (LAYER=5, DEP_BUDGET=6 with rationale), api golden (752 entries),
   cqrs-lint catalog (`module_catalog_data.go:298`), flake testModules (line 303; systemscenario
   line 306). Sibling followed the AGENTS.md procedure; no registration debt to surface.
5. **Gotcha documented** (`docs/agents/gotchas-testing.md`): the auto-commit daemon can absorb
   go.mod/go.sum MID-tidy; cure = verify each module with `go list ./...` after multi-module
   tidy; read-only drift checks use `go mod tidy -diff`. (Learned last session, owed since.)
6. **Verify chain v3 written and armed** (`/tmp/verify-chain.sh`, job 2D3, log
   `/tmp/verify-t08.log`): NEW pre-arm probe phase — `GOWORK=off go build ./...` in
   `PROBE_MODS=(systemscenario)` every 300 s; only when green does it enter the battle-tested
   v2 logic (gate wait-loop → `nix run .#verify`; load-class refusals retry; NON-LOAD stops for
   inspection). Added a regression re-probe branch (if a probed module regresses mid-chain, fall
   back to probe-wait instead of burning attempts). 12 h wall-clock bound. Terminal-state watcher
   armed (job 2DE, 12 h bound). Probe correctly red and cycling since 14:57 (latest 20:02:45).
7. **Mid-session status delivered** at 19:45 on request.

## b) PARTIALLY DONE

- **T03/f045 (GREEN #verify):** chain armed and healthy, but blocked on the sibling's wave
  landing (system v4.12.0 tag / replaces). ~5 h in probe-wait so far; their session is visibly
  active — as of 19:51 they have _untracked_ `systemscenario/zz_deadlock_repro_test.go` and
  `docs/status/deriver-deadlock-stacks-2026-10-09.txt`: they are debugging a deriver/bus
  deadlock RIGHT NOW (their plan's "bus deadlock" item). Load 32.85/44.88/37.91 at 20:05.
- **Self-healing verify scheduling (prior Q2):** implemented via chain v3, but its 12 h deadline
  (set at 14:56, expires ~02:56) bounds probe + attempts TOGETHER; a sibling wave landing after
  ~02:56 kills the chain with PROBE WINDOW EXPIRED instead of waiting (see e/1).

## c) NOT STARTED (carried, quiet-window/verify-gated)

- f046 receipts (TODO_LIST dedup-tail row + composed-verify re-record row) — due only after GREEN.
- T04 legs: f047 mysql-VM suite (+F52 AGENTS rows), f048 snapshot-migration mysql integration,
  f049 shuffle-seed replay (see d/2 — the seed list is GONE), f050 G-T13 ADTSet VM leg, f051
  nspawn (still blocked on root presence), f052 queue/mysql conformance.
- T05 legs: f053 calibration PASS loop, f054 SearchQuery count=5, f055 DG_NetworkRTT re-anchor,
  f056 benchmark baseline re-pin, f057 supersede-note on the 2026-09-19 variation doc.
- Owed "insert-before-symbol tool" gotcha (second-session deferral — I still have not searched
  prior docs/status reports for its context).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Rebuilt the chain wrapper in `/tmp` AGAIN.** This morning's /tmp wipe destroyed wrapper v2
   and I _knew that_ — my first act was noting the loss — yet I wrote v3 to the same volatile
   location. If my session dies, both the chain (job 2D3) and watcher (job 2DE) die with it, and
   another /tmp wipe loses the wrapper + log. Same mistake twice in one day.
2. **Lost prep artifact `/tmp/mysql-shuffle-seeds.txt`** (21 mysql-labeled seeds for f049,
   extracted last session). The wipe deleted it; I never persisted it in the repo or docs. f049
   now needs re-extraction from the TODO_LIST ~821 pointers. Prep work that lives only in /tmp is
   prep work you get to do twice.
3. **Wasted a background test run**: fired the systemscenario suite in parallel with the
   meta-tests before compile-checking the module; it died in seconds on the build break. Reading
   go.mod first (30 s) predicted it. Minor, but sloppy sequencing.
4. **multiedit staleness friction**: my first edit of can-run-composed-gate.sh bounced on the
   mtime guard (file touched 14:41, content identical). Correct handling (re-view, re-apply) but
   the lesson is general: in a tree with 2–3 active sibling sessions, ALWAYS re-view immediately
   before editing, even files you edited hours ago.

Not fuckups but worth owning: wrapper v3 inherits v2's log-duplication quirk
(`tail -15 "$LOG" >>"$LOG"` after failures) and its load-class classifier treats ANY `timed out
after` as load-class — if the 600s hang survives the sibling's hardening and verify hits it, the
chain could retry a hang 12× instead of stopping (I noticed this during reflection, did not
touch the running script — editing a live bash script mid-execution corrupts it).

## e) WHAT WE SHOULD IMPROVE

1. **Split the chain's deadlines**: probe-wait (sibling waves take hours) deserves its own long
   bound; the verify-attempt envelope a separate one. Current single 12 h bound conflates them.
2. **Persist ops tooling**: chain wrapper should live somewhere durable (or be `setsid`/nohup
   detached) with its log under `/tmp` at most as a symlink target. Artifacts like seed lists
   belong in the repo (docs/status appendix or scripts/testdata).
3. **Classify bare test-timeouts separately** from load-guard refusals in the chain: verify
   load markers (REFUSED / LOAD GATE FAILED / refusing:) justify auto-retry; a 600 s hang does
   not — it should stop for inspection after ≤2 occurrences.
4. **Pre-verify fixture-rot check**: tag waves (their plan rides one: systemscenario v4.0.0 +
   system v4.12.0) re-rot the cqrs-lint fixtures (the 42-tag-train class). A 30 s
   `go mod tidy -diff` × 3 fixtures before launching verify would save 90 min burns.
5. **Chain log hygiene**: stop appending the log's own tail into itself; number attempts in the
   failure header.
6. **TODO_LIST visibility**: add a row "T03/f045 blocked on sibling tag wave (system v4.12.0)"
   — currently the blocker's ownership lives only in status reports.
7. **Cover the `--wait` path in the gate's self-test**: the 6 cases exercise `--wait-loop` and
   the assert paths; nothing pins the two-ceiling passthrough I just added (the wait-for-quiet
   self-test covers its own side only).
8. **Direct-artifact claims**: I confirmed blockers #1/#2 via the artifacts themselves
   (tidy -diff, golden grep) and used the 2.965 s meta-test as corroboration — keep that order;
   a suspiciously fast suite run is never the primary evidence.

## f) NEXT UP TO 50 (ordered; nothing executed until instructed)

Verify path:

1. Monitor chain 2D3 / watcher 2DE to terminal state (log: /tmp/verify-t08.log).
2. On PROBE WINDOW EXPIRED (~02:56 if wave still in flight): re-arm with fresh deadline.
3. When systemscenario compiles again: standalone `GOWORK=off go test -count=1 .` (hang check)
   BEFORE trusting verify to absorb it.
4. After their tag wave lands: `go mod tidy -diff` × cqrs-lint fixtures (scan/bus/typed) —
   preempt the release-train rot class.
5. On VERIFY GREEN: f046 receipts (TODO_LIST ~860 dedup-tail row + ~820 re-record row; cite
   preflight 9/9, dup baseline 186, rc/date).
6. Post-wave: re-run `nix run .#check-file-size` and refresh the SURFACED TODO row (their wave
   will add/grow offenders).
7. Post-wave: recipes/doc-check pass (their await-symmetry + deadlock APIs may invalidate
   recipe fences — the MetaEngineStore doc-lie class).
8. Post-wave: api golden stability spot-check (their same-edit contract; verify gates it anyway).

Chain improvements (next wrapper version, applied when chain is idle):
9. Split probe vs verify deadlines.
10. Fix log self-append duplication.
11. Reclassify bare `timed out after` as stop-for-inspection (≤2 retries).
12. Detach with setsid/nohup; persist wrapper outside /tmp.
13. Add pre-verify fixture tidy -diff probe.

T04 (quiet windows, after verify green):
14. f047 `nix run .#integration-mysql-vm` (6 legs; check orphan QEMU port 33070 before/after).
15. F52: AGENTS.md integration-rows evidence from the VM run.
16. f048 storage snapshot-migration mysql integration (build tag `integration`, MYSQL_TEST_DSN;
userspace MariaDB recipe port 33061; TODO ~1018).
17. f049 RE-EXTRACT the 21 mysql shuffle seeds (list lost — see d/2) from TODO ~821 pointers,
then replay via `go test -shuffle=<seed>`; persist the list in-repo this time.
18. f050 G-T13 ADTSet mysql-VM leg (TODO ~580).
19. f051 `#integration-mysql-nspawn` — still blocked on root; surface in receipts again.
20. f052 queue/mysql conformance half.

T05 (same gate-passing window where possible):
21. f053 calibration-gate PASS loop.
22. f054 SearchQuery count=5 (calibration table; recipe at t18b-record:89).
23. f055 re-anchor DG_NetworkRTT + dgraph constants in a gate-passing window.
24. f056 `./scripts/benchmark-regression.sh --save benchmarks/benchmark-baseline.txt` with
provenance header (calibration PASS first).
25. f057 supersede-note on docs/benchmarks/2026-09-19_backend-comparison-variation.md.

Housekeeping:
26. Research + write the twice-owed "insert-before-symbol tool" gotcha (search prior
docs/status reports for context).
27. TODO_LIST row for the sibling-wave blocker (e/6).
28. `--wait` self-test case (e/7).
29. Consider a docs/status note at every session boundary, halt or not (this report exists only
because you demanded it).

## g) QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **Sibling wave ETA / stand-down policy.** Their session is visibly mid-wave (untracked
   deadlock repro at 19:51, load 32/44). My chain waits on them until ~02:56 then expires. Is
   there a known ETA or priority ordering for the system v4.12.0 tag wave, and should I keep the
   chain armed overnight (re-arming on expiry) or stand down until they finish?
2. **Hang takeover boundary.** If their wave lands and the 600s hang SURVIVES their
   timeout/quiet-window hardening, do I take over diagnosis inside systemscenario (their
   territory, after their wave goes quiet) or keep surfacing it to you/owners? (Their active
   `zz_deadlock_repro_test.go` suggests they may already be chasing this exact class.)
3. **Shared-host etiquette for T04/T05.** May the heavy legs (mysql QEMU VM, nspawn, benchmark
   campaigns) run while sibling sessions are active on this 32-core host, or strict
   quiet-window-only? I cannot judge the other sessions' criticality; the gate ceiling (10)
   effectively serializes everyone, which can idle my queue for many hours.

## Live Handles

- Chain: background job 2D3 (`/tmp/verify-chain.sh`); watcher: job 2DE; log: `/tmp/verify-t08.log`.
- Gate self-tests both green (10/10) after the `--wait` parity fix; daemon will absorb the edits.
- Nothing else of mine is running; no verify in flight (chain is in probe phase only).
