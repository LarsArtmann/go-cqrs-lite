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

- [ ] 🔥 **Update `METAENGINE-LAYOUT-PLANNING-MODEL.md`** — design doc still
      says "defaults to embedding"; on-disk calibration data contradicts this
      (Normalize wins WriteSpeed/StorageSpace on KV/LSM). Correct §4.
      _(Effort: S)_
- [ ] **Add calibration-correction addendum to ADR-0124** — record the KV-vs-LSM
      split and the measured ratios (KV: `1.8/0.48/0.63` embed, `0.5/1.0/1.3`
      normalize; LSM: `0.74/1.10/1.15` embed, `1.45/0.75/0.80` normalize).
      _(Effort: S)_
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
- [ ] **Converge `ReplanLayout` into `Store.Replan`** — `SetPriority` calls
      `Replan` but layout diffs require a separate `ReplanLayout` call. The
      in-memory plan now carries layout (`QueryAssignment.Layout`,
      `SerializableQuery.Layout`, `String()` renders it — all fixed), but the
      two planning passes have not merged into one.
      _(Effort: M)_
- [ ] **Refactor `layout_observability.go`** to call shared `resolvePriority`
      directly (currently delegates via `priorityForQuery` — correct but
      inconsistent with the convergence refactor).
      _(Effort: S)_

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

- [ ] 🔥 **Fix Dgraph `CounterBackend` DQL colon bug** — `keyVarDecls()`
      (`metaengine/dgraphengine/counter.go:155`) emits `$key%d string` (missing
      colon). DQL requires `$key%d: string`. `CounterIncrement` is completely
      broken for ≤20 keys. Single-character fix.
      _(Effort: XS)_
- [ ] **Fix Dgraph `JournalReadFrom` seq offset mismatch** — position-based
      resumption is off-by-one for Dgraph `StreamLogBackend`.
      _(Effort: S)_
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

- [ ] **Extract `mergeMostPermissiveProfile` from `doctor.go`** — `doctor.go`
      is 568 lines, exceeds the 350-line CI limit. Move the profile-merge logic
      + helpers into `doctor_profile.go`.
      _(Effort: S)_
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

- [ ] **Delete dead dirs `codec/testdata/` and `codec/reports/`** — testdata
      (fuzz + golden) and reports (coverage.out) remain after the alias
      conversion; nothing references them.
      _(Effort: XS)_
- [ ] **Write ADR for codec extraction** — every prior extraction has an ADR
      (retry → ADR-0064, idempotency → ADR-0065). Next number: ADR-0126.
      _(Effort: S)_
- [ ] **Complete go-codec project scaffolding** — missing `.golangci.yml`,
      `.github/workflows/ci.yml`, FEATURES.md, ROADMAP.md, SECURITY.md (the
      go-retry/go-idempotency repos set the pattern).
      _(Effort: M)_

---

## Deprecated Module Removal

> All four extracted packages (`go-codec@v0.1.0`, `go-idempotency@v0.1.2`,
> `go-retry@v0.3.1`, `go-flightrecorder@v0.2.0`) are published. Their
> `go-cqrs-lite/` shim modules are pure re-export aliases with zero internal
> source consumers. Removing them reduces module count, simplifies the
> workspace, and eliminates maintenance burden.

- [ ] 🔥 **Delete `codec/` shim module** — all source already imports
      `go-codec` directly. Remove dir + `go.work` entry + `flake.nix`
      `testModules` entry + `cmd/api-stability` modules list entry.
      _(Effort: S)_
- [ ] 🔥 **Delete `retry/` shim module** — zero internal source imports.
      Same removal checklist as codec.
      _(Effort: XS)_
- [ ] 🔥 **Delete `idempotency/` shim module** — all source (middleware,
      kvstore, sqlstore, integration, example) already imports
      `go-idempotency` directly. The `idempotency/store_test.go` and
      `property_test.go` tests still reference the shim — delete them too.
      _(Effort: S)_
- [ ] 🔥 **Delete `flightrecorder/` shim module** — zero internal source
      imports. Same removal checklist.
      _(Effort: XS)_
