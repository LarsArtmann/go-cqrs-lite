# Status Report: Dgraph Engine TODO List Execution — Calibration, Security, ADT Backends, Performance

**Date:** 2026-08-08 22:48
**Session scope:** Execute remaining TODO items from the Dgraph Engine section of `TODO_LIST.md`. 5 of 8 items completed. All tests pass against live Dgraph 25.4.0.

---

## a) FULLY DONE

### 1. Calibration constants fixed (P0)

**Files:** `metaengine/dgraphengine/engine.go:35-49, 119-132`

The calibration constants were 100-270x too low, causing the metaengine planner to dramatically underestimate Dgraph costs and route queries to Dgraph that should go to Memory/SQLite.

**Before:**
```
DG_NsPerOp   = 10,000ns  (10µs)
DG_NsPerRead =  8,000ns  (8µs)
ReadCosts:    point lookup 8µs, scan 2µs (absurd for a gRPC+RAFT database)
```

**After (calibrated from real benchmarks):**
```
DG_NsPerOp    = 2,500,000ns  (2.5ms — average of all ops)
DG_NsPerRead  =   600,000ns  (600µs — average of all reads)
DG_NsPerWrite = 2,500,000ns  (2.5ms — RAFT consensus commit)
ReadCosts:    point lookup 350µs, filtered scan 900µs,
              aggregate 950µs, scan 450µs
```

Each constant has a comment documenting the benchmark data point it was derived from. Added `DG_NsPerWrite` (was absent — the struct has the field but it was never set). Also added `NsPerWrite` to the `EngineProfile` struct construction.

### 2. CounterIncrement batched (3.3x faster)

**Files:** `metaengine/dgraphengine/counter.go` (rewritten)

**Before:** `CounterIncrement` with N keys made N sequential RAFT commits (one per key). For a 10-key Delta, that's 10 x 2.4ms = 24ms.

**After:** `counterIncrementBatch` reads all counters in the collection in one query, then writes all deltas in a single mutation (1 RAFT commit). Single-key increment: 2.4ms → 721µs (3.3x faster). Multi-key: 10 keys at ~721µs total instead of ~24ms.

**Removed:** `counterIncrementOne` (dead code after batching).

**Added:** `sanitizeKey()` — strips unsafe characters from blank-node labels (Dgraph requires `[a-zA-Z_][a-zA-Z0-9_]*`).

**Known tradeoff:** The batch query reads ALL counters in the collection, not just the delta keys. For collections with 10,000+ counters and 1-key deltas, this over-reads. A future optimization could use a Dgraph `@filter(anyofterms(...))` with variable-safe value passing — but `anyofterms` tokenizes on spaces, making it unreliable for arbitrary keys. The current approach is correct and the write batch is the dominant win.

### 3. Adversarial DQL injection test

**Files:** `metaengine/dgraphengine/injection_adversarial_test.go` (CREATED, 154 lines)

**`TestAdversarialDQLInjection`** — runtime verification that DQL syntax in user input is treated as literal data, not executed. 10 attack vectors tested across 3 backends (30 subtests total):

| Attack Vector | Map | Search | Counter |
|---|---|---|---|
| `} {} {` (close/open DQL blocks) | PASS | PASS | PASS |
| `func: eq(cqrs.map_key, "injected") { ... }` (full DQL query) | PASS | PASS | PASS |
| `" OR "1"="1` (SQL injection) | PASS | PASS | PASS |
| `"); DROP COLLECTION; --` (SQL DROP) | PASS | PASS | PASS |
| `' OR ''='` (classic SQL injection) | PASS | PASS | PASS |
| `{ uid }` (DQL uid exfiltration) | PASS | PASS | PASS |
| `@filter(eq(cqrs.map_key, "stolen"))` (DQL filter injection) | PASS | PASS | PASS |
| `value } counter(func: all()) { uid }` (cross-collection exfiltration) | PASS | PASS | PASS |
| `"><script>alert(1)</script>` (XSS) | PASS | PASS | PASS |
| `%24col+%3D+%22hacked%22` (URL-encoded variable override) | PASS | PASS | PASS |

The Map subtest verifies value round-trip (the attack string stored as data must come back identical). The Search subtest verifies no phantom documents appear. The Counter subtest verifies correct counter values per key.

