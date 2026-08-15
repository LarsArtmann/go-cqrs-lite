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

## Release / Tagging 🔥

> Blocked on user authorization (never tag/push without explicit instruction).
> The verify gate has been GREEN three times since ADR-0128 (latest:
  2026-08-15, 239 ok packages, lint 76/76) — this is the release checkpoint.

- [ ] [BLOCKED] **Tag the unpublished fixes parked behind replaces** — engine
      self-registration (sqlite/badger/pebble/pg v4.0.1 tags predate it) and
      the watermill `errors.Join` handler-independence fix. Tag engine
      v4.0.2+ ×4 (+ metaengine, system, stack/sqlite, stack/pebble consumers
      as needed — the SQL `JournalReadFrom` positional fix is INVISIBLE to
      consumers until these land; consumers on v4.0.1 double-process on
      resume). Then remove the 5 temporary replaces in `system/go.mod`.
      _(Effort: M)_
- [ ] [BLOCKED] **Cut `command/v4.6.1`** — v4.6.0 was published pinning
      `storage/memory v4.2.0` whose `ReadFrom` bug fails
      `commandtest.TestStoreSuite/ReadFrom` standalone. The pin bump is
      already in the tree; only the tag is missing.
      _(Effort: XS)_
- [ ] [BLOCKED] **Tag final v4.x patches of `transport/http` +
      `transport/grpc`** (deprecation notices included) — prerequisite for
      the v5 deletion (ADR-0127).
      _(Effort: S)_
- [ ] **Create GitHub Releases + trigger pkg.go.dev** for the 2026-08-13
      coordinated release (metadata/v4.4.0, event/v4.6.0, command/v4.6.0,
      query/v4.5.0) and the upcoming engine/watermill tags — never done.
      _(Effort: S)_
- [ ] **Consolidate indirect dep references** — after new module tags are
      published, the transitive `go-cqrs-lite/{codec,retry,idempotency,
      flightrecorder}/v4` indirect deps in ~49 consumer go.mod files clean
      up. Track and verify.
      _(Effort: M)_
- [ ] **Run the pre-tag checklist** — `nix run .#vulncheck` +
      `#check-arch` (verify covered the rest).
      _(Effort: S)_
- [ ] **Run calibration benchmarks against baseline** — verify
      `calibration-baseline.md` accuracy; add CI regression check.
      _(Effort: M)_

---

## Pin & Standalone-Build Hygiene

> Root cause class found twice in two sessions (system replaces; benchkit
> stale pins → SQLITE_BUSY standalone failures). The workspace gate masks
> it: `#verify` resolves local modules, CI runs GOWORK=off per-module — and
> the CI "Benchmarks" job is currently RED.

- [ ] 🔥 **Pin-drift meta-test** — compare each in-repo module's required
      versions of sibling modules against the latest tag
      (`git tag -l '<module>/v4*' | sort -V | tail -1`); fail on staleness.
      Would have caught both known skew classes at test time.
      _(Effort: M)_
- [ ] 🔥 **Repo-wide stale-pin sweep** — benchkit still pins
      `sqliteengine v4.0.1` (pre-JournalReadFrom-fix), `decider v4.3.0`,
      `event v4.6.0`… Mechanical bump of ~50 go.mod files, gate-verified.
      (Needs user sign-off on policy — see ROADMAP Open Questions.)
      _(Effort: M)_
- [ ] **`#verify-standalone` nix app (GOWORK=off per module) or explicit
      decision that CI owns that signal** — then CHECK CI after gates.
      Investigate how long the CI Benchmarks job has been red (`gh run list`)
      to size the blind window.
      _(Effort: M)_
- [ ] **Add CI leg for GOWORK=off standalone builds of leaf modules**
      (integration/, examples/, benchkit/) to catch pin rot early.
      _(Effort: S)_

---

## Metaengine — Layout Planning (Phase 6b)

> Core shipped 2026-08-11: priority system, embed-vs-normalize scoring,
> `ReplanLayout`, `ConfirmRebuild`, runtime backend add/remove, audit trail,
> 16-combination regression test, `cqrs-bench layout` CLI. KV/LSM scoring
> split with 60s on-disk calibration. Layout docs + ADR-0124 addendum
> corrected 2026-08-14. See ADR-0124, ADR-0125.

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
- [ ] **Converge `ReplanLayout` into `Store.Replan`** — two replan entry
      points with overlapping semantics (relayout.go still separate).
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
> rule with latency estimates, engine test parity shipped 2026-08-11.
> `JournalReadFrom` positional fixes (Dgraph + all SQL engines) shipped
> 2026-08-15 with shared-contract regressions.

