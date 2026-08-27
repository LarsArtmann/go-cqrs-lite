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

## Performance — 2026-08-16 audit backlog 🔥

> From the RAM/cache-line + IO audit session
> ([status](status/2026-08-16_03-10_perf-audit-cache-line-sql-batching-deserialize-wins.md),
> [Pareto plan](planning/2026-08-16_03-18_PERF-PARETO-SAFETY-FIRST-EXECUTION.md)).
> Shipped: workloadMeter cache-line pad (−46..51% contended ops), dialect-aware SQL
> batching (99→3276 rows on PG/MySQL/DuckDB; PG integration GREEN), pebble+bbolt
> deserialize JSON round-trip elimination (−46% ns / −53% allocs) — commits
> `cdc525fd5` + `a298ea388`. Wave 3 closed 2026-08-16 (measured numbers in
> [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md)).

- [x] **Durability tier→per-write-sync mapping** (storage/pebble
      hardcodes `pebble.Sync`; metaengine engines no NoSync path) — DONE
      2026-08-17 (Option A "align all layers"): `stack/pebble` maps
      Strict→sync, Normal→async WAL, Relaxed→DisableWAL+async (also fixes
      Relaxed forcing a memtable flush per write); new options
      `pebble.WithBackendAsyncWrites`, `pebbleengine.WithAsyncWrites`,
      `bboltengine.WithNoSync`; bbolt Normal≡Strict documented exception
      (no WAL); doc split brain fixed in `stack/durability.go`. Tier→options
      mapping tests + engine smoke tests green; throughput bench added
      (`BenchmarkEventAppendSync/Async`) — honest numbers PENDING a quiet
      window (see [BENCHMARKS.md](docs/BENCHMARKS.md): device queue
      saturated at load 3–4, sync≈async≈2.5 ms).
      _(Effort: M)_
- [x] **Benchstat baselines for the 3 new false-sharing control benches**
      (multiSeqCounter padded/unpadded, WorkerCounters, SSEReplaySeq) —
      committed 2026-08-17: [`benchmarks/2026-08-17_falsesharing-*.txt`](benchmarks/),
      benchstat tables + re-run commands in the
      [evidence doc](docs/benchmarks/2026-08-16_false-sharing-contention.md);
      SSEReplaySeq tie-breaker confirmed NO-PAD (padded +8% slower @16, tied @32).
      Not CI-gated (relative-comparison benches, see evidence doc rationale).
      _(Effort: XS)_

---

## Release / Tagging 🔥

> Blocked on user authorization (never tag/push without explicit instruction).
> Latest full-gate GREEN: 2026-08-16 13:15 (`#verify` run #4 EXIT=0, 591-line log,
> build+vet+test+race+lint 76/76, doc-check, api-surface; `#check-coverage` and
> `#check-duplication` also EXIT=0) — this is the release checkpoint. The
> 2026-08-16 chain (id v4.5.0, record v4.3.0, metadata v4.5.0, schema v4.3.0,
> event v4.7.0, query v4.6.0→retracted→v4.6.1, command v4.7.0→retracted→v4.7.1,
> middleware v4.5.0, metaengine v4.11.0, sqlite/pebble/pg engines v4.1.0,
> badger v4.0.2, watermill v4.5.0, mysql/bbolt/turso/iroh engines v4.0.0,
> storage v4.7.0→retracted→v4.7.1) is LIVE on the proxy.

- [x] **Land the stranded tag-chain repair commits on master** — done
      2026-08-22 wave: `092b5e8a8` was cherry-picked as `491379a2b` (on
      master, verified 2026-08-27 via `git merge-base --is-ancestor`);
      master's `command`/`query` go.mod pin `metadata/v4 v4.6.0` (the
      v4.4.0 pin-rot risk is gone). `4907b6afc` is obsolete: master's
      `metaengine/bench/go.mod` carries zero pseudo-versions (verified
      2026-08-27).
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

