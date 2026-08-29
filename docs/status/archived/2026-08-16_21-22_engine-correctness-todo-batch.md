# Status Report — 2026-08-16 21:22 — Engine Correctness TODO Batch (6/10 shipped, 1 mid-flight)

> **COMPLETION ADDENDUM (2026-08-16 ~22:30): the batch is now 9/10 done.**
> Items 1-9 shipped and verified; item 10 (nspawn) remains blocked on root.
>
> - **adttest suffix landed + twin bug fixed** — `Scenarios()` derives a per-RUN
>   token (`_<r><unix-nano>`); all 17 collection literals scoped. Verified from a
>   CLEAN server (recreated datadir): mysqlengine `-count=3` GREEN, DB shows only
>   suffixed collections with exact expected state per run. Stale test renamed to
>   `TestScenarios_AllADTs` (count-neutral) + the 4 missing scenario names added
>   to its coverage list.
> - **Item 2 shipped — MariaDB generated columns** (`mysqlengine/layout.go`):
>   VIRTUAL TEXT generated column per field (metadata-only ALTER — no table
>   rebuild; same mechanics as MySQL's hidden functional-index columns) +
>   composite `(collection, gc(N))` prefix index (1785-byte key, full-value
>   recheck — no truncation semantics). **Empirical discovery:** MariaDB 11.4
>   does NOT substitute generated columns into `JSON_UNQUOTE(JSON_EXTRACT(...))`
>   predicates (EXPLAIN `access_type: ALL` with the index listed in
>   possible_keys) — so `filterExpr` rewrites pushdown filters to the gc column
>   for laid-out fields; EXPLAIN then reports `ref` access on the composite
>   index. Integration test `TestMariaDBApplyLayout_GeneratedColumnFilter` pins
>   DDL + index usage + missing-field/long-value semantics + idempotency.
>   Answering open question 2: VIRTUAL dissolves the ALTER-risk concern —
>   in-place is safe and is what shipped.
> - **Items 8+9 shipped — benches** (`graph_bench_test.go`, `sort_bench_test.go`):
>   crossover tables recorded in `METAENGINE-LIVE-LATENCY-MODEL.md` §9. Graph:
>   iterative wins depth 1 (2-4x), parity depth 2, CTE wins depth ≥3 (up to 6x),
>   size-independent, identical shape on MariaDB 11.4 + MySQL 8.4 (Docker).
>   Sort: dual-key +26% vs single-expression on both servers; MySQL JSON-typed
>   arrow 2.5x faster than MariaDB dual-key. New TODO: depth-1 graph
>   short-circuit (XS, measured 2-4x win).
> - **`stack/mysql` suite GREEN `-count=3`** against the userspace MariaDB
>   (nspawn substitute; needed `GRANT ON`cqrs_%`.*` for derived multidb DBs).
>   Found + fixed a rerun-isolation bug: `createMySQLDB` now DROPs derived
>   databases before CREATE (testcontainers always fresh, shared servers were not).
> - **All wrap-up gates:** api-stability golden regenerated; `nix fmt` + doc-check
>   GREEN (910 refs); duckdbengine pushdown (CGo) 8/8 GREEN; metaengine core
>   tests GREEN; mysqlengine full suite `-count=2` GREEN.
> - **Ops lessons recorded:** `/dev/tcp` redirections silently fail in the tool
>   shell (mvdan/sh) → false "server down" readings; use `mysqladmin ping`.
>   `kill` is not a builtin here → `/run/current-system/sw/bin/kill`. A live
>   mysqld whose datadir was trashed keeps serving from unlinked inodes —
>   `pgrep -a mysqld` + `@@datadir`-vs-start-time before concluding server state.

Scope: the 10-item engine-correctness TODO batch from `paste_1.txt` (mysqlengine upsert audit,
MariaDB functional indexes, enginetest per-run suffixes, adttest graph/vector coverage,
convergence order-tolerance, quic ordering docs, 3 benches, nspawn integration).

Session evidence: pgengine matrix GREEN via Docker testcontainers (12.2s full run);
local engines GREEN under `-count=2`; shared MariaDB 11.4.12 running at `127.0.0.1:33061`
(userspace, no root) proving the shared-server failure mode live.

---

## 1. Fully done and verified

### 1.1 mysqlengine upsert semantics audit ✅

- Audited every upsert site vs pgengine: `MapSet` (`ON DUPLICATE KEY UPDATE value = VALUES(value)`
  ≡ `ON CONFLICT ... DO UPDATE SET value = excluded.value`), `CounterIncrement`
  (`value = value + VALUES(value)` ≡ `meta_counter.value + excluded.value`),
  `GraphAddEdge` (`INSERT IGNORE` ≡ `ON CONFLICT DO NOTHING`). All single-statement
  atomic, all routed through `conn()` so they join `RunInTx` exactly like pg.
- **Parity confirmed.** Affected-rows divergence (MySQL 1/2/0 vs PG 1) is unobservable —
  neither engine reads `RowsAffected`.
- Documented in `metaengine/mysqlengine/backends.go` MapSet doc: (a) ON DUPLICATE KEY fires on
  ANY unique key while ON CONFLICT names a constraint — meta_map has only its PK today, adding
  a second unique index would silently widen MySQL's trigger; (b) `VALUES()` is deprecated in
  MySQL 8.0.20+ but MariaDB lacks the alias form, so `VALUES()` is the correct dual-dialect choice.

### 1.2 quic pooled-stream ordering guarantee — verified + documented ✅

- **Verified by code reading:** pooled mode (`sendOpPooled`) is strict per-peer FIFO via three
  stacked mechanisms: (1) QUIC per-stream byte ordering, (2) sender serializes
  write-frame→read-ack under `pc.streamMu`, (3) receiver (`handlePooledStream`) processes frames
  sequentially in one loop. Non-pooled `sendOp` (default) has **NO cross-op ordering** — per-op
  streams handled by concurrent goroutines.
- Documented in three places: `WithStreamPooling` (guarantee, scope, head-of-line tradeoff),
  `sendOp` (no-ordering warning), `sendOpPooled` (mechanism + silent-drop loss semantics shared
  with `sendOp`).

### 1.3 Convergence suite order-tolerance audit ✅

- Swept ALL `waitFor*` helpers: `waitForMap` (single-value DeepEqual), `waitForCounter`
  (exact convergent CRDT value), `waitForSetContains` (membership), `waitForLogTail`
  (unordered multiset via fixed `sameLogTail`), `waitForMultimap` (set equality via
  `sameSetAny`), `waitForPeers` ×2 (count-based). **All order-tolerant.**
- One defect found and fixed: `sameLogTail` carried a stale contradictory doc paragraph
  claiming ORDER-sensitive comparison while the code compares an unordered multiset.
  Cleaned in `metaengine/irohengine/convergence_suite.go`.

### 1.4 adttest graph depth>2 + cycle scenarios ✅

- Added 3 scenarios to `Scenarios()`: `GraphDepth3Diamond` (D reachable via 2 paths + E at hop 3),
  `GraphCycle` (A→B→C→A + D→B re-entry, depth-4 walk must terminate, dedupe, exclude start),
  `GraphDepthBound` (chain depth-2 read excludes deeper nodes; off-by-one canary for CTE depth
  predicate vs iterative level counter).
- Updated `TestScenarios_AllFourteenADTs` 14→17 (test NAME now stale — see §4).
- Verified parity GREEN on: memory+sqlite, pebble, bbolt, badger, and **postgres (CTE vs
  iterative BFS divergence — the exact divergence class the TODO targeted — none found)**.

### 1.5 pgengine Vector — implemented (was a real capability bug) ✅

- **Discovery:** pgengine's Profile declared `ADTVector` in `Supports` + `DegradedADTs`, but the
  engine implemented **no VectorBackend at all** — and `Store.executeVectorSearch` has no
  degraded fallback, so any vector query routed to a pg-only deployment failed with
  `errUnsupportedVectorReads`. Declared-degraded was a lie.
- Implemented `metaengine/pgengine/vector.go`: `VectorInsert` (JSONB upsert), `VectorSearch`,
  `VectorSearchFiltered` (filter-then-score) over a new `meta_vector` table — brute-force O(N·D)
  using the shared `metaengine.VectorDistance`/`VectorMatchesFilters`/`TopKNearest` so semantics
  are identical to every other degraded engine.
- Verified against real Postgres: full ADT matrix GREEN including `Vector/postgres` and
  `VectorFiltered/postgres` parity vs the memory engine's in-memory index (the TODO's exact ask).
  CapabilityConformance GREEN.