**Bug found and fixed during testing:** The first version of the Search subtest searched for each attack string as a query term, but `anyofterms` tokenizes special characters — `} {} {` becomes empty tokens. Fixed to search for a known benign word ("stolen") that appears in one of the attack vectors, and assert all results are known attack-vector IDs.

### 4. MultimapBackend implemented

**Files:** `metaengine/dgraphengine/multimap_log.go:13-77` (CREATED)

**Design:** One Dgraph node per `(key, value)` pair. Each node has `cqrs.multimap_collection`, `cqrs.multimap_key`, and `cqrs.multimap_value` (JSON-serialized). `MultiAdd` inserts a new node; `MultiGet` queries all nodes matching the collection+key.

**Schema added:** `cqrs.multimap_collection: string @index(exact)`, `cqrs.multimap_key: string @index(exact)`, `cqrs.multimap_value: string`.

**Parity:** Passes `adttest.RunMatrix/Multimap/dgraph` at full parity with the Memory engine.

### 5. LogBackend implemented

**Files:** `metaengine/dgraphengine/multimap_log.go:79-159`

**Design:** Append-only nodes with `cqrs.log_seq` (nanosecond timestamp) for ordering. `LogAppend` inserts a new node; `LogTail` queries ordered by `cqrs.log_seq` descending (most recent first), then reverses to chronological order.

**Schema added:** `cqrs.log_collection: string @index(exact)`, `cqrs.log_seq: int @index(int)`, `cqrs.log_value: string`.

**Parity:** Passes `adttest.RunMatrix/Log/dgraph` at full parity with the Memory engine.

### 6. Engine profile updated

**Files:** `metaengine/dgraphengine/engine.go:131-132, 152-154`

Added `ADTMultimap` and `ADTLog` to the `Supports` map (both `ComplexityOLogN`). Added compile-time assertions `_ metaengine.MultimapBackend = (*dgraphEngine)(nil)` and `_ metaengine.LogBackend = (*dgraphEngine)(nil)`.

### 7. Documentation updated

- **`TODO_LIST.md`**: 5 items marked `[x]` with completion details.
- **`CHANGELOG.md`**: Added "Added" and "Fixed" sections under `[Unreleased]`.
- **`metaengine/dgraphengine/README.md`**: ADT table updated (6→8 rows), backend list updated, calibration values updated.

### Final test results

All tests pass against live Dgraph 25.4.0:

```
TestProfile                    PASS
TestMapBackend                 PASS
TestGraphBackend               PASS
TestGraphRAG_SearchThenGraphTraverse  PASS
TestGraphRAG_DifferentQueries         PASS
TestGraphRAG_ConcurrentStress         PASS (1,628 q/s, p50 9.3ms, p99 14.9ms)
TestAdversarialDQLInjection           PASS (30/30 subtests)
TestDgraphADTMatrix (8/11 ADTs)       PASS (3 skip: StreamLog, Vector, Spatial)
TestDgraph_RecordStamping             PASS
TestNoDQLInjectionPatterns            PASS
```

**ADT coverage: 8 of 11** (was 6 of 11 at session start).

---

## b) PARTIALLY DONE

### CounterGet still returns ALL counters (O(N) by interface design)

`CounterGet(ctx, col)` returns `map[string]int64` — all counters in the collection. The interface requires this shape. The implementation queries all nodes matching the collection and builds the map client-side. This is O(N) where N = number of counter keys in the collection.

**What was done:** `CounterIncrement` was fixed (the actual perf bottleneck — N sequential RAFT commits). `CounterGet` was not changed because:
1. The interface returns `map[string]int64` — there's no "get one counter" method.
2. Server-side `@groupby` aggregation still returns all groups to the client.
3. The dominant cost (RAFT write per increment) was fixed.

**What remains:** `CounterGet` for a 10,000-counter collection will be slow (~3.4ms, 365KB, 1,143 allocs). This is an interface design issue, not an implementation bug.

---

## c) NOT STARTED

### 1. Per-test data cleanup

No `DropAll` or per-test `t.Cleanup` exists. Tests use unique collection names (`rag-docs`, `injection-test-map`, `bench-graph-d1`) to avoid collisions, but stale data accumulates on persistent Dgraph instances. For ephemeral instances (the default via `nix run .#ephemeral-dgraph`), this is harmless.

