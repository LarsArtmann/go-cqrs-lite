# T03/f045 — Chain v4 Armed (Durable, Preflight Wired); Sibling Wave Half-Landed; Fixture Rot Preempted

Session 2026-10-09 21:51 → ongoing. Continuation of the halted 20:05 session under the
re-issued standing directive; pending questions resolved via plan-policy defaults
(keep/re-arm chain overnight, take the hang only if the wave dies quietly, strict
quiet-window discipline for T04/T05).

## What this session did

1. **Chain survived, then was replaced.** v3 (PID 2994481, armed 14:57) outlived its
   session — its parent is a still-alive `crush` process (411495). Killed it (mvdan/sh
   has NO `kill` builtin — `kill` silently no-ops with rc=2; use `pkill`) and armed
   **v4** detached (PID 222957, `setsid`, PPID 1): script + logs in
   `~/.local/state/crush-t03/` (survives /tmp wipes; the /tmp script file was already
   gone while the process lived on its fd).
2. **v4 fixes all four inherited v3 flaws**: separate probe deadline (12h, expires
   ~09:55) vs verify envelope; `preflight-composed.sh` (9/9) wired between gate and
   verify (was a manual step — now the standing constraint is mechanical);
   timeout-class failures stop for inspection after 2 hits; per-attempt logs (no
   self-append). Plus a NEW fixture `tidy -diff` probe (exit 46 on drift).
3. **Sibling wave is HALF-LANDED**: tags 15:49–18:40 (stack v4.5.0, cqrs-lint v4.15.0,
   deriver v4.3.4, metaengine…) but **`system/v4.12.0` + `systemscenario/v4.0.0` were
   never tagged** — systemscenario still pins `system/v4 v4.11.0`, no replace →
   compile-red GOWORK=off (`undefined: system.Clock/WithClock`, 4-arg `system.New`).
   Their deadlock repro was added 20:06 and deleted 20:50; siblings actively
   linting/building again at 22:00. Chain probes every 300s, correctly red.
4. **Predicted fixture rot PREEMPTED**: their tag wave bumped `tidwall/gjson`
   v1.19.0→v1.20.0 in cqrs-lint `busfixture`+`typedfixture` go.sums → tidied,
   `go list`-verified, converged ×3 rounds (the planned burn: a 90-min verify leg).
5. **Housekeeping debts cleared**: (a) the thrice-owed **insert-before-symbol gotcha**
   landed in `docs/agents/gotchas-tooling-build.md` (origin: 10-08 21:03 §d — multiedit
   header-anchor clobbering ×2; mtime guard fired DURING the write, practicing the
   adjacent rule); (b) TODO STATE rows for the blocker (verify-re-record row) and the
   f049 seed recipe (`build/shuffle-seeds.log` is in-tree and durable — the lost /tmp
   list was derived data; 7 mysql-manual seeds); (c) **`--wait` two-ceiling
   delegation now self-test-pinned** (case 1c) — first mutation SURVIVED (rc-based
   test can't distinguish delegation-refusal from trailing-assert-refusal) and the
   mutant harness itself was broken (`$SCRIPT_DIR` unresolvable in /tmp copies —
   copy `wait-for-quiet.sh` alongside); final pin asserts the `host never quieted`
   message, mutation-verified red, real 7/7.
6. `wait-for-quiet.sh` self-test re-confirmed 4/4 after the passthrough knobs
   (`WAIT_TIMEOUT`/`WAIT_INTERVAL`, defaults identical — no behavior change).

## State at write time (22:08)

Load 42.51 (siblings' golangci-lint ×3 + nix build); chain v4 probing red on schedule;
tree clean (daemon absorbing my edits).

## Next

Monitor `~/.local/state/crush-t03/verify-chain.log` → on GREEN: f046 receipts
(preflight 9/9, dup baseline 186, rc/date) → T04 (f047–f052) / T05 (f053–f057) in
quiet windows. On PROBE WINDOW EXPIRED (~09:55): re-arm with a fresh deadline.
On FIXTURE DRIFT / PREFLIGHT RED / TIMEOUT-CLASS / NON-LOAD FAILURE: inspect, fix,
re-arm. Never edit the running script in place; edit the durable file only when idle.
