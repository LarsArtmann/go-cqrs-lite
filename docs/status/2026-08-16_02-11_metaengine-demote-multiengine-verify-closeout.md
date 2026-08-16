# Status: Metaengine DemoteEngine + Multi-Engine Integration + Verify Closeout — 2026-08-16 02:11

Session goal: finish the remaining TODO_LIST §Phase 6b + §Layout-roles items —
the multi-engine integration test (two real backends), `DemoteEngine`, docs,
and the verify gate — plus resolve the three open questions left by the
previous session's report
(`docs/status/2026-08-15_22-57_metaengine-layout-calibration-replan-convergence.md`).

## DONE (code, verified GREEN)

### 1. Open questions resolved

- **Q1 (60s DuckDB re-run)**: DONE. Literal `-benchtime=60s` run of
  `BenchmarkColumnarLayoutCalibration_{Embed,Normalize}*` reproduced the
  calibration ratios within 2% (read 2.66x vs 2.62x, write 0.19x vs 0.20x,
  storage shape unchanged). Constants are LOCKED; provenance comment in
  `metaengine/layout_scoring.go` now records the confirmation run.
- **Q2 (MySQL ratios)**: accepted the QEMU-port-forward ratios. Rationale:
  only ratios feed the constants, the geomean is robust to a single engine's
  RTT inflation, and `integration-mysql-nspawn` needs interactive sudo.
- **Q3 (DemoteEngine timing)**: implemented now (below); design addendum
  written into `METAENGINE-LAYOUT-ROLES.md` §4.4 alongside the code.

### 2. `DemoteEngine` — TODO closed

- `metaengine/demote.go` (NEW, 327 lines): `Store.DemoteEngine(ctx, name,
  opts...)` + `WithDemoteRole` (Backup default / Migration) +
  `WithDemoteForce`. Active/DualUse → shadow, the inverse of PromoteEngine.
  - Preflight (`demotePreflight`, no mutation): refuses unknown/shadow
    engines, non-shadow target roles, last-routable-engine demotion, queries
    whose ADT no remaining engine supports, and a missing EventLog.
  - Atomic transition: role flip + replicator registration + EventLog
    snapshot + query re-assignment execute under ONE write-lock section via
    the new `replanWithTransition` hook (store.go), audited trigger
    `engine-demoted` (plan_audit.go).
  - Targeted catch-up: `replayToShadow` (through the replicator's own apply
    semantics via new `replicator.applyJobFilter`) fills the demoted engine's
    never-served collections; `applyReplay` with a served-query filter
    populates the new owners (non-idempotent folds demand `WithDemoteForce`).
    The replicator's applier goroutine starts only after mirror catch-up so
    history lands before buffered live events.
- Atomicity hardening (this is the interesting part):
  - `applyWithRecord` (store.go) now records to the EventLog, dispatches
    primary folds, and fans out replication under ONE read-lock section —
    so the demotion's write-locked EventLog snapshot splits history at
    exactly the routing flip: every event reaches each engine exactly once
    (never as both primary fold AND replay/replication, never neither).
  - `dispatchFoldsLocked` (runtime_backend.go): primary dispatch skips
    engines registered as replicas — closes the stale-assignment window.
  - `PromoteEngine` (roles.go) restructured to drain + flip inside the same
    transition lock (previously: flip under lock, replan separately — a
    straddle window existed).
- Tests (`metaengine/demote_test.go`, 349 lines): happy path (flip, un-route,
  mirror catch-up, re-routed results, live mirroring, audit trigger,
  promote-back), refusals (4 cases incl. EventLog-missing and
  only-routable-engine), non-idempotent guard + forced exactly-once, and a
  200-goroutine concurrent-apply race proving exactly-once under a racing
  demotion. All green with `-race`; full metaengine package green with
  `-race` (`ok 140.435s`).

### 3. Multi-engine integration test — TODO closed

