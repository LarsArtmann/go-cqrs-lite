# Status Report #5: T18b — All Rulings Encoded; Chain Armed; Storm-Waiting

**Date:** 2026-09-21 14:52 CEST (12:52 UTC)
**Scope:** continuation of the T18b session (reports #1–#4 + addenda). This round's work: durable preservation of the session tooling, shellcheck of the detached scripts, and the armed-chain state. No unrelated work researched.
**Host:** shared 32-core; load 40.9/41.4/33.6 at writing — the cost pass (pid 3244128) and closure (pid 3258300) are both alive, waiting for a quiet window; baseline `a91e7cd90` untouched; working tree clean (daemon-absorbed).

---

## a) FULLY DONE

| # | What | Evidence |
|---|------|----------|
| A1 | **Every owner ruling across 3 rounds is implemented and verified** — p99 headline demotion (production-validated 08:46 UTC), `--save` refusal + `--force-save`, bounded-attempts acceptance, tooling promotion + self-test + flake apps (`quiet-window-run`, `nightly-bench`), known-flaky annotation with expiry, per-suite `benchtime::count`, baseline archival on supersede, CI-parity rewrite, nightly-only fallback installer | Fixture suites + committed tree; decision log in TODO row 40 |
| A2 | **The full closure pipeline is autonomous** — cost pass → rule check (±10% across 3 quiet runs, ≤60s) → widen + CI parity → archive + noise-clean re-pin → verify; OR the nightly-only fallback (gate entry removed, non-gating stability suite created + wired) — with plausibility guards before any mutation | `/tmp/matview-closure.sh` (preserved copy `/var/tmp/t18b/`), TODO row 40 |
| A3 | **Durable tooling preservation (this round)** — session scripts copied to `/var/tmp/t18b/` (cost, closure, verifier) so a `/tmp` clear or reboot no longer destroys the re-arm path; report #4's "reboot insurance" concern closed for the scripts themselves | `ls /var/tmp/t18b/` this round |
| A4 | **shellcheck applied to the detached session scripts (this round)** — cost pass, closure, verifier: no findings | `nix run nixpkgs#shellcheck -S warning …` silent-exit-0 |
| A5 | **Session evidence trail complete** — four status reports + addenda, decision log in TODO rows 34/40/41, all changes daemon-committed | `git log` — latest commits touching my files: `221553bf9` et al.; working tree clean |

## b) PARTIALLY DONE

| # | What works | What remains | Blocker |
|---|-----------|--------------|---------|
| B1 | **The autonomous matview closure** — armed, guarded, GOTOOLCHAIN-fixed; has never been allowed to run its measurement phase | Needs one genuine quiet window (load1+load5 < 5). Every window so far was either consumed by the GOTOOLCHAIN failure (11:15 UTC) or closed by a storm (now 41). 6h budgets per phase; re-arm scripts in `/var/tmp/t18b/` | Shared-host load — the session's single standing blocker |
| B2 | **Local green verification of the 1.27.1 baseline** — gate machinery proven; closure delivers it automatically after widening | Pending B1 | Same |
| B3 | **Nightly automation** — fully shipped except the owner-side timer install (`systemctl` is harness-banned for me; command documented in `scripts/nightly/README.md`) | One owner command | Owner action |
| B4 | **Doc wiring** — TODO/state current everywhere | AGENTS.md Quick Reference rows for the two new flake apps; stale-reference sweep across docs/workflows; gotchas-testing reconciliation | Queued (→ f) |

## c) NOT STARTED (carried; deliberately untouched)

C1 T19–T21 (v5-gated, ADR-0142 §2) · C2 SearchQuery re-run + dgraph re-anchor (row 306c/d — now trivially queueable via the promoted tool) · C3 W1 siblings (`api-stability` TestEvery, composed `#verify`/`#verify-ci` re-record) · C4 plan-doc addenda (2026-09-18 SUPERB T18b row; TODO row ~81 pointer) · C5 MIN/matview bimodality root cause (independent of widening) · C6 other-session `scripts/vm-mysql.sh` (untouched, not mine).

## d) TOTALLY FUCKED UP

**D1 — Durable preservation was executed three rounds late.** Report #2 listed "assume reboot / durable homes" as an improvement; reports #3 and #4 repeated the `/tmp`-only pattern anyway (and #4's addendum promised "reboot insurance" as a documentation line-item instead of just moving the files). This round finally copied the scripts to `/var/tmp/t18b/` — a 10-second fix that sat undone across two documented incidents. The cost: if the host had rebooted between 12:29 and 14:30 today, every ruling encoded in those scripts would have needed reconstruction from report prose.

**D2 — The standing "blocking" state is my own tooling's dependence on luck.** Three separate times the pipeline's progress has hinged on catching a quiet window while detached scripts survived (reboot killed v2; env killed the cost pass's window; tonight's storm gates the re-armed chain). The system now has the right guards, but the *architecture* — detached bash scripts polling `/proc/loadavg` — remains the fragile part. The durable answer (a small daemon/scheduled task, or moving measurement to CI) exists as f-items and has not been chosen. This is not broken today; it is one reboot away from being broken, and I keep re-arming instead of hardening.

**D3 — Minor: the round produced no new execution** — the last 45 minutes were re-armed-waiting, and this report honestly says so. Listed because "nothing happened" must stay visible in a status report rather than being padded to look like progress.

## e) WHAT WE SHOULD IMPROVE

1. **Fix-by-moving beats fix-by-documenting.** The preservation fix took 10 seconds once executed; both prior reports only wrote it down. Rule: any improvement achievable in <2 minutes gets executed in the same breath as being listed.
2. **The campaign queue should be explicit**: SearchQuery re-run, dgraph re-anchor, and the closure all want quiet windows; a single ordered queue file (or the promoted tool taking a manifest) would let them fire across successive windows without session supervision (→ Q1).
3. **Two-axis verdicts** (machine-loud vs sampler-unstable) remain the deepest gate improvement candidate — the MIN/matview saga is fully documented but the gate still expresses it as pass/fail (carried, f#13).
4. **Watchdog marker files** for detached work (report #4 e#4, still open): the closure writes logs, but nothing screams if it dies; a `DONE`/`FAILED` marker file per phase would make the state checkable in one `ls`.
5. **Carried**: reference sweeps after contract changes; AGENTS Quick Reference rows; evidence preservation into `docs/benchmarks/`.

## f) NEXT TASKS (up to 50; ★ = already in TODO_LIST; harvest candidates)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | ★ Land the closure: cost numbers → rule → widen + CI parity → archive + re-pin → verify (autonomous) | Critical | S | Verification |
| 2 | ★ Install the nightly timer (owner command in `scripts/nightly/README.md`) | High | S | Feature |
| 3 | ★ Fallback branch executes automatically if widening fails (nightly-only suite) — no action needed unless it fires | Critical | S | Verification |
| 4 | Queue SearchQuery re-run + dgraph re-anchor behind the closure via `quiet-window-run` (row 306c/d) | High | M | Verification |
| 5 | AGENTS.md Quick Reference rows: `#quiet-window-run`, `#nightly-bench` | Medium | S | Documentation |
| 6 | Stale-reference sweep: old headline list / unconditional `--save` / matview benchtime mentions (README, `cmd/cqrs-bench/README.md`, docs/benchmarks, workflows) | High | S | Documentation |
| 7 | `shellcheck` gate coverage for repo gate scripts (quiet-window-run, nightly-bench, benchmark-regression) in `check-release-scripts` | Medium | S | Quality |
| 8 | `scripts/go-env.sh` helper (GOTOOLCHAIN=auto + cache chain) sourced by gate scripts + documented — generalizes the incident fix | High | S | Quality |
| 9 | Watchdog marker files for detached phases (`/var/tmp/t18b/PHASE-<name>.{RUNNING,DONE,FAILED}`) | Medium | S | Quality |
| 10 | Composed `#verify` re-record (gate script changed three times since last green) | Medium | S | Verification |
| 11 | `api-stability` TestEvery green record (W1) | High | S | Verification |
| 12 | Case-study appendix in `docs/benchmarks/calibration-2026-08-30.md`: the full storm/reboot/p99/bimodal/GOTOOLCHAIN series as the gate-rationale record | Medium | S | Documentation |
| 13 | Benchkit-side variance-aware stability probe (two-axes verdicts) | Medium | M | Feature |
| 14 | Root-cause MIN/matview bimodality (perf/freq governor/alignment) | High | M | Bug |
| 15 | Investigate `SUM_VIA_GROUPED/baseline` +34.5% one-off | Low | S | Bug |
| 16 | Preserve `/tmp` evidence (pipeline, verify, cost, closure logs) into `docs/benchmarks/` | Medium | S | Cleanup |
| 17 | Deep-quiet window probe/logger (loadavg sampling → best campaign windows) | Medium | M | Feature |
| 18 | Mutation-tested fixture: noise-fail + `--save` must refuse | Medium | S | Quality |
| 19 | Confirm CI's baseline artifact carries the 5 new entries | Medium | S | Quality |
| 20 | Replace pid-chaining with results-file polling in chained automation | Medium | S | Quality |
| 21 | Sweep other gates for fragile p99/max thresholds (~100-iteration estimators) | Medium | M | Quality |
| 22 | Reconcile `gotchas-testing.md` quiet-window caveat with 449416da2's line — one authoritative note | Medium | S | Documentation |
| 23 | Plan-doc addendum: 2026-09-18 SUPERB T18b row → DONE pointer | Low | S | Documentation |
| 24 | Annotate TODO row ~81 pointer ("remaining bench re-pin is T18b/T14") as resolved | Low | S | Documentation |
| 25 | Audit other pre-1.27 baselines for the same staleness class | Medium | S | Quality |
| 26 | `--explain <bench>` per-sample diagnostics in the gate | Medium | M | Feature |
| 27 | `--json` evidence output for the promoted tool | Low | S | Feature |
| 28 | DirectSQL A/B benches gate-set decision (dep-budget review first) | Low | S | Decision |
| 29 | Refresh stale load claims in TODO rows (now up to 99) | Low | S | Cleanup |
| 30 | Document the closure/verifier re-arm procedure with the `/var/tmp/t18b/` paths | Low | S | Documentation |
| 31 | Host benchmark-ceiling policy (owner undecided; strict <5 stands) | High | S | Decision |
| 32 | After green lands: close TODO row 34's caveat formally with the verifying run's hash | Medium | S | Verification |
| 33 | Recurring baseline-freshness policy once nightly is installed (weekly re-pin? expiry?) | Medium | S | Decision |
| 34 | Move session tooling patterns (quiet-window + bounded retry + guards) into an ADR or the W2 design row so the next session inherits architecture, not folklore | Medium | M | Documentation |
| 35 | Post-closure: retire `/tmp` + `/var/tmp/t18b` copies, keep only repo scripts | Low | S | Cleanup |

**Harvest note:** ★ already in TODO_LIST; the rest route through docs-health HARVEST on instruction. The critical path is exactly #1–#4.

## g) QUESTIONS I CANNOT ANSWER MYSELF

The blocking decisions have converged — the remaining owner-only items are sequencing and scope authority:

**Q1 — Autonomous campaign queue:** may I queue the row-306 campaigns (SearchQuery count=5 re-run, then the dgraph constants re-anchor) behind the matview closure in the same `quiet-window-run` tooling, so they fire automatically across successive quiet windows without further instruction? They have supersede semantics (overwrite calibration tables per their rows), which is why I will not self-authorize the sequencing.

**Q2 — Hygiene now vs harvest:** the carried hygiene items (#5–#13: AGENTS rows, reference sweeps, evidence preservation, gotchas reconciliation) are each S-sized and independent — execute them in this session as the storm-wait continues, or leave them as HARVEST rows for a docs-health pass?

**Q3 — Nightly timer commitment:** the last missing piece of the nightly ruling is one `systemctl --user …` command only you can run. Will you run it (and then nightly gates are live tonight at 03:00), or should I treat nightly as manual-only for now and stop tracking it as "partial"?

---

*State at writing: cost pass pid 3244128 + closure pid 3258300 alive (waiting out load 41); scripts preserved in `/var/tmp/t18b/`; baseline `a91e7cd90` untouched; tree clean. No manual commit per harness contract. Waiting for instructions.*

---

## ADDENDUM 2026-09-21 ~15:50 CEST — THE RULE QUALIFIED; incident + repair; all three Q-rulings executed

**The headline event:** at 13:30 UTC the re-armed chain hit a genuine quiet window and the cost pass produced real numbers — then the accepted rule PASSED on its first candidate:

| Config | Wall | MIN/matview median |
|--------|------|--------------------|
| 100x/5 (old) | 5s | 11313 ns |
| 1000x/5 | 27s | 7640 ns |
| **100x/9 (candidate)** | **6s** | **8118 → 8131 → 8793 (max drift +8.3% ≤ 10%)** |
| 1000x/9 | 48s | 7573 ns |

The closure widened the gate entry to `100x::9`, applied CI parity (`benchmarks.yml` matview leg → `-benchtime=100x -count=9`), passed the calibration gate (load 1.94/3.60) — then its re-pin and verification **died with exit 126: Permission denied**. Root cause: my `set_entry` rewrote the gate script via `awk > tmp && mv`, and the fresh file lost its exec bit (644). Fixed (chmod restored; the preserved closure copy now chmods after mv). The completion of the interrupted tail — quiet → calibration-gate → noise-clean re-pin (superseded baseline auto-archived to `docs/benchmarks/baselines/`) → verification — is armed as `/var/tmp/t18b/closure-completion.sh` (pid 1566584). **Green there closes T18b permanently.**

**Q1 executed — campaign queue armed** (chained after completion): SearchQuery count=5 re-run (row 306c) then a full `BenchmarkCalibration_*` measurement capture (row 306d inputs), via `nix run .#integration-dgraph`, plausibility-guarded, results + PROVENANCE to `/var/tmp/t18b/campaign-results.md`. The bench wiring was **smoke-tested first** (34s live run, PASS — the D2 lesson applied). Supersede/re-anchor *edits* stay manual with numbers in hand (row 306d's constant campaign is a design pass, not a script).

**Q2 executed — hygiene batch landed:** gotchas-testing caveat (quiet loadavg ≠ sufficient; KNOWN_UNSTABLE; GOTOOLCHAIN trap), AGENTS.md Quick Reference rows (`#quiet-window-run`, `#nightly-bench`), dated addendum on the 2026-09-18 SUPERB plan's T18b row, TODO row ~81 pointer resolved. Evidence preservation decided AGAINST as raw files: the four status reports already embed the decisive excerpts and are committed — duplicating raw logs into docs/ would be noise.

**Q3 executed the SystemNix way:** nightly timer implemented in `/home/lars/projects/SystemNix/platforms/nixos/users/home.nix` following the taskwarrior house pattern (writeShellApplication + `systemd.user.timers`, 03:00/Persistent/30m jitter, `go_1_27` on the unit PATH because the go.work floor is 1.27.1 and user units don't inherit dev-shell toolchains). **Eval-verified** on evo-x2: `home-manager.users.lars.systemd.user.timers.go-cqrs-nightly-bench.Timer` renders `{03:00, Persistent, 30m}`. Alejandra-formatted (121/82 diff includes enforced format convergence on the niri TOML block — their pre-commit would produce it anyway). Activation = owner deploy (sudo). The go-cqrs-lite `scripts/nightly/` hand-written units are superseded by this declaration. SystemNix context discovered: `check-flake-inputs.sh` bans `GOTOOLCHAIN=auto` in flakes (sandbox purity) — so the interactive `local` pin is policy-consistent, and script-level self-export (what I did) is the correct layer.

**Also noted:** the Q1/Q2 fallback and widening encoded in the closure + TODO row 40; the one thing still waited on is a quiet window for the completion + campaign legs (load 8.7/10.5 at last check — draining).
