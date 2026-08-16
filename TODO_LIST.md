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

- [x] 🔥 **MySQL/MariaDB byte guard for multi-VALUES batches** — DONE 2026-08-16:
      `sql.MaxStatementBytes` (8 MiB) + `RowsWithinByteCap` dual-cap in
      `SharedBatchInsertEvents` and `view.BatchSet`; verified on real MariaDB VM
      (2000×8 KiB regression test in `stack/mysql`).
- [x] **`MaxParametersForDialect` unit test** — DONE: table test in
      `storage/sql/batch_insert_test.go` (SQLite/PG/MySQL/DuckDB/unknown/nil).
- [x] **DuckDB verification of larger batch chunks** — DONE: full `./storage/...`
      tree green with CGO + 6×2 MiB view chunk test.
- [x] **Projectionhost live-checkpoint batching** — DONE 2026-08-16:
      `WithCheckpointEvery(n)`/`WithCheckpointInterval(d)` (opt-in, default
      unchanged), Stop/shutdown flush, crash window ≤ n−1 reprocess; race ×3
      green. Docs: readmodels.md §2.3.
- [x] **bbolt opt-in `db.Batch` group commit path** — DONE 2026-08-16:
      `bbolt.WithBatchCommit()` on `OpenWithOptions`/`NewBackendWith`; all
      write closures route through `writeTx`; idempotent-under-retry verified
      (conflicting writer ejected solo, batch-mates land); race-tested.
- [x] **PG `COPY FROM` bulk path** — DONE 2026-08-16: `StreamAppend` now chunked
      multi-VALUES (10k rows/stmt) by default; `pgengine.WithCopyAppend(n)`
      opt-in COPY via `db.Conn().Raw()` → pgx (no second pool); measured 1.41x
      @10k / 1.49x @100k rows vs batched INSERT; falls back inside RunInTx.
- [x] **Pebble tuning knobs** — DONE 2026-08-16: `stack/pebble`
      `WithMemTableSize`/`WithBlockCacheSize`/`WithWALBytesPerSync`/
      `WithPebbleCompression`; defaults byte-identical (pinned by test); block
      cache ref released after Open.
- [x] **SQLite durability tier when WAL off** — DONE 2026-08-16:
      `ApplySQLiteDurability` now applies every non-empty tier (Normal
      early-return was WAL-specific); tier application de-nested from
      `if cfg.WAL` in stack/sqlite/preset.go + stack/turso/backend.go.
      Tests: `WithoutWAL` table (relaxed=OFF≠FULL pin), preset-level
      `RelaxedWithoutWAL`; stack/sqlite/turso suites green.
- [ ] [BLOCKED] **Durability tier→per-write-sync mapping** (storage/pebble
      hardcodes `pebble.Sync`; metaengine engines no NoSync path) — real win
      (fsync per append) but a behavior change for existing Normal-tier
      consumers. AWAITS USER DECISION (status §g Q3).
      _(Effort: M)_
- [x] **Reconstruct payload adopt-variant** — DONE 2026-08-16:
      `event.ReconstructEventWithAdoptedPayload` (ownership-transfer contract;
      `Payload()` stays defensive) wired into pebble+bbolt deserialize.
      Measured: bbolt 2815→2521 ns/op (−10%), pebble 3316→2872 (−13%),
      −32 B/op. Equivalence + alias + race tests green.
- [x] **Bound `idempotencyTracker`** — DONE 2026-08-16: mutex-guarded
      `dedup.Ring`, default window 131072 IDs (~10 MB) via
      `WithIdempotencyCapacity` (≤0 = legacy unbounded). 1M-ID memory-bound,
      eviction, and concurrent exactly-once tests race-green.
- [x] **Envelope first-byte sniff in `UnwrapDecode`** — DONE 2026-08-16
      (go-codec sibling repo): first byte ≥ 0x80 (CBOR major types 4-7, never
      valid JSON) skips the doomed envelope parse. Fallback path 181ns/6
      allocs → 1.6ns/0 allocs (-99%, n=10 benchstat); envelope path unchanged
      (p=0.912). Pinned by `TestUnwrapDecode_FirstByteSniff` (all 128 high
      bytes) + `BenchmarkUnwrapDecode_FallbackRawCBOR`. CONSUMER NOTE: needs a
      go-codec tag before GOWORK=off builds pick it up (standing tagging
      question).