- `metaengine/bench/multi_engine_integration_test.go` (NEW, ~320 lines): the
  Phase 6b "two real backends" test. SQLite (Row) + Pebble (LSM), both
  CGo-free, driven through the FULL cutover lifecycle: plan → apply →
  `AddEngine(RoleMigration)` → `Backfill` (idempotent folds, no force) →
  live mirroring (waitFor) → `PromoteEngine` → `DemoteEngine` (whichever
  engine is NOT serving post-promote) → live mirroring again → final assert:
  BOTH engines serve identical, complete state for all 8 items via direct
  `MapGet` on each engine + `Execute` (handles SQLite's RawValueReader
  JSONValue and decoded-map return shapes).
- Second test: `Backfill` non-idempotent guard fires on live engines too.

### 4. API surface + gates

- api-stability golden regenerated: exactly 4 new exports
  (`DemoteEngine`, `DemoteOption`, `WithDemoteRole`, `WithDemoteForce`) —
  `docs/api_surface.txt`.
- `nix fmt` clean; `check-arch` PASSED (bench go.mod's pgengine/mysqlengine
  deps + the two SQL driver imports were already allowed in depguard and the
  layer map — no config changes needed); doc-check PASSED (868 refs, 41 pkgs).
- Lint: my modules 0 issues (fixed one `musttag` finding by adding the
  `json:"Title"` tag). One REMAINING repo lint finding is foreign (see
  Blocked).

## DONE (docs)

- `docs/planning/METAENGINE-LAYOUT-ROLES.md` §4.4: rewritten from "No demote
  in v1 / future API" to the shipped design (atomicity argument, catch-up
  semantics, no-Backfill-after-Demote warning, PromoteEngine hardening note).
- `docs/adr/0124-operator-driven-layout-planning.md`: new addendum —
  Row/Columnar calibration table (measurement-derived constants), the
  normalize-read sign-flip correction, ReplanLayout convergence (§5), and the
  DemoteEngine atomicity design (§7). Stale "remain analytical estimates"
  bullet updated.
- `docs/planning/METAENGINE-LAYOUT-PLANNING-MODEL.md`: Row/Columnar
  calibration addendum table with per-engine ratios and per-priority winners.
- `metaengine/layout_scoring.go`: provenance comment records the 60s
  confirmation run (within 2%).
- `CHANGELOG.md`: full Unreleased entry (calibrations, DemoteEngine,
  transition atomicity, integration test) + Changed entry for the
  ReplanLayout semantic change.
- `TODO_LIST.md`: Phase 6b items 1–4 + DemoteEngine checked off with
  one-line outcomes; NEW deferred TODO added — make the pre-existing KV/LSM
  calibration benches size-stable and re-derive those constants (known
  append-drift measurement flaw, flagged by the previous session).
- Skill references: `recipes.md` (roles recipe gains the DemoteEngine
  snippet + invariant I5 exactly-once; ReplanLayout comment corrected for
  the applies-config semantics) and `modules.md` (metaengine row now lists
  roles/promote/demote + calibration provenance).

## GREEN evidence (commands + results)

- `cd metaengine && GOWORK=off go test -tags "goexperiment.jsonv2" -count=1 .`
  → `ok 16.395s` (pre-existing items re-verified)
- 60s DuckDB bench → ratios within 2% (constants locked)
- `cd metaengine && GOWORK=off go test -count=1 -race .` → `ok 140.435s`
- `cd metaengine/bench && GOWORK=off go test -run TestMultiEngine -count=3
  -race .` → `ok 1.109s`
- `nix run .#lint` → only the foreign commandlifecycle finding remains
- `nix run .#check-arch` → all passed; doc-check → 868 refs valid
- api-stability `--update` → 4 new exports; `TestAPISurface*` green on
  re-run