- [x] 🔥 **Repo-wide stale-pin sweep** — done 2026-08-27 (plan P06/T02):
      19 modules bumped to latest published tags (metaengine v4.12.0,
      storage v4.8.0, engine patches, watermill/testutil/middleware et al),
      each GOWORK=off build-verified; stack/* and transport/* deliberately
      untouched (v5 deletes them); eventtest dead tags v4.0.0/v4.2.0
      documented (Go rejects them — module path has no /vN suffix). Note: — benchkit still pins
      `sqliteengine v4.0.1` (pre-JournalReadFrom-fix), `decider v4.3.0`,
      `event v4.6.0`… Mechanical bump of ~50 go.mod files, gate-verified.
      (Needs user sign-off on policy — see ROADMAP Open Questions.)
      _(Effort: M)_
- [x] 🔥🔥 **storage/pebble + storage/bbolt standalone builds RED** — fixed
      2026-08-22 wave (pins bumped to `event/v4 v4.8.0`); re-verified GREEN
      2026-08-27 17:35 (`GOWORK=off go build -tags "goexperiment.jsonv2" ./...`
      EXIT=0 for both modules, clean tree).
      The original 2026-08-16 note for the record: both pinned
      `event/v4 v4.6.0` but `serialization.go` called
      `event.ReconstructEventWithAdoptedPayload` (shipped after v4.6.0) —
      same workspace-masking class as the command/v4.7.0 incident.
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
> `docs/status/2026-08-15_22-04_metaengine-followup-closeout.md` §f.

- [ ] **Turso explicit CTE-probe test** — the sqliteengine probe covers
      local drivers; add a tursoengine test confirming it holds over the
      remote protocol.
      _(Effort: S)_
- [x] **Badger engine vector + graph parity audit** — has neither; audit
      against the pebble/bbolt precedent and either implement or document
      the gap.
      _(Effort: S)_ — done 2026-08-16: `badgerengine/vector.go` + `graph.go`
      implement both ADTs at pebble/bbolt parity (filtered k-NN included);
      suite green.
- [x] 🔥 **Vector payload binary encoding (spike Phase 0)** — replace the
      JSON vector payload with fixed-width little-endian float32 bytes on
      the brute-force engines (memory ceiling ~90ns/vector vs pebble
      ~17µs today; decode is ~190x of the cost). Format marker must keep
      old JSON rows readable (self-describing envelope precedent). This is
      the unconditional 80/20 from
      [`docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md`](docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md)
      §4 Option A / §5 Phase 0; int8 quantization (Phase 1) only if p99 is
      still above budget afterwards.
      _(Effort: M)_ — done 2026-08-17: `metaengine.EncodeVectorBinary` /
      `DecodeVectorBinary` / `DecodeVectorAuto` (`'b'` marker + uint32 LE
      dim + LE float32s); pebble/bbolt/badger write binary and read via the
      sniffing decoder so legacy JSON rows keep working in place
      (mixed-format collections pinned by per-engine tests). Measured
      ~31-35x VectorSearch win (pebble 10K: 172.2ms → 5.63ms/query;
      ~460-560ns/vector vs the ~85-95ns in-RAM ceiling). pgengine stays
      JSON (JSONB column — needs a BYTEA DDL migration; deliberate scope
      cut, noted in the spike doc §4). Phase 1 (int8) remains deferred
      until p99 is above budget. Follow-up benches (same day) cover all
      three LSM engines (pebble 457µs/5.23ms, bbolt 426µs/5.79ms, badger
      647µs/5.85ms at 1K/10K), filtered k-NN (1034µs @1K pebble), and a
      codec micro-bench (196ns binary vs 8.5µs JSON decode @D=128) — spike
      doc §2.
- [x] **Vector search at scale** — quantization/HNSW spike for LSM engines
      when collections exceed ~100K vectors (brute-force scan is O(N)).
      _(Effort: L)_ — done 2026-08-16: spike complete with measured
      baselines (memory ~90ns/vector vs pebble ~17µs — the 190x gap is JSON
      decode, not the scan); phased plan (binary float32 → int8 quantization
      → optional HNSW with filter fallback) in
      [`docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md`](docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md).
      Implementation of the phases is NOT done — Phase 0 (binary encoding)
      is the follow-up.
- [x] **`VectorResult` filtered k-NN** — metadata-filtered vector search
      (filter + top-k in one query); API currently returns bare top-k.
      _(Effort: M)_ — done 2026-08-16: `VectorSearchFiltered` +
      `VectorFilter`/`VectorFilterBackend` natively on memory, badger, pebble,
      bbolt, iroh (passthrough); the generic Store path falls back to
      filter-then-rank for any engine without the capability, so filtered
      k-NN works everywhere vectors work. Filters ride
      `Embedding.Metadata` and upsert clears stale metadata; adttest
      `VectorFiltered` scenario pins cross-engine parity.
- [x] **`GraphRemoveEdge`** — the edges table exists but nothing removes
      edges; tombstone events should drive edge removal (ADR-0114 style).
      _(Effort: M)_ — done 2026-08-16: `GraphRemoveEdge` + `HasGraphEdgeRemoval`
      on memory, badger, sqlite, pg, mysql, dgraph (symmetric both-direction
      delete), graphadapter, iroh (passthrough); `FoldEdgeRemove` folds
      removals into the store; adttest `GraphRemove` scenario. (pebble/bbolt
      have no graph ADT at all — unchanged.)
- [x] **Graph directed-vs-undirected option** — `GraphNeighbors` is
      directed-only today.
      _(Effort: S)_ — done 2026-08-16: `GraphNeighborsUndirected` +
      `HasUndirectedGraphSupport` on memory, badger, sqlite (recursive CTE
      both directions), pg, mysql (derived-table seed + OR-join recursive
      arm — the 4-arm form is illegal in both), dgraph (alias of directed:
      storage is symmetric), iroh (passthrough); graphadapter deliberately
      does NOT implement it (documented gap). (pebble/bbolt: no graph ADT.)
- [x] **mysqlengine upsert semantics audit** — confirm `MapSet` uses
      `INSERT ... ON DUPLICATE KEY UPDATE` consistently with pg's
      `ON CONFLICT` (atomicity + affected-rows parity).
      _(Effort: S)_ — done 2026-08-16: parity confirmed for `MapSet`,
      `CounterIncrement`, `GraphAddEdge` (single-statement atomic upserts;
      affected-rows difference unobservable); caveats documented in
      `mysqlengine/backends.go` (any-unique-key trigger widening, VALUES()
      deprecation note).
- [x] **MariaDB functional-index alternative** — generated columns + plain
      index instead of the current `ApplyLayout` no-op.
      _(Effort: M)_ — done 2026-08-16: `applyMariaDBLayout` adds a VIRTUAL
      TEXT generated column (no truncation, no table rebuild) + composite
      `(collection, gc(N))` prefix index per field; `filterExpr` rewrites
      pushdown filters to the column (MariaDB does NOT substitute generated
      columns into JSON expressions — verified 11.4); EXPLAIN-verified
      `ref` access; integration test `TestMariaDBApplyLayout_GeneratedColumnFilter`.
- [x] **enginetest per-run collection suffixes** — shared-server engines
      accumulate state under `-count>1` (documented constraint that bit the
      race runs); per-run suffixes in the helpers would remove the class.
      _(Effort: M)_ — done 2026-08-16: `enginetest.ScopedCollection` at all
      helper chokepoints + adttest `Scenarios()` run-suffix (per-RUN token,
      keeping cross-engine parity comparable within a run); mysqlengine
      verified GREEN `-count=3` against a shared MariaDB; twin fix in
      `stack/mysql` (DROP before CREATE for derived multidb databases).
- [x] **adttest: graph depth>2 + cycle scenarios in `RunMatrix`** — current
      matrix is depth-limited; the CTE/iterative divergence only shows at
      depth>2 with cycles/diamonds.
      _(Effort: S)_ — done 2026-08-16: `GraphDepth3Diamond`, `GraphCycle`,
      `GraphDepthBound` scenarios; parity GREEN incl. pg CTE path.
- [x] **adttest: Vector scenario on pgengine** — parity check for the
      degraded scan path against the in-memory index.
      _(Effort: S)_ — done 2026-08-16: pgengine gained a real degraded
      `VectorBackend` (was declared but unimplemented — queries would have
      failed); `meta_vector` DDL + shared `metaengine` distance helpers;
      GREEN against real Postgres.
- [x] **Convergence suite order-tolerance audit** — `sameLogTail` was the
      only order-asserting helper (fixed 2026-08-15); sweep the remaining
      `waitFor*` helpers for hidden order assumptions.
      _(Effort: S)_ — done 2026-08-16: all helpers order-safe; one stale
      contradictory doc paragraph on `sameLogTail` removed.
- [x] **quic pooled-stream ordering guarantee** — default `sendOp` uses one
      stream per op (no cross-op order); verify + document that pooled mode
      (`sendOpPooled`, one stream per peer) DOES order ops.
      _(Effort: S)_ — done 2026-08-16: per-peer FIFO verified (sender mutex
      serialization + sequential receiver loop + QUIC stream ordering);
      documented in `options.go`, `transport.go`, `pool.go`.
- [x] **Bench: CTE vs iterative BFS crossover** — at which depth does the
      MySQL CTE beat the iterative fallback? Feeds the planner's cost model.
      _(Effort: S)_ — done 2026-08-16: `graph_bench_test.go`; crossover at
      depth 2-3 (iterative wins depth 1 by 2-4x, CTE wins ≥3 up to 6x,
      size-independent 1k-100k, same shape MariaDB 11.4 + MySQL 8.4);
      table in `METAENGINE-LIVE-LATENCY-MODEL.md` §9. Follow-up item below.
- [x] **Bench: MariaDB dual-key sort cost** — measure the
      `CAST(... AS DECIMAL)` overhead vs MySQL's single JSON key.
      _(Effort: S)_ — done 2026-08-16: `sort_bench_test.go`; dual-key costs
      +26% vs single-expression on both servers; MySQL JSON-typed arrow
      form is 2.5x faster than MariaDB's dual-key (19 vs 47ms / 50k rows);
      table in `METAENGINE-LIVE-LATENCY-MODEL.md` §9.
- [x] **mysqlengine: depth-1 graph short-circuit** — measured 2-4x win
      (bench table in `METAENGINE-LIVE-LATENCY-MODEL.md` §9): route
      `GraphNeighbors(depth==1)` to the direct adjacency query (+ `AND
      to_node <> ?` to preserve start-node exclusion) instead of the
      recursive CTE.
      _(Effort: XS)_ — done 2026-08-17: directed + undirected depth-1 both
      short-circuit ahead of the CTE/iterative switch (the recursive arm
      contributes zero rows at depth 1, so the CTE seed alone is provably
      equivalent); per-engine CTE-parity tests against MariaDB 11.4;
      single-query drain consolidated into `queryGraphRows`.
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

- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Track which can be removed after migrations complete.
      _(Effort: M)_

---

## WithActor Hardening

> `event/command/query.WithActor` + `Tracing.ActorID` shipped and released
> (metadata/v4.4.0, event/v4.6.0, command/v4.6.0, query/v4.5.0 — 2026-08-13).
> id/v4.4.0 contains `actor_id.go` (verified in tag).

- [x] **Test-coverage gaps** — golden JSON for full `event.Event`/`command.
      BasicCommand` with ActorID; watermill wire-format preservation; CBOR
      roundtrip (events default to CBOR); SQL `MarshalMetadata` scan path;
      pebble/bbolt encode/decode; e2e decider (command→events) and projection
      (events→read) propagation; `TestQuery_AllMetadata`; json/v1 fallback
      `omitempty` behavior.
      _(Effort: M)_ — done 2026-08-17: most lanes were already shipped by the
      actor wave (`1153c7d11`/`842741cab`) and re-verified green (watermill
      round-trips, `TestMarshalMetadata_ActorRoundtrip`, `TestStoreMetadataRoundtrip`
      in pebble+bbolt, `integration.TestActorPropagationEndToEnd`,
      `TestQuery_AllMetadata`, `TestTracing_JSONv1Fallback`). Newly added:
      `event/golden_metadata_test.go` + `command/golden_metadata_test.go`
      (golden JSON + store-load round-trip) and
      `event/metadata_cbor_test.go`. The CBOR lane FOUND A REAL DEFECT: every
      module pinned `id/v4 v4.4.0`, which lacks `ActorID.MarshalBinary`
      (first tagged in `id/v4.5.0`) — GOWORK=off/published consumers silently
      lost the actor in CBOR. Fixed by bumping all 59 pins to v4.5.0 (see
      CHANGELOG Fixed entry).
- [x] **Ecosystem propagation checks** — scenario DSL actor support;
      `scheduling/`, `deriver/`, `commandlifecycle/` ActorID propagation;
      ActorID-from-context middleware; `id.ActorID.Validate()`.
      _(Effort: M)_ — done 2026-08-17: scenario (`TestGiven_When_ThenEvents_ActorMetadata`),
      deriver (`TestDeriver_Idempotent_PreservesActor`), commandlifecycle
      (`commandTracing` recorder), middleware (`middleware.CommandActorContext`),
      and `id.ActorID.Validate` were already shipped — verified green. The one
      genuine gap was `scheduling/`: `Timer[P].Actor` added (plain string in
      the "kind:raw" wire format — zero-dep module, `record.CommonMetadata`
      precedent) + versioned payload envelope in `scheduling/sqlstore` with
      legacy bare-payload fallback, so timer-initiated commands can carry the
      originating actor durably (see CHANGELOG Added entry).

---

## Code Quality / Infrastructure

- [x] **CHANGELOG honesty gate** — lint that every `pkg.Symbol` identifier
      cited in CHANGELOG Added/Changed entries exists in the api-stability
      golden. Kills the reverted-work fiction class mechanically (the
      2026-08-10/11 tombstone entries described `e406edcfb`, which was
      reverted by `a6613ef0d` before any tag — CHANGELOG never recorded the
      reversion; corrected 2026-08-16).
      _(Effort: S)_ — done 2026-08-16: `scripts/check-changelog-symbols.sh`
      (CI + nix-less grep gate) scopes `[Unreleased]` Added/Changed headings
      (inclusive), resolves `alias.Symbol` against golden last-segment,
      module-root prefix, and repo source-dir fallback (subpackage citations
      like `enginetest.X`); caught two inaccuracies in this session's own
      CHANGELOG fold on first run.
- [x] **api-stability: fail loudly on parse-skip** — the checker prints `skip
      <module>:` and proceeds when a file is unparseable, so a corrupted
      module looks identical to a legitimately-removed one in the golden
      (a silently-shrinking golden is the corruption tell, 07:12 report §e.2).
      Cheapest corruption tripwire available.
      _(Effort: XS)_ — done 2026-08-16: unparseable modules now return a hard
      error instead of `skip <module>:` + continue.
- [x] **BuildFlow pre-commit: `gofmt -l` syntax gate on staged `.go` files** —
      concurrent-session mid-write corruption entered the index twice on
      2026-08-16 (`func (w *workor)`, `fojection.`); a 1s syntax check on
      staged files blocks the class.
      _(Effort: XS)_ — done 2026-08-16: `scripts/check-staged-go.sh` (gofmt
      `-e -l`, `|| true` so gofmt's exit-2 on parse errors feeds the message
      check instead of tripping `set -e`), wired into the installed hook,
      `scripts/install-hooks.sh` (canonical restorer of BOTH post-BuildFlow
      gates), and `scripts/pre-commit.sh`. Negative test: staged corrupt file
      → precise parse errors, exit 1.
- [x] **Pre-gate load-sweep script** — run timing-assertion tests
      (`-run 'Latency|Timer|Deadline'`) under a CPU soaker before `#verify`;
      the 12:39 session burned two full gate cycles (~20 min each)
      discovering load-sensitive flakes one at a time.
      _(Effort: S)_ — done 2026-08-16: `scripts/load-sweep.sh` + `nix run
      .#load-sweep` (nproc-1 soakers, 8 timing-test modules, per-module
      GOWORK=off, logs to /tmp/load-sweep-*.log).
- [x] **Guard against local-path `replace` directives** — cqrs-lint rule,
      buildflow check, or both: reject `replace … => /home/…` (broke CI
      Release on every push until `ceb88738b`; dev-against-siblings belongs
      in `go.work` `use`, never in a published go.mod). Approach is a user
      question (11:00 report §g Q2).
      _(Effort: S)_ — done 2026-08-16 (approach decided: extend the existing
      CI-wired script, cheapest complete enforcement; cqrs-lint rule would be
      duplicate machinery): `check-replace-directives.sh` now rejects
      absolute-path targets outright; relative sibling replaces stay the
      documented convention (stripped by `tag-release.sh` at cut time).
- [x] **Clean up registered git worktrees** — `/tmp/cqrs-tagwt` (tag chain,
      commits `092b5e8a8`/`4907b6afc` stranded there), `/home/lars/projects/
      wt-head`, `/tmp/gcl-verify`, `go-cqrs-lite-pin` still registered.
      After the stranded-commit cherry-pick lands.
      _(Effort: XS)_ — done 2026-08-16: the valuable stranded commit
      `092b5e8a8` (metadata/v4.5.0 pin fix + retracts + tag-release.sh
      GOWORK=off-build gate) was cherry-picked to master as `491379a2b` and
      verified with standalone GOWORK=off builds of command+query;
      `4907b6afc` was obsolete (bboltengine v4.0.0 pin already on master);
      all four worktrees removed (pin's dirty docs were a dubious
      `npx`→`pnpm dlx` sweep over archive files — discarded deliberately).
- [x] **Infrastructure polish (nix apps + shared helpers)** — add
      `#check-lint-config`, `#verify-ci` (mirror GH Actions GOWORK=off
      per-module), wire `#sweep` to pre-commit/cron, consolidate engine
      `register.go` boilerplate (7 modules), add missing devShell tools
      (dprint, go-licenses, vulnix — their absence forces `--no-verify`).
      _(Effort: M)_ — done 2026-08-16: `#check-lint-config` (golangci config
      verify + depguard), `#verify-ci`, `#load-sweep`, `#integration-redis`
      apps added (flake eval verified); devShell already had dprint/
      go-licenses/vulnix (stale premise); `register.go` "consolidation" = 8
      files standardized with `//art-dupl:accept` directives — Go REQUIRES a
      per-package `init()` for driver self-registration (database/sql
      pattern), so cross-module dedup is impossible by design (documented in
      AGENTS.md #19); `#sweep` stays manual/cron (not in pre-commit — its
      5-minute lint pass would dwarf every other gate).
      cron line: `0 * * * * cd <repo> && nix run .#sweep`
- [x] **Fix `nix fmt` vs golangci-gci import-grouping conflict at the tooling
      level** — the class re-broke 95+ files once; it WILL recur on every
      future import addition. Make the treefmt formatter config aware of gci
      section rules (or vice versa).
      _(Effort: S)_ — done 2026-08-16 ("vice versa"): gci REMOVED from
      `.golangci.yml` formatters — treefmt goimports `-local` owns the
      3-group layout, CI's `nix fmt --fail-on-change` enforces it
      mechanically; one tool owns grouping, the formatter. treefmt-nix has
      no gci program at the pinned rev, and golangci-as-treefmt-formatter
      would type-check on every `nix fmt`.
- [x] **Enforce the heap-measurement contract mechanically** — tests calling
      `runtime.ReadMemStats` must never `t.Parallel()` (13 files fixed
      2026-08-14; repo-wide audit came back clean, but nothing prevents
      reintroduction). cqrs-lint rule or check-script grep.
      _(Effort: S)_ — done 2026-08-16: `scripts/check-heap-parallel.sh`
      (same-file grep tripwire, CI-wired); cross-file callers of soak runners
      remain a review concern (documented in-script).
- [x] **Audit benchkit's remaining wall-clock assertions** —
      `raceEnabled` branch at `benchkit_test.go:821` may share the
      load-sensitivity mis-model already fixed for `DurationAborts`/
      `CancelledContext` (flat 30s).
      _(Effort: S)_ — done 2026-08-16: confirmed the mis-model (5s non-race
      ceiling under parallel load); aligned `TestRun_SQLite_DurationAborts`
      to the flat-30s hang-detection model; verified 3x under `-race`
      (22s, green).
- [x] **Wire broker tests into CI** — `TestRedisStreamRoundtrip` passes only
      via manual `ephemeral-redis.sh`; add a `#integration-redis` nix app
      (mirrors `#integration-pg`). Then extend to broker edges the gochannel
      tests can't catch: redelivery duplicates, consumer-group rebalance,
      message size limits.
      _(Effort: M)_ — done 2026-08-16: `#integration-redis` app + CI
      `redis-integration` job; `ephemeral-redis.sh` default command now runs
      the watermill suite; new `watermill/broker_edge_redis_test.go` covers
      Nack redelivery (NackResendSleep), consumer-group exactly-once across
      two subscribers (20 msgs, no dupes/losses), and 2 MiB payload
      integrity — all green against a real ephemeral broker.
- [x] **Doc-check 0-warning CI tripwire** — warnings are at 0 since
      2026-08-15; add a regression guard so warning spam can't creep back.
      _(Effort: XS)_ — done 2026-08-16: doc-check now collects warnings
      (unreadable dirs, empty exports, unparseable files) and FAILS on any;
      the zero-references case (silent no-op verification) is an error too.
      Full verify file set passes with 0 warnings (1139 refs).
- [x] **Skill docs: capability diagnostics recipe** — `recipes.md` section on
      `CapabilityAudit`/Doctor's `--- Capability ---` section
      (consumer-facing declared-vs-implemented diagnostics), and a
      `modules.md` metaengine row note. Shipped 2026-08-16, undocumented.
      _(Effort: XS)_ — done 2026-08-16: recipes §2.12 (CapabilityAudit +
      Doctor snippet + RunCapabilityConformance gate) + modules.md metaengine
      row note; doc-check green (1139 refs).
- [x] **Duplication-baseline hygiene** — add `//art-dupl:accept` directives
      at the 9 intentional clone sites; dirty-tree guard for baseline
      re-pins; re-pin at next clean tree (current pin includes in-flight
      foreign code).
      _(Effort: S)_ 2026-08-16: wave-4 triage added 3 more directives
      (2 false-sharing bench mirrors, 1 cross-module test read-file idiom)
      and consolidated 2 groups (metaengine `enginesSnapshot`, event
      `seekableReadFrom`) — gate green, baseline untouched (annotations
      suppress live). The 9 legacy sites + dirty-tree guard remain.
      2026-08-16 (infra wave): dirty-tree guard DONE (`#check-duplication`
      refuses uncommitted baseline changes); 8 `register.go` driver files
      annotated (`//art-dupl:accept`).
      2026-08-16 (closeout): concurrent session finished (engine-correctness
      wave committed `d7e583c82`), so the 12 flagged groups in its files were
      annotated by this session: cross-engine dialect twins (mysql/sqlite
      graph_undirected dispatch + row scanners, pebble/pg vector prologues,
      badger/sqlite marshal-with-fallback helpers), same-file twins
      (memory_graph RLock+BFS prologues, badger directed/undirected guards,
      metaengine reflect extractors), and quickstart demo setup. Iterative —
      suppressing the visible 5 exposed 7 masked groups (art-dupl reports
      one group per region pair; always re-run to 0). Gate green with
      baseline untouched; re-pin DROPPED (live annotations are the
      documented preference — a deleted annotation re-flags its group,
      forcing re-justification). Directive placement gotcha: it must sit
      directly on/above the region start, not above the following func's
      doc comment.
- [x] **check-coverage.sh hardening** — meta-test asserting every EXPECTED
      key resolves to a real module dir (codec-dangle class); make
      `--update` auto-stamp the "verified" date.
      _(Effort: S)_ — done 2026-08-16: dangling EXPECTED keys fail fast with
      a precise message; `--update` rewrites the header's `(verified …)`
      date via sed. Gate green (all 11 modules within ±2%).
- [x] **duckdbengine suite split** — 76-91s observed under the (now 8m)
      ceiling; worth splitting the soak into its own budget.
      _(Effort: S)_ — done 2026-08-16: the soak's comment CLAIMED `-short`
      skip but the code never checked it (doc-vs-code split brain —
      `#verify-fast` was running the 80-100s soak); `testing.Short()` gate
      added, verified skipping under `-short`. Full `#test`/
      `#test-all-backends` still cover it.
- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin. (Last open item of the
      19-item Code Quality / Infrastructure section — 18/19 done.)
      _(Effort: M)_ — 2026-08-16: static review done (portability note added
      to the script header; no Linux-isms found, /dev/kvm check
      uname-guarded); hardware verification on a real Mac remains open.
- [x] **Delete junk from repo root** — `t/`, `result/` (16MB root-owned),
      `reports/coverage.out` (empty), `reports/jscpd-report.json`; drop the
      orphaned `stash@{0}` (WIP @ `e87be3143`, pre-recovery leftovers).
      _(Effort: XS)_ — done 2026-08-16: all three dirs trashed (root-owned
      `result/` moved via parent-dir rename), stash dropped (dropped-hash
      `cfca37601` recoverable from reflog-era output if ever needed).
- [x] **Per-module CHANGELOGs** — 6 of ~82 modules have one. Decide policy
      (root CHANGELOG only vs per-module) and either write them or document
      the decision.
      _(Effort: L)_ — done 2026-08-16 (decision: ROOT ONLY): survey found 4
      (not 6) module files, all orphaned (nothing read them) and stale —
      three described already-tagged work as `[Unreleased]`. Unmirrored
      content folded into the root CHANGELOG (the honesty gate caught two
      inaccurate symbol citations during the fold), files deleted, policy
      + enforcement documented in CONTRIBUTING.md and AGENTS.md #20.

---

## Correctness Defect Sweep (brutal-review backlog)

> From `docs/reviews/2026-08-14_14-25_brutal-self-review.md`. All other
> verified findings shipped 2026-08-15/16 — see CHANGELOG. These remain:

- [x] **Correctness-sweep leftovers** — `kv.Cache` shared `*T`;
      TypedQueryStore hardcoded JSON decode (`query/typed.go`); ghost
      `event.ErrBinaryNotFound` (document or delete).
      _(Effort: M)_ — done 2026-08-16: kv.Cache copy-isolates values (Get
      returns a deep copy, Set caches a private copy); all four blind stores
      decode non-envelope data via the configured codec + JSON↔CBOR
      cross-retry (ADR-0050 addendum); ErrBinaryNotFound deleted (orphan of
      the removed event/blob.go helpers).

---

## system/v4 Full-Code-Review Follow-Ups (2026-08-16/17)

> From `docs/reviews/2026-08-16_full-code-review-system.html`. All 5 P1 and
> the actionable P2/P3 findings are FIXED (commits a211ebcb2, 449e0e5a7,
> 42dfab5b0 — shipped with regression tests). These remain, routed:

- [x] **Count-by-name dispatch (P1-2)** — DONE 2026-08-18: metaengine
      `ExecuteQueryByName`/`ExecuteTypedByName` (unknown name →
      `ErrNoQueryForName`); `GetCount` dispatches by counter name; regression
      test covers two counters. Type dispatch kept, shadowing documented.
      Proposal: docs/adr/2026-08-17_system-v4-review-proposals.md.
      _(Effort: M, impact: correctness for multi-counter domains)_
- [x] **Named-bus API** — DONE 2026-08-18: `MultiBus.AddNamedPublisher`/
      `PublisherByName`/`Names` + `System.PublisherFor(target)` bind fan-out
      buses by their YAML `publish:` target; closers named
      `fanout-bus-<target>`. _(Effort: M)_
- [x] **Role wiring** — DONE 2026-08-18: dedicated RoleCommands/RoleQueries/
      RoleSnapshots instances bind stores from their own engines
      (`System.CommandStore`/`QueryStore` accessors); one engine may serve
      multiple roles; duplicates → `ErrDuplicateInstanceRole`; missing
      SnapshotBackend → `ErrNotSnapshotBackend`. _(Effort: M)_
- [x] **Reserved-config honesty** — DONE 2026-08-18: Mode documented
      introspection-only (README `mode: sync` example removed), Subscribe +
      CacheConfig.Engine documented reserved/not-read (removal at v5),
      Collections documented introspection-only, `Evolve(Internal())`
      documented recorded-but-not-enforced. _(Effort: S per field)_
- [x] **Durability wiring** — DONE 2026-08-18: `DriverConfig.Durability` +
      tier constants + `ValidateDurabilityTier`/`RejectDurabilityTier` in
      metaengine; sqlite maps tier → `PRAGMA synchronous` (conflicting
      operator pragma errors); memory rejects strict; system resolves
      per-engine tiers (conflict → `ErrDurabilityConflict`) and no longer
      silently defaults instances to "normal". _(Effort: M)_
- [x] **Durability breadth** — DONE 2026-08-18: pebble (WAL/sync/async/
      DisableWAL via `WithAsyncWrites`/`WithDisableWAL`), postgres
      (`synchronous_commit` DSN runtime param, conflict = config error),
      bbolt (normal aliases strict — no middle tier; relaxed = `WithNoSync`),
      badger (async floor via new `Option`/`WithAsyncWrites`) all map
      explicit tiers now; dgraph/duckdb/mysql/turso still reject.
      `BusConfig.Mode` endgame DECIDED: remove at v5, never implement.
      _(Effort: M)_
- [x] **EventAdapter backend contract doc** — DONE 2026-08-18:
      `system/doc.go` documents Atomic (all shipped engines) vs
      Transactional vs racy fallback for Save. _(Effort: S)_
- [x] **Release coordination: system/v4.5.0** — DONE 2026-08-18: all 7 tags
      cut, pushed, and VCS-verified (metaengine/v4.12.0, sqliteengine/v4.2.0,
      pebbleengine/v4.2.0, badgerengine/v4.1.0, bboltengine/v4.1.0,
      pgengine/v4.2.0, system/v4.5.0). Zero contingency tags were needed:
      record/v4.2.0, id/v4.5.0, watermill/v4.4.0 all satisfied the stripped
      standalone builds (the §g question dissolved). Post-wave: clean-room
      consumer build of system/v4@v4.5.0 GREEN; 10 stale-pin consumer modules
      standalone-build GREEN (no sweep needed); vulncheck + changelog-symbol
      gate + api-stability meta-tests GREEN. Also unblocked en route: reverted
      a gci/depguard .golangci.yml regression and trimmed AGENTS.md 399 → 369
      (BuildFlow ≤377 gate). GitHub Releases skipped (repo convention: one
      legacy entry only). _(Effort: M)_
- [ ] **Fix GitHub Actions billing** — BLOCKED on user (2026-08-18): every
      paid CI job (Release, Benchmarks, ci.yml) fails in 3-7s with "recent
      account payments have failed or your spending limit needs to be
      increased"; broken since ~2026-07-17. Local `nix run .#verify` remains
      the authoritative gate until billing is restored. _(Effort: S, user
      action: Billing & plans settings)_
- [x] **stack.Bundle cross-check** — VERIFIED 2026-08-17 (proposal §8,
      read-only): Bundle has no ack/WARN machinery at all (no CheckSafety
      equivalent) — nothing shares the fixed scream-store bugs, nothing to
      port. _(Effort: S, impact: correctness on the still-shipped stack path)_
- [x] **system/ coverage 74.4%** — DONE 2026-08-18: raised to 79.4% —
      CachedEventStore passthroughs/stats/capability fallbacks,
      CommandAdapter batch + time-filtered loads + journal reads,
      EvolveKey/Internal options, and the reifyTo JSON branch (sqlite
      explicit-fold test). _(Effort: M)_
- [x] **Host buildcache repair** — DONE 2026-08-18: /mnt/buildcache
      repaired (64% used, writable); the /tmp cache env workaround can be
      retired.

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
- [ ] **v5 decision: kvstore SA1019 exclusion** — keep the scoped
      `.golangci.yml` exclusion (`(middleware|idempotency)/.*_test\.go$`) as
      the permanent answer, or migrate the kvstore test matrices onto the
      go-idempotency contract suite before the cut.
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
> + CHANGELOG `[Unreleased]` entry in the same change.

- [x] **`AsRecord` bridges populate a validated `record.StreamRef`** 🔥 —
      adopt the shipped `Validate()` in all three bridges (event/command/
      query); `event.AsRecord` stamps streamType, command/query keep the
      empty-type form. Kills P1's interchange half; per owner decision there
      is NO new identity type (plan T04). ✅ 2026-08-22 (T04, prior session)
      _(Effort: M)_
- [x] **`record.Cause{Kind, ID}` + `CommonMetadata.Cause`** 🔥 — CauseKind
      iota (none/command/timer/derivation), zero value = none; deprecate
      `CausationID string`; event bridge maps `Causation.CommandID`/
      `Tracing.CausationID`. Kills P4's two-causation-homes (plan T05).
      ✅ 2026-08-22 (T05; shipped with a 5th kind `CauseUnknown` — the bare
      tracing chain does not discriminate the causer's kind)
      _(Effort: M)_
- [x] **`Record.ID string` + `Record.Encoding string`** 🔥 — `event.AsRecord`
      fills both so self-describing payloads survive the bridge (mixed
      JSON+CBOR round-trip test); command/query fill ID. Kills P5 (plan T08).
      ✅ 2026-08-22 (T08; `Encoding string` — the codec layer's self-describing
      form per the ADR-0044 envelope — instead of the sketched `uint8`, which
      would invent a parallel numeric mapping nothing else reads)
      _(Effort: M)_
- [x] **`record.Stamp{at, known}`** — replaces the three zero-time
      timestamps (`ClientCreatedAt`/`ServerReceivedAt`/`ServerStoredAt` →
      `Created`/`Received`/`Stored`); deprecate old fields. Kills P7 (plan
      T06). ✅ 2026-08-22 (T06; query bridge stamps `Received` — the honest
      home for the server clock the old field parked in `ClientCreatedAt`)
      _(Effort: M)_
- [x] **Structural `record.Actor{Kind, Raw}` mirror** — bridges stop
      stringifying the actor union; wire form stays only at the serialization
      edge. Kills P3 parse-tax (plan T07). ✅ 2026-08-22 (T07, via
      `metadata.RecordActor`; legacy `UserID` fallback upgraded to
      `ActorUser`)
      _(Effort: M)_
- [x] **Metadata capability interface** 🔥 — additive `MetadataCarrier`-style
      interface adopted by `query/audit` middleware; deprecate the two
      duck-typed interfaces (`query/audit.go:86,114`). Growing the exported
      `Command`/`Query` interfaces with `Metadata()` is BREAKING for
      hand-rolled implementations — decided to ride the v5 cut (owner input
      2026-08-22). Kills P6 (plan T09). ✅ 2026-08-22 (T09: `query.MetadataCarrier`
      + `query.PayloadCarrier` + `command.MetadataCarrier` shipped; audit.go
      migrated; Appendix C comparison memo; gate green `8c0f48ab0`)
      _(Effort: M)_
- [x] **`Decider.ExecuteRef`/`LoadRef` additive variants** — pair forms
      delegate + get `Deprecated: removed in v5`; migrate internal callers
      (scenario/, examples). One identity convention on the hot path (plan
      T10). ✅ 2026-08-22 (T10: ref forms are the real impls on `id.StreamRef`;
      `system/register.go` migrated — the only production pair-form caller;
      lockstep tests `decider/ref_forms_test.go`; gate green `8c0f48ab0`)
      _(Effort: M)_
- [x] **Brand `scheduling.TimerID` + `Timer.Actor id.ActorID`** —
      `id.Of[TimerMarker]`; JSON wire form via PrefixedString round-trip; add
      `id/v4` dep (Tier-1→Tier-0 legal) + `#check-arch`; consumer-pin sweep
      in the same wave. Kills P11 (plan T11). ✅ 2026-08-22 (T11 shipped
      `82517580d`: string-backed branded TimerID — DEVIATION from the
      `id.Of[TimerMarker]` sketch, timer IDs are semantic idempotency keys;
      Actor→`id.ActorID` wire-identical; SQL envelope actor stays a string
      column with boundary conversion; dep budget 0→2; gate green with T12)
      _(Effort: M)_
- [x] **`record.Type` consolidation** — define once, alias in event/command/
      query; collapse triplicated ParseType/IsZero. Kills P12 drift (plan
      T12, after the StreamRef bridge work). ✅ 2026-08-22 (T12: `record.Type`
      + parametrized `record.ParseType(s, emptyErr)`; event/command/query
      `Type` are aliases with deprecated sentinel-preserving wrappers;
      compile-time alias lockstep tests; golden regen; exclusive gate green
      `f518d24a0`)
      _(Effort: M)_
- [x] **`snapshot.NewSnapshot` constructor + codec stamp** — envelope-style
      encoding stamp (ADR-0044 pattern), `Validate`, invariants (non-nil
      State, Version ≥ 1). Kills P10 (plan T17). ✅ 2026-08-22 (validating
      ctor + `Validate`/`Ref`/`ErrInvalidSnapshot` + `Encoding record.Encoding`
      stamp on Snapshot; TypedStore + decider stamp on save; `SaveSnapshot`
      deprecated; per-module test+lint green, golden +4, arch exception
      snapshot→storage/memory test-only)
      _(Effort: M)_
- [x] **Snapshot wire-tag migration audit** — pebble/bbolt/sql/memory
      backends; decide keep-old-tags-until-v5; note into §v5 Unification
      (plan T18, after the constructor). ✅ 2026-08-22 (audit table in §v5
      Unification above; keep-old-tags decision recorded; pebble+bbolt now
      PERSIST the encoding stamp additively with roundtrip tests; SQL
      envelope stays authoritative)
      _(Effort: S)_
- [x] **Deprecation census artifact** — exact 36-alias list + snapshot wire
      tags + stale error codes → `docs/planning/v5-deprecation-sweep.md`
      checklist feeding §v5 Unification (plan T19). ✅ 2026-08-22 (artifact
      emitted: 42 aggregate-vocabulary symbols across 6 modules + 5
      record-bridge fields + tombstone API + wire-tag/error-code inventory
      + execution rules)
      _(Effort: S)_
- [x] **Tombstone v5 deletion prep** — verify migration doc accuracy +
      `listing.StatusMiddleware` bridge test coverage (plan T20).
      ✅ 2026-08-22 (doc verified against shipped APIs; found + fixed a
      REAL gap: OnTombstone/OnRebirth had zero trigger coverage anywhere —
      added `stack/materialize_tombstone_bridge_test.go` pinning the full
      StatusMiddleware→mark→OnTombstone/OnRebirth chain; deletion pre-reqs
      listed in the §v5 tombstone entry above)
      _(Effort: S)_
- [x] **Extended data-model review** — storage/*, system/, stack/,
      watermill/, middleware/ get the same rigor; findings appendix
      cross-linked to the core review (plan T21). ✅ 2026-08-22
      (`docs/reviews/2026-08-22_extended-data-model-review.md`: 15 findings
      E1-E15 + capability matrix; verdict: no any-typed values anywhere,
      the disease is cross-backend drift; pebble lying-doc fixed on the
      spot). Follow-ups extracted below.
      _(Effort: L)_
- [x] **bbolt error-family parity** (review E3) — replace bare `fmt.Errorf`
      in bbolt command/query marshal+reconstruct with errorfamily codes,
      matching pebble's pattern. ✅ 2026-08-27 (deep full-code review;
      `bbolt.serialize_command`/`reconstruct_command`/`serialize_query`/
      `reconstruct_query` Corruption contexts)
      _(Effort: S)_
- [x] **turso.Policy nil-map write guard** (review E9) — mutators panic on
      zero-value Policy; guard writes like reads. ✅ 2026-08-27 (nil-receiver
      no-op + lazy map init, zero-value tests added)
      _(Effort: S)_
- [x] **system.ShutdownDependency name validation** (review E10) — validate
      Before/After against DeploymentConfig.Engines at build; typo'd names
      currently silent-no-op. ✅ 2026-08-27 (`validateShutdownDependencies` in
      `system.New`, `ErrShutdownDependencyInvalid` sentinel, 3 integration
      tests)
      _(Effort: S)_
- [ ] **v5 items from extended review** — E1 (event-envelope Encoding →
      `record.Encoding`), E7 (watermill/middleware RetryConfig collision),
      E8 (typed Message Kind enum), E11 (AdapterCore.Encode error return),
      E13 (SQLTimerStore phantom param), E15 (middleware signature
      unification) — fold into the v5 cut wave after the sweep doc.
      _(Effort: M)_
- [x] **Stream/StreamRef/StreamID naming decision** — the trio is itself a
      naming smell; decide each shape's role before v5 (plan T22, after the
      owner's Option B decision). ✅ 2026-08-22 (plan Appendix D: trio
      KEEP with role table; real smell found = `record.StreamRef` string
      vs `id.StreamRef` struct name collision + `/` vs `:` separator
      divergence → v5 rename `record.StreamKey` added to the sweep doc;
      ActorKind mirror re-confirmed as accepted zero-dep pattern)
      _(Effort: S)_
- [ ] **Post-landing sweep for this series** — api-stability meta-tests,
      doc-check over skill refs, consumer-pin sweep for `record/v4` consumers
      under GOWORK=off (plan T24; MarshalBinary lesson).
      _(Effort: M)_
- [ ] **AGENTS.md memory: data-model conventions** — record T01 outcome +
      new conventions (validating-population pattern, capability-interface
      rule) once the PRs land (plan T25).
      _(Effort: S)_
- [x] **Report polish remnants** — programmatic TOC-anchor check + CSS
      template-diff audit for the core review (Related reviews / Next skills
      sections already added 2026-08-22) (plan T16). ✅ 2026-08-22 (anchor
      check: 0 broken after adding Amendments to the TOC + Related
      Reviews/Next Skills sections with cross-links to the extended review;
      CSS diff vs kit template: zero token drift — report is a clean subset
      + 2 report-specific additions)
      _(Effort: S)_
- [x] **Upstream skill fixes** — `docs/reviews` ↔ `docs/brainstorming`
      divergence in data-model-review skill docs; add "read prior reports" +
      "copy template, never transcribe" steps (plan T23).
      ✅ 2026-08-22 (`~/.config/crush/skills/data-model-review/`:
      output-guide.md now says `docs/reviews/` matching SKILL.md + repo
      truth; Step 5 gained both steps before writing — skill lives outside
      this repo, hence no repo commit)
      _(Effort: S)_

### Deep Full-Code Review (2026-08-27) — follow-ups

> 9 fixes landed inline (E3/E9/E10 + decider poll-clamp/WithoutCancel/
> ErrNilDecide, scheduling Due ordering, kv stale-cache invalidate,
> commandlifecycle version-cache recovery, projectionhost backoff cap).
> Report: `docs/reviews/2026-08-27_full-code-review.html`.

- [ ] **commandlifecycle Recorder unbounded versions map** — every command ID
      seeds a `versions` entry that is never evicted (only manual
      `ResetVersion`); long-running dispatch loops grow it forever. Bound it
      (LRU/TTL) or re-seed per emit batch.
      _(Effort: S)_
- [ ] **commandlifecycle AttemptMiddleware standalone leak** — the attempt
      tracker is only cleared by the OUTER middleware; standalone
      `AttemptMiddleware` usage grows `attempts` forever.
      _(Effort: S)_
- [ ] **projectionhost applyWithRetry ignores error family** — Rejection/
      Corruption (non-retryable) handler errors are still retried
      `dlqThreshold` times before DLQ; skip straight to DLQ for
      `!family.IsRetryable()`.
      _(Effort: S)_
- [ ] **projectionhost Stop timeout has no retry path** — after
      `shutdownTimeout` fires, `stopped=true` makes later Stop calls no-op
      while workers may still run; expose a force/re-drain path.
      _(Effort: S)_
- [ ] **snapshot.ReadPressure bounded tracking** — the `reads` map grows
      with distinct stream refs (inline `TODO(review-2026-08-27)` in
      `snapshot/read_pressure.go`); bounded LRU option.
      _(Effort: S)_
- [ ] **command/query constructor error-style drift** — `command.New` wraps
      sentinel errors while `query.New` returns raw strings; unify before v5.
      _(Effort: S)_
- [ ] **cqrs-lint version constant automation** — `cmd/cqrs-lint/main.go`
      `version` const drifted from the released tag (4.6.0 vs v4.7.0);
      `scripts/tag-release.sh` should bump it in the same wave.
      _(Effort: S)_
- [ ] **kv.Cache.Get miss-path double round-trip** — miss path encodes+
      decodes an extra copy for isolation (documented tradeoff); optional:
      cache the store-fresh value directly and copy only on hit.
      _(Effort: S)_
- [ ] **listing cursor cross-type ambiguity** — `ListOptions.After` matches
      ID strings across stream types when no Type filter is set; same ULID
      under two types can skip/repeat entries. Document or key cursor by
      (type, id).
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

- [x] **listing StatusMiddleware marks never reach the journal** 🔥 — done
      2026-08-27 (plan P05): migration-doc recipe rewritten to the shipped
      `listing.WithStatusClassifier`/`NewStatusClassifier` (type-driven status,
      wire-compatible with legacy TombstoneStatus ints; golden parity tests);
      `StatusMiddleware` Deprecated. Marks-to-journal dishonesty is moot: the
      reader no longer reads metadata marks at all.
      `MarkTombstone` returns a NEW event delivered to bus subscribers only;
      the journal copy saved by the decider never carries the mark, so
      `InMemoryStreamReader.rebuildCache` reads status `Undetermined` from
      the journal: `TombstoneExclude` (default) INCLUDES deleted streams and
      `TombstoneOnly` returns nothing. The migration-doc recipe
      (`docs/migration/tombstone-to-domain-events.md:118-141`) cannot work
      with the decider save-then-publish flow. Near-term: correct the doc +
      add a type-driven reader option (`WithDeleteTypes`); the real fix is
      the v5 type-based status (sweep items 18/19). Tests currently hide
      this by pre-stamping metadata before Save.
      _(Effort: M; doc part S)_
- [ ] **metaengine Record context is shared mutable state across Stores** 🔥 —
      `recHolder` lives on the Fold instance; `Store.Verify` builds a second
      Store from the same declarations and replays into it, so concurrent
      live `Apply` + Verify replay is a data race on `recHolder.rec` and can
      cross-attribute Record context. Pass the Record through the invoke
      closure as an argument instead of a shared cell.
      _(Effort: M)_
- [ ] **metaengine replay paths destroy Record context** 🔥 — `EventLog`
      stores only (Type, Payload); `Backfill`, `DemoteEngine` catch-up, and
      `Verify` replay with synthesized `record.Record{Type}`, so Record-aware
      folds rebuilt via replay silently diverge from live-built ones (Verify
      compares row counts only and cannot catch it). Add optional
      `Record record.Record` to EventInput/log entries (additive).
      _(Effort: M)_
- [x] **watermill CatchUpSubscriber checkpoints at handoff, not Ack** — done
      2026-08-27 (plan P04): checkpoint advances only on `msg.Acked()` in both
      replay and live phases; Nack stops the subscription with the checkpoint
      left behind the failed event; the 1024-entry ring is replaced by a
      last-replayed-ID watermark. Regression tests (no-ack crash, Nack stop,
      watermark suppression) race-verified 3x. Original note:
      doc says "after each message is Acked" but both replay and live phases
      save the checkpoint right after `output <- msg`; a crash (or Nack)
      between handoff and processing permanently skips events (at-most-once).
      Also: the 1024-entry dedup ring is bounded by a wrong invariant (the
      real overlap set is every event appended during replay, unbounded), so
      duplicates can still slip through under load. Checkpoint on
      `msg.Acked()`; replace the replay-side ring with a last-replayed-ID
      watermark.
      _(Effort: M)_
- [x] **watermill subscriber Close can panic (send on closed channel)** —
      verified 2026-08-27 against current code: `Close()` never closes
      output channels (only `runCatchUp`'s single-sender defer does) and
      `Subscribe` creates a fresh subscription per call (no shared map to
      overwrite). No reproducible panic path remained; the P04 ack-waiting
      refactor kept the single-sender invariant. Original note:
      `Close()` closes `outputCh` while handler goroutines may be blocked
      selecting on send; a send on a closed channel is always "ready". Never
      close `outputCh`; signal via `closeCh` only. Also `Subscribe` twice
      overwrites the handlers map entry (duplicate delivery on shared
      output).
      _(Effort: S)_
- [x] **system shutdown validation rejects the runtime's own synthetic
      engines** — done 2026-08-27 (plan P05.4/5): validation now runs against
      the populated engine set (configured + synthetic "default"/
      "projections"); empty names return ErrShutdownDependencyInvalid.
      Synthetic-name acceptance test added. Original note: `validateShutdownDependencies` checks against
      `deployment.Engines`, but the constructor auto-creates `"default"` /
      `"projections"` engines that the runtime sort honors — the
      `ShutdownDependencies` doc example (`Before: "projections"`) now fails
      `New()` with ErrUnknownEngine. Validate against the populated engine
      set; also classify empty names as ErrShutdownDependencyInvalid (not
      ErrUnknownEngine).
      _(Effort: S)_
- [ ] **pebble command/query duplicate check is check-then-commit + fail
      closed** — the existence check runs outside the write lock (concurrent
      duplicate Save silently overwrites instead of Conflict) and treats ANY
      Get error as "exists" (an Infrastructure read failure is reported as
      `ErrDuplicateCommand`). Take the per-ID shard lock around check+commit;
      return wrapped Infrastructure when the check itself fails.
      _(Effort: M)_
- [ ] **stream-not-found contract diverges across backends** — pebble/bbolt
      `Load`/`LoadFromVersion` return `(nil, nil)` for missing streams while
      memory/SQL (and the same stores' LoadToVersion) return
      ErrStreamNotFound; `event.EventSource` godoc is silent. Pin the
      contract on the interface, align backends (store-side change is
      v5-marked). Same class: SeekableJournal dangling-cursor behavior
      diverges (memory replays from start; SQL/pebble/bbolt return empty and
      stall) — pick the SQL contract, document on the interface.
      _(Effort: S doc / v5 align)_
- [ ] **schema upcaster registry hazards** — (a) upcaster returning the same
      pointer mutates the stored/shared event (README claims "original is
      never mutated"); (b) `(nil, nil)` return panics; (c) the registry
      force-stamps source+1 regardless of the returned version (a v1→v3
      upcaster is relabeled and double-transformed); (d) duplicate
      (type,version) registrations accepted with unstable sort; (e) doc
      claims construction-time chain validation that does not exist.
      Guard nil, use sort.SliceStable, verify version stamps post-upcast,
      reject duplicates, fix or implement the claimed validation.
      _(Effort: M)_
- [ ] **snapshot.TypedStore.Save bypasses NewSnapshot** — bare struct literal:
      no invariant validation (version 0 / zero refs persist; property test
      generates version 0) and no CreatedAt stamp, unlike every other write
      path. Route through the validating constructor.
      _(Effort: S)_
- [ ] **kv.Cache has no invalidation primitive + cache-aside race** —
      `Backend()`/`Store()` hand out raw writers that bypass the cache, a
      second Cache instance never invalidates, default TTL is 0 (unbounded
      staleness), and a Get-miss can pin a stale value after a concurrent Set
      (G1 read-old → G2 Set → G1 cache.Set(old)). Add `Invalidate`/
      `InvalidateAll` (additive), document the single-writer assumption and
      TTL recommendation. Also: `DeleteAll` with no configured prefix
      deletes EVERY key in the backend — document the blast radius or gate
      behind an explicit opt-in.
      _(Effort: M)_
- [ ] **catalog SchemaFromType silent-wrong schemas** — exported embedded
      (anonymous) struct fields are skipped although encoding/json flattens
      them onto the wire (generated docs/clients disagree with payloads;
      enshrined by `TestFromType_SkipsAnonymousEmbeddedFields`); recursive
      types (self-referential payloads) recurse to a stack overflow with no
      in-progress guard. Recurse into exported anonymous fields honoring
      their json tags; reserve a cache placeholder before recursing.
      _(Effort: M; goldens change)_
- [ ] **cqrs-lint C042 inspects the wrong argument** — the zero-expected-
      version rule checks `call.Args[2]` but `event.Store.Save` is
      `(ctx, ref, events, expectedVersion)` — the version is `Args[3]`; the
      rule can never fire on the canonical API (and misses
      `event.Version(0)` conversions).
      _(Effort: S)_
- [ ] **scenario DSL can pass vacuously** — `Given(...).When(cmd, decide)`
      with no `Then*` compiles and passes with zero assertions;
      `GivenProjection` without `ThenNoError`/`ThenError` swallows every
      handler error. Register a `t.Cleanup` guard failing the test when no
      terminal assertion ran. Also `scenario/doc.go`'s example signature is
      stale (missing `t *testing.T`).
      _(Effort: S)_
- [ ] **projectionhost hardening set** — `ReplayDeadLetters` calls
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
- [ ] **capability interfaces not adopted at three assertion sites** —
      middleware/actor.go, commandlifecycle/recorder.go, and
      watermill/command_protocol.go each re-declare a private
      `Metadata() command.Metadata` interface although
      `command.MetadataCarrier` exists exactly for this (ADR-0111 g).
      Replace the private clones (non-breaking).
      _(Effort: S)_
- [ ] **transport deprecation is not machine-readable** — transport/http and
      transport/grpc say "DEPRECATED" in prose but lack the Go-standard
      `// Deprecated:` paragraph, so staticcheck SA1019 never flags
      consumers; http's WebSocket section steers to grpc without noting it
      is deprecated too.
      _(Effort: XS)_
- [ ] **deriver has no cycle guard** — `Then`'s doc blesses A→events→B→events
      chains through the bus, but nothing bounds derivation cycles
      (deterministic IDs key on the source event ID, which changes every
      round). Opt-in depth guard via a hops counter in derived-command
      metadata, or document the hazard on AsHandler.
      _(Effort: S)_
- [ ] **scheduling multi-instance + retry semantics undocumented** — no
      claim/lease protocol: two Schedulers on one store double-fire every
      timer (undocumented single-active-instance requirement);
      dispatchWithRetry retries every family (Rejection retried 3× per poll
      forever) and errors.Join keeps only the last attempt's error;
      `MarkFired` deletes by ID with no epoch (a re-scheduled timer can be
      deleted mid-flight). Document now; ClaimingTimerStore (SKIP LOCKED)
      as additive follow-up.
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
- [ ] **eventtest fakes** — `LoadToVersion` returns a live sub-slice of the
      store's backing array (in-place sort corrupts the fake);
      `ReadAll`/`ReadFrom` return map-iteration order violating the
      Journal's documented OccurredAt ordering; `FakeBus.Publish` reads
      `publishChain` unlocked while `UsePublish` swaps it under mu.
      _(Effort: S)_
- [ ] **record.Stamp zero-time presence flip** — `NewStamp(time.Time{})` is
      known but JSON-round-trips to unknown (wire `at` is a value, so the
      zero time reads back as absent). Wire-compatible fix: make the wire
      field `*time.Time` (nil → unknown). Undocumented edge today.
      _(Effort: S)_
- [ ] **dispatcher/metadata README lies** — dispatcher README claims `M` is
      the message type (it is the middleware type), claims pre-computed
      chains (code rebuilds per Dispatch), and lists nonexistent symbols
      (`LifecycleMixin`, `CatalogDispatcher`, `Handlers()`); metadata README
      drops ActorID from its Tracing snippet and calls command/query
      Metadata "standalone structs" (they are aliases).
      _(Effort: S)_
- [ ] **id.ActorID vs record.Actor zero-semantics asymmetry** —
      `id.ActorID{User, ""}.IsZero() == true` but
      `record.Actor{ActorUser, ""}.IsZero() == false`; `record.Actor{User,""}.String()`
      ("user:") re-parses to an id-side zero, dropping the kind. Documented
      mirrors, asymmetric zeros — document both sides now, unify at v5.
      _(Effort: S doc / v5 unify)_

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

- [x] **PG integration test isolation under explicit DSN** — EXECUTED
      2026-08-27 by the goal-gap session BEFORE this item was moved to
      Declined (commit 5ec4b1b39): storage, storage/relational and benchkit
      now route through testutil/pgtestcontainer (per-test DBs under
      explicit DSN, PID-qualified names so parallel test binaries cannot
      collide), full `#integration-pg` GREEN. Revert explicitly if the
      decline rationale outweighs the verified fix. Original note:
      pg_integration tests assume "each test gets its own fresh database"
      (true with testcontainers when POSTGRES_TEST_DSN is unset), but
      `nix run .#integration-pg` points every package at ONE shared
      `cqrs_test` database: `TestPostgresEventStore_CRUD`'s global
      `ReadAll` sees other packages' events ("expected 2 events, got 27",
      pre-existing, reproduced twice 2026-08-22). Fix: create a per-test
      (or per-package) database even under an explicit DSN.
      _(Effort: S)_
