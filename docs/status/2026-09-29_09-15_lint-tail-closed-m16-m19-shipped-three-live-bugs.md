# Status Report — 2026-09-29 09:15: lint tail closed, M16+M17+M18+M19 shipped, 3 real bugs fixed

> Session window 07:02–09:15 CEST 2026-09-29. Continuation of the SUPERB
> post-wave plan (`docs/planning/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md`).
> Owner directive: full-list execution; ADR rulings/filings/billing prep-only.
> **Ended WAITING FOR INSTRUCTIONS** — questions at the end.

## What this session did

### Heavy-gate tail (carried over)

- **Full lint closed GREEN.** The prior session's 285-finding red was 100%
  phantom gci; the restored config left **8 real findings**, all fixed:
  exhaustive switch (`scheduling/sqlstore` gained `DialectDuckDB` — claim-core
  alias + explicit rejection, api golden regen'd) + 7 `stringscut/bytescut`
  modernize migrations to `CutLast` (storage/bbolt, cmd/cqrs-lint ×6).
  Also repaired `cmd/cqrs-lint/testdata/typedfixture/go.mod` (daemon pin-sweep
  missed it: go-error-family v0.10.1→v0.11.0) — that was the real cause of
  the version-rules test failures, not my segmentation edits.
- **154-file daemon commit triaged benign**: my `.golangci.yml` repair +
  routine dep UPgrades. Hash gate PASS, `check-go-version.sh` PASS (no
  recurrence #3).
- **`#check-duplication` GREEN** after consolidating a verbatim
  `dedupe`/`dedupeStrings` twin pair in cmd/doc-check (same package — real
  dedup, not an accept). Baseline 187, 0 new clones.

### M16 — queue M4 tail: DONE (design-gated half + sweep)

- **M16.1 clock seam — design doc**:
  `docs/planning/2026-09-29_07-20_queue-conformance-clock-seam-design.md`.
  All 10 fixed sleeps inventoried (60–650 ms; all lease/delay-lapse waits, one
  2 ms ms-truncation boundary). Options A (status quo minus), B (internal
  `nowFn` seam — recommended), C (exported `queue.WithClock` at v5).
  **Implementation is owner-gated** (question below).
- **M16.2 shared-DB parallel-migrate sweep — root cause FIXED**. Exposure
  matrix: pgtestcontainer-based suites (storage, relational, projectionhost,
  benchkit, pgengine) are per-test-database isolated incl. under explicit
  DSN; the live race was `metaengine/claimkit`'s raw-shared-DSN parallel pair
  - the production paths themselves (rolling deploys race the same DDL).
    Fix: **advisory-lock DDL serialization** (`pg_advisory_xact_lock
  0x63717273`) at all three PG DDL sites — `storage.PostgresInitSchema`
    (new `storage/postgres_ddl.go`), `pgengine` init + planned layouts (new
    `ddl_lock.go`), `claimkit` pg dialect (new `ddl_lock.go`). Three 8-way
    concurrent-construction regression tests added; **all green against
    ephemeral PostgreSQL** (3 targeted legs). MySQL: zero parallel+DSN exposure
    (verified). Migration step (`MigrateSnapshotColumnsToStream`) already
    concurrency-aware — left as-is.
- **M16.3**: TODO queue-M4 row struck with full receipts.

### M17 — system test-mass gap (Feedback #6): DONE, 2 real bugs found+fixed

- **M17.1 table tests** over the koanf/YAML surface (priority 3 levels +
  inline, materialized views + manifest path, mixed-pool instances, buses,
  nil-map invariants, error paths, legacy-env interplay). The tests FOUND two
  LIVE `system.LoadConfig` bugs:
  1. The documented `CQRS_INSTANCES__0__DURABILITY` override was **silently
     dropped** (koanf's env provider cannot index slices). Fixed:
     `applyIndexedInstanceEnvOverrides` post-unmarshal (durability/role/
     engine), loud errors on out-of-range indices and unknown fields.
  2. Setting such a var **corrupted the YAML instances list** (koanf merged
     the map-shaped `instances.<i>.<field>` key into the list and truncated
     it). Fixed: env provider excludes `CQRS_INSTANCES__*`.
- **M17.2 rapid properties** ×3 (YAML round-trip, env-beats-YAML, indexed
  override with in-range/out-of-range draws). rapid v1.3.0 added (test-only).
- **M17.3 real-engine lifecycle stress** (`systemtest`, Feedback-#4 split
  honored): 6 rounds × 8 concurrent creates on real sqlite → Count
  double-apply sentinel must equal dispatches exactly → racing
  GracefulClose/Close. Green.
- **M17.4 wiring determinism**: two identical constructs → byte-identical
  `system.Explain`. Green. Feedback-#6 row struck; CHANGELOG (41 symbol
  citations green).

### M18 — temporal property tests: DONE (+scope note)

- Three rapid properties in `metaengine/temporal_property_test.go`, all vs a
  reference model with explicit timestamps (deterministic, no sleeps):
  out-of-order + same-ts LWW collapse (shuffled application order; probes at
  every event ts and midpoints), retention-never-prunes-newest (MaxVersions=1
  adversarial + MaxAge windows), tombstone-as-of visibility (set→tombstone→
  rebirth at ±1ns boundaries). Green ×100 draws. Model subtlety pinned: the
  same-ts winner is the last-APPLIED write (application order, not value
  order).
- **M18.4 pebble/bbolt scope decision one-pager**:
  `docs/reviews/2026-09-29_pebble-bbolt-versioned-cells-scope-decision.md`
  (effort ~M/engine, the real cost is retention + the tiebreak-byte one-way
  door; recommendation DEFER with demand trigger). Owner ruling A–D pending.
  TODO row annotated.

### M19 — NATS JetStream leg: DONE (4/4 green against a real broker)

- `watermill-nats/v2 v2.2.0` added (test-only). Four tests:
  `TestNatsJetStreamRoundtrip` (bridge EventBus+CommandBus through real
  JetStream — the NATS sibling of the redis leg), Nack redelivery, group
  exactly-once (TrackMsgID + GroupedConsumer), 2 MiB payload roundtrip.
  All **green** via `nix shell nixpkgs#nats-server -c bash
  scripts/ephemeral-nats.sh ...`.
- Script fixes: stale "no maintained plugin" comment in
  broker_integration_test.go corrected; `ephemeral-nats.sh` now raises
  max_payload to 8MB via config file (no CLI flag exists in nats-server
  2.14.7 — CLI attempt failed).
- **Upstream plugin gaps discovered and worked around in-test** (documented
  in the test file): (a) stream AND consumer names derive verbatim from the
  topic → dotted bus topics (`cqrs.events`) are illegal names; the test ships
  a sanitized-name `StreamConfigurator` + custom `ResourceInitializer`;
  (b) subscribers never provision streams (GET only) — streams must
  pre-exist; (c) the plugin routes by stream-name-as-subject (marshaler takes
  `streamConfig.Name`). These belong in the watermill skill's backends.md —
  NOT YET WRITTEN (next step).
- `#integration-nats` flake app added (mirrors integration-redis; app
  evaluates; CI-leg decision folded into the questions). Lint clean, tidy
  done.

## What is NOT done (honest remainder)

- **M19 tail**: watermill skill backends.md gotcha entry + CHANGELOG M18/M19
  entries + doc-check pass — NOT yet written (next session, ~15 min).
- **Not re-run after today's edits**: `#check-duplication` (new test code
  since the green), api-stability verify pass (no exported changes today
  beyond the already-regen'd DialectDuckDB), `nix flake check` (deferred —
  load 41/30/23 ALL session, quiet gates never opened).
- **Untouched**: M21 (graph unify), M24 (enricher), M25, M26, M15 prep
  bundles, M20/M22/M23 bundles, F032, quiet-gated M04/M05/M11/M14.3.
- **NATS suite ran green ONCE** — no repeat/flake check yet.
- **Push**: not done (Q1 below). **Benchkit tag wave**: not cut (Q2 below).

## What I forgot / could do better

1. Repeated the `rg -rn` replace-flag mistake TWICE (garbled output, wasted
   cycles) and `source ../../scripts/go-env.sh` path errors twice in nested
   dirs — muscle memory both; the env-chain gotcha doc deserves a one-liner
   about "pwd first".
2. Four avoidable compile iterations on the NATS/property tests from sloppy
   multiedit anchors (ctx-declaration removal, const-after-use,
   ensure-before-const, gofumpt drift). Should have written the file once,
   read it, then edited.
3. Verified the stress test only under -count=1; no race-detector pass on the
   new system/stress tests this session (race lives in #verify, which is
   quiet-gated).
4. `waitFor` silently times out (redis-suite helper pattern) — my nats
   roundtrip copies it; the assert-after-wait saves correctness, but a
   wait-with-fail helper would localize failures faster.
5. No opportunistic quiet-window polling loop was set up despite knowing
   load gates were closed — a background load-watcher could have auto-flagged
   the first sub-5 window.

## Questions for the owner (blocking)

1. **Push master now?** (many daemon commits ahead of origin; all tags
   already pushed)
2. **Benchkit tag wave** — cut under the blanket sanction (M09 precedent) or
   hold for explicit go?
3. **Quiet-window strategy** for M04/M05/M11/M14.3 (load never dropped below
   14 today): (a) I auto-poll opportunistically, (b) you name a window,
   (c) raise the ceiling?
4. **Design rulings pack** (one class, one reply): M16.1 clock seam B-vs-C ·
   M18.4 pebble/bbolt A–D · M19 CI-leg (add to ci.yml now vs billing-gated
   later).
