# Session report: Unblock-the-Machine — go-directive root cause + Tier-1 execution

**Date:** 2026-09-22 02:30–04:00 CEST
**Plan:** [`docs/planning/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md`](../planning/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md) (Tier 1%→51% + parts of 4%)
**Outcome:** T01/T02/T03/T05 done, T06+T12 done, T20 partial (hook env), T04 armed-but-window-blocked. **The 4× downgrade-wave class is root-caused and mechanically dead.**

## The root cause (the session's finding)

**BuildFlow's `go-version-auto-configure` step caused every go-directive downgrade wave.**
Its rule — "the go directive is a floor and must be major.minor only; a patch component
pins the toolchain to one exact patch" — makes its `--fix` strip `go 1.27.1` → `go 1.27`,
which this repo's published dependencies (go-finding toolsdk ≥ 1.27.1) hard-fail on under
`GOTOOLCHAIN=auto`. The auto-commit daemon's post-commit BuildFlow cycles (full mode, with
repairs) re-applied the strip after every restore — waves 1–4, including a FIFTH wave that
struck mid-session at 02:41 and was caught red-handed.

Evidence trail (reproducible): `~/.local/state/buildflow/buildflow.db` step_outputs for
runs `20260922-003920-*` / `20260922-004155-*` show `go-version-auto-configure: success`
exactly at the rewrite mtimes; `buildflow -s go-version-auto-configure` reports the rule
verbatim ("major.minor only" warning on `example/taskmanager/go.mod`).

The same daemon wave (`4a540b02c`, 23:33) also corrupted `.golangci.yml` via
`golangci-lint-auto-configure` under the broken toolchain: depguard allow-list stripped
(−111 lines), `go:` pinned to 1.26.7, removed `goexperiment.jsonv2` tag re-added. The
hash-golden tripwire correctly refused it; restored from `4a540b02c^`.

## Fixes shipped (all committed; core narrative in `b91205f23`)

| Fix                                                                                                                                                                                                                              | Evidence                                                                                                                                                                      |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `.buildflow.yml`: `go-version-auto-configure` skipped (rationale in-file)                                                                                                                                                        | step no longer executes; directive survived a later `--fix` probe                                                                                                             |
| `scripts/check-go-version.sh`: drift lock — go.work floor (1.27.1) + lockstep go.mod equality; 5 new planted-fixture self-test legs incl. CI=true no-bypass; wired into existing `#verify`/nightly/`#check-release-scripts` legs | self-test 9/9; mutation-tested (neutered policy → self-test fails); live PASS via `nix run .#check-go-version`                                                                |
| `scripts/go-env.sh` (T05): one-file forced env chain                                                                                                                                                                             | idempotent-silent when correct; overrides ambient poison                                                                                                                      |
| Adopted in: `.githooks/pre-commit`, `check-coverage.sh`, `benchmark-regression.sh`, `preflight-composed.sh`                                                                                                                      | authored commits pass hooks under ambient `GOTOOLCHAIN=local` (proven live by `b91205f23`) — the `--no-verify` forcing is gone                                                |
| Directives restored to `go 1.27.1` (97 files)                                                                                                                                                                                    | workspace build green; system+metaengine+taskmanager `GOWORK=off` tests green; all six example suites green (incl. goal-shaped `TestDocs_ReadmeEvolutionFence` — F152 closed) |
| shellcheck debt → zero findings (`.shellcheckrc` policy + real fixes)                                                                                                                                                            | `shellcheck scripts/*.sh scripts/lib/*.sh` clean; real bug fixed: canonical-facts fixture printf arg ignored → `ROADMAP.md` never created                                     |
| `scripts/check-md-go.sh --self-test` (T06/T12): 4 gate behaviors pinned hermetically; `ARCHIVE_SEGMENT` mutation leg committed-row variant                                                                                       | 4/4 legs; mutation-tested; wired into `#check-release-scripts`                                                                                                                |
| md-go flake: version = pinned source rev (`4dd9437`), `ldflags -X main.version`                                                                                                                                                  | `nix eval .#md-go-validator.version` → `"4dd9437"`                                                                                                                            |
| `.golangci.yml` restored + templ regenerated from correct cwd                                                                                                                                                                    | `nix run .#check-lint-config` PASS; preflight templ phase PASS                                                                                                                |
| Baseline +1 (archived planning doc fence, sanctioned regen)                                                                                                                                                                      | `nix run .#check-md-go` green                                                                                                                                                 |

