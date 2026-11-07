# Post-park CI stabilization: eight defect classes, one upstream filing, M05 at the wire

**Session window:** 2026-11-06 22:00 → 2026-11-07 02:01 CET (resumed from the 21:28 park point).
**Resume state:** plan M01–M26 executed; ONE owner question parked (SSE: instrument vs quarantine); M05 (CI-green receipt) blocked on an unknown CI verdict.
**Live state at writing (02:01):** run **42318787854** on `1b5e6639e`: **28 legs green, 2 running, 0 red** — the closest to full green all night. Verify-fast is among the remaining two; the SSE test verdict has still never been reached (see §d).

---

## a) Summary

The parked question ("instrument or quarantine the SSE flake?") remains **unanswered and unanswerable tonight** — every candidate run's Verify-fast leg died on a *different* defect or was cancelled before the SSE test could report. Instead of one quiet CI watch, the session became an eight-defect stabilization campaign: the tree at park-time was **un-buildable on CI** (templ 1100-era files vs pinned v0.3.1070 lib), and each fix peeled the onion one layer further. All eight classes are now fixed, locally verified, and — for MySQL (twice) and Dgraph (once) — **CI-verified green**. The final candidate run is 28-green/0-red with two legs in flight.

## b) What was done (chronological, one fix per defect class)

1. **CI triage of run 42288656759** (4 red legs): templ 1100-era codegen broke build/Flake/CSP legs; MySQL QEMU conformance flake (`claims.go:55: worker 5 claim: invalid connection`).
2. **Templ whipsaw round 3 → root-caused to BuildFlow itself.** Regenerated with the gate binary (17687ea56), then discovered the true engine while committing: BuildFlow's pre-commit `templ-generate` step resolves `templ` from PATH (floating v0.3.1114) and re-broke the files **during my own commit's hook**; the daemon fossilized it (660f5e64d, f79e37fcf). Durable fix: `skip_steps: [templ-generate]` in `.buildflow.yml` (77564de36) — the flake's `check-templ` gate (flake.lock-pinned binary) stays the sole codegen authority. Verified: the hook now leaves gate-regenerated files alone; CSP leg green ever since.
3. **BuildFlow#93 filed** (upstream close-the-loop): `templStaleCLISkipReason` (templ_generate.go:268 at 6c58af740) guards only the CLI-OLDER direction (`cli.lessThan(required)`); a NEWER CLI sails through and writes `*_templ.go` referencing helpers absent from the pinned lib. Source-verified, duplicate-searched (open+closed), voice-checked, filed with pinned-commit citations.
4. **Tidy repair:** the concurrent session's 94-file churn added 2030 spurious `/go.mod` hash lines → 82 modules untidy → CI tidy leg red + flake vendorHash mismatch. Repaired via per-module `GOWORK=off go mod tidy` under nix go1.27.3 (mass-tidy.sh); **verified byte-identical restoration** of the pre-churn go.sums (0 files differ vs 456a42bb4) so the flake hashes re-validated without re-pinning.
5. **MySQL QEMU dead-connection flake (5/5 runs, three distinct surfaces):**
   - surface 1: `queue/conformance pinFencing` workers → bounded retry (5× @25ms) on the dead-conn class (`driver.ErrBadConn` / "invalid connection"); safe because a dead conn kills the claim tx pre-commit (2d06f93ac).
   - surface 2: `metaengine/mysqlengine` engine **init** (server-side RST: `packets.go:58 read tcp ... connection reset by peer`) → all four test construction sites route through `newEngineWithRetry` (4× @250ms), preserving the OQ-10 fail-loudly policy (exhaustion fatals, never skips) (83c0d92e0).
   - surface 3: `adttest ConcurrentClaimersDisjoint` (pre-classified INFRA flake 2026-11-05) → same bounded retry in the claim worker (f99acdd53).
   - **Result: MySQL QEMU leg GREEN twice consecutively.** The class matches the documented slirp/port-forward infra limit (CHANGELOG 2026-10-17; gotchas-testing 2026-09-19).
6. **Dgraph ephemeral tx-abort flake (2 consecutive runs):** designed per the engine's own "caller retries" contract — `metaengine.TransientCommitError` + `metaengine.IsTransientCommit` (capability marker in core, `errors.AsType`-based), `dgraphengine.RunInTx` wraps contention-class commit failures (`transientCommitError`, compile-time interface proof), and the Store batch apply retries the whole `RunInTx` up to 4× with jittered backoff (`runInTxWithRetry`; closure is engine-only per the RunInTx contract). Package-surface baseline re-pinned (+2/+1 reviewed), api golden regen, CHANGELOG entries, unit tests for all three behaviors (marker traversal, non-transient surfacing, budget exhaustion). **Local ephemeral Dgraph suite GREEN rc=0 (98s); CI Dgraph leg GREEN** (1eacdc475).
7. **`check-test-runtime.sh` EPIPE crash:** the display-only `sort | head -10` pipeline died under `pipefail+errexit` (GNU sort exits 2 on EPIPE) **before the verdict block** — CI flake-check leg went red with **zero BREACH/NEW rows** (it would have passed). Fixed with `|| true` + explanatory comment; shfmt(3.14.1)-clean; self-test 6/6; the guard pattern mutation-tested (unguarded aborts, guarded survives) (cb7c407a0).
8. **erraudit finding on my own new code:** `IsTransientCommit` used `errors.As` → migrated to `errors.AsType[TransientCommitError]` per the go-error-modernization skill (safe type-matching class, not a sentinel); local `erraudit-sweep.sh` 103/103 modules clean (1b5e6639e).

