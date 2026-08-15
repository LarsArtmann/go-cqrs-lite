# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- _(Effort: XS/S/M/L/XL)_ = rough size

---

## Metaengine — Layout Planning (Phase 6b)

> Core shipped 2026-08-11: priority system, embed-vs-normalize scoring,
> `ReplanLayout`, `ConfirmRebuild`, runtime backend add/remove, audit trail,
> 16-combination regression test, `cqrs-bench layout` CLI. KV/LSM scoring
> split with 60s on-disk calibration. See ADR-0124, ADR-0125.

- [ ] **Calibrate DuckDB (Columnar)** — the Columnar × ReadSpeed cell is an
      exact tie (2.65 vs 2.65); float-comparison fragility. Run 60s disk bench.
      _(Effort: M)_
- [ ] **Calibrate SQLite/Postgres/MySQL (Row)** — Row-layout multipliers remain
      analytical estimates, not benchmark-derived.
      _(Effort: M)_
- [ ] **Multi-engine integration test with two real backends** — current test
      uses one MemoryEngine + Backfill replay. Need two live engines with data,
      verify both serve correct query results after `AddEngine` + `Backfill`.
      _(Effort: M)_

### Layout roles (long-horizon, depend on a design doc first)

- [ ] **Fold-pipeline sync for Active+DualUse roles** — event → all
      Active+DualUse projections in one engine transaction (strong consistency).
      Needs transactional fold pipeline redesign.
      _(Effort: L)_
- [ ] **Async replication for Backup+Migration roles** — eventual consistency,
      failure-isolated. Needs replication subsystem design.
      _(Effort: L)_
- [ ] **Role transition API** — Backup→Active promote, Migration→Active cutover.
      Depends on the role model above.
      _(Effort: M)_
- [ ] **Real workload trace format** — JSON-lines spec, trace recorder, trace
      player for benchmark calibration.
      _(Effort: M)_
- [ ] **Aggregate boundary config** — `WithSharedCollection("Attachment")`
      opt-in for shared-by-type collections. Needs collection-grouping design.
      _(Effort: M)_
- [ ] **Per-fold mutex instead of global `foldMu`** — current `foldMu`
      serializes all fold execution; per-fold would allow parallel writes across
      different queries. High risk without soak testing.
      _(Effort: M)_
- [ ] **Multi-collection batch atomicity** — when one event triggers folds for
      multiple collections, all writes commit atomically in one engine
      transaction. Replaces `RelationalProjection`'s per-event tx.
      _(Effort: L)_

---

## Metaengine — Universal ADT Coverage (Phase 7)

> StreamLog on Dgraph, native graph on SQLite/Turso (iterative BFS), degraded
      rule with latency estimates, engine test parity all shipped 2026-08-11.

- [x] **Fix Dgraph `JournalReadFrom` seq offset mismatch** — DONE 2026-08-15:
      `JournalReadFrom` now skips `afterSeq` leading entries (positional
      semantics) instead of `gt(seq)` filtering, because Dgraph seqs are sparse
      UnixNano timestamps and the system adapters derive `afterSeq` from entry
      indexes. Exact-count local test + `enginetest.RunStreamLogBackendTest`
      parity wired in. Verified against live Dgraph.
- [x] **Fix `JournalReadFrom` raw-seq filtering on SQL engines** — DONE
      2026-08-15: sqlite/pg/mysql/duckdb filtered `seq > afterSeq` on a seq
      counter GLOBAL across collections, while the contract (and every caller)
      passes a position within the collection's journal → re-delivery whenever
      collections interleaved (e.g. commands written before events). All four
      engines now skip via `OFFSET` over the collection-filtered, seq-ordered
      result. Interleaved-collections phase added to
      `enginetest.RunStreamLogBackendTest(In)`; pg/mysql verified live;
      end-to-end regression `system.TestEventAdapter_ReadFrom_InterleavedCollections`
      (proven failing against the old sqliteengine).
- [ ] **Seq-carrying journal reads (perf follow-up)** — the OFFSET-based
      positional skip scans past skipped index rows (O(offset) per page);
      true-seq resumption (`JournalReadAllWithSeq` or `StreamLogEntry{Seq,
      Value}`, adapters resume on engine seqs) would make it O(log n) via
      index seek. Correctness is now guaranteed; this is purely a
      large-journal performance item.
      _(Effort: M)_
