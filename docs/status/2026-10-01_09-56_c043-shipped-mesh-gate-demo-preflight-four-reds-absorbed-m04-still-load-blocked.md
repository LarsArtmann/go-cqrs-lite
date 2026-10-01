# Status 2026-10-01 09:56 — C043 shipped, mesh gate demo, preflight four-reds absorbed, M04 still load-blocked

> Session 05:00–05:40 active work + background chain until ~06:35, gap until 09:55.
> Continuation of the SUPERB post-wave plan
> (`docs/planning/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md`).
> Prior session report: `2026-10-01_04-30_config-war-recurrence-killed-ddl-dedup-absorbed-breakages-m24-enricher.md`.
> The three owner questions from that report remain OPEN (relisted in §g with fresh evidence).

---

## a) FULLY DONE this session

1. **M24.4 — RequestScope doc tail (closes M24 completely).**
   - core.md §3.8 gains the "which request" counterpart block (WithRequestScope +
     CompositeEnricher example, recipes §2.42 pointer).
   - recipes.md §2.42 "Request Correlation — RequestScope Enricher" added and classified
     in `cmd/doc-check/recipes_catalog_meta2.go`; compile-verified by `TestRecipes`.
   - cqrs-htmx `usermgmt/audit_context.go` carries the dated drop-local-copy TODO for the
     next bump (upstream = `event.RequestScope` + `WithRequestScope` + `RequestScopeEnricher`).
   - doc-check green: 1,161 refs/54 pkgs (explicit) and 2,449 refs/86 pkgs (auto-discovery
     incl. module READMEs).
   - Fix worth noting: both doc fences initially used `decider.WithEnricher(...)` without
     the explicit `[State]` — inference is impossible from a ContextEnricher arg; corrected
     in both files (the recipes compile gate caught it, not me).