- [x] **bbolt deserialize benchmark** — DONE 2026-08-16:
      `BenchmarkEventDeserialize` (storage/bbolt): 2815 ns/op, 1210 B/op,
      20 allocs pre-adopt → 2521 ns/op with adopt; ledger updated.
- [x] **Measure-then-pad cache-line candidates** — DONE 2026-08-16
      ([evidence](docs/benchmarks/2026-08-16_false-sharing-contention.md),
      count=10 @-cpu=16,32): multiSeqCounter PADDED (trailing 128B size-class
      pad, 2.5-2.8x under two hot collections); worker counters NO PAD
      (padded mirror ~58% slower for the single-writer under reader spin —
      confirms prior analysis); SSEReplay.seq NO PAD (contradictory deltas;
      record() touches seq and mutex fields together). Control benches kept
      in-tree.
- [x] **Perf ledger** — DONE 2026-08-16: [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md)
      maps every shipped win to its runnable benchmark + baseline + last
      measured numbers. REMAINS: benchstat baselines for the 3 new benchmarks
      (kept open below).
      _(Effort: S)_
- [x] **Fix ignored `MarshalMetadataJSON` error** — DONE 2026-08-16:
      explicit discard with ADR-0126 constraint documented + nil fallback on
      marshal failure (zero metadata instead of partial JSON).
- [x] **ScanSlice `RowCount()` pre-size** — VERIFIED NO-OP 2026-08-16:
      database/sql has no RowCount API; cap-64 pre-size, `maps.Clone`, and
      `MergeCustomMaps(len+len)` already applied. Changing the exported
      generic signature would buy nothing.
- [x] **`OpenSQLiteInMemory` callers: unique shared-cache DSNs** — DONE
      2026-08-16: `OpenSQLiteInMemory` now generates a unique
      `file:<random>?mode=memory&cache=shared` DSN per call, removing the
      single-connection pool pin. All pooled connections share one in-memory
      schema, enabling read concurrency. The `view` test helper was updated
      to match. Race tests no longer need `SetMaxOpenConns(1)`.
      _(Effort: S)_

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

- [ ] [BLOCKED] 🔥 **Tag the wave-4 module batch** — `event` (DecorateJournal),
      `metadata` v4.5.1+ (BrandedString), `schema`, `metaengine` (capability
      audit + iroh exports), `metaengine/irohengine`, `projectionhost`, and
      `storage` v4.7.2 (SQLite `OpenSQLiteInMemory` shared-cache DSNs). Constraint: the
      batch must tag event+metadata+schema before/with projectionhost (its
      released go.mod needs them — the release flow strips the sibling
      replaces). Via `scripts/tag-release.sh` from a clean tree.
      _(Effort: M)_