- [ ] **mysqlengine vs MariaDB (nspawn) compatibility** — 3 pushdown tests
      emit MySQL-8 JSON path syntax (`>'$.x' = CAST(? AS JSON)`) that
      MariaDB rejects (Error 1064); ADTMatrix + HealthCheck fail with
      "invalid connection" in that env. Pre-existing (found 2026-08-15 while
      live-verifying the JournalReadFrom fix, which itself PASSES there).
      Decide: MariaDB dialect support or a real MySQL-8 test backend.
      _(Effort: M)_
- [ ] **Brute-force vector search on Pebble/bbolt** — Vector ADT currently
      memory-only. Add degraded O(N) brute-force for LSM engines.
      _(Effort: M)_
- [ ] **Native graph dispatch on Postgres/MySQL** — still degraded
      (`ComplexityON`); SQLite/Turso have native iterative BFS. Add recursive
      CTE variant for engines that support `WITH RECURSIVE`.
      _(Effort: M)_
- [ ] **Recursive CTE optimization for deep traversals** — current SQLite BFS
      is one query per node per level; a single recursive CTE would be faster
      for deep graphs (but libSQL/Turso lack `WITH RECURSIVE`).
      _(Effort: M)_

---

## cqrs-lint

> Per-module coaching migration complete: all adoption (F003-F029) and
> resilience (B029-B031) rules evaluate per-module. 86 per-module profiles
> verified by `integration_multimodule_test.go`. F001/F002/F005/F014 remain
> workspace-global by design (low leakage risk).

- [ ] **Add per-module regression tests for remaining migrated rules** —
      F004, F007, F009, F012, F017, F023-F029, B030 lack dedicated per-module
      tests (only F003/F013/F022/B029/B031 + the F006-F021 batch have them).
      _(Effort: M)_
- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Track which can be removed after migrations complete.
      _(Effort: M)_

---

## Codec Extraction

> `codec/` extracted into standalone `go-codec` repo (published `go-codec@v0.1.0`).
> `codec/` is now a deprecated re-export alias. 53 consumer modules migrated.
> `go.work` replace directive removed; `../go-codec` added to `use` block.

- [ ] **Complete go-codec project scaffolding** — missing `.golangci.yml`,
      `.github/workflows/ci.yml`, FEATURES.md, ROADMAP.md, SECURITY.md (the
      go-retry/go-idempotency repos set the pattern).
      _(Effort: M)_

---

## Deprecated Module Removal

> **DONE 2026-08-14** — all four shim modules deleted; see
> [ADR-0128](docs/adr/0128-extract-codec-and-remove-shim-modules.md) and the
> [status report](docs/status/2026-08-14_20-46_todo-execution-shims-layout-dgraph.md).
> `idempotency/{kvstore,sqlstore}` stay in their existing paths (consumer
> stability — decided, do not revisit). Remaining follow-up:

- [ ] **Consolidate indirect dep references** — after new module tags are
      published, the transitive `go-cqrs-lite/{codec,retry,idempotency,
      flightrecorder}/v4` indirect deps in consumer go.mod files will clean
      up. Track and verify.
      _(Effort: M)_

---

## Release / Tagging

- [ ] 🔥 **Tag the unpublished fixes parked behind replaces** — the old
  BLOCKED tag chain completed upstream (id/v4.4.0 contains `actor_id.go`;
  commandlifecycle/v4.0.0 + projections/v4.0.0 exist), so the old
  `commandlifecycle`/`id`/`record` replaces were removed 2026-08-14. Still
  unpublished: engine driver self-registration (v4.0.1 tags predate it) and
  the watermill handler-independence fix (`errors.Join`). `system/go.mod`
  carries temporary replaces for sqliteengine/badgerengine/pebbleengine/
  pgengine + watermill — remove after tagging engine v4.0.2+ and
  watermill/v4.5.0. Needs user approval (never tag without explicit
  instruction).
- [ ] **Run calibration benchmarks against baseline** — verify
      `calibration-baseline.md` accuracy; add CI regression check.
      _(Effort: M)_

---

## Code Quality / Infrastructure

