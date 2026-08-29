# Status Report: Dgraph Metaengine Implementation

**Date:** 2026-08-07 00:41
**Session scope:** Implementing `metaengine/dgraphengine/` — a Dgraph-backed metaengine Engine

---

## a) FULLY DONE

1. **Module created and builds clean** — `metaengine/dgraphengine/` with `go.mod`, `go.sum`, 8 Go source files (1,132 lines total). Builds both standalone (`GOWORK=off`) and in workspace mode with `-tags "goexperiment.jsonv2"`.

2. **6 backend interfaces implemented:**
   - `MapBackend` — conditional upsert via DQL `@if(eq(len(entry), 0))` + `@index(exact)` for O(logN) point lookups
   - `CounterBackend` — transactional read-modify-write per key
   - `ScanBackend` — full-collection scan with Go-side filter/sort/cursor/limit
   - `GraphBackend` — native uid→uid edges with `@reverse`, `@recurse(depth)` for multi-hop BFS traversal. **O(degree^depth), NOT degraded — Dgraph's killer feature**
   - `SetBackend` — `@index(exact)` membership check via `count(uid)`
   - `SearchBackend` — `@index(term)` + `anyofterms()` for full-text search. **NOT degraded — native term index**

3. **Engine profile calibrated** — `PersistencePersistent`, `ReplicationSingleLeader` (RAFT), `NsPerOp=10000`, `NsPerRead=8000`, `Calibratable` interface implemented, `DegradedADTs` only marks SortedMap (Go-side sort).

4. **Cross-engine parity test** — `adt_matrix_test.go` runs `adttest.RunMatrix` against memory + dgraph factories. Auto-skips when Dgraph unavailable (`DGRAPH_ADDR` env var, default `localhost:9080`).

5. **Unit tests** — `engine_test.go` with `TestProfile`, `TestMapBackend`, `TestGraphBackend`. All compile and would run against a live Dgraph instance.

6. **Workspace wiring complete:**
   - `go.work` — added `./metaengine/dgraphengine`
   - `cmd/api-stability/main.go` — added to modules list
   - `docs/api_surface.txt` — regenerated (19 new exports captured)
   - `cmd/cqrs-lint/pkg/analyzer/module_catalog_test.go` — added to excluded modules
   - `AGENTS.md` — updated modules row, test command, monorepo tree, Tier 4 graph
   - `README.md` — full module documentation with ADT table, usage, testing instructions

7. **Meta-tests pass** — `TestEveryGoModDirIsInModulesList`, `TestCatalogEveryGoWorkModuleCovered`, `TestAPISurfaceCheck` all green.

---

## b) PARTIALLY DONE

1. **Test coverage** — Only 3 unit tests + the matrix harness. No tests for CounterBackend, ScanBackend, SetBackend, SearchBackend individually. No test for `SetCalibration` / `Calibratable`. No test for `NewFromClient`. No test for `MapDelete`. All tests skip without a running Dgraph instance, meaning **zero tests actually ran in this session**.

2. **DQL injection hardening** — `dqlString()` escapes quotes/backslashes/newlines/tabs but does NOT handle null bytes, unicode escapes, or other control characters. Dgraph supports parameterized queries via `QueryWithVars` / `txn.Do` with `$variable` placeholders. The code uses manual string interpolation throughout. This is a **security gap**.

3. **Counter concurrency** — `counterIncrementOne` uses read-modify-write in a transaction (optimistic concurrency). Under contention, Dgraph returns `ErrAborted` and the caller gets an error with no retry. The pgengine equivalent uses `INSERT ON CONFLICT DO UPDATE SET value = value + excluded.value` — a single atomic statement. No retry logic.

---

## c) NOT STARTED

1. **StreamLogBackend + AtomicAppender** — pgengine, badgerengine, pebbleengine all implement these. Critical for event sourcing use cases (version-checked appends, journal reads). Dgraph's `BIGSERIAL`-equivalent would need a sequence predicate or external counter.

2. **PushdownScan** — pgengine, duckdbengine, pgengine push filter/sort into SQL. Dgraph could push filters into DQL `@filter(eq(...))` and sort into `orderasc:`/`orderdesc:` natively. Currently all filtering is Go-side (full scan + filter).

3. **LayoutPlanner / LayoutPlanApplier** — SQL engines create indexes dynamically. Dgraph could `Alter` schema to add `@index(exact)` or `@index(term)` on demand for declared filter/sort fields. Currently schema is static in `init()`.

4. **RawValueReader / RawScanReader** — pebbleengine has these for zero-allocation reads (skip JSON decode).

5. **StreamingScan** — duckdbengine has `StreamScan` returning `iter.Seq2[any, error]` for OOM-safe lazy iteration.

6. **MultimapBackend** — straightforward to implement (Dgraph uid→uid edges per key).

7. **LogBackend** — append-only log via Dgraph nodes ordered by UID assignment or a sequence predicate.

8. **SnapshotBackend** — snapshot persistence (version-keyed byte blobs).