### 1.6 enginetest per-run collection suffixes ✅ (the TODO item itself)

- New `metaengine/enginetest/runcollection.go`: process-unique `runToken` (pinnable via
  `ENGINETEST_RUN_TOKEN`) + exported `ScopedCollection(name)` with per-call disambiguator.
- Applied at every internal chokepoint: `RunStreamLogBackendTestIn`, `RunSeqSeekableStreamLogTestIn`,
  `RunAtomicAppenderTestIn`, `RunScanBackendTest`, `RunPushdownTest`, `assertTxCommitSetup`
  (→ `RunTransactionalTest` + `RunTransactionalBaselineTest`), `RunConcurrentTxTest`,
  `RunRecordStampTest`, `RunAutoCRUDSoak`.
- **Breaking helper signature change:** `RunPushdownTest`'s `run` closure now receives `col`
  (closures previously captured the literal collection — the hidden coupling that made
  suffixing impossible). Updated all 10 call sites (pgengine ×5, duckdbengine ×5).
- Verified: pebble/bbolt/badger/memory helper suites GREEN under `-count=2`.

---

## 2. Partially done / mid-flight

### 2.1 adttest.RunMatrix has the SAME shared-server bug — fix designed, not yet landed ⚠️

Running `mysqlengine` tests twice against the shared MariaDB exposed it live:

```
Counter:  memory=alpha=13  mysql=alpha=52   (4 runs of accumulation)
StreamLog: memory=version:3 mysql=version:12
```

The matrix's scenario collections are fixed constants ("counters", "events", ...) and the
RunMatrix doc merely DOCUMENTS "never run twice against the same server" — the exact
documented-constraint anti-pattern the enginetest TODO item set out to remove.

**Design (ready to implement):** per-RUN token only — NOT per-call. The cross-engine parity
check requires every factory inside one `RunMatrix` invocation to land in the SAME collection,
so the suffix must be computed once per process and be deterministic within it
(`name + "_" + runToken`). ~19 collection literals across `Scenarios()` in
`metaengine/adttest/harness.go` to wrap via a `scenarioCollection()` helper
(lines ~221–749: users, fruits, counters, graph, graph_rm, graph_und, graph_deep, graph_cycle,
graph_bound, sorted, log, tasks_by_user, events, vectors, vectors_filtered, docs, places).

**Current state: mysqlengine `-count>1` against a shared server is RED because of this.**

### 2.2 Benchmarks (items 8+9) — infrastructure ready, benches not written

- Ephemeral userspace MariaDB 11.4.12 is UP: `127.0.0.1:33061`,
  `MYSQL_TEST_DSN="cqrs:cqrs@tcp(127.0.0.1:33061)/cqrs_test?parseTime=true&multiStatements=true"`
  (log: `/tmp/mariadb-cqrs/mysqld.log`, socket `/tmp/mariadb-cqrs/mysql.sock`).
- Note: `TestGraphNeighbors_IterativeMatchesCTE` (correctness parity) already exists in
  mysqlengine — the missing piece is the crossover BENCH feeding planner cost constants.

---

## 3. Not started

- **Item 2 — MariaDB functional-index alternative** (generated columns + plain index instead of
  ApplyLayout no-op). Effort M. Ready to test against the live MariaDB.