- [ ] [BLOCKED] 🔥 **Land the stranded tag-chain repair commits on master** —
      cherry-pick `092b5e8a8` (command/query `retract` directives + metadata
      v4.5.0 pins + hardened `tag-release.sh` standalone-build gate) and
      `4907b6afc` (metaengine/bench pseudo-version tidy) from the tag worktree
      (`git merge-base --is-ancestor` confirms both NOT on master, verified
      2026-08-16). Master's `command`/`query` go.mod still pin `metadata/v4
      v4.4.0` — any future tag cut from raw master re-breaks consumers (the
      v4.7.0/v4.6.0 incident class). Regen the api-stability golden fresh on
      master instead of cherry-picking `d25e8a959`.
      _(Effort: S)_
- [ ] [BLOCKED] **go-codec F46: commit + tag the `UnwrapDecode` sniff** —
      the first-byte fast path (fallback 181ns/6 allocs → 1.6ns/0 allocs) sits
      UNCOMMITTED in `../go-codec` (no auto-commit daemon there); GOWORK=off
      consumers get nothing until it is tagged.
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

- [x] 🔥 **Pin-drift meta-test** — DONE 2026-08-16:
      `cmd/api-stability/pin_drift_test.go` `TestSiblingModulePinsResolve`:
      hard-fails unreplaced pins referencing nonexistent tags or
      pseudo-versions; staleness warns (16 stale today, all replace-governed)
      until the pin-sweep policy decision flips `enforceStaleness`.
      Handles nested-module tag pollution; skips hermetic nix builds.
- [ ] 🔥 **Repo-wide stale-pin sweep** — benchkit still pins
      `sqliteengine v4.0.1` (pre-JournalReadFrom-fix), `decider v4.3.0`,
      `event v4.6.0`… Mechanical bump of ~50 go.mod files, gate-verified.
      (Needs user sign-off on policy — see ROADMAP Open Questions.)
      _(Effort: M)_
- [ ] 🔥🔥 **storage/pebble + storage/bbolt standalone builds RED** (verified
      2026-08-16 14:20, `GOWORK=off go build` fails): both pin
      `event/v4 v4.6.0` but `serialization.go` calls
      `event.ReconstructEventWithAdoptedPayload` (shipped after v4.6.0; needs
      ≥ v4.7.0 + the unreleased adopt API or a local `../event` replace until
      tagged). Same workspace-masking class as the command/v4.7.0 incident.
      Fix: bump pins (or add sibling replaces) + add both to the GOWORK=off
      standalone gate.
      _(Effort: S)_
- [ ] **`#verify-standalone` nix app (GOWORK=off per module) or explicit
      decision that CI owns that signal** — then CHECK CI after gates.
      Investigate how long the CI Benchmarks job has been red (`gh run list`)
      to size the blind window.
      _(Effort: M)_
- [ ] **Add CI leg for GOWORK=off standalone builds of leaf modules**
      (integration/, examples/, benchkit/) to catch pin rot early.
      _(Effort: S)_
- [x] **`system/integration` DuckDB standalone failure** — FIXED
      2026-08-16: root cause was published duckdbengine v4.0.1 predating
      register.go's `metaengine.RegisterDriver` self-registration (workspace
      mode masked it). Added replace via `go mod edit` + tidy; standalone
      GOWORK=off suite green. Drop the replace once duckdbengine v4.0.2+
      tags the registration.

---

## Metaengine — Layout Planning (Phase 6b)

> Core shipped 2026-08-11: priority system, embed-vs-normalize scoring,
> `ReplanLayout`, `ConfirmRebuild`, runtime backend add/remove, audit trail,
> 16-combination regression test, `cqrs-bench layout` CLI. KV/LSM scoring
> split with 60s on-disk calibration. Layout docs + ADR-0124 addendum
> corrected 2026-08-14. See ADR-0124, ADR-0125.

- [x] **Calibrate DuckDB (Columnar)** — DONE 2026-08-15: benchmarked via
      `BenchmarkColumnarLayoutCalibration_*` (cgo-gated); the exact-tie cell is
      now a measured 0.08-margin Embed win; a literal 60s confirmation run
      reproduced ratios within 2%.
- [x] **Calibrate SQLite/Postgres/MySQL (Row)** — DONE 2026-08-15: measured on
      file-backed SQLite, Postgres 16 (`.#integration-pg`), MySQL
      (`.#integration-mysql-vm`); geomean read 1.27x / write 0.52x / storage
      0.35x; normalize-reads-cheaper myth corrected.
- [x] **Multi-engine integration test with two real backends** — DONE
      2026-08-15: `metaengine/bench/multi_engine_integration_test.go` drives
      SQLite + Pebble through plan → AddEngine(Migration) → Backfill →
      Promote → Demote → live mirroring; both engines serve identical results.
- [x] **Converge `ReplanLayout` into `Store.Replan`** — DONE 2026-08-15:
      ReplanLayout funnels through the single `replanWithTrigger` path and
      diffs plan snapshots; the duplicate scoring loop is deleted. Semantic
      change: ReplanLayout now APPLIES the priority config (SetPriority+Replan
      equivalent).

### Layout roles (long-horizon, depend on a design doc first)

> Shipped 2026-08-15 — engine roles, async shadow replication, promote/
> cutover, workload traces, shared-collection boundaries, per-fold locks.
> See CHANGELOG and
> [`docs/planning/METAENGINE-LAYOUT-ROLES.md`](docs/planning/METAENGINE-LAYOUT-ROLES.md).

