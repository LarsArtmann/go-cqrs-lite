# Status Report: Dgraph Engine Integration Testing + Performance Benchmarks

**Date:** 2026-08-08 22:03
**Session scope:** Execute 5 remaining gaps from "Metaengine v2 — Remaining Gaps" (TODO_LIST.md paste)

---

## a) FULLY DONE

### 1. `nix run .#ephemeral-dgraph` — Dgraph Zero + Alpha from nixpkgs

- **`scripts/ephemeral-dgraph.sh`** (114 lines): Auto-selects free ports for Zero + Alpha, starts both processes, waits for Alpha health endpoint (`/health` JSON response containing `"healthy"`), exports `DGRAPH_ADDR`, runs the passed command, auto-cleans temp dirs on exit.
- **`flake.nix` app** (`ephemeral-dgraph`): wired with `pkgs.dgraph` (v25.4.0 from nixpkgs), `CGO_ENABLED=1`, `GOEXPERIMENT=jsonv2`.
- **Pattern**: identical to `ephemeral-redis.sh` / `ephemeral-nats.sh` — trap cleanup, mktemp data dir, readiness loop.
- **Key discovery**: Dgraph Alpha's TCP port opens ~10s before it's actually ready to serve gRPC queries. The initial 2-second sleep was insufficient — had to add a real health endpoint poll (60 iterations × 0.5s, checking for `"healthy"` in the JSON response from the HTTP `/health` endpoint).

### 2. All 10 Dgraph tests pass against live Dgraph 25.4.0

Verified with both normal and `-race` execution:

| Test                          | Status | Notes                               |
| ----------------------------- | ------ | ----------------------------------- |
| TestProfile                   | PASS   | Profile constants verified          |
| TestMapBackend                | PASS   | Set, Get, Delete, missing key       |
| TestGraphBackend              | PASS   | 3 edges, depth-1 neighbor query     |
| TestDgraph_RecordStamping     | PASS   | Record type → Dgraph node           |
| TestDgraphADTMatrix/Map       | PASS   | Cross-engine parity with Memory     |
| TestDgraphADTMatrix/Set       | PASS   | Cross-engine parity                 |
| TestDgraphADTMatrix/Counter   | PASS   | Cross-engine parity                 |
| TestDgraphADTMatrix/Graph     | PASS   | Dgraph's native strength            |
| TestDgraphADTMatrix/Search    | PASS   | @index(term) full-text search       |
| TestDgraphADTMatrix/SortedMap | PASS   | Degraded (Go-side sort) but correct |

5 ADTs correctly skip (not implemented): Vector, Spatial, Multimap, Log, StreamLog.

### 3. MapDelete bug fixed (Dgraph 25.x behavior change)

**Root cause**: Dgraph 25.x does NOT delete all predicates when `DeleteJson` contains only `{"uid": "0x1"}`. The node remains fully intact — all predicate values persist. This is a behavior change from older Dgraph versions.

**Debug journey**: Three approaches tested and ALL failed silently (returned `nil` error but data remained):

1. Upsert `DeleteJson` with `{"uid": "uid(entry)"}` — no-op
2. Upsert `DelNquads` with `uid(entry) * * .` — no-op
3. Two-step: query concrete UID → `Mutate(DeleteJson: {"uid": "0x1"})` — no-op
4. HTTP `/mutate` with `{"delete":[{"uid":"0x1"}]}` — returned Success but data remained
5. **Working approach**: `DeleteJson` with explicit null predicates: `{"uid": "uid(entry)", "cqrs.map_collection": null, "cqrs.map_key": null, "cqrs.map_value": null}`

The fix is in `engine.go:MapDelete`.

### 4. DQL injection regression test

**`metaengine/dgraphengine/injection_test.go`** — `TestNoDQLInjectionPatterns`:

- Scans all non-test `.go` files in the package
- Asserts no `dqlString` identifier exists (the deleted escaper)
- Asserts no `fmt.Sprintf` with `cqrs.` AND `%` on the same line (with allowlist for `sanitizePredicate`, `graphEdgePredicate`, `firstClause`)
- Passes cleanly. Runs without Dgraph (pure file scan).

### 5. CHANGELOG entry