Also: TODO_LIST rows kept current throughout (M05 row refreshed, MySQL row → FIXED with watch-notes, Dgraph row → FIXED, templ row addendum naming the BuildFlow root cause + #93); preflight gates run before every push (check-templ, tidy gate, changelog-symbols, lint-scope per changed module).

## c) Files & changes (all pushed; HEAD = 1b5e6639e)

| Commit | Content |
| --- | --- |
| 17687ea56 (daemon) | docserver `*_templ.go` regenerated with the gate binary (1070-era) |
| 77564de36 | `.buildflow.yml`: `skip_steps: [templ-generate]` + rationale |
| 8500f8118 (daemon) | 92 go.sums restored (2030 junk lines stripped) |
| 2d06f93ac (daemon) | `queue/conformance/claims.go`: pinFencing dead-conn retry |
| 83c0d92e0 | `metaengine/mysqlengine/helper_test.go` (+2 test files routed): `newEngineWithRetry` |
| f99acdd53 (daemon) | `metaengine/adttest/claim_conformance.go`: claim-worker retry; package-surface baseline re-pin (841→843/169→170) |
| 1eacdc475 | `metaengine/transient_commit.go`(+test), `runtime_backend.go` (retry call), `dgraphengine/transaction.go` (marker), CHANGELOG, api_surface.txt |
| cb7c407a0 (daemon) | `scripts/check-test-runtime.sh`: EPIPE guard |
| 1b5e6639e | `transient_commit.go`: `errors.AsType` migration |
| BuildFlow#93 | upstream issue (templ-generate newer-CLI skew guard gap) |

Authored-commit note: five of the nine content commits landed as daemon `chore:` absorbs despite same-window staging (§d.2).

## d) Self-critique — what I got wrong or could do better

1. **I destroyed my own evidence once.** The first QEMU VM run was piped through `| tail -6` — the diagnostic middle was gone; the rerun then got refused by quiet-window (load ~90). Rule I should follow: **every load-sensitive local run writes a full log to `build/` on the first attempt.**
2. **The daemon won every commit race.** Five content commits landed as chore-mush. The owner ruling covers authored commits at *phase boundaries*; CI-fix cadence (commit→push→watch, ~8 rounds) gave the daemon ~1-3 min windows and it took them. Not content-lossy (verified each time), but history readability suffered and I kept writing authored messages that never shipped. Improvement: stage+commit in one tool call (no gap), or accept chore-mush for hotfix-class commits and say so in the report.
3. **Two CI rounds were spent on findings about MY OWN new code** (package-surface ratchet +841→843; erraudit `errors.As`). The preflight ritual (TODO_LIST row 211) doesn't include `#check-package-surface` or `erraudit-sweep` because it was built for docs/tidy/templ drift. **Extension rowed below (f.3):** when a change adds exported API or new error-matching code, those two gates join the preflight.
4. **Whack-a-mole tax on the MySQL class.** After the second distinct surface (engine init RST) I should have swept ALL mysql-adjacent dial/construction sites in one pass; instead the adttest surface took another round (~45 min of CI). The class-wide grep is now obvious in hindsight: `mysqlengine.New(`, `ClaimDue(`, `sql.Open("mysql"`. Rowed as a checkable habit (f.4).
5. **I briefly treated a load-starved local VM failure as new signal.** The local `mysqlengine.init: invalid connection` under load≈90 was the same class, not a new fact — I should read `/proc/loadavg` BEFORE interpreting any load-sensitive local result, not after.
6. **The SSE question starved.** Six hours, seven runs, zero SSE verdicts: Verify-fast died on build (3×), package-surface (1×), then runner preemption/cancellation (3×). Each superseding push was individually justified (a real fix), but I never considered batching fixes into fewer pushes to give one Verify-fast a full window — with hindsight, fixes 5b+5c (two MySQL surfaces) and 6 (Dgraph) could have ridden ONE push. Cost: the owner's parked question has no new data. Mitigation rowed (f.1).
7. **Whitespace discipline slipped twice** (test-file alignment; the adttest multiedit fuzzy-match re-indent). Both caught by gofmt/vet before push — but the multiedit's "applied to whitespace-equivalent text" notice deserved an immediate re-view, not a later one.

## e) Risks & follow-ups

- **M05 receipt is minutes away but not in hand** — 2 legs in flight at writing. If either reds: triage, don't assume (tonight's lesson: every red had a distinct, findable root cause).
- **Retry shields mask an infra truth:** the VM MySQL's connection-reset behavior and Dgraph's abort storms are environmental; the shields make tests tolerant, they do not fix the environment. If the classes escalate (exhausted retries), the next level is engine-side `errorfamily` classification (see g.2).
- **`metaengine/mysqlengine` init path has no shield beyond tests** — production consumers dialing flaky MySQL get no retry; that's policy, not oversight (see g.2).
- **pgengine 66.7s flake-check anomaly** (one run) never recurred after the EPIPE fix landed — possibly the crash distorted measurement; watch only.
- **Concurrent-session territory respected:** `.golangci.yml` tug-of-war and the go.sum churn were left to their owner (content landed via daemon per norm); my edits stayed disjoint.