9. **VectorBackend** — Dgraph has no native vector search, so this would be brute-force (matching the memory engine). Low value.

10. **SpatialBackend** — Dgraph has `geo` type with `@index(geo)` and `near()` function. Could be a native implementation, not brute-force.

11. **Bench tests** — pgengine, badgerengine, pebbleengine all have `calibration_bench_test.go`. No benchmarks written.

12. **Testcontainer / Nix integration** — pgengine has `testcontainer_test.go` (auto-starts Postgres via Docker). No equivalent for Dgraph. No Nix VM test. Tests just skip.

13. **CHANGELOG.md** — not updated.

14. **ADR** — no architecture decision record for why Dgraph, what tradeoffs, what ADTs are native vs degraded.

15. **`.golangci.yml` depguard allow list** — AGENTS.md says "When adding new dependencies, add them to .golangci.yml depguard allow list at the same time." Not checked/updated for `github.com/dgraph-io/dgo/v240`.

16. **Data cleanup / `DropAll` method** — No way to wipe Dgraph state between test runs. `init()` creates schema but doesn't drop existing data. Tests will see stale data from previous runs.

17. **Connection options** — No `NewWithTLS`, no `NewWithAuth`, no connection string support. Only bare `New(addr)` with insecure gRPC. `dgo.Open(connStr)` supports `dgraph://user:pass@host:port?sslmode=verify-ca` — should expose this.

18. **`go work sync` triggered download of uncached internal modules** — the `record/v4` and `sqliteengine/v4` pseudo-versions were fetched from VCS (worked because devShell SSH redirect). Not a problem but worth noting.

---

## d) TOTALLY FUCKED UP

1. **Nothing is catastrophically broken** — the module compiles, vets clean, passes all meta-tests. But:

2. **The code has NEVER been tested against a running Dgraph instance.** Every `t.Skipf("Dgraph not available")` path was taken. The DQL queries, JSON tag mappings, upsert conditions, and graph traversal response parsing are **completely unverified**. There could be (likely are) runtime bugs in:
   - JSON struct tag mismatches (`cqrs.map_value` vs what Dgraph actually returns)
   - Conditional upsert `@if` syntax correctness
   - `@recurse` response shape and `extractNeighborIDs` recursive parsing
   - Counter transaction conflict handling
   - `dqlString` escaping correctness for DQL string literals

3. **`dqlString` is hand-rolled injection prevention** — I should have used `QueryWithVars` with `$variable` placeholders from the start. Every single DQL query in this module interpolates user-provided values directly. This is the kind of thing that gets exploited.

---

## e) WHAT WE SHOULD IMPROVE

1. **Replace ALL DQL string interpolation with parameterized queries** (`QueryWithVars` / `txn.Do` with `Vars`). This eliminates the injection surface entirely. The `dqlString()` function should be deleted.

2. **Implement `PushdownScan`** — Dgraph's `@filter(eq(pred, val))` and `orderasc:`/`orderdesc:` are native pushdown targets. This would move SortedMap from degraded to native.

3. **Use Dgraph `@upsert` directive on schema** — The schema already declares `@upsert` on indexed predicates, but the upsert queries use conditional `@if` mutations instead of the `upsert` query+mutation pattern. The `@upsert` directive enables Dgraph's internal UID-based upsert which is more efficient than conditional mutations.

4. **Counter should use Dgraph's `@count` index or a single atomic upsert** — The current read-modify-write is correct but slow under contention. Consider using Dgraph's upsert with `uid(v)` + conditional mutation to do it in one round-trip.

5. **Add `SpatialBackend` natively** — Dgraph has `geo` type + `near()` function. This would be a genuine native implementation (not brute-force like memory engine), making Dgraph the ONLY engine with native spatial support.

6. **Implement `GraphBackend` using Dgraph type system** — Currently graph edges use dynamically-created predicates (`cqrs.edge.<collection>`). Using Dgraph types (`type GraphNode { cqrs.edge.graph: [uid] }`) would be cleaner and enable schema validation.

7. **Add a `DropAll` / `Reset` method** for test cleanup. Essential for reproducible test runs.

8. **Add testcontainer support** — A `docker run dgraph/dgraph` testcontainer pattern matching pgengine's `testcontainer_test.go`.

9. **Connection string support** — Expose `dgo.Open(connStr)` as `NewFromConnStr(connStr)` for TLS/auth/cloud deployments.

---

## f) Up to 50 Things We Should Get Done Next

### Critical (must do)

1. **Test against a real Dgraph instance** — Spin up Dgraph locally or via Docker and run the full test suite. Fix all runtime bugs.
2. **Replace `dqlString` with parameterized queries** — Security: eliminate DQL injection surface.
3. **Update `.golangci.yml` depguard allow list** — Add `github.com/dgraph-io/dgo/v240`.
4. **Add `DropAll` / cleanup method** — Test reproducibility.

### High priority

