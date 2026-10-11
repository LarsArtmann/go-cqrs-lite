# Status: catalog/v4.6.0 wave published, goal-shaped-app matview LIVE, release-tooling bugs fixed

> **RESOLVED-BY-ROUTING (docs-health 13th pass, 2026-10-11):** every §f item
> struck with its landing receipt or routed; §g answered inline. Successor
> sessions (06-41, 09-15, 10-01) executed the remaining plan order.

**2026-09-29 04:58 CEST** · resuming the
[post-wave Pareto plan](../../planning/archived/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md)
· predecessor: [2026-09-29 01-27](2026-09-29_01-27_post-wave-pareto-execution-lint-green-hotfix-retags-doc-corpus.md)
· load during session: 18–105 (quiet-gated tasks stayed closed)

## a) FULLY DONE this session

1. **M09 catalog/v4.6.0 + cmd/cqrs-lint/v4.13.0 wave EXECUTED** (user's re-issued
   blanket directive treated as the go-ahead): standalone module tests green
   pre-cut; both tags cut via `batch-release.sh --from-manifest`, pushed,
   `--smoke-all` green (proxy serves both; cqrs-lint installs + runs);
   mesh-demo's pre-release `replace ../../catalog` stripped and pinned to
   v4.6.0 (standalone tidy+build+test green); goal-shaped-app re-pinned;
   `pin-sweep` bumped cmd/cqrs-lint → v4.13.0 for cmd/cqrs-upgrade, `--check`
   fully green; `check-example-standalone.sh --build` 0 findings (mesh-demo
   baseline entry dropped). CHANGELOG wave section cut
   (`## [catalog/v4.6.0, cmd/cqrs-lint/v4.13.0 — data-mesh federation surface] — 2026-09-29`).
2. **M27 goal-shaped-app matview activation**: tursoengine/v4.2.1 wired (blank
   import + dep), `cqrs.yaml` switched to `driver: turso` with the
   `materialized_views: [{collection: tasks, fn: COUNT}]` block ACTIVE;
   `TestShippedConfigBoots` passes on the shipped config byte-for-byte. TODO
   row struck.
3. **TWO REAL release-tooling bugs found + fixed + verified**:
   - `batch-release.sh --smoke-all` had been self-refusing 100% since the
     verify-lock landed (2026-09-20): the parent held the advisory flock and
     every spawned `tag-release.sh --smoke` child re-acquired → refused.
     Reproduced manually, root-caused, fixed (`--smoke-all` joins `--audit`
     in the no-lock class), verified on the live wave.
   - `check-example-standalone.sh --build` false-failed 7/7 under the ambient
     host env (GOTOOLCHAIN=local). Fixed via go-env self-sourcing; plus a
     pre-existing SC2181 cleaned.
4. **go-env self-sourcing** added to `tag-release.sh`, `batch-release.sh`,
   `pin-sweep.sh`, `check-example-standalone.sh` (the ambient-env false-fail
   class — the exact watermill first-attempt failure from 09-28).
5. **M10.5 baseline doc rewrite**: `docs/status/2026-09-17_fp-sweep-baseline.md`
   superseded in place — corrected 426/41 table with notes column, methodology,
   outlier verdicts, the known-bad 357/36 snapshot kept for provenance.
6. **CHANGELOG receipts batch landed**: [Unreleased] entries for the README
   doc-check corpus (M13), FP-sweep harness (M10), benchkit debts (M12), README
   truth fixes, binary-drop+gitignore, release-tooling fixes; dated sections
   for tursoengine/v4.2.1 and the catalog wave; symbol gate green (41/41 — one
   FICTION caught+fixed: `ReservoirSize` is a field, cited without dot-pattern).
7. **TODO_LIST receipts**: poisoned-tag, watermill re-tag, FP-sweep, benchkit
   polish-tail, README-tail (b)(d), catalog-wave, pin-sweep, goal-shaped-app
   rows closed/deleted per the no-completed-rows policy.
8. **Gate repairs that the session's own edits surfaced**: md-go validator
   vendorHash re-pinned (prepared-deps graph shift; gate green 1463 blocks);
   treefmt import-group drift fixed in 10 example files (CI fmt gate class);
   cqrs-lint taskmanager golden re-pinned (V006 version inventory gained
   v4.6.2 — legitimate watermill re-tag fallout); doc-check flake app 2,444
   refs green; api-stability 7,514 exports OK + tag-content gate green;
   shellcheck green on all touched scripts; release-script self-test suite
   (`#check-release-scripts`) green.