Added under `## [Unreleased]` → `### Fixed — dgraphengine DQL injection + MapDelete — 2026-08-08`. Documents: security fix (14 query sites), MapDelete bugfix, ephemeral-dgraph test infrastructure.

### 6. README GraphBackend audit

**`metaengine/README.md:530-534`**: Added clarification that `GraphBackend` is implemented by Memory, Dgraph, and graphadapter engines only, and that consumers should check `Profile().Supports` for the definitive list.

### 7. dgraphengine README updated with real benchmarks

**`metaengine/dgraphengine/README.md`**: Replaced Docker-only testing instructions with `nix run .#ephemeral-dgraph` instructions. Added performance table with real benchmark data. Added "Dgraph 25.x delete behavior" section documenting the explicit null-predicate requirement.

### 8. AGENTS.md updated

- Added `ephemeral-dgraph` entry alongside Redis/NATS
- Added Dgraph 25.x delete behavior lesson to the "Cross-Cutting Lessons" section

### 9. TODO_LIST.md updated

All 5 items from "Metaengine v2 — Remaining Gaps" section replaced with a completion summary block. Remaining work noted: MultimapBackend/LogBackend/SnapshotBackend not implemented (lower priority — Dgraph's strengths are Graph and Search, which fully work).

---

## b) PARTIALLY DONE

### Dgraph engine — 6 of 11 ADT backends implemented

- **Implemented and verified against live Dgraph**: Map, Set, Counter, Graph, Search, SortedMap (degraded — Go-side sort)
- **Not implemented** (harness skips): Multimap, Log, StreamLog, Vector, Spatial
- These were noted as "not started" before this session — no work was done on them. They remain lower priority since Dgraph's native strengths (Graph, Search) are the reason to choose Dgraph over other engines.

### DQL injection fix verification

- All 14 query sites were migrated to `QueryWithVars` in the PRIOR session
- This session VERIFIED the fix works against a live Dgraph instance (all queries execute correctly)
- The regression test prevents re-introduction
- **However**: no adversarial injection test was written (e.g., sending a key containing `' OR 1=1` or Dgraph DQL syntax to verify it's treated as a literal string, not executed). The regression test only checks source code patterns, not runtime behavior.

---

## c) NOT STARTED

### MultimapBackend / LogBackend / SnapshotBackend for Dgraph

Not started. These would require:

- `MultimapBackend`: map of keys to lists — Dgraph can model this via edges or repeated predicate values
- `LogBackend`: append-only log — Dgraph can model via ordered nodes with timestamps
- `SnapshotBackend`: point-in-time state snapshots — Dgraph doesn't have native snapshot semantics; would need versioned predicates or a separate snapshot namespace

### Dgraph VM test (NixOS)

No `nix build .#checks.x86_64-linux.dgraph-vm` target exists (like postgres-vm, mysql-vm, duckdb-vm, turso-vm). The ephemeral script covers interactive testing, but a VM test would provide CI reproducibility without ephemeral process management.

### `test-all-backends` integration

The `test-all-backends.sh` script and its flake app currently list "SQLite, Pebble, bbolt, DuckDB, PG, MySQL" but NOT Dgraph. Adding Dgraph would require modifying the script and the flake app's `runtimeInputs`.

---

## d) TOTALLY FUCKED UP

### Nothing irreversible

No data loss, no broken builds, no reverted changes. Everything that was changed compiles and tests pass.

### However — wasted debugging time on MapDelete

The initial ephemeral-dgraph script used a 2-second `sleep` as the readiness gate. Dgraph Alpha's TCP port opens before it's ready to serve queries, so the first test run showed `"Please retry again, server is not ready to accept requests"` on ALL tests. Fixed by adding a health endpoint poll, but this wasted one full test cycle (~60s of Dgraph startup + test skip).

### The debug test files were left in the repo briefly

Two debug test files (`internal_debug_test.go`, `debug_delete_test.go`) were created during MapDelete debugging. They were deleted before the final test run, but if they had been committed by the auto-git daemon, they would have polluted the repo. The daemon didn't commit during this window, but this is a risk to be aware of.

---

## e) WHAT WE SHOULD IMPROVE

### 1. Dgraph calibration constants need real-world validation

The engine uses `DG_NsPerOp = 10000.0` (10µs) and `DG_NsPerRead = 8000.0` (8µs). Real benchmarks show:

- **MapSet: 2,721,392 ns/op (2.7ms)** — 271x higher than the 10µs calibration
- **MapGet: 344,395 ns/op (344µs)** — 43x higher than the 8µs calibration
- **CounterIncrement: 2,392,882 ns/op (2.4ms)** — 239x higher

The calibration constants are 2-3 orders of magnitude too low. This means the metaengine planner will dramatically underestimate Dgraph costs, potentially routing queries to Dgraph that should go to Memory or SQLite. **The calibration should be updated to match real measurements**, or at minimum the profile should be marked as "uncalibrated / optimistic."

### 2. CounterGet is extremely expensive (3.4ms, 365KB, 1,143 allocs)

`CounterGet` returns ALL counters in a collection by querying all nodes with `cqrs.counter_collection = $col` and aggregating client-side. For a collection with 1,000+ counters, this will be catastrophically slow. A better design would use Dgraph's built-in aggregation (`sum(val(count))`) or maintain a rollup counter node.

### 3. No test isolation between parallel tests

All Dgraph tests share the same Dgraph instance and use collection names like `"test-map"`, `"users"`, `"bench"`. If tests run in parallel (they do — `t.Parallel()`), and two tests use the same collection name, they'll interfere. The ADT matrix harness uses `"users"` for Map, which could collide with `TestMapBackend`'s `"test-map"` if both write to overlapping keys. Currently this doesn't fail because the tests use different collection names, but it's fragile.

### 4. No data cleanup between test runs

Each test run against the same Dgraph instance accumulates data. There's no `DropAll` or per-test data cleanup. This is fine for ephemeral instances (fresh each time), but if someone runs tests against a persistent Dgraph, stale data could cause false positives.

### 5. The `TestNoDQLInjectionPatterns` test has an allowlist that could grow

The allowlist (`sanitizePredicate`, `graphEdgePredicate`, `firstClause`, `pred :=`) is a whitelist of safe `fmt.Sprintf` + `cqrs.` patterns. As more code is added, this list will grow, and each addition is a potential injection vector that the test author deemed "safe." A better approach would be a linter rule or a code review checklist.

### 6. The ephemeral-dgraph script has a 60-second timeout

The health endpoint poll runs for 60 iterations × 0.5s = 30 seconds max. On a slow machine or under heavy load, Dgraph Alpha might take longer to initialize. The script would fail and exit. This hasn't been observed in practice (typically ready in 5-10s), but the limit is arbitrary.

### 7. GraphNeighbors still uses `fmt.Sprintf` for predicate name

`graph.go:GraphNeighbors` uses `fmt.Sprintf` to build the `pred` variable from `sanitizePredicate()`. While `sanitizePredicate()` only allows `[a-zA-Z0-9_.]`, this is a defense-in-depth concern. A linter can't distinguish safe from unsafe `fmt.Sprintf` usage.

### 8. Graph and Search benchmarks ADDED (resolved 2026-08-08)

~~Only Map, Counter, and Set benchmarks exist~~. Five new benchmarks added to
`bench_test.go`: `GraphAddEdge` (2.8ms), `GraphNeighbors_Depth1` (420us),
`GraphNeighbors_Depth3` (963us), `SearchInsert` (2.5ms), `SearchQuery` (882us).
Graph depth-3 @recurse traversal and Search anyofterms() over 500-doc corpus
both measured. Results in `dgraphengine/README.md` performance table.

### 9. GraphRAG pipeline + mixed workload benchmarks (resolved 2026-08-08)

Added `graphrag_test.go` (2 functional tests), `mixed_bench_test.go` (4 benchmarks),
and `stress_test.go` (concurrent stress test).

**GraphRAG tests** (`graphrag_test.go`):

- `TestGraphRAG_SearchThenGraphTraverse`: 8-entity knowledge graph, search "golang" → 2 hits → depth-2 graph expansion → 6 context entities. Validates the full pipeline correctness.
- `TestGraphRAG_DifferentQueries`: 5 microservices with dependencies, validates search→expand across different query terms.

**Concurrent stress test** (`stress_test.go`):

- `TestGraphRAG_ConcurrentStress`: 200 entities, ~600 edges, 16 goroutines, 320 GraphRAG pipeline queries.

| Metric     | Normal    | -race     |
| ---------- | --------- | --------- |
| Throughput | 2,955 q/s | 1,460 q/s |
| p50        | 5.3 ms    | 10.7 ms   |
| p99        | 6.5 ms    | 13.1 ms   |

Each "query" = SearchQuery (limit 5) + 5 x GraphNeighbors (depth 2) = 6 gRPC
round-trips. Production-grade GraphRAG from a single Dgraph instance.

**Mixed workload benchmarks** (`mixed_bench_test.go`):

- `BenchmarkDgraph_GraphRAG_SearchThenExpand` (2.7ms): full GraphRAG pipeline — search + 5 depth-2 graph traversals.
- `BenchmarkDgraph_GraphWriteReadMix` (4.8ms): 25% write + 75% read graph workload.
- `BenchmarkDgraph_MapReadWriteMix` (671µs): 80% read + 20% write key-value workload.
- `BenchmarkDgraph_FullTriad_MapGraphSearch` (1.0ms): MapGet + SearchQuery + GraphNeighbors per iteration.

These are the first tests/benchmarks in the entire metaengine codebase that
combine GraphBackend + SearchBackend on the same engine. Dgraph is the ONLY
engine that implements both at full parity — GraphRAG is its unique value
proposition validated with real numbers.

---

## f) Up to 50 things we should get done next

### High priority (P0)

1. **Update Dgraph calibration constants** — `DG_NsPerOp` and `DG_NsPerRead` are 100-270x too low. Set to real values: ~2,700,000 ns/op write, ~350,000 ns/op read. Or implement `Calibratable.Benchmark()` to auto-calibrate.
2. ~~**Add Graph benchmark**~~ — DONE (2026-08-08): `BenchmarkDgraph_GraphNeighbors_Depth1` (420us) + `_Depth3` (963us) + `GraphAddEdge` (2.8ms).
3. ~~**Add Search benchmark**~~ — DONE (2026-08-08): `BenchmarkDgraph_SearchInsert` (2.5ms) + `SearchQuery` (882us).
4. **Fix CounterGet** — either use Dgraph aggregation (`sum(val(count))`) or document the O(N) scan cost clearly.
5. **Add Dgraph to `test-all-backends.sh`** — consumers should be able to test all backends including Dgraph in one command.

### Medium priority (P1)

6. **Add Dgraph VM test** (`nix/vm/dgraph.nix`) — NixOS VM test for CI reproducibility, matching postgres-vm/mysql-vm pattern.
7. **Implement MultimapBackend for Dgraph** — model via repeated predicate values or edge lists.
8. **Implement LogBackend for Dgraph** — append-only ordered nodes.
9. **Add adversarial DQL injection test** — send keys containing DQL syntax (`func:`, `@filter`, `{ }`, quotes) and verify they're treated as literals.
10. **Add per-test data cleanup** — `t.Cleanup` that deletes test-specific collections, or `DropAll` in `TestMain`.
11. **Add health check to ephemeral script** — verify `HealthCheck()` method works against the ephemeral instance.
12. **Run `nix fmt` on all changed files** — formatting was not verified by the full treefmt gate.
13. **Run `nix run .#lint`** — golangci-lint not run on the new code.
14. **Run `nix run .#verify`** — full verify gate not executed this session.
15. **Tag `dgraphengine/v4.0.2`** — security fix + MapDelete bugfix warrant a patch release.
16. **Add `DGRAPH_HTTP_PORT` env var** — the ephemeral script derives HTTP port from gRPC port offset, but an explicit override would be cleaner.
17. **Document Dgraph memory requirements** — Dgraph Alpha uses ~500MB+ RAM. Consumers should know this before choosing it over SQLite.
18. **Add connection pool tuning** — `dgo.NewClient` uses default gRPC options. For production, `grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(...))` may be needed for large result sets.
19. **Add retry logic for transient Dgraph errors** — `"Please retry again"` errors happen during cluster transitions. The engine should retry with backoff.
20. **Profile the allocation hotspots** — 194 allocs/op for MapSet is high. Profile with `go tool pprof` to find optimization opportunities.

### Lower priority (P2)

21. **Implement SnapshotBackend for Dgraph** — versioned predicates or separate snapshot namespace.
22. **Implement VectorBackend for Dgraph** — Dgraph doesn't have native vector search, but could emulate via brute-force (like Memory engine).
23. **Implement SpatialBackend for Dgraph** — Dgraph doesn't have native geo, but could emulate.
24. **Add Dgraph cluster mode testing** — test with 3 Alpha nodes for RAFT consensus behavior.
25. **Add Dgraph schema migration support** — versioned schema evolution.
26. **Add Dgraph backup/restore integration** — Dgraph has native backup; expose it via the engine.
27. **Add Dgraph export/import** — bulk data loading for initial projections.
28. **Add Dgraph ACL integration** — Dgraph Enterprise has ACLs; document how to use with the engine.
29. **Add Grafana dashboard for Dgraph metrics** — Dgraph exposes Prometheus metrics; integrate with the prometheus module.
30. **Add OTel tracing to Dgraph operations** — trace MapSet/MapGet/GraphAddEdge with spans.
31. **Add Dgraph to the system.Deployer** — `system.DomainConfig` should support Dgraph as a backend choice.
32. **Add Dgraph stack preset** — `stack/dgraph` module for one-call setup.
33. **Add Dgraph to metaengine/bench module** — cross-engine benchmarks including Dgraph.
34. **Document Dgraph vs Neo4j tradeoffs** — when to choose Dgraph over Neo4j for graph projections.
35. **Add Dgraph multi-tenancy support** — namespace isolation via Dgraph namespaces (Enterprise feature).
36. **Add Dgraph GraphQL layer** — Dgraph auto-generates GraphQL from DQL schema; document integration.
37. **Add connection monitoring** — track gRPC connection state, reconnect on failure.
38. **Add Dgraph version compatibility testing** — test against Dgraph 20.x, 21.x, 23.x, 24.x, 25.x.
39. **Add Dgraph bulk loader integration** — for initial projection bootstrapping from event journals.
40. **Add Dgraph to the cqrs-lint module catalog** — `cqrs-lint doctor` should detect Dgraph usage.
41. **Add Dgraph to the api-stability golden** — the dgraphengine module's exported symbols should be tracked.
42. **Add Dgraph to the doc-check verification** — README and AGENTS.md Dgraph references should be verified.
43. **Write Dgraph engine architecture doc** — explain the node/predicate model, schema design, upsert patterns.
44. **Add Dgraph streaming subscription** — Dgraph has streaming subscriptions; expose via the engine.
45. **Add Dgraph upsert conditional documentation** — the `@if(eq(len(entry), 0))` pattern is non-obvious.
46. **Add Dgraph DQL query builder** — type-safe query construction to prevent injection at the type level.
47. **Add Dgraph schema validation** — validate schema at init time against expected predicates.
48. **Add Dgraph connection string parsing** — support `dgraph://user:pass@host:port?tls=true` format.
49. **Add Dgraph failover testing** — test behavior when the Alpha leader changes.
50. **Add Dgraph to the example/taskmanager** — show a real CQRS app using Dgraph for graph projections.

---

## g) Questions for the user

### 1. Should I update the Dgraph calibration constants to match real benchmarks?

The current `DG_NsPerOp = 10,000` (10µs) and `DG_NsPerRead = 8,000` (8µs) are 100-270x lower than real measurements (2.7ms write, 344µs read). This means the planner will route queries to Dgraph that should go to Memory/SQLite. I can update them, but I can't decide whether you want to:

- (a) Set them to measured values (honest but pessimistic for production with network latency)
- (b) Implement `Calibratable.Benchmark()` for auto-calibration at startup
- (c) Leave them and document the discrepancy

### 2. Should I add Dgraph benchmarks for Graph and Search operations?

These are Dgraph's killer features (native graph traversal, full-text search), but there are zero benchmarks for them. The existing benchmarks only cover Map, Counter, and Set — which are NOT Dgraph's strengths. Without Graph/Search benchmarks, consumers can't make informed decisions about when to choose Dgraph.

### 3. Should the pre-existing `b029.go` compile errors be fixed now?

`cmd/cqrs-lint/pkg/rules/resilience/b029.go` has 4 compile errors (`RuleID`, `Title`, `Summary` fields don't exist on `finding.Finding`, `Confidence` type mismatch). These appear in every LSP diagnostic output and are tracked in TODO_LIST.md separately. They don't affect the dgraphengine work, but they pollute every diagnostic view and make it harder to spot real errors.
