# Status 2026-09-29 06:41 — config-war recurrence caught, M14 counts gate shipped, M27 receipts closed

> **RESOLVED-BY-ROUTING (docs-health 13th pass, 2026-10-11):** every §f item
> struck with its landing receipt or routed; §g answered inline. Same-day
> successors (09-15) and the 10-01 wave executed the queue.

**Session window:** 05:32–06:41 CEST. Continuation of the SUPERB plan
([2026-09-28_22-00](../planning/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md)).
Load at start 14.4, at close 24.4/36.3/44.4 — quiet gates stayed closed all session.

---

## a) FULLY DONE (verified this session)

1. **M27 receipts closed.** CHANGELOG `[Unreleased]` → Changed: goal-shaped-app
   matview-LIVE entry (turso driver + `materialized_views` active vs published
   `tursoengine/v4.2.1`, `TestShippedConfigBoots` byte-for-byte pin). Symbol
   gate green (40 citations). M27 is now 100% done including receipts.
2. **AGENTS.md TL;DR rule 2 extended:** the four release/gate scripts
   (`tag-release.sh`, `batch-release.sh`, `pin-sweep.sh`,
   `check-example-standalone.sh`) self-source `scripts/go-env.sh` since
   2026-09-29; new go-invoking gate scripts must do the same.
3. **M14 finished (except quiet-gated M14.3 stamp protocol).**
   - **ADT anchor found** (the missing piece from last session):
     `metaengine.AllADTs()` (`metaengine/enum_validation.go`) = **11 planner
     ADTs**; `adttest.Scenarios()` (harness.go:205, "11-ADT test matrix")
     covers all 11 with 20 scenarios. The write-side `ADTDueClaim`/`ADTDedup`
     are explicitly NOT planner ADTs (types.go says so) — the skill's old
     "12 ADTs" census was silently omitting `StreamLog`.
   - **Canonical-facts gate leg 5 shipped** (`check_engine_counts` in
     `scripts/check-canonical-facts.sh`): derives 12 engine implementations
     (11 `metaengine/*engine` dirs + in-core `memory_engine.go`), 11
     self-registering drivers (`RegisterDriver` register.go sites — irohengine
     deliberately does not register), 11 planner ADTs; fails any canonical-doc
     citation that disagrees; REQUIRES each unit cited at least once.
     Self-test mutation legs added (wrong counts + missing-unit claim) and
     green; shellcheck clean; wired into nightly-gates.yml automatically
     (it already runs `--self-test` + gate).
   - **Live rot killed (4 docs, 5 sites):** ROADMAP:57 "10 engine backends"
     (missing BigTable + Iroh) → "12 engine implementations";
     ROADMAP:176 "All 10 drivers" census → 11 (+bigtable, iroh-exclusion
     noted); ROADMAP:74 "all 12 ADTs" → "all 11 planner ADTs"; FEATURES:246
     - FEATURES:1493 "10 ADTs" → 11 planner ADTs; FEATURES:1500 driver
       parenthetical rewritten gate-derived; NEW FEATURES row
       "Canonical engine/ADT counts" citing the gate; skill `modules.md` +
       `core.md` "12 ADTs" → 11 planner ADTs + write-side note.
   - **M14 CHANGELOG bullet** added; gates re-run green (canonical-facts,
     changelog-symbols, doc-check 1,132 refs over the edited skill docs).
