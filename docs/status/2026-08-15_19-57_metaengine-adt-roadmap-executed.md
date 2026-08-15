# Status Report: Metaengine ADT Roadmap — 4 TODO Items Executed

**Date:** 2026-08-15 19:57
**Scope:** The 4 metaengine TODO items from `TODO_LIST.md` (MariaDB compat, vector on LSM engines, native graph on PG/MySQL, SQLite recursive CTE) + on-sight fixes found along the way.
**Verification style:** every engine change live-verified against real servers (docker: MySQL 8.4, MariaDB 11.8, Postgres 16) plus full per-module suites.

---

## a) FULLY DONE

### 1. mysqlengine MariaDB dialect support (TODO #1) — DECIDED & SHIPPED

**Decision:** dialect support (not a MySQL-8 test backend). Preserves MySQL 8 behavior
byte-for-byte AND fixes MariaDB, which is what every nix test env actually runs
(`nix/vm/mysql.nix` = `pkgs.mariadb`).

- `SELECT VERSION()` dialect detection at engine construction (`metaengine/mysqlengine/dialect.go`): "mariadb" vs "mysql"; exported `Dialect()`.
- MariaDB SQL rendering: `JSON_EXTRACT(value,'$.f')` paths, `JSON_UNQUOTE(JSON_EXTRACT(...)) = ?` filters with natively-bound scalar params, no functional-index DDL (unsupported → graceful skip). MySQL 8 keeps `value->'$.f'` + `CAST(? AS JSON)` — zero behavior change.
- **Empirically probed before writing code** (docker MySQL 8.4 + MariaDB 11.8): MariaDB 11.8 rejects BOTH `->` and `CAST AS JSON` with Error 1064; the chosen MariaDB filter form returns identical rows to the MySQL form on both servers for strings, numbers (int/float), bools, IN lists, and range cursors.
- Evidence: full `metaengine/mysqlengine` suite **3/3 stable against MySQL 8.4 AND MariaDB 11.8** (fresh DB per run), including the previously-failing 3 pushdown tests, ADTMatrix, and HealthCheck.

### 2. ADTMatrix/stream-log cross-test interference — ROOT-CAUSED & FIXED (was lumped into TODO #1)

The "ADTMatrix fails in nspawn env" class was **pre-existing at HEAD**: parallel tests
(`RunStreamLogBackendTest`, `RunAtomicAppenderTest`, ADTMatrix StreamLog scenario) all
wrote to the SAME fixed `collection="events", stream="s1"` on one shared server.
Reproduced at HEAD against a fresh MySQL 8 database (2 parallel-run failures).

- Added `enginetest.RunAtomicAppenderTestIn(t, eng, col)` (mirrors the existing `RunStreamLogBackendTestIn`); mysqlengine tests now use unique collections.
- Evidence: mysqlengine suite green 3/3 against both servers after fix; HEAD fails intermittently before it.

### 3. Native graph dispatch on Postgres/MySQL via recursive CTE (TODO #3)

- `metaengine/pgengine/graph.go` + `metaengine/mysqlengine/graph.go`: `meta_graph_edges` table (PK + `(collection, from_node)` index), `GraphAddEdge` (PG `ON CONFLICT DO NOTHING` / MySQL `INSERT IGNORE`), `GraphNeighbors` = single `WITH RECURSIVE` walk (depth-capped, UNION dedup, DISTINCT nodes, start node excluded).
- Profiles upgraded `ADTGraph: ComplexityON degraded → ComplexityODegree native` — planner no longer emits the DEGRADED "add a graph engine" diagnostic for PG/MySQL.
- Node-key encoding mirrors sqliteengine (`encodeNodeKey`) for cross-engine parity.
- Evidence: new graph tests (depth semantics, diamond dedup, cycle safety, depth 0, idempotent edge add) pass on **PG 16 (testcontainer, full 320s suite ok)**, **MySQL 8.4**, **MariaDB 11.8**; adttest Graph scenario now runs against both engines with memory parity (previously skipped).

### 4. Recursive CTE for SQLite deep traversals (TODO #4)

- `sqliteengine` probes `WITH RECURSIVE` once at construction; when supported (plain SQLite — verified on modernc), `GraphNeighbors` runs as ONE recursive-CTE query instead of one query per node per level. Drivers/servers lacking CTE support (some libSQL/Turso deployments) auto-fall back to the retained iterative BFS — graceful degradation, no operator knob.
- Evidence: sqliteengine suite green, incl. new 100-node deep-chain test (full + partial depth) and diamond-dedup test; CTE probe unit test.

### 5. Brute-force vector search on Pebble/bbolt (TODO #2)