- Full `nix run .#verify`: Build ✓ Vet ✓ Test stage FAILED in 3 cmd/*
  packages — transient, see Blocked; all metaengine packages PASSED inside
  that run (metaengine 34.6s, metaengine/bench 17.2s incl. the new tests).
  All 3 packages re-ran green afterwards.

## BLOCKED / foreign interference (not mine, do not "fix")

1. **A second session is concurrently editing this repo.** Evidence: new
   staged files (tursoengine/register_internal_test.go, storage/* injection
   tests), a foreign status report at 01:33, and go.mod replace churn.
   During my `#verify` run, go.mod files briefly contained
   `=> ./event/eventtest` replaces pointing at a nonexistent directory —
   api-stability/cqrs-lint/cqrs-bench tests failed on exactly that
   (`reading ../../event/eventtest/go.mod: no such file or directory`). The
   replaces are gone from the tree now; all three packages re-ran GREEN
   serially afterwards (api-stability ok 1.4s, cqrs-bench ok 194.9s,
   cqrs-lint ok 33.8s). Nothing in my diff caused it.
2. **`commandlifecycle/recorder.go:196` exhaustruct lint finding** —
   introduced by the other session's actor commit `1153c7d11`
   (`event.Metadata` missing fields). Left for that session; blocks a fully
   green `#lint`.
3. **Full-package `-race` on metaengine/bench could not complete**: at 02:10
   the build broke on `sqliteengine/dsl.go:115: se.ownsDB undefined` — a
   foreign mid-edit state that appeared AFTER my green bench run at 02:09.
   Targeted race coverage of my tests is green (above); re-run the full
   package race once the other session's sqliteengine edit lands.
4. The `#verify` gate never reached its lint/doc-check/race stages (aborts at
   Test). Those stages were run individually and are green except items 2–3.

## NEXT STEPS

~~1. Once the concurrent session lands its sqliteengine/tursoengine work:
   re-run `cd metaengine/bench && GOWORK=off go test -count=1 -race .` (full
   package) and `nix run .#lint` (expect only the commandlifecycle finding
   or its fix).~~ done — full `#verify` GREEN (13:15 run #4, see 13-15 report) incl. lint/race stages; the exhaustruct finding fixed at `5b8a9a615`

~~2. Full exclusive `nix run .#verify` on a QUIET tree (per AGENTS.md, never
   concurrent with another session's edits) — everything except the foreign
   findings has passed individually.~~ done — verify run #4 GREEN on a quiet tree (13-15 report)

3. Optional deferred: TODO_LIST's new "size-stable KV/LSM calibration
   benches" item (re-derive KV/LSM constants without append-drift).
~~4. If desired: `git tag` per-module releases per CONTRIBUTING release
   process (metaengine has new exports — minor bump, non-breaking).~~ done — `metaengine/v4.11.0` + 8 engine tags + `watermill/v4.5.0` tagged 2026-08-16 (chain)

## What I discovered / lessons

1. **The demote double-apply problem is solved by lock structure, not
   sequencing**: record+dispatch+replicate under one read lock + transition
   under one write lock makes the EventLog snapshot split history at exactly
   the routing flip. No skip-sets, no watermarks, no epoch tracking needed —
   the original session's "skip-set on replicate" design sketch was
   unnecessary (and insufficient: it wouldn't have covered the re-routed
   replay side).
2. **Post-promote non-idempotent caveat**: `WithDemoteForce` assumes the
   receiving engines never held the re-routed collections. After a
   promote-cycle (the new owner previously mirrored those collections),
   forced demote would double-count non-idempotent folds — recover via
   remove+re-add+backfill instead. Documented in §4.4; the integration test
   deliberately uses idempotent folds, where the whole class vanishes.
3. **Execute/MapGet return shapes vary by engine**: SQLite's RawValueReader
   fast path returns `metaengine.JSONValue` (raw JSON bytes) from Execute,
   and MapGet returns `map[string]any` — tests must normalize both shapes.
4. **Auto-commit daemon + concurrent session**: `nix fmt` (treefmt) rewrote
   files between my read and edit (mtime bump failures), and mid-verify
   go.mod churn produced a false-failing gate. Multi-session repos: expect
   transient red, always re-run failures serially before diagnosing your own
   code. The exit-code-after-pipe lie (AGENTS.md) applies to tail-piped
   verify output too — capture to a file.
5. **SQLite per-connection locking**: `newSQLiteEngine` sets
   `SetMaxOpenConns(1)`; the integration test holds one store per SQLite DB,
   which sidesteps modernc/sqlite writer contention entirely.
