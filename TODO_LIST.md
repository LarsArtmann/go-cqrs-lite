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

## Harvested from docs-health audit (2026-08-29)

> Verified still-true gaps surfaced while annotating and archiving ~450
> 2026-08 reports. Every item was re-verified against code on 2026-08-29.
> Source reports now live under `docs/status/archived/`.

**Metaengine**

- [ ] **pgEngine/mysqlEngine `LayoutPlanApplier` + planned-layout schema evolution** — SHARPENED 2026-08-29 (session 2): verified NEITHER engine has ANY planned-table support (no ApplyLayoutPlan/registerLayout/planned read-write paths at all) — this is not an evolution patch but a full port of sqliteengine/planned.go (302 lines) + duckdbengine/layout_planner.go (349 lines) to two dialects: CREATE TABLE from LayoutPlan.ColumnType, typed write path (no name heuristics), planned reads, information_schema-driven ALTER TABLE ADD COLUMN evolution, planned filter/sort pushdown. Effort: L (needs ephemeral PG + MariaDB matrix). SQLite/DuckDB appliers exist.
      _(Effort: M)_ — sources: status 2026-08-02_21-18/22-17
- [✓] **DuckDB counter SQL pushdown** — DONE 2026-08-29 (todo-execution session 2) — `CounterIncrement` batches deltas into chunked multi-row upserts (256/stmt) instead of one round trip per key; the filter builders were unified onto a single WHERE/AND-connector `appendDuckDBFilter` (layout_planner + explain now share it). `CounterGet`'s full map IS the CounterBackend contract — no cheaper shape exists; batch INSERT was the real win.
      _(Effort: S)_ — source: status 2026-08-08_01-33