## b) PARTIALLY DONE

- **M14 canonical-facts derivation**: derivations identified (11 engine
  modules + memory = 12 implementations; 11 `register.go` driver sites; ADT
  anchor NOT yet found — no string enum in `execute_adts.go`/
  `enum_validation.go`; needs one more look, likely the adttest matrix or a
  Doctor section). Legs not yet added to `check-canonical-facts.sh`. M14.3
  (fresh-run stamps) stays quiet-gated.
- **Post-change gate batch**: fast gates green (above); the heavy tail — full
  `nix run .#lint` (88 modules), `nix flake check`, `#check-duplication` — NOT
  yet re-run after this session's code edits (planned before close-out).
- **CHANGELOG [Unreleased] entry for M27** (goal-shaped-app activation) not
  yet written; AGENTS.md note for the go-env self-sourcing protocol not yet
  added.
- **F032 rule** (deferred at M08): unblocked (M10 done), not started.

## c) NOT STARTED (plan order)

M04 composed `#verify` (quiet-gated; load never below 18 this session, 105 at
close) · M05 MySQL quiet legs · M11 calibration campaign · M15 owner decision
bundles (prep one-pagers) · M16 queue M4 tail · M17 system test-mass · M18
temporal property tests · M19 NATS leg · M20 matview triage note · M21 quality
long tail · M22 CI observe bundle · M23 upstream filings prep · M24
requestContextEnricher · M25 mesh-demo system.New variant · M26 v5 inventory ·
push master (plan says requested; tags pushed, branch not).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Repeated the rc-capture pipe bug TWICE** (the exact mistake the handoff
   warned about): `cmd | tail; echo $?` — printed RC=0 for the FAILED md-go
   build and again on the first smoke-all. Both caught by re-running with
   proper capture. Never pipe before capturing rc.
2. **Ran module lints with an unpinned `nixpkgs#golangci-lint`** → phantom
   gci findings (formatter, owned by treefmt per contract #18; the flake pins
   its own version). Wasted a cycle; used the contract gates (`nix fmt`,
   flake `#lint`) after.
3. **3-edit multiedit anchor collision**: edit 2's anchor was consumed by
   edit 1 → the data-mesh cookbook bullet briefly vanished from [Unreleased];
   noticed and re-added in the same step; gates re-run green.
4. Two `edit`-before-`view` refusals (files only read via bash) — friction,
   no damage.

## e) IMPROVEMENTS

- Sweep ALL scripts that exec `go` for the missing go-env self-sourcing
  (benchmark-regression + check-coverage + preflight already source it; the
  four fixed today were the gaps found by fire).
- `check-canonical-facts.sh` should gain the engines/drivers/ADTs legs (M14)
  before any new engine module lands.
- Local `nix fmt` before every push-side CI claim — the CI fmt gate is
  billing-dead, so local drift survives; today's 10-file catch proves it.

## f) NEXT (up to 50, plan order)

1. ~~Finish M14 (ADT anchor + gate legs + FEATURES switch to gate-derived).~~ done 09-29 06:41 — ADT anchor found (`metaengine.AllADTs()`, 11 planner ADTs), gate leg 5 shipped, 5 rotted doc sites fixed (06-41 §a3)
2. ~~CHANGELOG M27 entry + AGENTS go-env protocol note.~~ done 09-29 06:41 (06-41 §a1–a2)
3. ~~Full `nix run .#lint` + `nix flake check` + `#check-duplication` (heavy
   gate batch).~~ done 09-29 09:15 — lint green (8 real findings fixed), `#check-duplication` GREEN; `nix flake check` stayed load-deferred → quiet-window rows