- **Item 10 — `nix run .#integration-mysql-nspawn`**: needs root (`systemd-nspawn`), which this
  session cannot escalate (sudo banned in tool policy). The userspace MariaDB provides the same
  real-env verification for engine modules, but the full nspawn path ALSO runs `stack/mysql` —
  that suite has not run yet (can run it against the userspace server instead).

---

## 4. Totally fucked up / loose ends (own mistakes first)

1. **`TestScenarios_AllFourteenADTs` name is now stale** — asserts 17 scenarios; rename to
   `TestScenarios_AllSeventeen` or count-neutral `TestScenarios_Coverage`.
2. **api-stability golden NOT regenerated** — `enginetest.ScopedCollection` is a new exported
   symbol in the metaengine module. Must run
   `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update` before
   any gate run, or `#verify` goes RED on a 3-4 min cycle.
3. **No `nix fmt` / doc-check / lint pass yet** this session (doc-comment edits + new files are
   unformatted-unverified; golines may reflow the long doc comments).
4. duckdbengine pushdown tests vetted but NOT executed (CGo runtime untested this session).
5. Auto-commit daemon already committed the vector/graph/ordering work (`98a46e95b`) mid-session
   — fine, but means the mid-flight adttest fix must land as its own commit.
6. MariaDB grant gotcha burned a cycle: TCP from 127.0.0.1 maps to the `localhost` account —
   needed `'cqrs'@'localhost'` IN ADDITION to `'cqrs'@'%'`.

---

## 5. What I could have done better

- Run a shared-server `-count=2` pass BEFORE building the enginetest suffix infra — would have
  exposed the adttest twin bug at design time instead of after.
- Batch the doc-comment writes and run `nix fmt` once at the end instead of leaving formatting
  unverified across 6 touched modules.
- Rename the stale test in the same edit that changed its count.

---

## 6. Next tasks (ordered)

1. Land the adttest per-run scenario suffix (design in §2.1) + re-run mysqlengine `-count=2` GREEN.
2. Rename `TestScenarios_AllFourteenADTs`.
3. Regenerate api-stability golden (`ScopedCollection` export).
4. `nix fmt` + doc-check + golangci on touched modules (metaengine, pgengine, mysqlengine,
   duckdbengine, irohengine).
5. Run duckdbengine pushdown tests (CGo) against the userspace server or temp DB.
6. Item 2: MariaDB generated columns in `ApplyLayout` — `ALTER TABLE meta_map ADD COLUMN
   gc_<field> <type> GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(value,'$.<field>')))`,
   - plain index; guard by `e.dialect == "mariadb"`; verify EXPLAIN uses the index.
7. Bench: CTE vs iterative BFS crossover across depth 1–6 × row counts → record crossover table
   → feed `ReadCosts`/planner notes in METAENGINE-LIVE-LATENCY-MODEL.md.
8. Bench: MariaDB dual-key `CAST(... AS DECIMAL)` sort vs MySQL single JSON key (same dataset,
   both engines) → record overhead ratio.
9. Run `stack/mysql` suite against the userspace MariaDB (nspawn substitute).
10. Extract the userspace-MariaDB startup into `scripts/dev-mariadb.sh` (rootless alternative to
    nspawn) if adopted.
11. Re-run `nix run .#verify` exclusively (nothing else heavy) once 1–5 land.
12. Consider documenting pgengine's `meta_vector` table in pgengine README capability table.

## 7. Questions (cannot resolve myself)

1. Is passwordless sudo available to agent sessions for `integration-mysql-nspawn`, or should the
   rootless userspace-MariaDB pattern become the canonical real-env verification?
2. For MariaDB generated columns: ALTER the live `meta_map` in-place (online DDL risk on large
   tables) vs an opt-in layout path — which do you prefer?
3. Should `ENGINETEST_RUN_TOKEN`-style pinning also be added to the adttest suffix (CI
   correlation), or keep adttest tokens fully internal?