- [x] **`DemoteEngine` (role transition v2)** — DONE 2026-08-15: atomic
      drain-then-unroute via `replanWithTransition` (role flip + replicator +
      EventLog snapshot under one write lock), targeted mirror catch-up +
      re-routed replay, `engine-demoted` audit trigger; PromoteEngine hardened
      through the same atomic path. Exactly-once proven by a concurrent-apply
      race test. See METAENGINE-LAYOUT-ROLES.md §4.4.
- [x] **`Re-derive KV/LSM layout constants from size-stable benches`** — DONE
      2026-08-16: both benches fixed (memory EmbedWrite drifted via unbounded
      append; disk EmbedWrite's typed assertion never matched the map form disk
      engines decode into, so the mutation silently no-oped — the TODO's
      premise was wrong for the disk bench), re-measured exclusively with
      median-of-10, and constants updated (KV norm write 0.84; LSM norm
      1.67/0.62/0.98). Includes a NEW real-bytes storage bench
      (`BenchmarkDiskLayoutCalibration_Storage`): the JSON model overstated
      normalize's LSM storage advantage ~2x (real 0.86x geomean — per-child
      seq-suffixed keys eat the dedup saving). All 16 matrix winners preserved;
      LSM × Balanced margin 0.01 → 0.28. See
      [ADR-0124 addendum](docs/adr/0124-operator-driven-layout-planning.md)
      and
      [`docs/status/2026-08-16_14-09_kv-lsm-recalibration-size-stable-benches.md`](docs/status/2026-08-16_14-09_kv-lsm-recalibration-size-stable-benches.md).

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

- [x] **Seq-carrying journal reads (perf follow-up)** — DONE 2026-08-16:
      `metaengine.SeqSeekableStreamLog` capability
      (`JournalReadAllWithSeq`/`JournalReadFromSeq`, `StreamLogEntry{Seq,Value}`)
      implemented by 8 engines (memory, sqlite, pg, mysql, duckdb, pebble,
      bbolt, badger; turso inherits via sqliteengine; dgraph/iroh intentionally
      out per design §7). Resume is a pure `collection+seq` index seek —
      O(log n) per page instead of O(offset) — and gap-tolerant by
      construction. `enginetest.RunSeqSeekableStreamLogTest` conformance gates
      every engine; memory gap-tolerance test covers journal-entry deletion.
      `system` EventAdapter + AdapterCore resume on true engine tokens
      (zero-cursor reads skip journal scanning entirely; cursor resolution
      paged 512/batch). Measured (sqlite, 100k-entry drain, page 500,
      benchstat): **761.8 ms ±17% → 106.8 ms ±20% = 7.1x** —
      [ledger](docs/BENCHMARKS.md). Design +
      implementation: [`docs/planning/SEQ-CARRYING-JOURNAL-READS.md`](docs/planning/SEQ-CARRYING-JOURNAL-READS.md)
      (IMPLEMENTED).
- [x] **Engine capability conformance test** — DONE 2026-08-16 (F60 +
      Doctor wiring): `metaengine.CapabilityAudit` (root package) enforces
      the three rules (over-declaration, under-declaration, Degraded ⊆
      Supports); `adttest.RunCapabilityConformance` gates all 10 engines;
      `Store.Doctor` renders a `--- Capability ---` section per engine so
      lying engines surface at runtime. adttest delegates to the root
      implementation (dependency direction: adttest → metaengine).
      Follow-up same day: irohengine graph over-declaration fixed —
      `Replicated` now forwards `GraphAddEdge`/`GraphNeighbors` as local
      passthrough (+ `ErrGraphBackendNotImplemented`, regression tests);
      all 9 engines green in the conformance loop.
- [ ] **Run the 9-engine conformance loop under `#test-integration`** —
      mysql/dgraph/turso rows only execute against real servers (they skip
      cleanly without them; loopback/quic have their own matrices).
      _(Effort: S)_
- [ ] **iroh graph `WriteOp` replication** — `GraphAddEdge`/`GraphNeighbors`
      are local passthrough; the replication wire protocol has no graph
      WriteOp kind, so edges do NOT converge across peers (documented on the
      methods). Replicate edges or keep the honest local-only note in Doctor.
      _(Effort: M)_