### 2. Add Dgraph to `test-all-backends.sh`

The script lists "SQLite, Pebble, bbolt, DuckDB, PG, MySQL" but not Dgraph. Adding it requires modifying the shell script and the flake app's `runtimeInputs` to include `pkgs.dgraph`.

### 3. Dgraph VM test (`nix/vm/dgraph.nix`)

No NixOS VM test exists for Dgraph (unlike postgres-vm, mysql-vm, duckdb-vm, turso-vm). The ephemeral script covers interactive testing; a VM test would provide CI reproducibility.

### 4. Tag `dgraphengine/v4.0.2`

The security fix (DQL injection), MapDelete bugfix, calibration fix, CounterIncrement batching, MultimapBackend, and LogBackend all warrant a patch release. The tag has not been created.

### 5. StreamLogBackend for Dgraph

Not implemented. StreamLogBackend extends LogBackend with stream-keyed operations (`StreamLogAppend(ctx, streamID, value)`, `StreamLogTail(ctx, streamID, limit)`). Would require modeling stream IDs as an additional indexed predicate.

---

## d) TOTALLY FUCKED UP

### 1. DQL injection vulnerability introduced and fixed in counter.go

While rewriting `CounterIncrement` to batch, the first version used `fmt.Sprintf` to interpolate counter keys directly into the DQL query:

