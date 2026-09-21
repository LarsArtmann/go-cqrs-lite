# Status Report — M20 One-Pagers + M16/M13/M08-Local Wave (owner directive: M20 first)

> Point-in-time snapshot, 2026-09-21 ~15:05 CEST, master @ `ffbfdcf07` + pending.
> Continues `2026-09-21_14-12_w0-burn-class-guards-w3-rulings-execution.md` under the
> [owner-unblock-trust plan](../planning/2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md)
> ("execute the WHOLE list", M20 flagged important by the owner).

## a) DONE this wave (all verified, committed, pushed)

| Item | Evidence |
| ---- | -------- |
| **M20 F109** lease one-pager | [`docs/planning/2026-09-21_engine-single-writer-lease-one-pager.md`](../planning/2026-09-21_engine-single-writer-lease-one-pager.md) — verified: lease semantics only in `queue/`+`claiming/` task claims; rec = `EngineConfig.SingleWriter` advisory `<dsn>.cqrs-lease` flock; ADR-0146 candidate |
| **M20 F110** AggregateOn one-pager | [`docs/planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md`](../planning/2026-09-21_aggregateon-querydecl-seam-one-pager.md) — QueryOption on `QueryDecl`, `MatViewSpecReporter` capability, scalar-covered-first (defect A guard) |
| **M20 F111** Scan-default survey | [`docs/planning/2026-09-21_scan-default-v5-survey.md`](../planning/2026-09-21_scan-default-v5-survey.md) — census found `system.Find` inherits the 100 cap + goal-shaped-app scans uncapped; rec = unbounded at v5 + lint nudge; feeds G-T14 |
| **M20 F112–F114** routing + strikes | TODO rows for lease/AggregateOn-step/G-T14 updated with delivered links, open remainder kept visible; docs-truth tail row struck fully resolved |
| **M16** alias bundle | 5→(iterative unmasking)→11 ambiguous-alias sites fixed by importing the exact package in each fence; **doc-check zero warnings** (1194 refs); `TestRecipes` harness green 21s |
| **M16** §2.11 live-latency sync | every §2.11 claim re-verified against code: 1s default probe interval (`probe.go:89`), 5s timeout, hysteresis 0.20 (`store_routing.go:17`), `StartAutoReplan` stop-func shape, `FormatLiveLatency` — section accurate, no edits needed |
| **M16** status indexes | live-reports table (17 files, one-liners) in `docs/status/README.md`; per-day wave table (132 rows) in `docs/status/archived/README.md` |
| **M13** FEATURES census | scripted bidirectional diff: header said 82, disk has **96**; 6 missing rows added (goal-shaped-app, scheduler-otel-status, storage/backuptest, system/integration, testutil/pgtestcontainer, testutil/mysqltestcontainer); sub-package rows explained; census note names the method; my own row-draft errors caught by source verification (env names are `DATABASE_URL`/`POSTGRES_TEST_DSN`, not invented ones) |
| **M13** guarantee stamps | Architecture-Guarantees section got dated last-verified stamps separating this session's doc-gate evidence from the 09-20 full `#verify` |
| **M08 local** autoretry | `.github/workflows/autoretry.yml`: failed CI/Nightly-Gates runs rerun **failed jobs only, attempt==1 bound** — infra transients get one extra shot, real breakage stands, billing sees at most one partial rerun |
| **M08 local** leg-set trim | CI per-module matrix stopped burning a nix-installing runner on the package-less root module and the zero-test cqrs-lint fixture (examples KEPT — their tests run nowhere else); actionlint clean |
| CHANGELOG wave | guard-wave entry in `[Unreleased]` (11 symbol citations verified honest by the gate) |
| docs-truth row | struck: Scan docs (09-17), alias advisories (today, was 3→actually 5 sites→11 fences), overflow probe embedded (09-18) |

## b) PARTIALLY DONE

- **M10** — vm-mysql hardening landed last wave; **the real `#integration-mysql-vm`
  run is the only remainder**. Deferred repeatedly today because system load ran
  35–72 (attempt-burn class; the guards we built exist for exactly this). Will run
  in the first load1<5 window.
- **M08** — nightly 04:12 triage + TagContent clean-run confirm need remote Actions
  evidence (owner question Q2 unanswered). Local scope done (see a).

## c) NOT STARTED (remaining plan items)

M11/M12 upstream filings (approved; repros need CPU — load-blocked today),
M14 (READMEs into doc-check gate), M15 (quickstart drift guards),
M17/M18 (goal-shaped-app adoption demos), M19 (FilterOp — gated on Q1 pin-wave
timing), M22–M27 polish tails. M13's per-module fresh-run stamps (needs quiet CPU).

## d) WHAT WE GOT WRONG / NEARLY WRONG

1. **Daemon-race on the M20 commit** — the auto-commit daemon absorbed the three
   one-pagers mid-commit; my commit landed as the final cite-fix only. Content was
   never at risk (same wave, pushed together) but attribution split across commits.
2. **One-report-per-alias misled the M16 estimate** — "3 advisories" was 3 *aliases*;
   fixing one instance unmasked the next (11 fences total), the same iterative
   unmasking AGENTS contract #14 documents for art-dupl. Should have swept all
   instances per alias from the start.
3. **Two invented details caught before commit** — pgtestcontainer env name and a
   driver-registration claim in my first FEATURES row draft failed source
   verification; fixed pre-commit. The verify-your-own-citations discipline worked.

## e) IMPROVEMENTS

- The alias-sweep should be a `cmd/doc-check` mode: `--list-all-ambiguous` emitting
  every instance, not the first per alias. Candidate for the M23/md-go-validator wave.

## f) NEXT TASKS

1. **M10 VM run** in the first quiet window (load1<5), then F52 AGENTS rows + TODO
   evidence + strike.
2. **M11/M12 filings** (verify-before-filing: fresh exhaustruct repro, go/types race
   repro, turso pin-bump probe) → github-voice → file → TODO links (owner approved).
3. M14 → M15 → M17/M18 → M22 → M23 → M25 → M26 → M27; M19/M13-stamps when CPU allows.
4. Wave-close: composed `preflight-composed.sh` + `#verify` in a quiet window.

## g) QUESTIONS (still open, unchanged from 14:12 report)

- **Q1**: run M19 now vs hold for the metaengine/record pin-tag wave?
- **Q2**: is `gh` Actions access available for M08 remote evidence, or billing-gated?
- **Q3**: file upstream issues direct (M11/M12) or show drafts first? ("go file" is on
  record from the W3 bundle — proceeding on that unless overridden.)