- [ ] **irohengine optional-capability forwarding audit** — `Replicated` was
      caught dropping graph dispatch (fixed 2026-08-16); audit the remaining
      optional capabilities (Close, Transactional, StreamLogBackend, probers)
      for the same interface-embedding promotion gap.
      _(Effort: S)_
- [ ] **Surface capability drift beyond tests** — Doctor: note iroh graph
      non-replication in the Capability section; EXPLAIN: plan-time warning
      banner from `AuditCapability`; plus a metaengine meta-test pinning
      `adttest` stays delegating-only (no logic re-growth in the wrong
      package).
      _(Effort: S)_
- [ ] **DuckDB real aggregation pushdown (`AggregateReader`)** — approved by
      DiscordSync census review; `CounterGet` currently loads all rows into
      Go maps instead of pushing GROUP BY to columnar SQL. Highest-leverage
      DuckDB item.
      _(Effort: L)_
- [ ] **Dgraph engine hardening** — `Transactional` (RunInTx) support or an
      ADR documenting why not; per-test collection isolation for the shared
      persistent server; add Dgraph to `test-all-backends`/CI matrix
      (AGENTS quick-ref lists no Dgraph backend today).
      _(Effort: M)_
- [ ] **DuckDB native graph via recursive CTE** — DuckDB supports
      `WITH RECURSIVE`; mirror the pgengine `meta_graph_edges` implementation
      (currently intentionally degraded).
      _(Effort: M)_
- [ ] **Turso explicit CTE-probe test** — the sqliteengine probe covers
      local drivers; add a tursoengine test confirming it holds over the
      remote protocol.
      _(Effort: S)_
- [ ] **Badger engine vector + graph parity audit** — has neither; audit
      against the pebble/bbolt precedent and either implement or document
      the gap.
      _(Effort: S)_
- [ ] **Vector search at scale** — quantization/HNSW spike for LSM engines
      when collections exceed ~100K vectors (brute-force scan is O(N)).
      _(Effort: L)_
- [ ] **`VectorResult` filtered k-NN** — metadata-filtered vector search
      (filter + top-k in one query); API currently returns bare top-k.
      _(Effort: M)_
- [ ] **`GraphRemoveEdge`** — the edges table exists but nothing removes
      edges; tombstone events should drive edge removal (ADR-0114 style).
      _(Effort: M)_
- [ ] **Graph directed-vs-undirected option** — `GraphNeighbors` is
      directed-only today.
      _(Effort: S)_
- [ ] **mysqlengine upsert semantics audit** — confirm `MapSet` uses
      `INSERT ... ON DUPLICATE KEY UPDATE` consistently with pg's
      `ON CONFLICT` (atomicity + affected-rows parity).
      _(Effort: S)_
- [ ] **MariaDB functional-index alternative** — generated columns + plain
      index instead of the current `ApplyLayout` no-op.
      _(Effort: M)_
- [ ] **enginetest per-run collection suffixes** — shared-server engines
      accumulate state under `-count>1` (documented constraint that bit the
      race runs); per-run suffixes in the helpers would remove the class.
      _(Effort: M)_
- [ ] **adttest: graph depth>2 + cycle scenarios in `RunMatrix`** — current
      matrix is depth-limited; the CTE/iterative divergence only shows at
      depth>2 with cycles/diamonds.
      _(Effort: S)_
- [ ] **adttest: Vector scenario on pgengine** — parity check for the
      degraded scan path against the in-memory index.
      _(Effort: S)_
- [ ] **Convergence suite order-tolerance audit** — `sameLogTail` was the
      only order-asserting helper (fixed 2026-08-15); sweep the remaining
      `waitFor*` helpers for hidden order assumptions.
      _(Effort: S)_
- [ ] **quic pooled-stream ordering guarantee** — default `sendOp` uses one
      stream per op (no cross-op order); verify + document that pooled mode
      (`sendOpPooled`, one stream per peer) DOES order ops.
      _(Effort: S)_
- [ ] **Bench: CTE vs iterative BFS crossover** — at which depth does the
      MySQL CTE beat the iterative fallback? Feeds the planner's cost model.
      _(Effort: S)_