4. **`.golangci.yml` config-war recurrence DETECTED + REPAIRED** (the
   session's most important find — detail in §d/§e):
   - Full `nix run .#lint` (first since the wave) came back RED with 285
     findings across 47 modules — **100% phantom `gci` formatter findings**,
     zero real issues.
   - Diagnosis: daemon commit `e52027a7d` (tonight 01:36:23, "auto-commit 46
     changed files", i.e. BETWEEN sessions) had damaged `.golangci.yml` in
     FOUR ways: re-added `gci` to `formatters.enable` (removed 2026-08-16,
     contract #18), downgraded `go: 1.27.1` → `1.26.7`, re-added the
     `goexperiment.jsonv2` build tag (removed 2026-09-19, contract #10), and
     DELETED the entire 90-line depguard allow-list block.
   - `check-golangci-hash.sh` tripwire correctly FAILED (RC=1) — M06's
     tripwire works; it just had not been run since 01:36.
   - Repair: restored `e52027a7d~1:.golangci.yml` wholesale (the gate's own
     sanctioned restore path); hash gate PASS; `#check-lint-config` + full
     `#lint` re-run launched (in flight at report time, log
     `/tmp/lint-full-2.log`).

## b) PARTIALLY DONE

- **Heavy gate batch** (post-wave tail): full `#lint` executed once (red —
  config damage, see above), repair applied, second full run IN FLIGHT at
  report time. `nix flake check` and `nix run .#check-duplication` NOT run.
- **M16** — reading/recon only: the queue M4 TODO row was located (remaining
  = (c) clock seam design-gated + (e) shared-DB parallel-migrate sweep; (a)
  (b) (d) (f) already done). No design doc, no sweep yet.

## c) NOT STARTED (this session)

M17 (system test-mass: config-loader table tests, rapid fuzz, shutdown
stress, determinism), M18 (3 temporal rapid properties + pebble/bbolt scope
note), M19 (ephemeral-nats.sh + watermill-nats roundtrip + `#integration-nats`
flake app), M21 (BFS unify, passthrough conventions, retry backport review,
templ watch note), M24 (`requestContextEnricher` upstream into `event/`),
M25 (mesh-demo system.New coeffect-gate variant), M26 (v5 inventory doc),
M15.1/M15.2/M15.4 owner-decision prep bundles, M20 matview triage note, M22
CI watch notes, M23 verified filing drafts (HOLD), F032 named-`[]byte` lint
rule, quiet-gated M04/M05/M11 (+M14.3 stamps), closing status report + push.

## d) TOTALLY FUCKED UP (and fixed, or still open)

1. **The daemon's 01:36 config damage (fixed, recurrence #2 of the class).**
   This is the second `.golangci.yml` war (first was 2026-09-26, repaired
   2026-09-28). The 46-file auto-commit did not just revert a line — it
   silently downgraded the go floor and deleted depguard, which would have
   caused EXACTLY the directive-downgrade wave class (F154) plus phantom
   lint findings, had it shipped into a release. Caught only because the
   plan scheduled the full-lint tail. The hash tripwire detects this class
   perfectly — the gap is that NOTHING ran it between 01:36 and 05:47.
2. **My multiedit ordering mistake** on `check-canonical-facts.sh`: edit 2
   consumed the `run_all()` function (I used it as an anchor and re-emitted
   it inside another edit's replacement). Caught immediately by structure
   inspection, repaired in the next edit, self-test green. Cost: one extra
   tool round-trip. Lesson: never use a still-needed function as an edit
   anchor inside a multiedit batch.
3. **Self-test leg-5 assertion bug (caught by the gate's own design):** my
   first mutation planted only 2 of 3 expected failures ("1 self-registering
   drivers" was honest). Fixed by planting a wrong drivers count (7) so the
   leg genuinely exercises all three failure modes (wrong engines, wrong
   drivers, missing ADT unit claim).

## e) WHAT WE SHOULD IMPROVE

1. **Run the config tripwire trio after EVERY session, or make it
   un-skippable:** the trio (`check-formatters`, `check-depguard`,
   `check-golangci-hash`) lives in `#verify-fast`, but sessions keep ending
   with "fast gates green" meaning only the gates they happened to run. The
   01:36 damage survived 4+ hours undetected. Candidate: a `#verify-fast`
   run as a session-closing ritual, or wiring the trio into the auto-commit
   daemon itself.
2. **Protect `.golangci.yml` from the daemon structurally:** the `.githooks/`
   directory exists — a pre-commit hook rejecting changes to
   `.golangci.yml`/`flake.nix` unless `GOLANGCI_INTENTIONAL=1` is exported
   would make the daemon's damage class impossible to land silently (owner
   decision — it changes local git behavior).
3. **The 01:36 46-file auto-commit should be autopsied:** what heuristic
   moved the daemon to rewrite a config it was told (via golden hash) is
   load-bearing? If the daemon re-runs `golangci-lint fmt` or similar on the
   tree, that tool invocation is the recurrence vector.
4. **Full `#lint` under load 14–44 took ~50 min.** Consider slicing the lint
   app into per-module phases (like `verify-ci`) so partial re-runs after a
   config repair don't re-lint 88 modules from scratch.
5. **Hand-maintained count census rows keep rotting** (M14 killed this
   class for engines/drivers/ADTs — 5 sites fixed). Remaining hand-counted
   claims spotted but out of scope: FEATURES:332 "12 ADT harness self-tests"
   (adttest has 4 _test.go files — claim shape unclear), FEATURES "150
   specs", various "N engines implement X" capability rows. A follow-up
   derivation sweep could pin the scenario-count claims
   (`len(adttest.Scenarios())` = 20) the same way.

## f) NEXT — up to 50, in execution order

1. ~~Await `#lint` re-run result (background, `/tmp/lint-full-2.log`) — expect
   green with the restored config.~~ done 09-29 09:15 — green after 8 real findings fixed (09-15 §heavy-gate)
2. ~~`nix run .#check-duplication` (clean tree required — daemon permitting).~~ done 09-29 09:15 — GREEN, baseline 187, 0 new clones
3. ~~`nix flake check` (heaviest leg; consider deferring into M04's quiet
   window if load stays >5).~~ routed — deferred exactly as suggested; still quiet-blocked at 10-11
4. ~~M16.1 clock-seam design proposal (ADR-0122 `WithClock`, delete fixed
   sleeps) — design doc under `docs/planning/`.~~ done 09-29 09:15 — `2026-09-29_07-20_queue-conformance-clock-seam-design.md`
5. ~~M16.2 shared-DB parallel-migrate sweep (`t.Parallel` + shared DSN across
   engine suites).~~ done 09-29 09:15 — advisory-lock DDL serialization at all 3 PG DDL sites; 3 concurrent regression tests green (09-15 §M16)
6. ~~M16.3 queue M4 receipts + TODO row strike.~~ done 09-29 09:15 (09-15 §M16.3)
7. ~~M17.1 system config-loader table tests (koanf/YAML + env merge).~~ done 09-29 09:15 — found+fixed 2 LIVE LoadConfig bugs (09-15 §M17)
8. ~~M17.2 config-loader rapid fuzz.~~ done 09-29 09:15 — 3 rapid properties, rapid v1.3.0 test-only
9. ~~M17.3 lifecycle/shutdown stress with real engines.~~ done 09-29 09:15 — 6 rounds × 8 concurrent creates on real sqlite, green
10. ~~M17.4 determinism test (same domain+deployment → identical wiring).~~ done 09-29 09:15 — byte-identical `system.Explain`
11. ~~M17.5 Feedback-#6 row receipt.~~ done 09-29 09:15 — row struck with receipts
12. ~~M18.1 rapid property: out-of-order stamps + same-ms LWW collapse.~~ done 09-29 09:15 (09-15 §M18)
13. ~~M18.2 rapid property: retention-never-prunes-newest.~~ done 09-29 09:15
14. ~~M18.3 rapid property: tombstone-as-of visibility.~~ done 09-29 09:15
15. ~~M18.4 pebble/bbolt join-scope decision note (owner question).~~ done 09-29 09:15 — `docs/reviews/2026-09-29_pebble-bbolt-versioned-cells-scope-decision.md`; owner ruling A–D pending
16. ~~M19.1 `scripts/ephemeral-nats.sh` (mirror `ephemeral-redis.sh`).~~ done 09-29 09:15
17. ~~M19.2 watermill-nats roundtrip test (mirror redis leg: roundtrip, Nack
    redelivery, group exactly-once, 2 MiB payloads).~~ done 09-29 09:15 — 4/4 green vs real JetStream
18. ~~M19.3 `#integration-nats` flake app + CI-leg decision.~~ done 09-29 09:15 — app added; CI-leg decision folded into the owner questions (billing-gated)
19. ~~M21.1 `graphNeighborsFallback` → `GraphBFS` unify (nil-vs-empty decision
    - typed-key param).~~ done 10-01 — M21.1 GraphBFSNodes unify (10-01_04-30 §a)
20. ~~M21.2 ephemeral-script passthrough conventions unify.~~ done 10-01 (M21.2)
21. ~~M21.3 contention-retry backport review (turso/badger transient-abort
    class).~~ routed — no landing receipt found through 10-11; lives in TODO_LIST §Code Quality
22. ~~M21.4 templ clone-group watch note (no lever until art-dupl templ
    support).~~ superseded 10-10 — templ residue moved into the baseline via surgery (10-10_12-34; AGENTS contract #14c)
23. ~~M24.1 read local copy of cqrs-htmx `audit_context.go`.~~ done 10-01 (M24 design step; 10-01_04-30 §a)
24. ~~M24.2 design `event/` enricher API (`WithEnricher` surface).~~ done 10-01 — shipped as `RequestScopeEnricher`/`WithRequestScope`
25. ~~M24.3 implement + tests + api golden regen in the same edit.~~ done 10-01 — code+tests+golden green (10-01_04-30 §a)
26. ~~M24.4 core.md section + cqrs-htmx TODO note.~~ done 10-01 09:56 — core.md §3.8 + recipes §2.42 compile-verified; doc-check 2,449 refs green
27. ~~M25.1 mesh-demo system.New coeffect-gate demo design.~~ done 10-01 09:56
28. ~~M25.2 implement the variant (runtime `DomainConfig.Events` gate demo).~~ done 10-01 09:56 — `gate` subcommand
29. ~~M25.3 verify + core.md §9 row + receipts.~~ done 10-01 09:56 — tests+standalone green; receipts in CHANGELOG/TODO
30. ~~M26.1 v5 inventory: every v5-gated row → readiness checklist (deps,
    order).~~ done 10-01 — `docs/planning/2026-10-01_v5-cut-readiness-checklist.md`
31. ~~M26.2 scan-default flip note (execute ON the v5 branch per ADR-0123).~~ routed — executes on the v5 branch; checklist row stands
32. ~~M15.1 quick-rulings one-pager (M22/Q3, push cadence, claiming V006,
    iroh P99, T18b (a)(b), DSN strict, sync/embedded scope, dgraph one-RPC,
    CapabilityGaps→Doctor, `#test-examples` gate, docs-health cadence,
    benchkit tag-wave timing).~~ done 10-01 — owner-decision pack (10-01_04-30 §a)
33. ~~M15.2 ADR-level rulings pack (SingleWriter ADR-0146, Direction ruling
    ADR-0147, v5-encryption 4 questions, ADR-0138 demand-check).~~ done 10-01 — pack shipped; the ADR numbers were subsequently consumed by other topics (actual ADR-0146/0147 differ), direction later ruled via ADR-0151
34. ~~M15.4 user-action list (GH billing, ERRAUDIT_PAT, evals claude CLI,
    benchkit LICENSE).~~ done 10-01 — in the pack
35. ~~M20.1 matview v2 demand-triage note.~~ done 10-01 (watch pack)
36. ~~M22.1–M22.4 CI observe bundle notes (first runs, weekly leg, shuffle
    watch, CatchUp observe-only).~~ done 10-01 (watch pack)
37. ~~M23.1 turso A+B issue drafts (verify-then-file; HOLD filing).~~ done 10-01 — verified drafts; filing owner-held
38. ~~M23.2 turso-go (a)(b)(c) drafts (HOLD filing).~~ done 10-01 — source-verified on v0.8.1
39. ~~M23.3 md-go-validator upstream asks (owner repo).~~ done 10-01 (filing pack)
40. ~~M23.4 go-graph-rag consumer update draft.~~ done 10-01 (filing pack)
41. ~~F032 named-`[]byte` lint rule in cqrs-lint (unblocked by M10; guard the
    watermill bug class).~~ done 10-01 09:56 — shipped as C043 (6 tests, FP-clean, RULES.md/README)
42. ~~Quiet-gated M04: `preflight-composed.sh` → `can-run-composed-gate
    --wait-loop` → exclusive `nix run .#verify` (dedup row (a) + CI
    re-record rows close GREEN).~~ routed — open at 10-11 (TODO_LIST §CI)
43. ~~Quiet-gated M05: `#integration-mysql-vm` + shuffled-seed replay from
    `build/shuffle-seeds.log` (+ F52/Green-MySQL receipts).~~ routed — open at 10-11
44. ~~Quiet-gated M11: `calibration-gate.sh` → SearchQuery count=5 →
    supersede-note → dgraph constants re-anchor.~~ routed — open at 10-11
45. ~~Quiet-gated M14.3 per-module fresh-run stamp protocol.~~ routed — open at 10-11
46. ~~CHANGELOG section cut for this session's receipts at close.~~ done — receipts cut across the 09-15/10-01 batches; symbol gate green throughout
47. ~~Closing status report (supersede or extend this one) + status-index row.~~ done 09-29 09:15 — successor report `2026-09-29_09-15` + index rows
48. ~~Push master (carried-over owner question).~~ routed — still open at 10-11 (branch ahead 4)
49. ~~`nix fmt` before any lint verdict; treefmt drift check on touched files.~~ done — standing practice since; re-confirmed by the 10-11 triage session (nix fmt as the canonical first move)
50. ~~`.githooks` daemon-shield proposal (see §e.2) into the owner bundle.~~ routed — shield NOT implemented; the damage class recurred (10-10 gci re-add, 10-11 re-add+removal); durable-guard item rowed in the 10-11 triage report §f16

## g) QUESTIONS (cannot figure out myself)

1. ~~**Push `master` now?** It is N auto-commits ahead of origin (carried over
   from yesterday, still unanswered). The tags are already pushed; only the
   master branch ref lags.~~ still open at 10-11 — branch ahead 4; owner never answered
2. ~~**Benchkit tag wave — cut under the standing blanket directive, or hold
   for an explicit go?** (M15.1 lists "benchkit tag-wave timing" as a quick
   ruling; the M09 precedent says blanket = sanction, but this one was
   explicitly posed yesterday and not answered.)~~ ANSWERED by action — benchkit/v4.7.0 tagged in the 10-08 wave
3. ~~**How do you want the quiet-window work (M04/M05/M11) obtained, given
   load never dropped below 14 this session (24/36/44 at close)?**
   (a) I keep auto-polling in future sessions and run them opportunistically
   when `load1 < 5`, (b) you name a dedicated quiet window (e.g. a specific
   night hour) and I run the full quiet batch then, or (c) raise the ceiling
   (the `can-run-composed-gate` threshold is configurable) and accept noisy
   verify evidence.~~ still open at 10-11 — de-facto (a) happened every session without landing a window; owner ruling pending (M15.1)

---

**Standing context:** three questions from the 04-58 report remain unanswered
(push master, benchkit wave, quiet strategy) — re-asked above. Blanket
directive continues to be treated as sanction for wave/tag mechanics per the
M09 precedent; ADR rulings, public filings, billing stay owner-gated.