- [ ] **Seq-carrying journal reads (perf follow-up)** — the OFFSET-based
      positional skip scans past skipped index rows (O(offset) per page);
      true-seq resumption (`JournalReadAllWithSeq` or `StreamLogEntry{Seq,
      Value}`, adapters resume on engine seqs) would make it O(log n) via
      index seek. Correctness is now guaranteed; this is purely a
      large-journal performance item.
      _(Effort: M)_
- [ ] **Engine capability conformance test** — plan-time `Supports`-declared
      vs actually-implemented interface check per engine (brutal review
      finding: 6 engines over-declare; e.g. pg/mysql/duckdb declare Set/Log/
      Multimap/Graph/Vector with no native implementations).
      _(Effort: M)_
- [ ] **DuckDB real aggregation pushdown (`AggregateReader`)** — approved by
      DiscordSync census review; `CounterGet` currently loads all rows into
      Go maps instead of pushing GROUP BY to columnar SQL. Highest-leverage
      DuckDB item.
      _(Effort: L)_
- [x] **mysqlengine vs MariaDB (nspawn) compatibility** — DONE 2026-08-15:
      dialect detection via `SELECT VERSION()` ("mariadb" vs "mysql") at
      engine construction; MariaDB gets `JSON_EXTRACT`/`JSON_UNQUOTE` SQL
      with natively-bound scalar params, MySQL 8 keeps `->` + `CAST(? AS JSON)`;
      functional-index DDL skipped on MariaDB (unsupported, graceful
      degradation). Also fixed the "ADTMatrix fails" class: parallel tests
      shared fixed collections ("events"/"s1") on one server — stream-log
      tests now use unique collections (`RunStreamLogBackendTestIn`, new
      `RunAtomicAppenderTestIn`). Verified live against MySQL 8.4 AND MariaDB
      11.8 (docker), 3x stable each. Known MariaDB limitation: JSON-path
      ORDER BY is text-ordered (multi-digit numbers sort lexicographically).
- [x] **Brute-force vector search on Pebble/bbolt** — DONE 2026-08-15:
      `VectorInsert`/`VectorSearch` on both engines via `keycodec.VectorKey`
      prefix (`vec\x00<col>\x00<id>`, JSON-encoded float32 dims) + exported
      `metaengine.VectorDistance` for numeric parity with MemoryVectorIndex.
      bbolt now declares `ADTVector: ComplexityON` degraded; adttest Vector
      scenario runs against both engines (previously skipped) with
      cross-engine parity.
- [x] **Native graph dispatch on Postgres/MySQL** — DONE 2026-08-15:
      `meta_graph_edges` table + `GraphAddEdge` (INSERT IGNORE / ON CONFLICT
      DO NOTHING) + `GraphNeighbors` via a single `WITH RECURSIVE` CTE
      (depth-capped, DISTINCT, start-node excluded). Profiles upgraded to
      `ComplexityODegree`, no longer degraded — planner emits no DEGRADED
      diagnostic for graph. Verified live: PG 16 (testcontainer), MySQL 8.4,
      MariaDB 11.8. DuckDB intentionally left degraded (follow-up candidate).
- [x] **Recursive CTE optimization for deep traversals** — DONE 2026-08-15:
      sqliteengine probes `WITH RECURSIVE` support once at construction; when
      available (plain SQLite) `GraphNeighbors` runs as a single recursive-CTE
      query instead of one query per node per level. Drivers/servers without
      CTE support (some libSQL/Turso deployments) auto-fall back to the
      iterative BFS — graceful degradation, no operator configuration.
- [ ] **Dgraph engine hardening** — `Transactional` (RunInTx) support or an
      ADR documenting why not; per-test collection isolation for the shared
      persistent server; add Dgraph to `test-all-backends`/CI matrix
      (AGENTS quick-ref lists no Dgraph backend today).
      _(Effort: M)_

---

## cqrs-lint

> Per-module coaching migration complete: all adoption (F003-F029) and
> resilience (B029-B031) rules evaluate per-module. 86 per-module profiles
> verified by `integration_multimodule_test.go`. F001/F002/F005/F014 remain
> workspace-global by design (low leakage risk). F030 (deprecated transport
> imports) shipped 2026-08-14 — 203 rules total.

- [ ] **Add per-module regression tests for remaining migrated rules** —
      F004, F007, F009, F012, F017, F023-F029, B030 lack dedicated per-module
      tests (only F003/F013/F022/B029/B031 + the F006-F021 batch have them).
      _(Effort: M)_
- [ ] **Teach E005 about `system.RegisterCommand`** and regenerate the
      taskmanager lint golden — kills 10 enshrined false positives.
      _(Effort: S)_
- [ ] **Wishlist (parked from consumer feedback rounds)** — `--doctor --fix`
      auto-write; stale-suppression detection as default (not `--strict`
      only); config-disabled rules in health breakdown; feature-profile-aware
      C008 (`monetary: false` → auto-INFO).
      _(Effort: M each)_
- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Track which can be removed after migrations complete.
      _(Effort: M)_

---

## WithActor Hardening

> `event/command/query.WithActor` + `Tracing.ActorID` shipped and released
> (metadata/v4.4.0, event/v4.6.0, command/v4.6.0, query/v4.5.0 — 2026-08-13).
> id/v4.4.0 contains `actor_id.go` (verified in tag).

- [ ] **Document `WithActor` in skill references** — `core.md` options
      section + `modules.md` Tracing fields (consumer-facing gap).
      _(Effort: S)_
- [ ] **Test-coverage gaps** — golden JSON for full `event.Event`/`command.
      BasicCommand` with ActorID; watermill wire-format preservation; CBOR
      roundtrip (events default to CBOR); SQL `MarshalMetadata` scan path;
      pebble/bbolt encode/decode; e2e decider (command→events) and projection
      (events→read) propagation; `TestQuery_AllMetadata`; json/v1 fallback
      `omitempty` behavior.
      _(Effort: M)_
- [ ] **Ecosystem propagation checks** — scenario DSL actor support;
      `scheduling/`, `deriver/`, `commandlifecycle/` ActorID propagation;
      ActorID-from-context middleware; `id.ActorID.Validate()`.
      _(Effort: M)_

---

## Code Quality / Infrastructure

- [ ] **Infrastructure polish (nix apps + shared helpers)** — add
      `#check-lint-config`, `#verify-ci` (mirror GH Actions GOWORK=off
      per-module), wire `#sweep` to pre-commit/cron, consolidate engine
      `register.go` boilerplate (7 modules), add missing devShell tools
      (dprint, go-licenses, vulnix — their absence forces `--no-verify`).
      _(Effort: M)_
- [ ] **Fix `nix fmt` vs golangci-gci import-grouping conflict at the tooling
      level** — the class re-broke 95+ files once; it WILL recur on every
      future import addition. Make the treefmt formatter config aware of gci
      section rules (or vice versa).
      _(Effort: S)_
- [ ] **Enforce the heap-measurement contract mechanically** — tests calling
      `runtime.ReadMemStats` must never `t.Parallel()` (13 files fixed
      2026-08-14; repo-wide audit came back clean, but nothing prevents
      reintroduction). cqrs-lint rule or check-script grep.
      _(Effort: S)_
- [ ] **Audit benchkit's remaining wall-clock assertions** —
      `raceEnabled` branch at `benchkit_test.go:821` may share the
      load-sensitivity mis-model already fixed for `DurationAborts`/
      `CancelledContext` (flat 30s).
      _(Effort: S)_
- [ ] **Wire broker tests into CI** — `TestRedisStreamRoundtrip` passes only
      via manual `ephemeral-redis.sh`; add a `#integration-redis` nix app
      (mirrors `#integration-pg`). Then extend to broker edges the gochannel
      tests can't catch: redelivery duplicates, consumer-group rebalance,
      message size limits.
      _(Effort: M)_
- [ ] **Doc-check 0-warning CI tripwire** — warnings are at 0 since
      2026-08-15; add a regression guard so warning spam can't creep back.
      _(Effort: XS)_
- [ ] **Duplication-baseline hygiene** — add `//art-dupl:accept` directives
      at the 9 intentional clone sites; dirty-tree guard for baseline
      re-pins; re-pin at next clean tree (current pin includes in-flight
      foreign code).
      _(Effort: S)_
- [ ] **check-coverage.sh hardening** — meta-test asserting every EXPECTED
      key resolves to a real module dir (codec-dangle class); make
      `--update` auto-stamp the "verified" date.
      _(Effort: S)_
- [ ] **duckdbengine suite split** — 76-91s observed under the (now 8m)
      ceiling; worth splitting the soak into its own budget.
      _(Effort: S)_
- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin.
      _(Effort: M)_
- [ ] **Build `example/metaengine-quickstart` in CI** — not in flake
      `examplePaths` (getting-started, readme-quickstart, taskmanager are);
      never built by `#verify`/CI.
      _(Effort: XS)_
- [ ] **`storage/backuptest`: wire into bbolt/pebble or delete** — orphan
      module; no engine go.mod depends on it (verified 2026-08-15).
      _(Effort: S)_
- [ ] **Delete junk from repo root** — `t/`, `result/` (16MB root-owned),
      `reports/coverage.out` (empty), `reports/jscpd-report.json`.
      _(Effort: XS)_
- [ ] **One bench system** — keep benchkit + cqrs-bench; delete the redundant
      harnesses (metaengine/bench, integration/ bench files, v2-era baseline);
      make the CI regression check fail on breach.
      _(Effort: M)_
- [ ] **`DecorateJournal` for `VersionedSeekableJournal`** — schema upcasting
      path still hand-wraps Journal+upcasters; the DecorateStore-equivalent
      for journals is the missing piece of ADR-0126.
      _(Effort: M)_