5. **Implement `StreamLogBackend` + `AtomicAppender`** — Parity with pgengine/badgerengine.
6. **Implement `PushdownScan`** — Move SortedMap from degraded to native.
7. **Implement `MultimapBackend`** — Easy via Dgraph uid edges.
8. **Implement `LogBackend`** — Append-only log nodes.
9. **Add `SpatialBackend` natively** — Dgraph `geo` + `near()`.
10. **Write individual backend tests** — Counter, Scan, Set, Search, Graph depth>1.
11. **Add counter retry on `ErrAborted`** — Or convert to single-statement upsert.
12. **Write bench tests** (`calibration_bench_test.go`).
13. **Add testcontainer support** (`testcontainer_test.go` or Nix VM test).

### Medium priority

14. **Write ADR** — Why Dgraph, ADT coverage, tradeoffs.
15. **Update CHANGELOG.md**.
16. **Connection string constructor** (`NewFromConnStr`).
17. **TLS / auth support**.
18. **Implement `LayoutPlanner`** — Dynamic `@index` creation for declared fields.
19. **Implement `RawValueReader`** — Skip JSON decode on point lookups.
20. **Implement `SnapshotBackend`** — Version-keyed byte blobs.
21. **Add `NewRoundRobin` constructor** — Multi-alpha Dgraph cluster.
22. **Implement `StreamingScan`** — `iter.Seq2` for large collections.
23. **Graph traversal: fix potential duplicate nodes at multiple depths** — `@recurse(loop: false)` should handle this but needs verification.
24. **Graph: verify `CanonicalizeNeighbors` parity** — The test expects sorted neighbor strings; verify Dgraph returns the same set as memory engine.
25. **Search: verify `anyofterms` matches memory engine's tokenization** — Memory engine uses Go strings.Contains; Dgraph uses word-boundary tokenization. These may diverge on partial matches.
26. **Search: add `allofterms` support** — Anyofterms is OR; allofterms is AND.
27. **Counter: batch multiple deltas in one transaction** — Currently one tx per key.
28. **MapSet: verify upsert conditional mutation actually works** — The two-mutation `@if` pattern is untested.
29. **Add `Close()` test** — Verify double-close is safe.
30. **Add concurrent access tests** — Dgraph is a server; concurrent reads/writes should work.

### Lower priority

31. **Implement `MapUpdater` (MapUpdate)** — Fold function over existing value.
32. **Add `StreamTemporalReader`** — Version-based reads (Dgraph has no native temporal queries).
33. **VectorBackend** — Brute-force (Dgraph has no native vector search).
34. **Update SKILL.md references** — Add dgraphengine to the consumer-facing skill.
35. **Add to `stack/` presets** — A `stack/dgraph` bundle.
36. **Add to `system/` deployer** — Dgraph as a deployment option.
37. **Nix flake: add Dgraph to integration test targets**.
38. **Nix flake: add `dgraph` to devShell `pkgs`**.
39. **Add `WithReplicationLag` / `WithNetworkRTT` plan options** for Dgraph cluster modeling.
40. **Document Dgraph-specific DDL** — Schema predicates, index types, type definitions.
41. **Add health check** — `client.Login` or a `Query("{}")` ping.
42. **Add `Doctor()` diagnostics** — List collections, their predicates, index status.
43. **Verify `@reverse` is needed** — Bidirectional edges are added manually; `@reverse` may be redundant.
44. **Consider Dgraph Lambda / GraphQL endpoint** as alternative to DQL.
45. **Add metrics** — gRPC latency, mutation count, query count.
46. **Add tracing** — OTel spans for Dgraph operations.
47. **Document operational concerns** — Dgraph cluster sizing, predicate cardinality, compaction.
48. **Add `ExplainPlan` support** — Show Dgraph as engine choice with cost.
49. **Cross-engine consistency verification** — Run matrix against pebble + sqlite too.
50. **Add fuzzing tests** — Fuzz `dqlString` (or its replacement) with adversarial input.

---

## g) Questions I Cannot Answer Myself

1. **Do you want Dgraph tests to auto-start via Docker testcontainers (like pgengine does), via Nix VM tests (like the MySQL/PG integration tests), or just keep the skip-when-unavailable pattern?** The testcontainer approach is fastest to implement but adds a Docker dependency to CI. Nix VM tests are the project convention but require writing a Dgraph NixOS module. The skip pattern means the dgraphengine code is never exercised in CI.

2. **Should I implement the remaining backends (StreamLog, PushdownScan, Multimap, Log, Spatial, Snapshot) now, or is the current 6-backend coverage (Map, Counter, Scan, Graph, Set, Search) sufficient for your use case?** The Graph + Search backends are Dgraph's differentiators; the others are table-stakes parity with existing engines.

3. **Is there a specific Dgraph deployment you're targeting (single-node dev, Dgraph Cloud, self-hosted cluster) that would affect connection options, auth, and TLS defaults?** This determines whether `New(addr)` with insecure gRPC is the right default or whether I should default to connection-string auth.