2. **Receipts (plan step 2).** 7 new CHANGELOG bullets (M24/GitHub-#35, M25, GraphBFSNodes,
   ExecDDLLocked, turso v0.8.1 re-verification, ephemeral passthrough, C043) + Fixed bullets
   (SIGPIPE, config-war #7, eventtest nil-payload, typedfixture re-tidy, file-size splits).
   TODO_LIST: #35 W3 row annotated DONE, GraphBFS unify row struck, ephemeral passthrough row
   struck. `check-changelog-symbols.sh` + `check-changelog-coverage.sh` green (one fiction
   caught and fixed: `jsontext.Value` is stdlib, not in the api golden — reworded).

3. **NATS flake check (plan step 4).** Second full suite run against ephemeral nats-server:
   all 4 tests PASS in 2.048s (roundtrip, Nack redelivery, group exactly-once, 2 MiB
   payloads). Two consecutive clean runs → not a flake. Receipt lives here; no M22 watch row
   was due (CI-watch tally unchanged).

4. **M25 — mesh-demo runtime coeffect-gate twin.**
   - `example/mesh-demo/gate.go`: orders context composed via `system.New` over the memory
     engine, folds reusing the bilateral contract payload types verbatim (`OrderPlacedPayload`,
     `InvoiceReceivedPayload`, `OrderCompletedPayload`, EvolveKey("OrderID")).
   - `mesh-demo gate` subcommand: typo'd import (`invoice.issud`) → `ErrDanglingEventSubscription`
     naming the dangling subscription and its consumer projection; contract universe composes
     cleanly. `gate_test.go` pins both sides.
   - go.mod gains `system/v4 v4.10.0` (+ transitive); `check-example-standalone.sh` green.
   - README layout row + runnable section; core.md §3.9 pointer line; CHANGELOG bullet.

5. **F032 — cqrs-lint C043 `named-byte-slice-payload`.**
   - Type-aware rule: named type with underlying `[]byte` passed as the `event.New` payload
     (arg index 4) → Warning/high; `jsontext.Value` exempt (the fast path's other named case);
     silent without TypesInfo (no syntactic guessing). Detector matches via
     `IsQualifierFor(gf, sel, "go-cqrs-lite/event")`.
   - 6 tests via `BuildContextWithTypes` with a local stub whose import path contains the
     qualifier fragment (typed resolution works exactly like production).
   - Root-cause helper fix: `analyzer.BuildContextWithTypes` now MkdirAll's fixture
     subdirectories (was flat-file-only; subpackage fixtures failed to write).
   - Registration + catalog entry + RULES.md regenerated + README headline 208→209 rules
     (correctness 42→43) + meta-test counts. Full cqrs-lint suite green (19 packages).
   - FP check: 0 C043 findings across watermill (the fixed bridge), mesh-demo,
     projectionadapter. FAQ entry cross-referenced. CHANGELOG bullet.

6. **File-size ratchet: three inherited NEW offenders absorbed (no behavior change).**
   `storage/sql/helpers.go` 358 (M16 zero-checkpoint fix), `analyzer/scanner.go` 442,
   `cmd/doc-check/main.go` 356 (09-29/30 waves) → split at cohesive seams:
   `storage/sql/checkpoint.go` (load/save pair), `analyzer/scanner_consts.go` (const-decl +
   alias-resolution family), `doc-check/json_report.go` (the `--json` wire shape).
   Gate back to "no new offenders, no growth". All affected module tests green.

7. **Preflight four-reds absorbed (M04.2 precondition work).** `preflight-composed.sh`
   initially RED on templ / api-stability / duplication / error-taxonomy:
   - **templ**: 5 docserver `_templ.go` files stale → regenerated from the right cwd; gate green.
   - **api-stability**: golden 7,523 vs 7,524 → regen (exactly `+NewC043Detector`, diff
     inspected); `TestEvery` then failed on 8 untidy modules (release-train rot:
     bboltengine, dgraphengine, duckdbengine, mysqlengine, otelobserver, queue/mysql,
     queue/postgres, systemtest) → all tidied, `TestEvery` green.
   - **error-taxonomy**: (i) `watermill.publish_event_failed` doc row had no literal mint
     site — root fix: `publishAll` now takes a `wrapFailure func(error, T) error` closure and
     each adapter owns its code at a literal call site (`wrapEventPublishFailure` /
     `wrapCommandPublishFailure`); loop stays shared, families stay un-forked, both codes
     now gate-visible. (ii) `storage.postgres_ddl_lock` (minted by storage root,
     M16.2-era) added to the doc's DDL row. Gate green: 526 codes / 457 doc claims.
   - **duplication**: 1 new clone group in `catalog/docserver/specview.templ` (noscript
     blocks) — templ groups have no accept-directive lever, baseline is the sanctioned
     escape → re-pinned 187→186 groups (daemon-committed first, dirty-tree guard honored),
     gate green.
   - **Final preflight: ALL 9 PHASES GREEN** (lint-config, templ, bench-gate, coverage,
     api-stability, duplication, go-version, turso-version, error-taxonomy).

## b) PARTIALLY DONE

- **M04 composed verify.** M04.2 (preflight) DONE GREEN. M04.1 wait-loop LAUNCHED 05:35,
  TIMED OUT after 3600s / 20 attempts: 14× "load above ceiling 10", 6× "tree changed during
  the 60s stability window" (daemon + what looks like a concurrent session editing).
  M04.3 (`nix run .#verify`) never ran; M04.5 receipts pending. NOTE: the wait-loop ceiling
  is `VERIFY_MAX_LOAD` default **10**, not 5 as the prior summary claimed — discovery worth
  keeping. At 09:56 load1 spiked to 23.4 but load15 was 5.46, i.e. a near-quiet window
  existed ~09:40–09:50, AFTER the chain had already died (see §d).

- **Session close.** verify-fast/api-stability final confirmation partially covered by
  preflight phases (api-stability + lint-config + duplication all green standalone); the
  composed `#verify` leg and the closing report are this document.

## c) NOT STARTED (all quiet-gated, never got a window)

- **M05** MySQL quiet legs (`#integration-mysql-vm` + shuffled-seed replay from
  `build/shuffle-seeds.log`, QEMU port 33070 pre-flight) + M05.5 receipts.
- **M11** calibration campaign (`calibration-gate.sh`, SearchQuery count=5 supersede check,
  dgraph constants re-anchor).
- **M14.3** per-module fresh-run stamps.
- **`nix flake check`.**
- **M15.3** routing owner decision-pack answers (A1–A14/B1–B5/C1–C4) — no answers arrived.

## d) TOTALLY FUCKED UP

- **The monitoring gap (my fault, biggest miss of the session).** I launched the M04 chain
  in background at 05:35, checked load once at 05:39, and then effectively sat idle. The
  chain died at ~06:35 (1h timeout); I only noticed at 09:55 when the owner pinged. That is
  3h20m of dead time during which (a) the chain was not re-launched, (b) a re-launch with a
  longer `--max-wait` was never considered, (c) usable read-only work (grep|head audit,
  fp-sweep review, seed-log pre-read for M05.4, flake.nix read-through for the flake check)
  went undone, and (d) the near-quiet window around 09:40–09:50 arrived with no chain alive
  to consume it. Lesson: background gates need a supervisor loop or periodic job_output
  polling, not fire-and-forget.
- **Root cause of the tree-stability failures found (post-hoc):** the status index shows TWO
  concurrent sessions ran inside my wait-loop window —
  `2026-10-01_05-54_catalog-eventcatalog-integration-research...` and
  `2026-10-01_06-15_embedded-eventcatalog-static-server-shipped.md` (the latter shipped
  `catalog/eventcatalog.StaticServer`!). Their edits explain the 6× "tree changed during
  the 60s stability window" refusals. So the wait-loop wasn't only load-blocked — the repo
  was genuinely mid-edit by other sessions. Any re-launch must expect this and the
  supervision must poll, not assume.
- **Inherited (not mine, absorbed): the prior session's close-out overstated cleanliness.**
  Its report claimed gates green, yet this session's first preflight found FOUR red gates
  (templ staleness, api golden drift, taxonomy drift, file-size NEW offenders) that predate
  my first edit. The prior session verified canonical-facts + doc gates but not these. No
  harm done — preflight exists precisely for this — but "verify before claiming verified"
  slipped.

Nothing was broken-and-left-broken: every red found this session was fixed and re-verified
green in the same session.

## e) WHAT WE SHOULD IMPROVE

1. **Supervise background chains**: re-launch `can-run-composed-gate --wait-loop` on timeout
   (or raise `--max-wait`), poll `job_output` at least every ~20 min, and use dead time for
   the read-only backlog.
2. **Document the real quiet ceiling** (`VERIFY_MAX_LOAD` default 10, not 5) in the plan/TODO
   rows so future sessions don't gate on an imaginary number. Consider whether 10 is even
   right given the langserver floor (see §g Q2).
3. **The langserver load floor is structural**: 10+ `golangci-lint-langserver` instances up
   to 22h+ old pin load ≥10-12. Until they are reaped or rate-limited, every quiet-gated task
   in this repo is effectively blocked. This needs an owner decision, not a workaround.
4. **Tree-stability vs the daemon + concurrent sessions**: 6 of 20 wait-loop attempts died
   on "tree changed during stability window" even when load was irrelevant. A
   `can-run-composed-gate` that ignores daemon-only commits (commit != tree edit) would
   remove a whole failure class.
5. **Carried systemic items** (unchanged, still open): tursogo pin-vs-constant gate leg;
   grep|head-under-pipefail audit across scripts/; scoped `nix fmt` (full-tree fmt woke the
   langservers); daemon config-corruption pre-commit tripwire; error-taxonomy gate cannot
   see dynamic-mint codes (C043-style blind spots) — the wrapFailure refactor is the
   template for making mints literal.
6. **Preflight is proven** — this is the second arc where it caught sub-5-minute reds before
   a 100-minute verify burned. Keep it mandatory before every composed attempt.

## f) NEXT (prioritized, up to 50)

**The M04 arc (blocking trust debt):**

1. Re-launch `scripts/can-run-composed-gate.sh --wait-loop` (consider `--max-wait` beyond
   3600s, e.g. 10800) — supervised, polled.
2. When GREEN: `nix run .#verify` EXCLUSIVE (no edits during).
3. M04.5 receipts: dedup (a) row + CI re-record row closed GREEN (or triage list).
4. If load <10 never returns: owner decision needed (§g Q3) — do not force.

**Quiet-window queue (same window as M04 if it holds, in order):**
5. M05.2 `nix run .#integration-mysql-vm` (port 33070 pre-flight).
6. M05.4 shuffled-suite seed replay from `build/shuffle-seeds.log`.
7. M05.3 triage + M05.5 receipts (F52 AGENTS rows + Green-MySQL row).
8. M11.1 `calibration-gate.sh` PASS check.
9. M11.2 SearchQuery count=5 re-run (supersede if medians move >5%).
10. M11.3 supersede-note on the 2026-09-19 capture; M11.4 baseline re-pin decision.
11. M11.5 dgraph constants re-anchor.
12. M14.3 per-module fresh-run stamps (quiet CPU).
13. `nix flake check`.

**Small code/docs tail (not quiet-gated):**
14. grep|head-under-pipefail audit across scripts/ (3 sites fixed, class remains).
15. turso pin-vs-constant gate leg (check-canonical-facts extension).
16. cqrs-lint: consider C043 mutant-discrimination + suppression fixtures if the meta-suites
want them (suite currently green without).
17. Re-run `scripts/fp-sweep.sh` 12-repo baseline to measure C043's real-world FP rate.
18. watermill skill backends.md: note the publishAll wrapFailure shape (mint-site locality).
19. cqrs-htmx: execute the audit_context.go TODO at next go-cqrs-lite bump (adapter +
`event.RequestScopeEnricher` swap) — sibling-repo change.
20. mesh-demo: consider a `gate` mention in the data-mesh skill reference (§2.41) if the
demo should surface there too (currently only core.md §3.9 points at it).
21. Consider `--max-wait` and daemon-aware stability for `can-run-composed-gate.sh` (item 4
above as a code change, with --self-test).
22. Load-sweep (`nix run .#load-sweep`) after the timing-path-adjacent refactors
(publishAll closure change is not timing-path, but cheap to confirm).
23. Nightly-bench baseline sanity after this session's code lands (bench-gate preflight leg
already green, so low risk).

**Release-train hygiene:**
24. `nix run .#vulncheck` (per-module standalone build) — not run this session.
25. `nix run .#check-arch` — not run this session (new deps: mesh-demo system/v4 — examples
are outside dep budgets, but verify).
26. `nix run .#check-coverage` ran green inside preflight; re-run after M04's full verify.
27. Watermill module: after publishAll refactor, confirm no consumer pinned the old private
signature (impossible — unexported; tests green).
28. When owner answers push question: push master (currently ~30+ unpushed daemon commits).

**Session close (after M04 resolves either way):**
29. Final `#verify-fast` confirmation.
30. api-stability verify pass (already green standalone; re-confirm at close).
31. Closing status report + index row + canonical-facts gate.
32. Route decision-pack answers to TODO_LIST rows (M15.3) when they arrive.

**Backlog candidates (from this session's observations):**
33. error-taxonomy gate: teach the scanner closure-owned mints OR document the
wrapFailure-locality convention for shared helpers.
34. `analyzer.BuildContextWithTypes`: consider a `--self-test`-style fixture test proving
subdir support stays (I added MkdirAll without a dedicated regression test; covered
indirectly by c043 tests).
35. doc-check: the json_report.go split leaves `main.go` at 290 — fine, but consider moving
findRepoRoot pair out too if it grows again.
36. C042/C043 share the "positional arg index" fragility (arg 4/arg 3) — if event.New's
signature ever grows, both rules need review; a comment or test fixture per constructor
shape would pin it.
37. The `#integration-nats` flake tally: log today's second green run in the M22 watch notes
when the CI leg exists.
38. Langserver lifecycle: propose a systemd user timer or Crush hook that reaps
langserver instances older than N hours (owner-gated).
39. Re-check the three "absorbed breakage" scripts under the NEXT daemon config-war
recurrence (tripwire trio mutation test still green per preflight lint-config).
40. M27 (goal-shaped-app activation + catalog pin tail) — still gated on M09 owner go-ahead.

## g) QUESTIONS FOR THE OWNER (cannot be resolved from inside the session)

1. **Push `master` now?** ~30+ daemon commits sit unpushed (tags are already public; A2 in
   the decision pack recommends yes). The branch has this session's C043 + M25 + M24 docs +
   gate repairs, all green standalone.
2. **Kill the stale golangci-lint-langserver instances?** 10+ running, up to 22h+ old, zero
   CPU each but they anchor the load floor ≥10-12, which is the ONLY thing standing between
   this repo and every quiet-gated task (M04/M05/M11/M14.3/flake-check). They are likely
   orphaned from earlier sessions; I lack permission to kill processes I didn't start.
3. **If load <10 never returns: which fallback for M04?** Concrete evidence now: 20 attempts
   over 3600s all red (14 load, 6 tree-stability), and a near-quiet window (load15 ≈ 5.5)
   appeared ~3h AFTER the chain timed out. Options: (a) run `#verify` at a raised
   `VERIFY_MAX_LOAD` (e.g. 15) accepting some timing-test flake risk, (b) you name a window
   (e.g. overnight) and I supervise a long `--max-wait` chain, (c) accept the standalone
   per-phase greens (preflight 9/9) + per-module tests as this cycle's verification record.

---

_Arte in Aeternum — the window existed; the chain was dead. Fix the supervision, not the gate._

---

## Postscript 11:01 CEST (owner re-pinged)

- Load at 11:01: **11.45 / 7.49 / 9.67** — load5 is UNDER the ceiling (10); load1 is 1.45
  over. A window is actively forming; with the three owner questions answered (push?
  kill langservers? fallback?), a supervised re-launch could plausibly catch it.
- Row-ownership: this report supersedes nothing substantive — the 05-54 and 06-15 reports
  belong to OTHER sessions and keep their own rows.