- [ ] **Decide + implement (or permanently drop) `brandedString` extraction
      into `record/`** — the asrecord clone pair is larger than the helper;
      needs a judgment call.
      _(Effort: S)_
- [ ] **Per-module CHANGELOGs** — 6 of ~82 modules have one. Decide policy
      (root CHANGELOG only vs per-module) and either write them or document
      the decision.
      _(Effort: L)_

---

## Correctness Defect Sweep (brutal-review backlog)

> From `docs/reviews/2026-08-14_14-25_brutal-self-review.md` — verified
> findings, not yet fixed. Grouped by module.

- [ ] **SQL injection surface** — `FilterOp`/column allowlists
      (`storage/sql/where.go`); quote ORDER BY columns
      (`storage/view/query.go:137`); stop leaking DSNs in errors
      (`tursoengine/register.go:69`).
      _(Effort: M)_
- [ ] **Resource leaks** — `sqliteengine`/`tursoengine` self-opened
      `*sql.DB` `Close()` leak.
      _(Effort: S)_
- [ ] **Core defects** — singleflight leader-ctx capture (`decider/load.go`);
      per-handler command middleware (`memory_bus.go`); query audit fake
      RequestIDs (`audit.go`); `Pagination.Offset()` underflow; `kv.Cache`
      shared `*T`; TypedQueryStore hardcoded JSON decode (`query/typed.go`);
      ghost `event.ErrBinaryNotFound` (document or delete).
      _(Effort: M)_
- [ ] **Planner cost model** — graph cost `branching^depth`; volume without
      silent default; filter selectivity.
      _(Effort: M)_
- [ ] **Strong types** — `record.NewStreamRef` validation (+ `Split()` on
      `/`); `id` global-mutex ULID entropy (sharded).
      _(Effort: M)_
- [ ] **Security hygiene** — SECURITY.md v3 table stale; govulncheck failures
      swallowed in release.yml; remove iroh fork pin (`git.coopcloud.tech`
      supply-chain flag).
      _(Effort: S)_

---

## Docs Honesty

- [ ] **Reconcile the ADR-0114 tombstone story** — `DeletePolicy` rename never
      landed; FEATURES/CHANGELOG/AGENTS/migration-guide/DOMAIN_LANGUAGE tell
      slightly different stories. Land DeletePolicy or rewrite all four to
      one truth.
      _(Effort: M)_
- [ ] **README feature table honesty** — stop selling tombstone soft-delete
      as the headline capability; lead with what consumers actually import.
      _(Effort: S)_
- [ ] **Skill reference recipes** — catch-up drain pattern (projectionhost
      TOCTOU fix); `WithoutViewAutoMigrate` (README-only today); `Increment`
      non-clamping philosophy (FAQ).
      _(Effort: S)_
- [ ] **README feature-table honesty** — stop selling tombstone soft-delete
      as the headline deletion story (ADR-0114 made deletion a domain event).
      _(Effort: S)_
- [ ] **`integration/README.md`** lists 5 of ~15 suites — enumerate or
      point at the flake apps.
      _(Effort: XS)_
- [ ] **Revive or retire `docs/sessions/SESSION_MILESTONES.md`** — stale
      since 2026-08-11; also fix module-count drift (82 vs 86 vs 88) in
      every doc that hardcodes it.
      _(Effort: S)_

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
      v4.x patch releases of both modules (see Release section), drop
      the modules from `go.work`/flake `testModules`/api-stability list, then
      delete at the v5 cut.
      _(Effort: M)_
- [ ] **Write v5 migration guide** — document the path from v4 (stack presets,
      v1 tiers, transport/*, manual RelationalProjection/view reads) to v5
      (`system.System`, auto-projection, watermill/go-sse delivery).
      Before/after examples for each v1 tier, including
      `relational → metaengine` (consumer-pulled).
      _(Effort: L)_
- [ ] **v5 decision: kvstore SA1019 exclusion** — keep the scoped
      `.golangci.yml` exclusion (`(middleware|idempotency)/.*_test\.go$`) as
      the permanent answer, or migrate the kvstore test matrices onto the
      go-idempotency contract suite before the cut.
      _(Effort: S)_
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
- **`System.WaitReady(ctx)` API** — declined in the file-renamer TOCTOU review:
  the catch-up drain closes the race; a readiness API would be a false
  guarantee. See `docs/feedback/reviewed/2026-08-13_file-renamer_drain-live-toctou-race-review.md`.
- **file-renamer circuitbreaker/dlq modules** — rejected; failsafe-go + the
  FAQ pointer cover it. See
  `docs/feedback/reviewed/2026-08-13_file-renamer_extract-circuitbreaker-and-dlq-review.md`.