- [ ] **Update `cmd/cqrs-lint` after shim deletion** — module catalog
      ImportHints, E001 tier-0 list, F007/F008 detectors, architecture rules.
      The linter should only recognize the external packages after shims
      are gone.
      _(Effort: M)_
- [ ] **Update AGENTS.md module map** — remove the four shim rows, note
      that the external repos are the canonical source.
      _(Effort: S)_
- [ ] **Update SKILL.md + references** — remove deprecated module docs,
      point consumers to the external repos.
      _(Effort: M)_
- [ ] **Add deprecated `COSESign1String` / `COSEEncrypt0String` backward-compat
      aliases in `codec/alias.go`** — the pre-extraction names were silently
      dropped. Add them before any codec shim removal to avoid breaking
      consumers. (Moot if codec/ is deleted in the v5 cut instead.)
      _(Effort: XS)_
- [ ] **Regenerate API surface golden** after any module deletion —
      `cmd/api-stability` `modules` slice must match the actual module set.
      Meta-test `TestEveryGoModDirIsInModulesList` enforces this.
      _(Effort: XS)_
- [ ] **Run `nix run .#verify` after all deletions** — catch dangling
      references, broken builds, stale golden files.
      _(Effort: S)_

### Cleanup opportunities to consider alongside deletion

- [ ] **Audit `idempotency/kvstore/` and `idempotency/sqlstore/` module
      paths** — should they move out of the `idempotency/` namespace now
      that the parent shim is gone? (e.g., `kvstore/` or keep as-is for
      consumer stability)
      _(Effort: S)_
- [ ] **Consolidate indirect dep references** — after new module tags are
      published, the transitive `go-cqrs-lite/{codec,retry,idempotency,
      flightrecorder}/v4` indirect deps will clean up. Track and verify.
      _(Effort: M)_
- [ ] **Verify `go.work` use block ordering** — external deps should be
      alphabetically consistent after removing local shim modules.
      _(Effort: XS)_

---

## Release / Tagging

- [BLOCKED] 🔥 **Tag `id/v4.3.0` → re-tag record/command/metaengine → tag
  `commandlifecycle/v4.0.0`** — `id.ActorID` (commit `7e374b753`) was never
  released; published `id/v4.2.0` lacks `actor_id.go`, so consumer
  `GOWORK=off` builds of `record/v4`, `command/v4`, `metaengine/v4` fail.
  Fix chain: tag `id/v4.3.0` → re-tag `record/v4.2.0`, `command/v4.5.0`,
  `metaengine/v4.9.0` → tag `commandlifecycle/v4.0.0` +
  `commandlifecycle/projections/v4.0.0` → bump 66 downstream `go.mod` requires.
  Needs user approval (never tag without explicit instruction).
- [ ] **Run calibration benchmarks against baseline** — verify
      `calibration-baseline.md` accuracy; add CI regression check.
      _(Effort: M)_

---

## Code Quality / Infrastructure

- [ ] 🔥 **Clear the stale-GREEN backlog** — multiple 2026-08-11 sessions
      shipped code without running `nix run .#verify` (layout convergence, fold
      inference, ADT coverage, codec extraction, per-module coaching). Run the
      full gate (`#verify`: build + vet + test + race + lint + doc-check +
      doc-assertions) and fix everything it surfaces.
      _(Effort: M)_
- [ ] **Audit/remove duplicate `replace` directives** — `system/go.mod` has 3
      replace directives, `record/go.mod` has 1. Verify each is still needed
      (several were temp workarounds for the ActorID gap).
      _(Effort: S)_
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
- [ ] **Fix `check-arch` module-catalog coverage (94 gaps)** — the coverage
      meta-check in `scripts/check-module-layers.sh` fails on master: ~47
      modules lack `LAYER`/`DEP_BUDGET` entries (e.g. `transport/http`). All
      existing budget checks pass; the gate is red purely on missing catalog
      data. Pre-dates the WAL unification (verified at `6aaca6b0e`).
      _(Effort: M)_
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