## f) Next things (ranked, not just rowed)

1. **Read run 42318787854's final verdict**; if green: tick the M04/M05 TODO row (CI-green receipt), addendum-index this report. If red: triage the leg with the same discipline (logs → class → fix → verify).
2. **SSE verdict harvest:** the first run whose Verify-fast COMPLETES answers whether `TestSSE_TaskViewStreamsOverHTTP` still fails on CI. If it fails on an otherwise-green run → the parked owner question becomes blocking; if green ×2 → recommend closing the recurrence row.
3. **Preflight ritual extension (row 211):** add "API-adding changes ⇒ `#check-package-surface` + `erraudit-sweep.sh` locally BEFORE push" (tonight cost: 2 CI rounds).
4. **Class-wide shield sweep habit (row):** after any dead-conn/timeout flake on surface #2, grep the sibling surfaces (`mysqlengine.New(`, `ClaimDue(`, `sql.Open`) in the SAME edit.
5. **EPIPE audit (row, XS):** sweep `scripts/*.sh` for other display-only `| head` pipelines under pipefail+errexit; guard or document each (the check-test-runtime one cost a CI round; siblings are latent).
6. **MySQL engine-side classification decision (see g.2).**
7. **BuildFlow#93 follow-up:** if the owner wants it fixed fleet-side, the patch is small (guard fires on inequality, message names the direction) — it is Lars's own repo; needs an owner go (concurrent session active there tonight).
8. **Untagged-backlog wave** (~20 modules — pre-existing row, re-derive fresh at execution).
9. **Verify-launcher promotion** (fold preflight + 5.3 extension into `scripts/verify-launch.sh`).
10. **F032 upstream detector fix** (toolspec imports: integration, not adoption).
11. **Nested per-group caps** (transcripts trimCall kept — expressibility gap rowed upstream).
12. **erraudit pin bump** (needs upstream tag; 4 manual storage/turso rows retire with it).
13. **templ re-sync** when nixpkgs ships a true 1100-era generator (re-bump lib+components together, regenerate, remove the skip_steps entry).
14. **crm read-storm re-measure**; **BuildFlow BD003 ×3**; **cqrs-htmx usermgmt release wave** (syncs 11 indirect kv pins).
15. **pgengine-66s watch row** (only if it recurs).
16. **M05 addendum + 21:28-report cross-link** once the verdict lands.
17. **Quiet-window discipline note** in gotchas-testing: load check BEFORE interpreting local VM/ephemeral results (§d.5).
18. **Report-index row** for this file (docs/status/README.md) — daemon usually handles; verify.

## g) Questions I cannot answer myself

1. **The parked one, unchanged: SSE flake — instrument or quarantine?** (`systemscenario.TestSSE_TaskViewStreamsOverHTTP`; 3rd occurrence class; unreproducible locally ×9; the drop-counter instrumentation plan sits in the TODO row.) Tonight added NO new data — Verify-fast never completed. If the current run completes Verify-fast, the answer may arrive by itself.
2. **Should dead-connection tolerance move into the ENGINES (production) as `errorfamily` Transient classification**, so consumers dialing flaky MySQL/Dgraph get caller-side retry semantics for free — or is test-only tolerance the correct scope (production retry = caller policy per the architecture)? I chose test-only + the TransientCommitError capability (which IS a production seam, but only for whole-tx aborts). This is a policy call.
3. **Daemon-vs-authored commits on hotfix cadence:** five content commits became `chore:` absorbs tonight. Acceptable (content verified, norm documented), or does the owner want hotfix commits protected (e.g., a daemon pause flag, or my committing via a single staging+commit call with zero gap)?

— session park point: report written 02:0x CET; **WAITING** on (1) run 42318787854's final verdict and (2) the owner's answers.
