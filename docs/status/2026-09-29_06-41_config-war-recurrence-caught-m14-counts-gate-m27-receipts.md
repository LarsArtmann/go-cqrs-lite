# Status 2026-09-29 06:41 — config-war recurrence caught, M14 counts gate shipped, M27 receipts closed

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
     + FEATURES:1493 "10 ADTs" → 11 planner ADTs; FEATURES:1500 driver
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

1. Await `#lint` re-run result (background, `/tmp/lint-full-2.log`) — expect
   green with the restored config.
2. `nix run .#check-duplication` (clean tree required — daemon permitting).
3. `nix flake check` (heaviest leg; consider deferring into M04's quiet
   window if load stays >5).
4. M16.1 clock-seam design proposal (ADR-0122 `WithClock`, delete fixed
   sleeps) — design doc under `docs/planning/`.
5. M16.2 shared-DB parallel-migrate sweep (`t.Parallel` + shared DSN across
   engine suites).
6. M16.3 queue M4 receipts + TODO row strike.
7. M17.1 system config-loader table tests (koanf/YAML + env merge).
8. M17.2 config-loader rapid fuzz.
9. M17.3 lifecycle/shutdown stress with real engines.
10. M17.4 determinism test (same domain+deployment → identical wiring).
11. M17.5 Feedback-#6 row receipt.
12. M18.1 rapid property: out-of-order stamps + same-ms LWW collapse.
13. M18.2 rapid property: retention-never-prunes-newest.
14. M18.3 rapid property: tombstone-as-of visibility.
15. M18.4 pebble/bbolt join-scope decision note (owner question).
16. M19.1 `scripts/ephemeral-nats.sh` (mirror `ephemeral-redis.sh`).
17. M19.2 watermill-nats roundtrip test (mirror redis leg: roundtrip, Nack
   redelivery, group exactly-once, 2 MiB payloads).
18. M19.3 `#integration-nats` flake app + CI-leg decision.
19. M21.1 `graphNeighborsFallback` → `GraphBFS` unify (nil-vs-empty decision
    + typed-key param).
20. M21.2 ephemeral-script passthrough conventions unify.
21. M21.3 contention-retry backport review (turso/badger transient-abort
    class).
22. M21.4 templ clone-group watch note (no lever until art-dupl templ
    support).
23. M24.1 read local copy of cqrs-htmx `audit_context.go`.
24. M24.2 design `event/` enricher API (`WithEnricher` surface).
25. M24.3 implement + tests + api golden regen in the same edit.
26. M24.4 core.md section + cqrs-htmx TODO note.
27. M25.1 mesh-demo system.New coeffect-gate demo design.
28. M25.2 implement the variant (runtime `DomainConfig.Events` gate demo).
29. M25.3 verify + core.md §9 row + receipts.
30. M26.1 v5 inventory: every v5-gated row → readiness checklist (deps,
    order).
31. M26.2 scan-default flip note (execute ON the v5 branch per ADR-0123).
32. M15.1 quick-rulings one-pager (M22/Q3, push cadence, claiming V006,
    iroh P99, T18b (a)(b), DSN strict, sync/embedded scope, dgraph one-RPC,
    CapabilityGaps→Doctor, `#test-examples` gate, docs-health cadence,
    benchkit tag-wave timing).
33. M15.2 ADR-level rulings pack (SingleWriter ADR-0146, Direction ruling
    ADR-0147, v5-encryption 4 questions, ADR-0138 demand-check).
34. M15.4 user-action list (GH billing, ERRAUDIT_PAT, evals claude CLI,
    benchkit LICENSE).
35. M20.1 matview v2 demand-triage note.
36. M22.1–M22.4 CI observe bundle notes (first runs, weekly leg, shuffle
    watch, CatchUp observe-only).
37. M23.1 turso A+B issue drafts (verify-then-file; HOLD filing).
38. M23.2 turso-go (a)(b)(c) drafts (HOLD filing).
39. M23.3 md-go-validator upstream asks (owner repo).
40. M23.4 go-graph-rag consumer update draft.
41. F032 named-`[]byte` lint rule in cqrs-lint (unblocked by M10; guard the
    watermill bug class).
42. Quiet-gated M04: `preflight-composed.sh` → `can-run-composed-gate
    --wait-loop` → exclusive `nix run .#verify` (dedup row (a) + CI
    re-record rows close GREEN).
43. Quiet-gated M05: `#integration-mysql-vm` + shuffled-seed replay from
    `build/shuffle-seeds.log` (+ F52/Green-MySQL receipts).
44. Quiet-gated M11: `calibration-gate.sh` → SearchQuery count=5 →
    supersede-note → dgraph constants re-anchor.
45. Quiet-gated M14.3 per-module fresh-run stamp protocol.
46. CHANGELOG section cut for this session's receipts at close.
47. Closing status report (supersede or extend this one) + status-index row.
48. Push master (carried-over owner question).
49. `nix fmt` before any lint verdict; treefmt drift check on touched files.
50. `.githooks` daemon-shield proposal (see §e.2) into the owner bundle.

## g) QUESTIONS (cannot figure out myself)

1. **Push `master` now?** It is N auto-commits ahead of origin (carried over
   from yesterday, still unanswered). The tags are already pushed; only the
   master branch ref lags.
2. **Benchkit tag wave — cut under the standing blanket directive, or hold
   for an explicit go?** (M15.1 lists "benchkit tag-wave timing" as a quick
   ruling; the M09 precedent says blanket = sanction, but this one was
   explicitly posed yesterday and not answered.)
3. **How do you want the quiet-window work (M04/M05/M11) obtained, given
   load never dropped below 14 this session (24/36/44 at close)?**
   (a) I keep auto-polling in future sessions and run them opportunistically
   when `load1 < 5`, (b) you name a dedicated quiet window (e.g. a specific
   night hour) and I run the full quiet batch then, or (c) raise the ceiling
   (the `can-run-composed-gate` threshold is configurable) and accept noisy
   verify evidence.

---

**Standing context:** three questions from the 04-58 report remain unanswered
(push master, benchkit wave, quiet strategy) — re-asked above. Blanket
directive continues to be treated as sanction for wave/tag mechanics per the
M09 precedent; ADR rulings, public filings, billing stay owner-gated.
