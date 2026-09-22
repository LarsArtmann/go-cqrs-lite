# Status Report — Full-Execution Session: Pareto Plan T01–T17 (disjoint waves under heavy concurrency)

**Date:** 2026-09-22 02:33 → 03:31 CEST
**Mandate:** "GET SHIT DONE! The WHOLE TODO LIST!" — execute
`docs/planning/2026-09-22_01-25_SUPERB-unblock-prove-deliver-pareto-plan.md`.
**Execution reality:** a second session was executing the same plan's
unblock tier in parallel (gate, skip_steps, hook hardening, go-env.sh) plus
the standing goal-closure lane; this session pivoted to disjoint,
verifiable waves and verified the overlapping ones after they landed.

## a) FULLY DONE (verified this session)

| #  | What                                                                                                                             | Evidence                                                                                                   |
| -- | -------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| A1 | **Directive-wave root cause NAILED** — BuildFlow's `go-version-auto-configure` auto-fix (pre-commit hook) canonicalizes `go 1.27.1` → `go 1.27`; every wave was an authored-commit hook run + daemon absorption; two MORE waves (5th ~02:39, plus the failed-commit replay) hit during this session and were caught red-handed by an armed watcher (`/var/tmp/godirective-watcher/`, ps snapshots) | my own hook output: "go-version-auto-configure 96 fixed"; empirical matrix: go 1.26.7/1.27.1 binaries never rewrite; fix = `.buildflow.yml` skip_steps (concurrent session, dry-run-verified "skipped via skip_steps config") |
| A2 | **T01 restore** ×3 (waves 5+6 undo) + build/test legs: workspace build green, system 4.4s, metaengine 14.6s, all six examples build+vet | census 97×1.27.1 after each restore; authored restore commit died on foreign mid-flight hook states ×3 — daemon absorbed; concurrent session landed the authored root-cause commit `b91205f23` |
| A3 | **check-workspace-sync.sh silent-death fixed** — example extraction still parsed the old flake form; `set -e`+pipefail aborted with rc=1 and NO output, blocking every authored commit's hook leg | `OK: go.work ↔ flake.nix are in sync` post-fix; awk over `exampleModules = [...]` + `\|\| true` guards |
| A4 | **T03 remote evidence** — 4/4 v4.9.0-wave Releases verified (sqlstore's tag was pushed but release.yml never ran → release created manually with provenance note); system/v4.9.0 notes curated; pkg.go.dev indexes v4.9.0 (`On`/`WithRacySave` render; `ErrRacySaveRefused` at system/errors.go:24) | `gh release list/view/create` receipts; **F153 filed: pkg.go.dev "License: UNKNOWN" hides ALL docs** |
| A5 | **T13 consumer answered** — go-graph-rag (own repo) TODO_LIST "Closed outside this repo" gained the #3/#5-fixed-in-v4.9.0 note + re-test invite | go-graph-rag TODO_LIST +10 lines |
| A6 | **T14 skill-ref sweep** — core.md + recipes.md pyramids → fluent `Evolve(...).On(...).Done()` (recipes catalog trailer fixed); FAQ third-party-engine entry; F151 206→207; at-least-once contract in readmodels.md; getting-started canary (README note + seam-naming failures); F152 fence green | TestRecipes ok; doc-check 1200 refs/54 pkgs; getting-started tests ok |
| A7 | **T15 scheduler-otel-status suite** — claim-flow test (schedule→Due-claim→MarkFired→Metrics counts→second-poll-empty) + rate-math test, race-clean; flake comment "all six carry suites" TRUE again | `go test -race` ok 1.0s; flake.nix comment updated |
| A8 | **T16 Turso grouped-matview fail-closed** — `ErrGroupedViewBugRefused` + `WithKnownGroupedViewBug()` opt-in at `tursoengine.New`; refusal test; repro/bench/property suites opt in explicitly; API golden regenerated; CHANGELOG + readmodels.md caveat updated | full tursoengine suite ok 4.2s; check-changelog-symbols honest |
| A9 | **T07 partial** — pin-sweep --check GREEN; cqrs-lint over all six examples: taskmanager C017 (memory→SQLite DLQ + lifecycle wiring) + S010 (documented wire-vs-at-rest nolint ON the finding line) + F031 (explicit WithLimit) fixed, readme-quickstart C028 ×2 fixed → zero error-severity findings; all six example suites green | per-example lint PASS; ALL-SIX-GREEN test leg |
| A10 | **T06/T12 verified+wired** — check-md-go --self-test 4/4 PASS (concurrent session shipped it); FEATURES docs-gates row + release-checklist mention added | self-test output; flake:1047 + ci.yml:85 wiring |
| A11 | **T17 canonical T18b record** — `docs/benchmarks/2026-09-20-21_t18b-record.md` (closure receipt, 4 gate-semantics changes, widening rule, incidents); `/var/tmp/t18b` retired (chain green 2026-09-21 18:14 UTC) | md-go gate green incl. the new doc (1460 valid) |
| A12 | **preflight-composed phantom-app lie fixed** — cited nonexistent `#can-run-composed-gate`; now points at `#quiet-window-run` (which also needed the inner `--` syntax) | script line 190 |
| A13 | **Master pushed** (3856cabe7..e70cc519b, 7 commits) — CI evidence live; fresh CI will confirm pin-sweep/CGo legs against the restored state | git push receipt |
| A14 | **T04 armed** — preflight-composed 6/6 GREEN (lint-config, templ, bench-gate, coverage, api-stability, duplication); `quiet-window-run -- nix run .#verify` waiting out load (ceiling 5; load spiked 109→857 from sibling sessions; 6h deadline) | job logs; /var/tmp/t04-verify.log |

## b) PARTIALLY DONE

- **T10 upstream filings** — guardrail-honored: filings are [BLOCKED] on
  owner approval (plan's "owner-approved" label conflicts with the TODO
  rows' blocks). Repro prep done for exhaustruct: upstream module is the
  **gaijin fork** (`dev.gaijin.team/go/exhaustruct/v5`; `4meepo` 404-GONE);
  singlechecker harness built at `/tmp/exhaustruct-repro`; synthetic
  minimal shapes do NOT trigger the panic — the definitive repro is the
  historical pre-fix tree (~`eea1c3c66^`). All banked in the TODO row.
- **T05/T20 (concurrent lane)** — verified landed: go-env.sh works
  (GOTOOLCHAIN local→auto, cache chain redirected off the FULL
  /mnt/buildcache); adopted in hook + benchmark-regression + check-coverage
  + preflight-composed.

## c) NOT STARTED (deliberate)

- T08/T09 tag waves — blocked on T04 green (verify) + the goal-closure
  lane's metaengine files settling.
- T25 owner bundle, T26 v5 train, T27 long tail — per plan tiers.

## d) TOTALLY FUCKED UP (honest ledger)

1. **Three authored commits died on foreign mid-flight states** (hook
   syntax error mid-edit, transient workspace-sync failure from the other
   session's flake.nix churn, shellcheck leg on their WIP scripts). I kept
   retrying instead of pivoting to the daemon-absorb reality ~15 minutes
   earlier.
2. **The exhaustruct repro rabbit hole** — spent ~20 min trying to build a
   minimal synthetic repro before conceding the historical-tree route; the
   harness + module-path correction are still net-positive but the trigger
  isolation remains open.
3. **A wrong quiet-window invocation** (`-- nix run .#verify` missing the
   inner `--`) burned one launch; the script's error output taught the fix.

## e) WHAT WE SHOULD IMPROVE

- **The pre-commit hook is a concurrency hazard**: any session's commit
  runs BuildFlow over the WHOLE shared tree, including other sessions'
  half-written files (syntax-error hook, shellcheck on WIP). The
  `--staged-only` flag is passed but some legs are repo-wide. Candidate:
  gate only staged files everywhere, or serialize commits per-repo.
- **`/mnt/buildcache` hit 100% AGAIN** (live at 02:49; ambient-GOMODCACHE
  toolchain unzips die). The monitoring row now has two recurrences
  recorded; needs the du-breakdown before any clear policy.

## f) NEXT (priority order)

1. Watch T04's verify land (6h window); record S04 in the composed-verify
   row; expect the two FOREIGN file-size ratchet violations
   (planner.go 562→579, query_constructors.go 407→411 — goal-closure
   lane's growth) to fail it — their session must shrink or re-baseline.
2. T08/T09 tag waves once verify green + lane settles (V006 golden regen
   included).
3. Turso defect-A onset characterization (M40) + the three drafts, all
   owner-gated for filing.
4. F153 license investigation (LICENSE propagation to submodules on
   pkg.go.dev).
5. Harvest this report on the next docs-health pass.

## g) QUESTIONS FOR THE OWNER

1. **F154 (BuildFlow upstream)**: file the go-version-auto-configure
   patch-floor bug upstream? Fleet-wide blast radius; repro solid.
2. **F153**: is a per-module LICENSE (or a LICENSE symlink strategy)
   wanted so pkg.go.dev renders docs, or do we accept hidden godoc?
3. **Filings bundle**: approve the three M11/M12 drafts + Turso defect A+B
   issue for filing (all verify-before-filing gated)?