- [ ] 🔥 **Clear the stale-GREEN backlog** — `system/` standalone (GOWORK=off)
      was red on master: unpublished driver self-registration + watermill
      handler-independence fix (both now parked behind replaces, see
      Release/Tagging) and a missing middleware golden `.json` (fixed
      2026-08-14). Remaining: run the full gate (`nix run .#verify`) after
      large changes and fix anything it surfaces.
      _(Effort: M)_
- [ ] **Infrastructure polish (nix apps + shared helpers)** — add
      `#check-lint-config`, `#verify-ci` (mirror GH Actions GOWORK=off
      per-module), wire `#sweep` to pre-commit/cron, consolidate engine
      `register.go` boilerplate (7 modules).
      _(Effort: M)_
- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin.
      _(Effort: M)_
- [ ] **Write actual Redis/NATS broker roundtrip tests** — the sanctioned
      broker path is `watermill/` + official plugins (ADR-0025 superseded the
      native `transport/nats|redis` modules — see
      `docs/planning/nats-transport-design.md`). Add `watermill-redisstream` +
      `watermill-nats` as test-only deps (excluded from dep budget), then write
      real roundtrips against `ephemeral-redis.sh` / `ephemeral-nats.sh`. The
      current stubs in `watermill/broker_integration_test.go` skip
      unconditionally EVEN when `REDIS_PORT`/`NATS_PORT` is set — replace them,
      they are corpse tests. Verify broker edges the gochannel tests can't
      catch: redelivery duplicates, NATS queue groups, message size limits.
      _(Effort: M)_

---

## v5 Unification (Phase 8: Deletion + Cut)

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Phases 1-7 done (type foundation, dead-code removal, self-registration,
> backend porting, record-typed folds, auto-projection, layout planning,
> universal ADT coverage). Phase 8 is the breaking cut.

- [ ] **Delete `stack.Materialize`** — auto-projection replaces it.
      _(Effort: S)_
- [ ] **Delete `storage.RelationalProjection` + `storage/view` (SQLViewStore)**
      — multi-collection batch atomicity + auto-projection replaces them.
      _(Effort: M)_
- [ ] **Delete `graph.GraphProjection`** — auto-projection + graphadapter
      replaces it.
      _(Effort: S)_
- [ ] **Delete `stack.Bundle` + all 8 stack presets** — `system.System` is the
      only composition root. `stack/` module deleted entirely.
      _(Effort: M)_
- [ ] **Delete `stack.RunProjections`** — `projectionhost.Host` is the only
      projection runner.
      _(Effort: S)_
- [ ] **Delete deprecated compat shells from ADR-0126** — `schema.VersionedStore`
      + `NewVersionedStore`, `signing.Rejecting*` forwarders,
      `encryption.ErrInnerStoreNot*` aliases, `metadata.CustomData`. Internal
      code is already off them (compat tests pin external behavior).
      _(Effort: S)_
- [ ] **Delete `transport/http` + `transport/grpc` modules** (ADR-0127) —
      delivery is `watermill/` + go-sse + cqrs-htmx. `example/taskmanager` is
      migrated (metaengine.ServeSSE on the task_views watcher); cqrs-lint F030
      coaches consumers off the deprecated imports. Remaining steps: tag final
      v4.x patch releases of both modules (deprecation notices included), drop
      the modules from `go.work`/flake `testModules`/api-stability list, then
      delete at the v5 cut.
      _(Effort: M)_
- [ ] **Write v5 migration guide** — document the path from v4 (stack presets,
      v1 tiers) to v5 (`system.System`, auto-projection). Before/after examples
      for each v1 tier.
      _(Effort: L)_
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate.
      _(Effort: M)_

---

## Declined / Rejected (do not re-litigate)

> Full rationale in the linked ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
- **Rewrite `check-module-layers.sh` as Go NOW** — deferred. The script is stable
  (348 lines). Revisit when complexity grows significantly.
- **Fix LogBackend same-nanosecond collision** — `time.Now().UnixNano()` could
  theoretically collide under extreme concurrency (1-in-a-billion). The
  performance cost of an atomic counter per collection is not worth the
  theoretical correctness gain. Accepted tradeoff: a counter may be off by 1.
- **Migrate F001/F002/F005/F014 to per-module coaching** — these (event count,
  schema versioning, catalog) are workspace-global by design; low leakage risk.