- [ ] **Bench: MariaDB dual-key sort cost** — measure the
      `CAST(... AS DECIMAL)` overhead vs MySQL's single JSON key.
      _(Effort: S)_
- [ ] **Run `nix run .#integration-mysql-nspawn`** (needs root) — real-env
      verification incl. `stack/mysql`; live verification so far used docker
      probes only.
      _(Effort: M)_

---

## cqrs-lint

> Per-module coaching migration complete: all adoption (F003-F029) and
> resilience (B029-B031) rules evaluate per-module. 86 per-module profiles
> verified by `integration_multimodule_test.go`. F001/F002/F005/F014 remain
> workspace-global by design (low leakage risk). F030 (deprecated transport
> imports) shipped 2026-08-14 — 203 rules total.

- [x] **Add per-module regression tests for remaining migrated rules** —
      ~~F004, F007, F009, F012, F017, F023-F029, B030 lack dedicated per-module
      tests~~ STALE (verified 2026-08-16): the "F006-F021 batch" file
      (`coaching_permodule_extra_test.go`) was extended past F021 — all listed
      rules now have per-module tests; B030 is in `b029_b031_permodule_test.go`.
- [x] **Teach E005 about `system.RegisterCommand`** and regenerate the
      taskmanager lint golden — killed the 10 enshrined false positives
      (scanner records the first generic type arg; done 2026-08-16).
- [x] **Wishlist (parked from consumer feedback rounds)** — all four shipped
      2026-08-16: `--doctor --fix` removes stale whole-line suppressions
      (trailing-on-code left manual); stale-suppression warnings run in every
      output format (stderr-only, `--quiet` still silences); health score
      shows an "Excluded from score by config" footer for disabled rules;
      `features.monetary` (`on`/`off`/`unknown`) overrides the money
      heuristics: C008 downgrades to Info on `off`, S006 skips entirely.
- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Track which can be removed after migrations complete.
      _(Effort: M)_

---

## WithActor Hardening

> `event/command/query.WithActor` + `Tracing.ActorID` shipped and released
> (metadata/v4.4.0, event/v4.6.0, command/v4.6.0, query/v4.5.0 — 2026-08-13).
> id/v4.4.0 contains `actor_id.go` (verified in tag).

- [x] **Document `WithActor` in skill references** — core.md §3.8 + recipes
      §2.21 + modules.md row all shipped (verified 2026-08-16).
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

- [ ] **CHANGELOG honesty gate** — lint that every `pkg.Symbol` identifier
      cited in CHANGELOG Added/Changed entries exists in the api-stability
      golden. Kills the reverted-work fiction class mechanically (the
      2026-08-10/11 tombstone entries described `e406edcfb`, which was
      reverted by `a6613ef0d` before any tag — CHANGELOG never recorded the
      reversion; corrected 2026-08-16).
      _(Effort: S)_
- [ ] **api-stability: fail loudly on parse-skip** — the checker prints `skip
      <module>:` and proceeds when a file is unparseable, so a corrupted
      module looks identical to a legitimately-removed one in the golden
      (a silently-shrinking golden is the corruption tell, 07:12 report §e.2).
      Cheapest corruption tripwire available.
      _(Effort: XS)_
- [ ] **BuildFlow pre-commit: `gofmt -l` syntax gate on staged `.go` files** —
      concurrent-session mid-write corruption entered the index twice on
      2026-08-16 (`func (w *workor)`, `fojection.`); a 1s syntax check on
      staged files blocks the class.
      _(Effort: XS)_
- [ ] **Pre-gate load-sweep script** — run timing-assertion tests
      (`-run 'Latency|Timer|Deadline'`) under a CPU soaker before `#verify`;
      the 12:39 session burned two full gate cycles (~20 min each)
      discovering load-sensitive flakes one at a time.
      _(Effort: S)_
- [ ] **Guard against local-path `replace` directives** — cqrs-lint rule,
      buildflow check, or both: reject `replace … => /home/…` (broke CI
      Release on every push until `ceb88738b`; dev-against-siblings belongs
      in `go.work` `use`, never in a published go.mod). Approach is a user
      question (11:00 report §g Q2).
      _(Effort: S)_