- `metaengine.VectorDistance` exported (wraps the memory engine's distance math → numeric parity).
- `keycodec.VectorKey/VectorPrefix` (`vec\x00<col>\x00<id>`, JSON float32 dims) — on-disk layout shared with the LSM key-shape family.
- `pebbleengine/vector.go` + `bboltengine/vector.go`: `VectorInsert`/`VectorSearch` (prefix scan + in-Go distance + top-k). bbolt profile now declares `ADTVector: ComplexityON` degraded (pebble already declared it — now actually implemented).
- Evidence: per-engine vector tests (cosine/euclidean/dot metric-parity vs `MemoryVectorIndex`, upsert-overwrite, k-truncation, empty collection) green; **adttest Vector scenario now RUNS on pebble+bbolt (previously auto-skipped) with cross-engine parity**.

### 6. On-sight fix: bash-map key mangling regression (3rd occurrence)

`scripts/check-module-layers.sh` + `scripts/check-coverage.sh` at HEAD contained
daemon/formatter-mangled keys (`LAYER[cmd / cqrs - gen]`, `[storage / memory]`) —
silently disabling layer + budget checks for ALL multi-segment modules. Same class
as the 2026-08-14 incident documented in AGENTS.md.

- Restored plain-path keys (`cmd/cqrs-gen`, `storage/memory`, ...) across both scripts.
- Evidence: `TestLayerScriptKeysMapToModules` green (was RED at HEAD); `nix run .#check-arch` **passes** again — which also validates all budgets with my changes in place.

### 7. Housekeeping

- api-stability golden regenerated (`--update`, 4067 exports; includes concurrent session's in-flight exports); api-stability meta-tests green.
- TODO_LIST.md: all 4 items marked `[x]` with implementation notes and known limitations.
- pebbleengine README corrected (claimed GraphBackend implemented — it never was; now documents vector + honest graph story).
- doc-check green (797 references).
- Scoped gofumpt/goimports applied to touched files; post-format all suites re-run green.

---

## b) PARTIALLY DONE

1. **Final `nix run .#verify` gate** — NOT yet run. Reason: a concurrent session is
   mid-refactor in `metaengine` core (roles.go, replicator, runtime_backend, planner,
   query, memory_engine, ...) and its tests currently hang/fail (see d1). The shared
   tree cannot pass the gate until that work settles. All MY modules are verified
   green per-module. **Effort: S once the tree is stable.**
2. **Coverage baseline** — `nix run .#check-coverage` is RED: metaengine documented
   83.3% vs actual 48.9%. Root cause is the concurrent refactor's hanging replicator
   tests (10m timeout aborts the run mid-package), NOT my changes. I deliberately did
   NOT `--update` the baseline to avoid encoding a half-finished state. **Effort: S
   after (1).**
3. **Turso remote CTE behavior** — the CTE probe makes local Turso/libSQL correct by
   construction, but I could not live-verify against a REMOTE Turso server (no
   credentials). The probe guarantees correctness either way; only the perf mode
   (CTE vs iterative) is unverifiable from here. **Effort: S with a Turso URL.**

---

## c) NOT STARTED

1. **DuckDB recursive-CTE graph** — intentionally left degraded (TODO scoped PG/MySQL).
   DuckDB supports `WITH RECURSIVE`; same recipe applies. Priority: Medium. Effort: M.
2. **MariaDB functional-index replacement** — ApplyLayout currently no-ops on MariaDB
   (queries work, full scans). Proper fix: virtual generated column + index, or
   type-aware `JSON_VALUE(... RETURNING)` once verified. Priority: Medium. Effort: M.
3. **badgerengine vector backend** — trivially portable from pebble/bbolt (same
   keycodec shapes). Not in TODO scope. Priority: Low. Effort: S.
4. **Search/Spatial over-declaration cleanup** (pebble declares, doesn't implement) —
   pre-existing; needs the engine over-declaration census (TODO_LIST ~L150). Priority: Medium.
5. **nspawn MariaDB live re-verification** — needs root; docker verification stands in.
   Priority: High before next release. Effort: S (run `nix run .#integration-mysql-nspawn`).

---

## d) TOTALLY FUCKED UP

1. **Concurrent session's in-flight refactor breaks metaengine core** (NOT my work, not
   touched by me): `TestVerify_DetectsDrift` (features4_test.go:570), `TestPromoteEngine_Cutover`
   (replicator_test.go:272), `TestReplication_BufferOverflowMarksStale` (hangs 9m47s →
   10m suite timeout aborts the whole package run). Files: roles.go (new), runtime_backend.go,
   replicator*, planner.go, query.go, memory_engine.go + ~10 more, modified minutes
   apart during my session. Severity: blocks the full verify gate + coverage run.
   Mitigation: none from my side — wait for that session to finish, then re-run.
2. **The daemon/formatter keeps mangling bash-map keys in scripts/** — this is the
   THIRD occurrence (AGENTS.md documents two reverts). The `TestLayerScriptKeysMapToModules`
   guard exists and DID catch it (red at HEAD), yet the broken script still got committed —
   meaning the failing test isn't gating the daemon's auto-commits. Root cause unaddressed.
   Mitigation: I restored both scripts; suggest a pre-commit hook (see f10).
3. **pgtestcontainer per-test DB provisioning dies on leftover databases** — if a
   previous run is killed, `test_1 already exists` fails every subsequent test until
   manual `DROP DATABASE`. Hit it; worked around manually. Should self-heal
   (`DROP ... IF EXISTS` or unique suffix).

---

## e) WHAT WE SHOULD IMPROVE

1. **Empirical SQL verification before engine work** — the MariaDB fix took one probe
   program (~40 lines) and 3 minutes, and overturned two "well-known" assumptions
   (MariaDB supports `->`; universal `JSON_EXTRACT = ?` param form — both false).
   Pattern to keep: probe docker servers first, then write generators.
2. **Server-backed engine tests need per-test DATABASES, not just unique collections** —
   pgengine does this (pgtestcontainer); mysqlengine now fakes it with unique
   collections. A `mysqltestcontainer` helper (or per-test schema in `mustNewMySQLEngine`)
   would remove the whole class. Impact: every shared-server engine suite.
3. **Engine README/API truth-drift** — pebbleengine README claimed a GraphBackend that
   never existed. The over-declaration census TODO (Supports vs implemented interface
   check) should also sweep READMEs.
4. **adttest fixed collections ("events", "s1") are a footgun** — the harness
   documents the constraint but nothing enforces it. A harness-level random collection
   suffix (opt-in per Factory) would make server-backed matrices safe by default.

---

## f) Top next tasks (harvest-ready)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Re-run `nix run .#verify` once the concurrent metaengine refactor lands; fix fallout | Critical | S | Quality |
| 2 | Rebaseline `check-coverage.sh` (metaengine 83.3% → actual) after (1) | Critical | S | Quality |
| 3 | Live-verify MariaDB dialect fix in the real nspawn env (`integration-mysql-nspawn`, needs root) | High | S | Quality |
| 4 | pgtestcontainer: auto-heal "database already exists" on re-run after killed run | High | S | Bug |
| 5 | Add pre-commit guard rejecting spaced `LAYER[...]`/`DEP_BUDGET[...]` keys in scripts/ (daemon-mangle prophylaxis) | High | S | Quality |
| 6 | DuckDB recursive-CTE graph backend (recipe: pgengine/graph.go) | High | M | Feature |
| 7 | Engine over-declaration census: align Profile().Supports/DegradedADTs with implemented interfaces (pebble Search/Spatial first) | High | M | Cleanup |
| 8 | DuckDB `AggregateReader` aggregation pushdown (existing TODO) | High | L | Feature |
| 9 | MariaDB ApplyLayout: virtual generated-column + index instead of no-op | Medium | M | Feature |
| 10 | MariaDB type-aware JSON ORDER BY (numbers currently text-sorted) | Medium | M | Bug |
| 11 | badgerengine VectorBackend (port from pebble/bbolt via keycodec) | Medium | S | Feature |
| 12 | adttest: opt-in per-Factory collection suffix for server-backed matrices | Medium | S | Quality |
| 13 | Journal true-seq resumption (existing TODO, O(offset) → O(log n)) | Medium | M | Feature |
| 14 | Dgraph engine hardening (existing TODO: RunInTx decision, CI matrix) | Medium | M | Quality |
| 15 | Engine declaration-vs-README sweep (pebble README bug class) | Low | M | Documentation |

---

## g) Top question

**Should the nix MySQL test environments stay MariaDB-only, or should we add a real
MySQL-8 backend (nixpkgs `mysql84` nspawn/VM) to the integration matrix?**

Context: I shipped dialect support so BOTH work, and live-verified both via docker.
But the nix envs only ever exercise MariaDB — a MySQL-8-only regression (e.g. in the
`->`/`CAST` rendering path) would only surface in production or via my ad-hoc docker
runs. Adding a MySQL-8 service doubles the VM/nspawn matrix cost; I can't judge that
tradeoff for your infra. My recommendation: yes, add it — the two dialects genuinely
diverge (this session proved it empirically) and the mysqlengine is now the only
engine claiming dual-dialect support.