- [✓] **Watcher hardening** — DONE 2026-08-29 (todo-execution session 2) — the internal `chan any` is now DOCUMENTED as the intentional type-erased hub transport (one collection can host Watcher[A] and Watcher[B]; the typed surface is Watcher[V]'s output), `WithReificationFailureHook` fires on any→V reification failure (alongside the meter counter), and `BenchmarkWatcherNotificationLatency` measures Apply→notification (~3.3µs/op memory).
      _(Effort: S/M)_ — sources: status 2026-08-02_21-19/22-17
- [✓] **mapUpdateReplicationRule covers FoldUpdate only** — DONE 2026-08-29 (todo-execution session) — add FoldMultiInsert/FoldAppend.
      _(Effort: XS)_ — source: status 2026-08-03_08-26
- [PARTIAL ✓ 2026-08-29 session 2] **dgraphengine gaps** — DONE: `Transactional`/`RunInTx` (writes join one dgo txn, read-your-writes, serialized, nesting rejected; commit/rollback/concurrent/nesting tests green on Dgraph 25.4.0), capability conformance test (HealthCheck/Prober/TransactMeasurer/Calibratable + internal compile-time pins), empty-collection MapScan test. REMAINING (pre-existing, fails on pristine HEAD too): `TestDgraphADTMatrix/GraphDepthBound` + `/GraphDepth3Diamond` cross-engine parity — dgraph depth-bound traversal returns fewer nodes than memory (`dgraph="B"` vs `memory="B","C"` for A→B→C→D→E depth 2). Repro: `nix run .#integration-dgraph -run TestDgraphADTMatrix/GraphDepthBound`. Suspect `@recurse` depth/projection handling vs `extractNeighborIDs` in graph.go.
      _(Effort: M)_ — source: status 2026-08-11_19-14
- [✓] **irohengine `HealthChecker`** — DONE 2026-08-29 (todo-execution session 2) — `Replicated` composes local-engine health (metaengine.HealthChecker passthrough) with transport liveness via the new `LivenessReporter` capability (`InProcessNetwork.Healthy`/`Shutdown`, peer `Healthy`, `quic.QuicTransport.Healthy`); a closed transport surfaces as `ErrTransportClosed` through HealthCheck instead of silent drops. Note: quic/go.mod carries a `replace => ../` sibling for the unpublished symbol until the next tag wave (tag-release.sh strips it).
      _(Effort: S)_ — source: status 2026-08-08_02-24
- [✓] **Engine per-pattern `ReadCosts` calibration** — DONE 2026-08-30 — badger/bbolt/pebble migrated off the legacy `NsPerRead` scalar onto 4-field `ReadCosts`, calibrated from the new per-pattern benches (`BenchmarkCalibration_<Eng>_FilteredScan/_CounterScan/_FullScan`, 10K-row map + 1K counters, medians of 3, 2026-08-30): badger 1100/650/165/630, bbolt 750/620/100/660, pebble 700/830/125/700 (point-lookup/filtered-scan/aggregate/scan, ns). Point-lookup constants re-anchored to the fresh Get medians (bbolt was ~2x conservative: 1500→750; pebble 1300→700; badger 1200→1100). On KV engines `ReadAggregate` executes `CounterGet` (prefix scan) — the aggregate benches calibrate that path. The `.golangci.yml` SA1019 exclusion for `EngineProfile).NsPerRead` is DELETED (zero internal uses remain); exported constants kept (api-surface stable) and now feed `ReadCosts.NsPerPointLookup`.
      _(Effort: M)_ — source: lint-gate repair 2026-08-29 (436a9c9cb)
- [✓] **iroh QUIC test hardening** — DONE 2026-08-30 — verified 4 of 5 items already landed by the concurrent session (`TestNormalizeAny` table + round-trip in `latency_internal_test.go`; `TestEvictPooledStream_ReopenOnNextSend` error-injection + `TestQuicPooledThousandOps` 1K-op/1-stream stress; framing constants deduplicated — both `loopback/frame.go` and `quic/frame.go` alias `irohengine.FrameHeaderSize`/`ErrFrameTooLarge`; `WithStreamPooling()` documented in quic/README options table with pinning-test refs). This session added the missing piece: `TestRing_ProductionCapacity10K` in dedup — drives a ring at the QUIC transport's production capacity (`NewRing(10000)` in quic/transport.go) through 30K adds, pinning bounded Len, the exact eviction window, and graceful re-add of an evicted op.
      _(Effort: S)_ — sources: status 2026-08-08_03-20/21-45
- [✓] **`VectorCount` optional capability + Doctor/EXPLAIN WARN** — DONE 2026-08-29 (todo-execution session 2) — `metaengine.VectorCounter` (`VectorCount` + `VectorCollections`, no payload transfer); Memory and pg engines implement it (pg via SQL COUNT/DISTINCT); Doctor gained a `--- Vectors ---` section; Doctor/ExplainPlan WARN per engine serving k-NN by full scan without size introspection.
      _(Effort: S)_ — source: status 2026-08-17_14-25
- [✓] **mysqlengine sort-path layout integration** — DONE 2026-08-29 (todo-execution session 2) — sort fields now get a DECIMAL(65,10) numeric twin column (`gcn_<f>_<h>`) plus composite (collection, gcn, gc) index; ORDER BY and keyset cursor predicates render the twin columns (numeric cursor → gcn; text cursor → gc), keeping dual-key numeric/text semantics while letting the index drive the sort. Fields laid out earlier as filter-only fields gain the twin on a later sort-field request. Verified live on MariaDB 11.4: sort order, keyset pagination, index present (`idx_map_sort_gc_*`). MySQL 8.4 path uses functional key parts directly (unchanged).
      _(Effort: M)_ — source: status 2026-08-16_22-50
- [✓] **metaengine README capability table** — DONE 2026-08-29 (todo-execution session) — missing GraphEdgeRemoval / UndirectedGraph / VectorFilterBackend rows.
      _(Effort: XS)_ — source: status 2026-08-16_21-22

**Storage / SQL**

- [x] **SQL-injection hardening tail** — DONE 2026-08-29/30: `ValidateJournalIdentifiers` guards Table + TimestampColumn interpolation in `JournalReader.ReadAll`/`LoadFromStart`, `ResolveCursorTimestamp`, and `KeysetPositionQuery` (fail-fast Infrastructure rejection); multi-condition fuzz target (`FuzzBuildWhereClauseChecked_MultiCondition`), persisted hostile corpus (`storage/sql/testdata/fuzz/...`, loaded by every CI run), nightly fuzz workflow (`.github/workflows/fuzz.yml`); gosec runs in golangci (G304/G115 excluded globally); `KeysetPositionQuery` "" shape fixed via `KeysetPositionQueryChecked` (Deprecated wrapper kept for v4). Wave-1 CHANGELOG backfill landed 2026-08-30.
      _(Effort: S)_ — source: status 2026-08-16_14-54
- [x] **`ScanSlice` pre-size** — DONE 2026-08-29 (plan V3 T06): optional capacity hint on `ScanSlice`; `JournalReader` threads its bounded limit (capped 4096) into the drain-path scans; benchmark added.
      _(Effort: XS)_ — sources: status 2026-08-16_03-10/07-12
- [x] **system Command/QueryAdapter `metaJSON, _ :=`** — DONE 2026-08-29 (plan V3 T05): defensive nil-on-failure + deterministic marshal mirroring the event adapter; note: today's string-typed metadata makes the error path unreachable, so the honest fix is hardening + the first metadata roundtrip tests.
      _(Effort: XS)_ — source: status 2026-08-16_17-38
- [x] **storage/relational one-tx-per-event** projection writes — ALREADY IMPLEMENTED, verified 2026-08-30: `RelationalProjection.Handle` (storage/relational/projection.go:134-155) begins one `BeginTx` per event, constructs the SINGLE `sqlSink` on that tx, commits on success, deferred-Rollback on error — every sink write and any raw SQL the handler runs share the event's transaction. Pinned by `storage/relational/tx_test.go` (sink exposes the active tx, raw SQL commits with sink writes, raw SQL rolls back on handler error).
      _(Effort: M)_ — source: status 2026-08-16_03-10
- [x] **backuptest tag + replace drop** — DONE 2026-08-29: `storage/backuptest/v4.1.0` cut with wave B3 (fetchable tag); bbolt/pebble pins bumped + sibling replaces dropped (plan V3 T07).
      _(Effort: XS)_ — source: status 2026-08-16_19-52
- [✓] **Test-suite consolidation tail** — DONE 2026-08-29 (todo-execution session 2) — storage/sql command/query stores run `commandtest`/`querytest.RunStoreSuite` against SQLite in-memory (LoadFromTimestamp subtest already existed in the commandtest suite); `querytest` gained its own self-test against the memory reference store, mirroring commandtest.
      _(Effort: S)_ — source: status 2026-08-08_01-39

**Docs / tooling**

- [✓] **Engine READMEs** — DONE 2026-08-29 — mysql/sqlite/turso/badger READMEs already exist (filled by earlier waves; verified non-empty); pebble's stale "Graph: O(N^d) BFS" comment replaced with an explicit NOT-supported note (profile omits ADTGraph); pgengine README covers the vector path and mysqlengine README covers LayoutPlanner (verified). modules.md gained dgraphengine + irohengine rows (session 2).
      _(Effort: S)_ — sources: status 2026-08-16_18-09/22-50
- [x] **CHANGELOG unreleased-block fold** — DONE 2026-08-29: the 2026-08-16-era block was folded into the top `[Unreleased]` window (header note in CHANGELOG.md records it); no separate `[Unreleased — earlier …]` heading remains.
      _(Effort: S)_ — source: status 2026-08-16_19-01
- [x] **v5 doc coverage** — DONE 2026-08-29 (plan V3 T30): faq.md gained a v5-deletion overview; AGENTS Codec Defaults carries the v5 note; method-level `Deprecated:` marker decision recorded as an ADR-0123 addendum; storage/pebble verified to have NO stack import (only a stale doc comment, fixed) — nothing blocks P13 stack deletion; stack/bench decision: DELETE with the rest of stack/ at the v5 cut (it benchmarks presets and has no reason to outlive them).
      _(Effort: M)_ — sources: status 2026-08-18_13-59, 2026-08-17_16-27
- [x] **Fix "~41-byte" figure** — DONE 2026-08-29 (plan V3 T31): all 3 active sites corrected to 43–46 bytes (archived status frozen as-is).
      _(Effort: XS)_ — source: status 2026-08-16_14-44
- [x] **AGENTS.md gotchas missing** — DONE 2026-08-29 (plan V3 T32): one consolidated storage-engine/MySQL-VM gotcha added within budget.
      _(Effort: S)_ — sources: status 2026-08-16_14-44/04-00
- [x] **SEVEN-TIER-MODEL.md reconciliation** — DONE 2026-08-29 (plan V3 T31): rewritten as Tier-3-with-Tier-0-core (planner deps dedup/record/id verified from go.mod); deleted flightrecorder/retry/codec rows removed.
      _(Effort: XS)_ — source: status 2026-08-08_08-23
- [✓] **BENCHMARKS.md + skill refs** — DONE 2026-08-29 — durability cells filled (pebble sync/async measured in a quiet window); modules.md bboltengine row present; session 2 added VectorCounter/HealthChecker/Transactional + projectionhost contract notes to modules.md (doc-check green, 931 refs).
      _(Effort: S)_ — source: status 2026-08-17_15-17
- [ ] **Release docs** — CONTRIBUTING Release Process lacks the pin-bump-before-tag recipe + GOPRIVATE verification commands; durability-tier-mapping ADR never written; Introspection/Doctor don't surface effective durability tiers.
      _(Effort: S)_ — source: status 2026-08-18_20-39
- [x] **catalog/docserver follow-ups** — DONE 2026-08-29 (plan V3 T36): docs-ui.css GET test added; go-snaps decision = stays counted in catalog's production dep budget (documented in check-module-layers.sh); cId-value-change CHANGELOG note written; README deps table added; templ drift gate shipped as `nix run .#check-templ`; CSP nonce support landed (Config.EnableCSP, per-request nonces on every script); EventCatalog CLI real-render validation executed against @eventcatalog/core ^4.6.3 — it caught a real exporter bug (producers/consumers now emit versioned reference strings).
      _(Effort: S)_ — source: status 2026-08-16_20-38
- [x] **benchmark-regression gate hardening** — DONE 2026-08-29 (plan V3 T37): fixture tests pin median computation, thresholds, and the save-after-compare ordering (`nix run .#check-bench-gate`, 9 cases); baseline-regen runbook added to BENCHMARKS.md; actionlint added to devShell (workflows lint-clean after fixing the two shellcheck infos in the doc-assertions step); `nix run .#verify-module -- <path>` scoped mode shipped. Remaining: 25% threshold re-tune deferred until live CI accumulates per-run variance (noted in the runbook).
      _(Effort: S)_ — source: status 2026-08-16_18-06
- [x] **Consumer asks (feedback)** — DONE 2026-08-29 (plan V3 T39): first-class snapshot encryption shipped as `snapshot.NewTransformedStore` + `encryption.SnapshotStateCodec`/`RotatingSnapshotStateCodec` (envelope key IDs + StaticKeyResolver give rotation without a migration window; integration compose test included); `DoWithValue[T]` landed in the external go-retry repo; OTel README gained the exporter-lifecycle/shutdown-flush section with the stop-accepting-then-flush ordering.
      _(Effort: S)_ — source: docs/feedback 2026-08
      _(Effort: S each)_ — sources: feedback 2026-07-17/2026-08-21
- [x] **Design questions** — RESOLVED 2026-08-29 (plan V3 T40):
      (1) ULID entropy: already implemented as the lock-free epoch design —
      documented as accepted in ADR-0131 (id/entropy.go);
      (2) Pebble calibration basis: post-Flush/pre-Compact chosen and
      documented in ADR-0132 (bench comment figure also corrected to 43–46);
      (3) command.Bus/MemoryBus removal: DECLINED — see Declined section.
      _(Effort: S)_ — sources: status 2026-08-16_17-39/14-44, 2026-08-03_20-30
- [ ] **Env**: /mnt/buildcache re-broken 2026-08-29 (golangci cache mkdir fails) — /tmp cache redirects required again until repaired.
      _(Effort: XS, environment)_

---

## Release / Tagging 🔥

> Blocked on user authorization (never tag/push without explicit instruction).
> **2026-08-29 13:18: full `nix run .#verify` GREEN (plan V3 T02)** — build +
> vet + test + race + lint 76/76 + check-arch + depguard + doc-check (1154
> refs) at commit `50a9a212d`; this replaces the 2026-08-16 release-checkpoint
> claim. **B1+B2+B3 CUT 2026-08-29** (18 tags incl. snapshot/v4.4.0,
> listing/v4.3.0, storage chain, backuptest/v4.1.0 — see CHANGELOG); B4–B7
> still need your sign-off
> ([plan](docs/planning/2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md)). The
> 2026-08-16 chain (id v4.5.0, record v4.3.0, metadata v4.5.0, schema v4.3.0,
> event v4.7.0, query v4.6.0→retracted→v4.6.1, command v4.7.0→retracted→v4.7.1,
> middleware v4.5.0, metaengine v4.11.0, sqlite/pebble/pg engines v4.1.0,
> badger v4.0.2, watermill v4.5.0, mysql/bbolt/turso/iroh engines v4.0.0,
> storage v4.7.0→retracted→v4.7.1) is LIVE on the proxy.

- [ ] [BLOCKED] 🔥 **Tag the pending v4 patch wave** (supersedes the stale
      2026-08-16 wave-4 list — audited 2026-08-27, plan task T01): ALREADY
      SHIPPED since that note: `event/v4.8.0` (DecorateJournal),
      `metadata/v4.6.0` (BrandedString), `metaengine/v4.12.0` (capability
      audit + iroh exports), `storage/v4.8.0` (OpenSQLiteInMemory shared-cache
      DSNs) — all LIVE on the proxy. REMAINING (prod-code-changed since
      latest tag, via `git diff <tag>..HEAD -- '*.go'`):
      (a) hardening-fixes wave: `event`, `command`, `query`, `dedup`,
      `dispatcher`, `middleware`, `scheduling`, `kv`, `commandlifecycle`
      (+projections), `system`, `idempotency/kvstore`,
      `idempotency/sqlstore`, `encryption`, `signing`;
      (b) snapshot chain in dependency order: `snapshot` → `decider` →
      `storage` → `storage/memory` → `storage/pebble` → `storage/bbolt`
      (+ `storage/turso`, `storage/backuptest`);
      (c) wave-4 leftovers: `schema`, `projectionhost`,
      `metaengine/irohengine`;
      (d) engine patches: `dgraphengine`, `duckdbengine`, `mysqlengine`,
      `graphadapter`, `projectionadapter`, `tursoengine`,
      `metaengine/irohengine/{loopback,quic}`.
      Order constraints: event+metadata+schema before/with projectionhost
      (released go.mod needs them — release flow strips sibling replaces);
      snapshot before decider before storage before pebble/bbolt; per-module
      cut→push interleave (GOPRIVATE resolves siblings via VCS fetch).
      Via `scripts/tag-release.sh` from a clean tree; full module-order plan:
      `docs/planning/2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md`.
      Do NOT tag `stack/*`, `storage/view`, `storage/relational` (v5 deletes
      them); `transport/*` gets final deprecation-only v4.x tags — separate
      item below.
      _(Effort: L)_
- [ ] [BLOCKED] 🔥 **Dead tags: `event/v4/eventtest/v4.0.0` + `v4.2.0`**
      (found 2026-08-27, T02 pin sweep). The eventtest module path
      (`.../event/v4/eventtest`) has no trailing `/vN`, so Go rejects v4.x
      versions for it (`invalid version: module path must match major
      version` — verified via `go mod download`). Consumers correctly stay
      on v0.3.0 (latest usable). Remedy needs user action: delete the two
      remote tags, or document-and-ignore; do NOT tag eventtest above v0.x
      unless the module path gains a major-version suffix (breaking move).
      _(Effort: XS)_
- [ ] [BLOCKED] **go-codec F46: commit + tag the `UnwrapDecode` sniff** —
      the first-byte fast path (fallback 181ns/6 allocs → 1.6ns/0 allocs) sits
      UNCOMMITTED in `../go-codec` (no auto-commit daemon there); GOWORK=off
      consumers get nothing until it is tagged. NOTE (2026-08-16): the same
      uncommitted tree drops `event.NewEvent` allocs 3→2, so
      `TestAllocs_NewEvent_*` FAILS under workspace-mode gates (`#verify-fast`)
      while passing GOWORK=off — update the allocs expectations in the same
      change that tags go-codec.
      _(Effort: XS)_
- [ ] [BLOCKED] **Ratify one shipped judgment call** — iroh latency P99
      bound 50→150ms (worst-of-30 sample inflates under gate load). Shipped +
      gated green; keep or revisit. (The `OpenSQLiteInMemory` pool-pin
      judgment call was resolved: replaced with shared-cache DSNs.)
      _(Effort: XS)_
- [ ] **Replace-drop sweep (after the wave-4 tags)** — system ×6,
      cqrs-bench ×7, event ×2, schema ×2, projectionhost ×2, integration ×2
      local `replace` directives exist only because wave-3/4 code is untagged;
      drop + tidy + GOWORK=off re-verify each module. Every one is documented
      droppable-on-tag; the sweep cost compounds until then.
      _(Effort: M)_
- [ ] **Create GitHub Releases for the 2026-08-16 tags** — 20 tags, none have
      releases (only storage/v4.7.1 got one). `gh` auth never verified from
      this environment. Optionally curated notes for the 8 core modules.
      _(Effort: S)_
- [ ] **Document the retract-and-republish pattern** in CONTRIBUTING.md
      Release Process (what happened to command/v4.7.0, query/v4.6.0,
      storage/v4.7.0 and the exact remedy), and audit recently published tags
      with the hardened script's standalone-build gate once `092b5e8a8` lands.
      _(Effort: S)_
- [ ] [BLOCKED] **Tag final v4.x patches of `transport/http` +
      `transport/grpc`** (deprecation notices included) — prerequisite for
      the v5 deletion (ADR-0127).
      _(Effort: S)_
- [ ] **Create GitHub Releases + pkg.go.dev fetch triggers** — folded into
      the 2026-08-16 GitHub Releases item above.
- [ ] **Consolidate indirect dep references** — after new module tags are
      published, the transitive `go-cqrs-lite/{codec,retry,idempotency,
      flightrecorder}/v4` indirect deps in ~49 consumer go.mod files clean
      up. Track and verify.
      _(Effort: M)_
- [ ] **Run the pre-tag checklist** — `nix run .#vulncheck` +
      `#check-arch` (verify covered the rest) + GOWORK=off `go test ./...`
      on the module AND its test subpackages (the command/v4.6.0
      commandtest failure class).
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

- [ ] **`#verify-standalone` nix app (GOWORK=off per module) or explicit
      decision that CI owns that signal** — then CHECK CI after gates.
      Investigate how long the CI Benchmarks job has been red (`gh run list`)
      to size the blind window.
      _(Effort: M)_
- [ ] **Add CI leg for GOWORK=off standalone builds of leaf modules**
      (integration/, examples/, benchkit/) to catch pin rot early.
      _(Effort: S)_

---

## Metaengine — Universal ADT Coverage (Phase 7)

> StreamLog on Dgraph, native graph on SQLite/Turso (iterative BFS), degraded
> rule with latency estimates, engine test parity shipped 2026-08-11.
> `JournalReadFrom` positional fixes (Dgraph + all SQL engines) shipped
> 2026-08-15 with shared-contract regressions. MariaDB dialect + numeric-safe
> sort + CTE probe, LSM vector search, native graph on PG/MySQL, and the
> SQLite recursive CTE shipped 2026-08-15 — see CHANGELOG.
>
> Follow-ups below harvested from
> `docs/status/archived/2026-08-15_22-04_metaengine-followup-closeout.md` §f.

- [ ] **Turso explicit CTE-probe test** — the sqliteengine probe covers
      local drivers; add a tursoengine test confirming it holds over the
      remote protocol.
      _(Effort: S)_
- [ ] [BLOCKED] **Run `nix run .#integration-mysql-nspawn`** (needs root) — real-env
      verification incl. `stack/mysql`; live verification so far used docker
      probes only. Partially covered 2026-08-16: userspace MariaDB 11.4
      verified mysqlengine (`-count=3`) + stack/mysql (`-count=3`) — but the
      nspawn env runs the full app-level flow.
      _(Effort: M)_

---

## cqrs-lint

> Per-module coaching migration complete: all adoption (F003-F029) and
> resilience (B029-B031) rules evaluate per-module. 86 per-module profiles
> verified by `integration_multimodule_test.go`. F001/F002/F005/F014 remain
> workspace-global by design (low leakage risk). F030 (deprecated transport
> imports) shipped 2026-08-14 — 203 rules total.

- [ ] **Audit `.golangci.yml` exclusion blocks + RE-ENABLE depguard** — FOUND 2026-08-29 (session 2): depguard is DISABLED (`linters.disable`) and its settings block was dropped in the 2026-08-29 lint-config refactor, orphaning `check-depguard.sh` (now reports a loud SKIP instead of erroring). Per-module dependency-prefix enforcement is OFF until re-enabled; check-arch layer budgets still apply. Original audit item: `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Track which can be removed after migrations complete.
      _(Effort: M)_

---

---

## Code Quality / Infrastructure

- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin. (Last open item of the
      19-item Code Quality / Infrastructure section — 18/19 done.)
      _(Effort: M)_ — 2026-08-16: static review done (portability note added
      to the script header; no Linux-isms found, /dev/kvm check
      uname-guarded); hardware verification on a real Mac remains open.
      2026-08-30 (C10): header now says the claim is static-review-only and
      names the verification route — a GitHub Actions macOS runner leg
      (blocked on macOS CI runner, same hardware constraint as D9).

---

---

## system/v4 Full-Code-Review Follow-Ups (2026-08-16/17)

> From `docs/reviews/2026-08-16_full-code-review-system.html`. All 5 P1 and
> the actionable P2/P3 findings are FIXED (commits a211ebcb2, 449e0e5a7,
> 42dfab5b0 — shipped with regression tests). These remain, routed:

- [ ] **Fix GitHub Actions billing** — BLOCKED on user (2026-08-18): every
      paid CI job (Release, Benchmarks, ci.yml) fails in 3-7s with "recent
      account payments have failed or your spending limit needs to be
      increased"; broken since ~2026-07-17. Local `nix run .#verify` remains
      the authoritative gate until billing is restored. _(Effort: S, user
      action: Billing & plans settings)_

---

## v5 Unification (Phase 8: Deletion + Cut)

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Phases 1-7 done (type foundation, dead-code removal, self-registration,
> backend porting, record-typed folds, auto-projection, layout planning,
> universal ADT coverage). Phase 8 is the breaking cut.
>
> **Pre-cut deprecation wave shipped 2026-08-17:** every API below that dies
> at v5 now carries a `Deprecated: removed in v5 (ADR-0123)` doc marker —
> `stack` Bundle/Materialize/RunProjections + all 8 presets, `storage/view`,
> `storage/relational`, `graph.GraphProjection`; `record.NewStreamRef` got a
> NOTE documenting its v5 signature change. Internal callers stay warning-free
> via a scoped `.golangci.yml` SA1019 exclusion keyed to the marker phrase.
> The deletions themselves are still TODO. See CHANGELOG `[Unreleased]` →
> `### Deprecated`.

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
- [ ] **Delete `storage/sql.BuildWhereClause`** — deprecated 2026-08-15
      (interpolates column names/operators); `BuildWhereClauseChecked` is
      the validated replacement.
      _(Effort: XS)_
- [ ] **Breaking `record.NewStreamRef` validation** — v4 kept the constructor
      non-breaking and added `StreamRef.Validate()` + `ErrInvalidStreamRef`
      (2026-08-16); `Split()` accepts the empty-streamType form that
      command/query asrecord produces. At v5, change to
      `NewStreamRef(streamType, entityID string) (StreamRef, error)` rejecting
      an empty entityID at construction (empty streamType stays legal) and
      migrate the call sites. Note: `id.NewStreamRef` is a separate,
      unrelated function.
      **Owner-confirmed 2026-08-22** (decision memo:
      `docs/planning/2026-08-22_03-52_core-data-model-v5-execution-plan.md`
      Appendix B): the struct `record.Stream` counter-proposal from the
      2026-08-22 core data-model review is REJECTED — the string type
      survives; identity-shape convergence (P1) is delivered by validated
      bridge population (T04) + `Execute(ref)` convention (T10) + the
      Stream/StreamRef/StreamID naming decision (T22).
      _(Effort: M)_
- [ ] **Delete `transport/http` + `transport/grpc` modules** (ADR-0127) —
      delivery is `watermill/` + go-sse + cqrs-htmx. `example/taskmanager` is
      migrated (metaengine.ServeSSE on the task_views watcher); cqrs-lint F030
      coaches consumers off the deprecated imports. Remaining steps: tag final
      v4.x patch releases of both modules (see Release section), drop
      the modules from `go.work`/flake `testModules`/api-stability list, then
      delete at the v5 cut.
      _(Effort: M)_
- [ ] **Delete deprecated tombstone metadata API (ADR-0114 completion)** —
      remove `event.DetectTombstone`/`MarkTombstone`/`MarkRebirth`/
      `TombstoneStatus`/`Metadata.Tombstone`; make deletion purely
      event-type-driven (metaengine already is; auto-projection replaces
      `stack.Materialize`, whose `OnTombstone`/`OnRebirth` are the last
      metadata-triggered path; `listing` needs type-based status to replace
      `StatusMiddleware` + the `event.DetectTombstone` call at
      `listing/in_memory.go:155`). The 2026-08-10 attempt (`e406edcfb`) was
      reverted (`a6613ef0d`) before release; docs realigned 2026-08-16.
      Owner decision M20 (2026-08-11): full rename deferred to v5.
      **T20 prep check (2026-08-22):** migration doc verified accurate
      against shipped code (StatusMiddleware signature, Materialize
      metadata trigger, metaengine Remove); bridge chain
      StatusMiddleware→OnTombstone/OnRebirth now has test coverage
      (`stack/materialize_tombstone_bridge_test.go` — previously ZERO
      trigger coverage anywhere). Pre-reqs before deleting: land
      type-driven status in `listing` (replaces the DetectTombstone call),
      migrate `example/taskmanager` off `OnTombstone`, regen golden,
      execute the tombstone section of
      `docs/planning/v5-deprecation-sweep.md`.
      _(Effort: M)_
- [ ] **Honest snapshot wire tags at v5 (T18 audit 2026-08-22)** — rename
      `snapshot.Snapshot` JSON tags (`aggregateId`/`aggregateType` →
      `streamId`/`streamType`) only together with the per-backend wire
      changes, keeping old tags readable until then. Audit findings:
      **memory** = in-process struct copy, no wire (nothing to do);
      **pebble** = CBOR local `serializableSnapshot` with OLD vocabulary
      (`aggregate_id`/`aggregate_type` tags) + legacy-JSON fallback — rename
      needs dual-read (try new tags, fall back to old) or a key-prefix
      version bump; **bbolt** = CBOR local struct already on honest
      `stream_type`/`stream_id` tags (only struct-level JSON tags change,
      wire stays); **SQL** = schema columns `aggregate_type`/`aggregate_id`
      (postgres/sqlite/mysql/duckdb) — rename = ALTER TABLE + backfill per
      dialect. The new `Encoding` stamp: persisted by pebble/bbolt/memory
      (additive `omitempty` field, old rows decode as Unknown); NOT
      persisted by SQL (no column — the ADR-0044 envelope inside State
      stays authoritative there; add a column only if SQL consumers need
      the stamp outside State). Pre-req for the SQL rename: migration
      scripts under `storage/sql/migrations/` + `nix run .#integration-pg`
      over the renamed schema.
      _(Effort: M)_
- [ ] **Write v5 migration guide** — document the path from v4 (stack presets,
      v1 tiers, transport/*, manual RelationalProjection/view reads) to v5
      (`system.System`, auto-projection, watermill/go-sse delivery).
      Before/after examples for each v1 tier, including
      `relational → metaengine` (consumer-pulled).
      _(Effort: L)_
- [✓] **v5 decision: kvstore SA1019 exclusion** — DECIDED 2026-08-29 (todo-execution session 2): the scoped exclusion is PERMANENT — the test matrices use go-idempotency's MemoryStore exactly as its deprecation prescribes ("development and testing") as the reference implementation in conformance matrices; production paths use the real stores. Rationale recorded in the .golangci.yml comment next to the exclusion. Migration-guide outline (v4→v5) written at docs/migration/V5-OUTLINE.md. go-idempotency already ships contract_test.go, so a parallel suite here would duplicate it.
      _(Effort: S)_
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate.
      _(Effort: M)_

---

## Core Data Model v4.x/v5 (2026-08-22 review + plan)

> Source: [core data-model review](docs/reviews/2026-08-22_core-data-model-review.html)
> (12 findings) + [execution plan](docs/planning/2026-08-22_03-52_core-data-model-v5-execution-plan.md)
> (25 tasks, subtask breakdown + ripple research in its appendices).
> Owner decision 2026-08-22 (plan Appendix B): string `record.StreamRef`
> SURVIVES v5 with a validating constructor; the review's struct
> `record.Stream` proposal is rejected. Decision/reference tasks (T01–T03,
> T14, T02) are DONE — not duplicated here. Metaengine ripple of the whole
> series: LOW for v4.x additive (all keyed literals, no Record wire
> serialization; plan Appendix A).
>
> Every PR below lands additive-first with `Deprecated: removed in v5`
> markers on what it supersedes, api-stability golden regen + `#verify-fast`
>
> - CHANGELOG `[Unreleased]` entry in the same change.

- [ ] **v5 items from extended review** — E1 (event-envelope Encoding →
      `record.Encoding`), E7 (watermill/middleware RetryConfig collision),
      E8 (typed Message Kind enum), E11 (AdapterCore.Encode error return),
      E13 (SQLTimerStore phantom param), E15 (middleware signature
      unification) — fold into the v5 cut wave after the sweep doc.
      _(Effort: M)_
- [ ] **Post-landing sweep for this series** — api-stability meta-tests,
      doc-check over skill refs, consumer-pin sweep for `record/v4` consumers
      under GOWORK=off (plan T24; MarshalBinary lesson).
      _(Effort: M)_
- [x] **AGENTS.md memory: data-model conventions** — DONE: AGENTS.md internal-contract #21 records the ADR-0111 T04–T12 conventions (record.Type aliases, typed Encoding stamp, single StreamRef, string-backed caller-keyed IDs, typed ActorID, AsRecord populate-never-drop, capability interfaces).
      _(Effort: S)_

### Deep Full-Code Review (2026-08-27) — follow-ups

> 9 fixes landed inline (E3/E9/E10 + decider poll-clamp/WithoutCancel/
> ErrNilDecide, scheduling Due ordering, kv stale-cache invalidate,
> commandlifecycle version-cache recovery, projectionhost backoff cap).
> Report: `docs/reviews/2026-08-27_full-code-review.html`.

- [✓] **commandlifecycle Recorder unbounded versions map** — DONE 2026-08-29 (todo-execution session) — every command ID
      seeds a `versions` entry that is never evicted (only manual
      `ResetVersion`); long-running dispatch loops grow it forever. Bound it
      (LRU/TTL) or re-seed per emit batch.
      _(Effort: S)_
- [✓] **commandlifecycle AttemptMiddleware standalone leak** — DONE 2026-08-29 (todo-execution session) — the attempt
      tracker is only cleared by the OUTER middleware; standalone
      `AttemptMiddleware` usage grows `attempts` forever.
      _(Effort: S)_
- [✓] **projectionhost applyWithRetry ignores error family** — DONE 2026-08-29 (todo-execution session) — Rejection/
      Corruption (non-retryable) handler errors are still retried
      `dlqThreshold` times before DLQ; skip straight to DLQ for
      `!family.IsRetryable()`.
      _(Effort: S)_
- [✓] **projectionhost Stop timeout has no retry path** — DONE 2026-08-29 (todo-execution session) — after
      `shutdownTimeout` fires, `stopped=true` makes later Stop calls no-op
      while workers may still run; expose a force/re-drain path.
      _(Effort: S)_
- [✓] **snapshot.ReadPressure bounded tracking** — DONE 2026-08-29 (todo-execution session) — the `reads` map grows
      with distinct stream refs (inline `TODO(review-2026-08-27)` in
      `snapshot/read_pressure.go`); bounded LRU option.
      _(Effort: S)_
- [✓] **command/query constructor error-style drift** — DONE 2026-08-29 (todo-execution session) — `command.New` wraps
      sentinel errors while `query.New` returns raw strings; unify before v5.
      _(Effort: S)_
- [✓] **cqrs-lint version constant automation** — DONE 2026-08-29 (todo-execution session) — `cmd/cqrs-lint/main.go`
      `version` const drifted from the released tag (4.6.0 vs v4.7.0);
      `scripts/tag-release.sh` should bump it in the same wave.
      _(Effort: S)_
- [✓] **kv.Cache.Get miss-path double round-trip** — DECLINED 2026-08-29 (todo-execution session): the suggested "cache the store-fresh value, copy only on hit" swap moves the same one encode+decode — under the copy-isolation contract the returned value must not alias the cached entry. A real reduction needs a raw-bytes TypedStore API (Get returning encoded bytes for a single decode copy); revisit if hot-path profiles demand it.
      _(Effort: S)_
- [✓] **listing cursor cross-type ambiguity** — DONE 2026-08-29 (todo-execution session 2) — `ListOptions.After` godoc documents the hazard (same ULID under two types can skip/repeat around the page boundary) and prescribes the mitigations: paginate with a concrete Type, or guarantee globally unique IDs. Cursor keyed by (type, id) stays the v5 option.
      _(Effort: S)_

### Deep-Review Gap Wave (2026-08-27, second session) — follow-ups

> Concurrent review wave (11 module clusters, agent-swept). 10 fixes landed
> inline (DecorateStore streaming forwarding; Conflict/Corruption family
> preservation across dispatcher/command/query/memory/pebble/eventstore;
> scheduling WithMaxRetries clamp + corrupt-row skip; query pagination
> divide-by-zero + audit detached-context; DuckDB PK duplicates; dedup nil
> Add; event constructor-family passthrough + Orchestration alias; middleware
> flight-recorder detached context; bbolt bucket/NotFound guards; pebble
> batch leak). Unfixed findings below, priority-ordered.

- [x] **metaengine Record context is shared mutable state across Stores** 🔥 — DONE
      2026-08-29 (plan V3 T24): the Record is passed through the invoke closure
      as a value (all 11 fold kinds); the recHolder cell, recordSetter fields,
      and the internal SetCurrentRecord hooks are gone; `RecordAwareFold` kept
      as a Deprecated source-compat interface. Regression race test added
      (`TestOnRecordFolds_ConcurrentStoresNoRace`, red before / green after).
      _(Effort: M)_
- [x] **metaengine replay paths destroy Record context** 🔥 — DONE 2026-08-29
      (plan V3 T24): `EventInput.Record` added (additive); `EventLog.RecordEvent`
      stores it on live applies; `Backfill`, `DemoteEngine` catch-up, and
      `Verify` replay now carry the original record instead of a synthesized
      `record.Record{Type}`.
      _(Effort: M)_
- [✓] **pebble command/query duplicate check is check-then-commit + fail-open** — DONE 2026-08-29
      (todo-execution session) — the existence check runs outside the write lock (concurrent
      duplicate Save silently overwrites instead of Conflict) and treats ANY
      Get error as "exists" (an Infrastructure read failure is reported as
      `ErrDuplicateCommand`). Take the per-ID shard lock around check+commit;
      return wrapped Infrastructure when the check itself fails.
      _(Effort: M)_
- [✓] **stream-not-found contract diverges across backends** — DOC-PINNED 2026-08-29 (todo-execution session 2) — `event.EventSource` godoc now pins (nil, ErrStreamNotFound) as the contract, names pebble/bbolt's (nil, nil) as a v5-marked divergence, and tells consumers to treat both shapes as "no events". `event.SeekableJournal` godoc pins the dangling-cursor contract (empty tail; memory's replay-from-start is the divergence). Store-side alignment waits for v5.
      _(Effort: S doc / v5 align)_
- [✓] **schema upcaster registry hazards** — DONE 2026-08-29 (todo-execution session) — (a) upcaster returning the same
      pointer mutates the stored/shared event (README claims "original is
      never mutated"); (b) `(nil, nil)` return panics; (c) the registry
      force-stamps source+1 regardless of the returned version (a v1→v3
      upcaster is relabeled and double-transformed); (d) duplicate
      (type,version) registrations accepted with unstable sort; (e) doc
      claims construction-time chain validation that does not exist.
      Guard nil, use sort.SliceStable, verify version stamps post-upcast,
      reject duplicates, fix or implement the claimed validation.
      _(Effort: M)_
- [✓] **snapshot.TypedStore.Save bypasses NewSnapshot** — DONE 2026-08-29 (todo-execution session) — bare struct literal:
      no invariant validation (version 0 / zero refs persist; property test
      generates version 0) and no CreatedAt stamp, unlike every other write
      path. Route through the validating constructor.
      _(Effort: S)_
- [✓] **kv.Cache has no invalidation primitive + cache-aside race** — DONE 2026-08-29 (todo-execution session) —
      `Backend()`/`Store()` hand out raw writers that bypass the cache, a
      second Cache instance never invalidates, default TTL is 0 (unbounded
      staleness), and a Get-miss can pin a stale value after a concurrent Set
      (G1 read-old → G2 Set → G1 cache.Set(old)). Add `Invalidate`/
      `InvalidateAll` (additive), document the single-writer assumption and
      TTL recommendation. Also: `DeleteAll` with no configured prefix
      deletes EVERY key in the backend — document the blast radius or gate
      behind an explicit opt-in.
      _(Effort: M)_
- [✓] **catalog SchemaFromType silent-wrong schemas** — DONE 2026-08-29 (todo-execution session 2) — embedded
      (anonymous) struct fields now FLATTEN into generated schemas, matching
      encoding/json promotion (named embeds stay named, `json:"-"` embeds are
      skipped, parent fields win conflicts, unexported-type embeds still
      promote — the Go wire behavior the old tests missed); self-referential
      types terminate via an in-progress cycle guard (opaque-object
      placeholder, concurrent same-type builds retry on the completed cache
      entry). The two tests that enshrined the skip behavior now pin
      promotion; wire-parity and mutual-cycle tests added.
      _(Effort: M; goldens change)_
- [✓] **cqrs-lint C042 inspects the wrong argument** — DONE 2026-08-29 (todo-execution session) — the zero-expected-
      version rule checks `call.Args[2]` but `event.Store.Save` is
      `(ctx, ref, events, expectedVersion)` — the version is `Args[3]`; the
      rule can never fire on the canonical API (and misses
      `event.Version(0)` conversions).
      _(Effort: S)_
- [✓] **scenario DSL can pass vacuously** — DONE 2026-08-29 (todo-execution session) — `Given(...).When(cmd, decide)`
      with no `Then*` compiles and passes with zero assertions;
      `GivenProjection` without `ThenNoError`/`ThenError` swallows every
      handler error. Register a `t.Cleanup` guard failing the test when no
      terminal assertion ran. Also `scenario/doc.go`'s example signature is
      stale (missing `t *testing.T`).
      _(Effort: S)_
- [✓] **projectionhost hardening set** — DONE 2026-08-29 (todo-execution session 2) — all seven findings fixed with regression tests (`hardening_test.go`): ReplayDeadLetters holds `handleMu`; Reset clears the checkpoint FIRST; `WithBatchSize` clamps <= 0; Start-after-Stop rebuilds workers (fresh stop channels) so the documented Stop→Reset→Start recipe works; CheckStaleness/CheckProjectionStaleness treat a FAILED worker as stale; retryable-family errors are NOT parked in the DLQ (restart path retries, loud WorkerFailed on exhaustion); corrupt SQLite DLQ rows are skipped + counted (`SkippedCount`). The original finding list read: `ReplayDeadLetters` calls
      `projection.Handle` outside `handleMu` (races a running worker);
      `Reset` clears read-model state BEFORE the checkpoint (crash window
      skips pre-checkpoint events; doc says checkpoint first);
      `WithBatchSize` accepts <=0 (worker exits "caught up" processing
      nothing); the documented Stop→Reset→Start recipe cannot work (Start
      rejects after first start); `CheckStaleness` reports fresh for a dead
      worker (lag==0 ambiguity); DLQ admits Transient/Infrastructure errors
      (permanent silent gap until manual replay); one corrupt SQLite DLQ row
      bricks List/ReplayDeadLetters.
      _(Effort: M)_
- [✓] **capability interfaces not adopted at three assertion sites** — DONE 2026-08-29 (todo-execution session) —
      middleware/actor.go, commandlifecycle/recorder.go, and
      watermill/command_protocol.go each re-declare a private
      `Metadata() command.Metadata` interface although
      `command.MetadataCarrier` exists exactly for this (ADR-0111 g).
      Replace the private clones (non-breaking).
      _(Effort: S)_
- [✓] **transport deprecation is not machine-readable** — DONE 2026-08-29 (todo-execution session) — transport/http and
      transport/grpc say "DEPRECATED" in prose but lack the Go-standard
      `// Deprecated:` paragraph, so staticcheck SA1019 never flags
      consumers; http's WebSocket section steers to grpc without noting it
      is deprecated too.
      _(Effort: XS)_
- [✓] **deriver has no cycle guard** — DONE 2026-08-29 (todo-execution session) — `Then`'s doc blesses A→events→B→events
      chains through the bus, but nothing bounds derivation cycles
      (deterministic IDs key on the source event ID, which changes every
      round). Opt-in depth guard via a hops counter in derived-command
      metadata, or document the hazard on AsHandler.
      _(Effort: S)_
- [✓] **scheduling multi-instance + retry semantics undocumented** — DONE 2026-08-29 (todo-execution session 2) — Scheduler godoc now documents all three hazards: the single-active-instance requirement (no claim/lease; two Schedulers double-fire), dispatchWithRetry's family-blind retries (Rejection retried forever per poll; errors.Join keeps only the last attempt's error), and the MarkFired no-epoch race (use fresh timer IDs per logical deadline). ClaimingTimerStore (SKIP LOCKED) remains the additive follow-up.
      _(Effort: S doc / L claim protocol)_
- [ ] **metaengine routing/lifecycle follow-ups** — `Calibration` setters
      race concurrent `Profile()` readers (documented Plan→Calibrate→Probe
      ordering makes it likely); `CheckRouting`'s cache signature omits the
      plan version (stale diagnostics after Replan) and live NsForRead;
      reassignment is strict argmin so the hysteresis deadband only gates
      suggestions (assignments flap under oscillation); engines over-
      declaring `Supports` produce execution-time hard errors with no
      plan-time diagnostic or routing penalty (CapabilityAudit renders a
      banner but is not a rule); graph BFS fallback dedups nodes by
      `fmt.Sprint` (int(1) collides with "1" on mixed-type nodes);
      OnRecord folds returning Embedding/IndexedText/Point/MultiEntry/
      Append receive an always-zero Record silently.
      _(Effort: M-L, several independent)_
- [✓] **eventtest fakes** — DONE 2026-08-29 (todo-execution session) — `LoadToVersion` returns a live sub-slice of the
      store's backing array (in-place sort corrupts the fake);
      `ReadAll`/`ReadFrom` return map-iteration order violating the
      Journal's documented OccurredAt ordering; `FakeBus.Publish` reads
      `publishChain` unlocked while `UsePublish` swaps it under mu.
      _(Effort: S)_
- [✓] **record.Stamp zero-time presence flip** — DONE 2026-08-29 (todo-execution session) — `NewStamp(time.Time{})` is
      known but JSON-round-trips to unknown (wire `at` is a value, so the
      zero time reads back as absent). Wire-compatible fix: make the wire
      field `*time.Time` (nil → unknown). Undocumented edge today.
      _(Effort: S)_
- [✓] **dispatcher/metadata README lies** — DONE 2026-08-29 (todo-execution session) — dispatcher README claims `M` is
      the message type (it is the middleware type), claims pre-computed
      chains (code rebuilds per Dispatch), and lists nonexistent symbols
      (`LifecycleMixin`, `CatalogDispatcher`, `Handlers()`); metadata README
      drops ActorID from its Tracing snippet and calls command/query
      Metadata "standalone structs" (they are aliases).
      _(Effort: S)_
- [✓] **id.ActorID vs record.Actor zero-semantics asymmetry** — DOC-PINNED 2026-08-29 (todo-execution session 2) — `ActorID.IsZero` godoc documents the asymmetry (record.Actor{user, ""} is NOT zero; "user:" re-parses to an id-side zero dropping the kind) and instructs consumers to treat IsZero as "no meaningful actor" until the v5 unification.
      _(Effort: S doc / v5 unify)_

---

## Declined / Rejected (do not re-litigate)

- **command.Bus / MemoryBus removal (v5 candidate)** — DECLINED 2026-08-29
  (plan V3 T40): the in-process bus is 47 lines, is the delivery mechanism
  for the documented saga/example flows, and complements rather than
  competes with `watermill/` (brokers for multi-process topologies).
  Removing it would break examples and the saga pattern for zero
  simplification. Re-evaluate only if the saga pattern moves to a dedicated
  orchestration module.

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