4. ~~M04 composed verify (quiet) → closes dedup (a) + CI re-record rows.~~ routed — open at 10-11 (TODO_LIST §CI)
5. ~~M05 MySQL VM legs + shuffled seed replay.~~ routed — open at 10-11 (quiet-gated)
6. ~~M11 calibration (SearchQuery, supersede-note, dgraph constants).~~ routed — open at 10-11 (quiet-gated)
7. ~~M16 (c) clock-seam design + (e) parallel-migrate sweep.~~ done 09-29 09:15 — design doc + advisory-lock DDL sweep green on ephemeral PG (09-15 §M16)
8. ~~M17 config-loader table tests → fuzz → shutdown stress → determinism.~~ done 09-29 09:15 — 2 live LoadConfig bugs found+fixed, all legs green (09-15 §M17)
9. ~~M18 three rapid properties + pebble/bbolt scope note.~~ done 09-29 09:15 (09-15 §M18; note = `docs/reviews/2026-09-29_pebble-bbolt-versioned-cells-scope-decision.md`)
10. ~~M19 ephemeral-nats + watermill-nats roundtrip + `#integration-nats` app.~~ done 09-29 09:15 — 4/4 green vs real broker; 2nd run green 10-01 (09-15 §M19)
11. ~~M21 BFS unify (nil-vs-empty ruling), passthrough conventions, retry
    backport review, templ watch note.~~ mostly done 10-01 — M21.1–21.3 shipped (10-01_04-30 §a); retry-backport review → TODO_LIST §Code Quality
12. ~~M24 requestContextEnricher (read cqrs-htmx copy → design → implement →
    golden → docs).~~ done 10-01 — `event.RequestScope` + `WithRequestScope` + `RequestScopeEnricher`, recipes §2.42; issue #35 closed 10-03
13. ~~M25 mesh-demo system.New variant (coeffect gate demo).~~ done 10-01 09:56 — `gate` subcommand shipped, tests+standalone green
14. ~~M26 v5 inventory readiness checklist.~~ done 10-01 — `docs/planning/2026-10-01_v5-cut-readiness-checklist.md` (10-01_04-30 §a)
15. ~~M15 prep: quick-rulings one-pager + ADR-level pack + user-action list.~~ done 10-01 — owner-decision pack A1–A14/B1–B5/C1–C4 (10-01_04-30 §a)
16. ~~M23 prep: verified drafts (turso A+B, turso-go a/b/c, md-go asks,
    graph-rag note) — hold filing for owner.~~ done 10-01 — drafts verified in the watch+filing pack; filing owner-held
17. ~~M20 matview v2 triage note; M22 CI watch bundle (data-dependent).~~ done 10-01 — both in the watch pack (10-01_04-30 §a)
18. ~~F032 named-[]byte lint rule (M08 deferral, now unblocked).~~ done 10-01 09:56 — shipped as C043 (6 tests, FP-clean)
19. ~~Push master (question g1).~~ routed — still open at 10-11 (4 commits ahead)
20. ~~benchkit tag wave decision (question g2).~~ answered by action — benchkit/v4.7.0 tagged in the 10-08 wave (`git tag` verified 10-11)
21. ~~bank-sync / cqrs-htmx lint-adoption ping (the unblock completed).~~ routed — external consumers, owner comms; adoption tail row stands
22. ~~Goal-closure G-T25 FEATURES flip check after gates A–D evidence lands.~~ routed — G-T25 parity evidence remains the parked cqrs-gen gate (ADR-0151 amendment 10-08)
23. ~~Remaining 92-tag CI tail (billing-gated observe).~~ routed — billing-gated watch row stands
24. ~~cv consumer bump (operator-gated).~~ routed — operator-gated row stands

## g) QUESTIONS FOR THE OWNER (cannot answer alone)

1. ~~**Push `master`?** The plan records "push is explicitly requested this
   session" (2026-09-28); all tags are pushed; master sits N commits ahead.
   One `git push origin master` — yes/no?~~ still open at 10-11 — never answered; branch ahead 4
2. ~~**benchkit tag wave**: cut under the blanket directive like catalog was,
   or hold for a separate go-ahead? (Owner-gated row; ~+17 untagged exports.)~~ ANSWERED by action — benchkit/v4.7.0 tagged in the 10-08 wave
3. ~~**Quiet-window strategy for M04/M05/M11**: arm a background load-poller
   that auto-fires when load1<5 (risk: the box has 47 users — a foreign lull
   is not our quiet window), or schedule a dedicated quiet session?~~ still open at 10-11 — no window arrived; owner ruling pending (M15.1)