## T03 — remote evidence (recorded)

- Last pre-fix CI runs on the corrupted tree failed correctly (pin-sweep, coverage, CGo
  legs) — CI did its job; re-runs expected after push.
- 10 releases render, incl. `system/v4.9.0` with curated notes (fluent `.On` +
  fail-closed Save headline). `scheduling/sqlstore/v4.1.1` cut 00:51 by the concurrent lane.
- pkg.go.dev indexes `system/v4@v4.9.0` (valid go.mod, tagged, Imports: 34); docs hidden
  by design (proprietary license — "License: UNKNOWN" is expected, not a defect).
- Proxy probe: `go get system/v4@v4.9.0` resolves clean; `go doc` shows
  `ErrRacySaveRefused`/`WithRacySave`/`On`.
- **T13 done:** re-test invitation filed as
  [go-graph-rag#2](https://github.com/LarsArtmann/go-graph-rag/issues/2) (voice-checked).

## T04 — composed verify: armed, window-blocked (not failed)

Preflight composed went **GREEN on all 6 phases** (after the lint-config restore, templ
regen, and go-env adoption — the api-stability "not tidy" was `/mnt/buildcache` ENOSPC in
disguise). The `can-run-composed-gate --wait-loop` (2h deadline) stayed blocked: load1
spiked 448→908 from concurrent-lane heavy jobs on the 32-core host. **Next session: re-run
preflight, then the wait-loop + `nix run .#verify` in a real window; record S04.**

## Incidents / follow-ups for the owner

1. **`/mnt/buildcache` is 100% full** (155G rust + 20G sccache) — the host's default
   GOCACHE/GOMODCACHE point there; every unchained go command ENOSPCs. The
   `buildcache-alarm.timer` exists; space needs an owner decision (rust-cache prune?).
   Interim: `scripts/go-env.sh` redirects this repo's runs to `/`-backed paths.
2. **Ambient session env carries `GOTOOLCHAIN=local` + buildcache paths** (nixpkgs Go
   wrapper default; the fish guard fixes it only for fish login shells — crush/daemon
   envs bypass it). go-env.sh defends this repo; a session-wide fix is an owner call.
3. **Upstream BuildFlow fix (fleet-wide value):** `go-version-auto-configure` must learn
   dependency-driven patch floors (strip a patch only when no dependency requires it);
   its current rule silently breaks any repo whose deps pin a patch. Also worth filing:
   auto-configure under a too-old toolchain rewrote `.golangci.yml` and patched adjacent
   goldens — repairs should refuse to run when the toolchain can't even load the module.
   BuildFlow repo HEAD (`9663c02`) is ahead of the installed binary (`557fe59`) — check
   whether a fix already landed before filing.
4. **The concurrent goal-closure lane shipped T15** (scheduler-otel-status suite, +126-line
   `main_test.go`, absorbed by daemon `dd3691be0`/`442d168f8`) — TODO row strike pending
   the lane's own close-out.
5. TODO_LIST rows for this session's strikes (T01/T02/T05/T06/T12/T13, F152) deferred to
   the next docs-health harvest — the file is lane-contended (guardrail G1).

## T18b watcher note (guardrail G2 — record only)

The armed chain **completed green**: `closure-completion.log` ends
"T18b CLOSURE COMPLETE: widened 100x/9, baseline re-pinned (archived), verification PASS"
(2026-09-21 18:14 UTC); campaign + rootcause results are in `/var/tmp/t18b/`. T17's
canonical-record write-up (`docs/benchmarks/2026-09-20-21_t18b-record.md`) is the
remaining work; the arc itself is finished. `/var/tmp/t18b/` copies can be retired once
the record cites them.