- [ ] **Clean up registered git worktrees** — `/tmp/cqrs-tagwt` (tag chain,
      commits `092b5e8a8`/`4907b6afc` stranded there), `/home/lars/projects/
      wt-head`, `/tmp/gcl-verify`, `go-cqrs-lite-pin` still registered.
      After the stranded-commit cherry-pick lands.
      _(Effort: XS)_
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
- [ ] **Skill docs: capability diagnostics recipe** — `recipes.md` section on
      `CapabilityAudit`/Doctor's `--- Capability ---` section
      (consumer-facing declared-vs-implemented diagnostics), and a
      `modules.md` metaengine row note. Shipped 2026-08-16, undocumented.
      _(Effort: XS)_
- [ ] **Duplication-baseline hygiene** — add `//art-dupl:accept` directives
      at the 9 intentional clone sites; dirty-tree guard for baseline
      re-pins; re-pin at next clean tree (current pin includes in-flight
      foreign code).
      _(Effort: S)_ 2026-08-16: wave-4 triage added 3 more directives
      (2 false-sharing bench mirrors, 1 cross-module test read-file idiom)
      and consolidated 2 groups (metaengine `enginesSnapshot`, event
      `seekableReadFrom`) — gate green, baseline untouched (annotations
      suppress live). The 9 legacy sites + dirty-tree guard remain.
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
- [ ] **CI `shfmt -d` drift check on `scripts/`** — the pre-commit hook
      formats staged files only; a CI check on the whole tree catches
      formatter drift before it reaches a hook (root cause of the 4x key
      mangling).
      _(Effort: XS)_
- [ ] **`reset_db` helper for docker mysql/mariadb/pg test loops** —
      reset-before each run, not just after; a shared-DB `-count>1` run
      polluted state mid-loop on 2026-08-15.
      _(Effort: XS)_
- [ ] **Re-run soak suite** (`SOAK_SKIP_*` unset) after the graph/vector
      engine additions.
      _(Effort: S)_
- [ ] **quic convergence flake watch in CI** — `TestQuicConvergenceSuite`
      under `-race -count=3` in the matrix; the Log order-tolerance fix
      landed 2026-08-15, watch for recurrence.
      _(Effort: S)_
- [ ] **metaengine-quickstart: graph + vector demos** — 4 engines now ship
      them; the example covers maps only.
      _(Effort: S)_
- [ ] **Prune docker test images** — mysql:8 + mariadb:11 probe images
      (~1.2GB) after the 2026-08-15 container cleanup; keep
      postgres:16-alpine (testcontainers reuses it).
      _(Effort: XS)_
- [ ] **`#verify` per-module timeout headroom** — quic convergence hit a
      15s near-miss under load; revisit per-package `-timeout` tuning.
      _(Effort: XS)_
- [ ] **Delete junk from repo root** — `t/`, `result/` (16MB root-owned),
      `reports/coverage.out` (empty), `reports/jscpd-report.json`; drop the
      orphaned `stash@{0}` (WIP @ `e87be3143`, pre-recovery leftovers).
      _(Effort: XS)_
- [x] **One bench system** — DONE 2026-08-16, revised on research: keep benchkit
      (SDK) + cqrs-bench (CLI) + stack/bench (go test -bench entry, feeds the
      gate); metaengine/bench SLIMMED, not deleted (31 of 36 benchmarks test
      planner internals + layout calibration via direct engine imports that the
      factory-driven, engine-agnostic benchkit cannot reach; only the 5
      benchkit-redundant ones removed). Deleted: 15 integration bench files +
      1 bench func, v2-era `benchmarks/` dir + root baseline. CI regression
      gate now FAILS on breach (median-based `scripts/benchmark-regression.sh`,
      25% threshold, artifact-baseline self-refresh).
      Plan: `docs/planning/2026-08-16_15-09_one-bench-system-consolidation.md`.
      _(Effort: M)_
- [x] **`DecorateJournal` for `VersionedSeekableJournal`** — DONE 2026-08-16:
      `event.DecorateJournal(journal, sourceT)` added (ADR-0126 journal
      counterpart; preserves Journal + SeekableJournal + StreamingJournal +
      io.Closer, applies the transform per 128-event chunk on streaming reads;
      new `event.ErrInnerStoreNotStreaming` sentinel). The old hand-wrapper
      silently dropped StreamingJournal (bbolt/pebble/memory/eventstore all
      implement it). `schema.NewVersionedSeekableJournal` is now a deprecated
      shell delegating to `DecorateJournal` + `UpcastSourceTransform`;
      canonical recipe documented in skill core.md.
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