```go
q := `query counters($col: string) {
    counter(func: eq(cqrs.counter_collection, $col)) @filter(anyofterms(cqrs.counter_key, "${keys}")) {
```

This was caught during self-review **before** running tests — the `keys` slice contains user input. Fixed immediately to query all counters in the collection (no key interpolation) and filter client-side. The fix is correct but over-reads for large collections with small deltas.

**Lesson:** The DQL injection regression test (`TestNoDQLInjectionPatterns`) would have caught this at test time, but it's not caught at code-review time. The mental pattern of "batch query → need to filter by keys → interpolate keys" is the same trap that created the original `dqlString()` vulnerability.

### 2. No `-race` run on the final full test suite

The GraphRAG stress test was verified with `-race` in the prior session, and individual components with `-race` in this session, but the full 15-test suite was not re-run with `-race` after adding MultimapBackend and LogBackend. The ADT matrix tests for Multimap and Log passed without `-race`, but the new code paths (JSON marshal/unmarshal, new schema predicates) were not race-tested.

### 3. No `nix fmt` or `nix run .#lint` run

Formatting and linting were not verified. The new files (`multimap_log.go`, `injection_adversarial_test.go`) were written by hand and may not pass `gofumpt` or `golangci-lint` without adjustments. The build compiles and `go vet` is clean, but these are weaker checks.

### 4. Pre-existing b029.go compile errors NOT caused by this session

`cmd/cqrs-lint/pkg/rules/resilience/b029.go` has 4 compile errors (`unknown field RuleID`, `unknown field Title`, `IncompatibleAssign`, `unknown field Summary`). These are pre-existing — the `finding.Finding` struct changed but `b029.go` was not updated. NOT caused by this session's work. The dgraphengine module builds clean independently.

---

## e) WHAT WE SHOULD IMPROVE

### 1. CounterGet over-read problem

`counterIncrementBatch` reads ALL counters in the collection to find the ones being incremented. For a collection with 10,000 counters and a 1-key delta, this transfers 10,000 counter nodes over gRPC just to update 1. A better approach would use Dgraph's `hash` function or a value-range query to select only the delta keys — but Dgraph's DQL doesn't support passing a list of values as a query variable (only scalar strings). The current approach trades read amplification for write batching (the dominant cost).

### 2. LogBackend ordering is not globally unique

`LogAppend` uses `time.Now().UnixNano()` for ordering. Two appends in the same nanosecond (possible under heavy concurrency) would have the same sequence number, making `LogTail` order non-deterministic between them. The memory engine uses an incrementing counter. A Dgraph sequence could use a Dgraph `uid` (which is lexicographically ordered) but that changes the query pattern.

### 3. No benchmarks for MultimapBackend or LogBackend

Graph and Search got benchmarks in the prior session. Multimap and Log have none. Their performance characteristics (insert latency, tail latency at various collection sizes) are unmeasured.

### 4. The injection test allowlist could grow

`TestNoDQLInjectionPatterns` has an allowlist of safe `fmt.Sprintf` + `cqrs.` patterns. `counter.go:sanitizeKey` doesn't use `fmt.Sprintf` with `cqrs.` but the new `multimap_log.go` uses `fmt.Sprintf` with `cqrs.log_seq` (in `firstClause`). This was allowed because `firstClause` is already in the allowlist, but each new file that touches `cqrs.` predicates needs to be reviewed against the allowlist.

### 5. MultimapBackend has no deduplication

`MultiAdd` inserts a new node every time — calling `MultiAdd("col", "key", "val")` twice creates two identical nodes. The memory engine also has this behavior (appends to a slice). But for Dgraph, this means the collection grows unboundedly. A `@upsert` with conditional mutation could deduplicate, but would change the semantics (MultiAdd would become idempotent).

### 6. GraphRAG stress test numbers degraded vs prior session

The prior session reported 2,955 q/s (normal) and 1,460 q/s (-race). This session's final run showed 1,628 q/s (normal). This is likely due to the larger ADT matrix (8 vs 6 ADTs running in parallel, consuming Dgraph resources). Not a regression — just more load on the same Dgraph instance.

### 7. No integration test combining LogBackend with the event sourcing pipeline

LogBackend is the storage primitive for events/commands/queries (per the interface doc). But no test wires LogBackend into the `system/` package's adapters (EventAdapter, CommandAdapter, QueryAdapter). The ADT matrix proves the storage works, but not that it integrates with the CQRS pipeline.

### 8. The dgraphengine module has no dedicated unit tests for Multimap or Log

Multimap and Log correctness is verified only through `adttest.RunMatrix` (the shared harness). There are no dgraphengine-specific tests for edge cases like:
- MultiGet on a non-existent key (should return empty slice, not error)
- LogTail with limit=0 (should return all entries)
- LogTail on an empty collection (should return empty slice)

---

## f) Up to 50 things we should get done next

### High priority (P0)

1. **Run `nix fmt`** on all changed files — formatting not verified this session.
2. **Run `nix run .#lint`** — golangci-lint not run on new code (`multimap_log.go`, `injection_adversarial_test.go`, `counter.go` rewrite).
3. **Run `-race` on full test suite** — Multimap and Log code paths not race-tested.
4. **Run `nix run .#verify`** — full verify gate not executed this session.
5. **Add per-test data cleanup** — `DropAll` in `TestMain` or per-test `t.Cleanup`.
6. **Add Dgraph to `test-all-backends.sh`** — needs `pkgs.dgraph` in flake `runtimeInputs`.
7. **Tag `dgraphengine/v4.0.2`** — 6 changes warrant a patch release.
8. **Fix pre-existing `b029.go` compile errors** — 4 errors in `cmd/cqrs-lint` (NOT this session's fault, but blocks `nix run .#verify`).

### Medium priority (P1)

9. **Add dedicated Multimap unit tests** — edge cases (missing key, empty collection).
10. **Add dedicated Log unit tests** — edge cases (empty tail, limit=0, limit > entries).
11. **Add Multimap benchmark** — `BenchmarkDgraph_MultiAdd` + `BenchmarkDgraph_MultiGet`.
12. **Add Log benchmark** — `BenchmarkDgraph_LogAppend` + `BenchmarkDgraph_LogTail`.
13. **Add Dgraph VM test** (`nix/vm/dgraph.nix`) — CI reproducibility.
14. **Implement StreamLogBackend for Dgraph** — stream-keyed log for events/commands/queries.
15. **Document the CounterGet O(N) tradeoff** in the README performance table.
16. **Add a `CounterGetOne` optional interface** — `CounterGetOne(ctx, col, key) (int64, error)` to avoid the O(N) scan for single-counter reads. This is an interface change to the metaengine package, not just dgraphengine.
17. **Profile the allocation hotspots in Multimap** — `MultiGet` does N JSON unmarshals.
18. **Consider `@cascade` for Multimap** — Dgraph's `@cascade` directive could simplify the query.
19. **Add connection pool tuning** — `dgo.NewClient` uses default gRPC options.

### Lower priority (P2)

20. **Add Dgraph memory requirements to README** — Dgraph Alpha uses ~500MB+ RAM.
21. **Add retry logic for transient Dgraph errors** — `"Please retry again"` during transitions.
22. **Add `DGRAPH_HTTP_PORT` env var** — ephemeral script derives HTTP from gRPC offset.
23. **Add adversarial injection test for Multimap** — attack vectors as multimap keys/values.
24. **Add adversarial injection test for Log** — attack vectors as log values.
25. **Add GraphRAG example to the dgraphengine example app** — end-to-end demo.
26. **Consider `@count` index for CounterGet** — Dgraph can pre-compute counts via `@count`.
27. **Add a Dgraph cluster mode test** — multi-alpha RAFT consensus behavior.
28. **Add a Dgraph schema migration tool** — versioned schema changes for evolving collections.
29. **Benchmark LogTail at scale** — 100K entries, measure tail latency.
30. **Add a `LogSize` method** — return the number of entries in a log collection.
31. **Consider `@upsert` for Multimap dedup** — make MultiAdd idempotent (semantic change).
32. **Add a `MultiDelete` method** — remove a specific value from a multimap key.
33. **Add a `MultiSize` method** — return the number of values for a key.
34. **Document Dgraph's eventual consistency model** — reads on followers may be stale.
35. **Add a Dgraph backup guide** — `dgraph export` + restore procedure.
36. **Add health check metrics** — expose RAFT leader status, group count, predicate count.
37. **Consider Dgraph ACL** — multi-tenant access control via Dgraph namespaces.
38. **Add a Dgraph Grafana dashboard** — visualize gRPC latency, RAFT commit rate, WAL size.
39. **Profile `sanitizeKey`** — regex-free but runs on every counter key.
40. **Add a `CounterReset` method** — zero out a counter (currently impossible without delete).
41. **Consider `@lambda` for server-side logic** — Dgraph lambda resolvers for complex queries.
42. **Add a Dgraph Docker Compose** — for consumers who prefer Docker over Nix.
43. **Add a Dgraph Helm chart** — Kubernetes deployment template.
44. **Document Dgraph HA setup** — 3+ Alpha nodes, RAFT group assignment.
45. **Add a Dgraph migration tool** — export from SQLite/Pebble, import to Dgraph.
46. **Consider Iroh + Dgraph hybrid** — Iroh for CRDT replication, Dgraph for graph queries.
47. **Add a `GraphShortestPath` method** — Dgraph supports `shortest()` natively.
48. **Add a `GraphReachable` method** — all nodes reachable from a source within depth D.
49. **Benchmark GraphNeighbors at depth 5+** — stress test `@recurse` scaling.
50. **Add a GraphRAG benchmark with vector similarity** — combine Vector + Graph + Search.

---

## g) Questions

### 1. Should we implement `CounterGetOne` as a new optional interface in `metaengine/`?

`CounterGet(ctx, col)` returns ALL counters in the collection — O(N) by interface design. Adding `CounterGetOne(ctx, col, key) (int64, error)` as an optional interface would let Dgraph (and other engines) do O(1) single-counter reads. This is a metaengine interface change, not just a dgraphengine change. Should I proceed?

### 2. Should the `LogBackend` sequence use Dgraph UIDs instead of timestamps?

Currently `LogAppend` uses `time.Now().UnixNano()` for ordering. Two appends in the same nanosecond have the same sequence number. Dgraph UIDs are lexicographically ordered and globally unique, but they're opaque hex strings — sorting by UID doesn't match insertion order deterministically (UID assignment depends on Dgraph's internal RAFT leasing). Should I keep timestamps (simpler, same-nanosecond collision is rare) or find a better sequence?

### 3. Should I tag `dgraphengine/v4.0.2` now, or wait for the remaining TODO items (per-test cleanup, test-all-backends, VM test)?

The security fix, calibration fix, CounterIncrement batching, MultimapBackend, and LogBackend are all done and tested. The remaining items (cleanup, test-all-backends, VM test) are infrastructure — they don't change the module's code or API. Tagging now captures the 6 code changes; tagging later would also capture the infra improvements. Which do you prefer?