- [x] **SQL injection surface** — allowlists + ORDER BY quoting.
      SHIPPED 2026-08-15 (`storage/sql.ValidateIdentifier`/`ValidateOperator`,
      `BuildWhereClauseChecked`, view query validation — see CHANGELOG).
      tursoengine DSN-redaction SHIPPED 2026-08-16 (`redactDSN` on every open
      error, `tursoengine/register.go`). Fuzz `ValidateIdentifier` against
      sqlite/pg/mysql metacharacter sets SHIPPED 2026-08-16
      (`storage/sql/validate_fuzz_test.go` — 3 fuzz targets, 451K execs, 0
      crashes). Item closed.
      _(Effort: S)_
- [x] **Resource leaks** — `sqliteengine`/`tursoengine` self-opened
      `*sql.DB` `Close()` leak. DONE 2026-08-16: `sqliteengine.OwnDB(eng)` marks
      self-opened DBs as engine-owned (pinned by `close_ownership_test.go`);
      both `NewSQLiteEngineFromDSN` and `tursoengine.New` use it.
- [x] **Core defects** — DONE 2026-08-16 (commit `06e046c2f`, each pinned by a
      dedicated regression test): singleflight leader-ctx capture
      (`decider/load.go`, `context.WithoutCancel`); per-handler command
      middleware (`command/memory_bus_test.go`); query audit fake RequestIDs
      (`query/audit_test.go`); `Pagination.Offset()` underflow
      (`query/pagination_test.go`). STILL OPEN: `kv.Cache` shared `*T`;
      TypedQueryStore hardcoded JSON decode (`query/typed.go`); ghost
      `event.ErrBinaryNotFound` (document or delete).
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
- [ ] **Serialize or re-budget the system projection-wait tests** —
      `TestSystem_ResetProjection_RestartAndReplay` (tight 5s
      `waitForProjectionProcessed` budget) overlaps the `t.Parallel`-ed
      `TestSystem_HealthCheck_FailedProjection`; load-flaky on busy machines
      (observed 2026-08-16, pre-existing). The deterministic "phase-2 replay
      dead" defect this was originally filed under turned out to be a test
      fixture bug — the shared-cache in-memory SQLite DSN was wiped once engines
      began closing self-opened `*sql.DB`; fixed by `5d66308c3` (file-backed
      `sqliteFileDSN`), verified green 3x. Evidence:
      `docs/status/2026-08-16_03-44_withactor-resume-gate-investigation-two-defects.md` §h.
      _(Effort: S)_
- [ ] **Enforce api-stability golden regen mechanically** — three consecutive
      feature commits (`842741cab`, `313d14b02`, plus sqliteengine DSN/OwnDB)
      shipped new exports without regenerating `docs/api_surface.txt`; every
      fresh checkout of those revisions fails the gate. Add the checker as a
      pre-commit hook step (fast: ~1s GOWORK=off run) so the drift cannot land.
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
      non-clamping philosophy (FAQ); MariaDB dialect + numeric-safe sort
      recipe (recipes.md §2.x).
      _(Effort: S)_
- [ ] **storage/view validated-WHERE rollout review** — the validated-checked
      WHERE changes (query.go/store.go, landed 2026-08-15) need an API-doc
      pass in the skill references.
      _(Effort: S)_
- [ ] **docs/DOMAIN_LANGUAGE.md additions** — "dialect", "capability probe",
      "degraded ADT" became load-bearing terms with the MariaDB/CTE-probe
      work; define them.
      _(Effort: XS)_
- [ ] **pebbleengine/bboltengine README symmetry** — note graph=unsupported
      (pebble) and audit bbolt, matching the new vector rows.
      _(Effort: XS)_
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
- [ ] **Delete `storage/sql.BuildWhereClause`** — deprecated 2026-08-15
      (interpolates column names/operators); `BuildWhereClauseChecked` is
      the validated replacement.
      _(Effort: XS)_
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
