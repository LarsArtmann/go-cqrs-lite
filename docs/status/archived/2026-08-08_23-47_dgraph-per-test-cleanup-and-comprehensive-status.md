# Dgraph Engine: Per-Test Cleanup, Remaining TODOs & Comprehensive Status

**Date:** 2026-08-08 23:47
**Session scope:** Started executing the 2 remaining TODO items from prior sessions + comprehensive status assessment
**Working tree:** Clean (auto-committed by git daemon)

---

## Executive Summary

The Dgraph engine for `metaengine/dgraphengine` has received **6 of 8** planned TODO items across multiple sessions. This session began executing the final 2 items — per-test data cleanup and test-all-backends integration — but was interrupted after completing only the first (TestMain + DropAll). A comprehensive audit reveals several code quality issues, unverified formatting/lint state, and 3 open design questions that need decisions.

**Bottom line:** The engine works and passes all tests against live Dgraph, but it has NOT been through `nix fmt`, `nix run .#lint`, or a full `-race` suite run since the Multimap/Log addition. These are mandatory before tagging.

---

## a) FULLY DONE (Verified)

### Across all sessions (code + tests committed):

| #  | Item                                  | Evidence                                                                                                                                                                         |
| -- | ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **Calibration constants fixed**       | `DG_NsPerOp=2.5M`, `DG_NsPerRead=600K`, `DG_NsPerWrite=2.5M` — measured from real benchmarks, documented inline in `engine.go:35-51`                                             |
| 2  | **CounterIncrement batched**          | `counter.go`: single read + single RAFT commit for N-key deltas (was N sequential commits). 2.4ms → 721µs (3.3x faster)                                                          |
| 3  | **Adversarial DQL injection test**    | `injection_adversarial_test.go`: 10 attack vectors × 3 backends (Map, Search, Counter) = 30 subtests. All pass.                                                                  |
| 4  | **MultimapBackend implemented**       | `multimap_log.go:1-105`: MultiAdd/MultiGet, passes `adttest.RunMatrix` parity vs Memory engine                                                                                   |
| 5  | **LogBackend implemented**            | `multimap_log.go:107-159`: LogAppend/LogTail, passes `adttest.RunMatrix` parity                                                                                                  |
| 6  | **GraphRAG functional tests**         | `graphrag_test.go`: 2 tests (8-entity knowledge graph, multi-query). Both pass with `-race`.                                                                                     |
| 7  | **Concurrent GraphRAG stress test**   | `stress_test.go`: 200 entities, 600 edges, 16 goroutines, 320 queries. 2,955 q/s normal, 1,460 q/s with `-race`.                                                                 |
| 8  | **Mixed workload benchmarks**         | `mixed_bench_test.go`: 4 benchmarks combining Map+Graph+Search. GraphRAG pipeline 2.7ms.                                                                                         |
| 9  | **Single-ADT benchmarks**             | `bench_test.go`: 5 new benchmarks (GraphAddEdge, GraphNeighbors d1/d3, SearchInsert, SearchQuery) + helpers                                                                      |
| 10 | **README rewritten**                  | `README.md`: leads with GraphRAG as headline, pipeline diagram, performance tables, 8-ADT coverage table                                                                         |
| 11 | **DQL injection hardened (static)**   | All queries use `QueryWithVars` with `$variable` placeholders — verified by `injection_test.go` source pattern scanner                                                           |
| 12 | **Compile-time interface assertions** | `engine.go:353-364`: 10 `var _` assertions (Engine, MapBackend, CounterBackend, ScanBackend, GraphBackend, SetBackend, SearchBackend, MultimapBackend, LogBackend, Calibratable) |
| 13 | **HealthCheck implemented**           | `engine.go:172-175`: gRPC query-based liveness probe                                                                                                                             |

### This session only:

| #  | Item                      | Status                                                                                                                                                                                 |
| -- | ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 14 | **TestMain with DropAll** | `main_test.go` created (57 lines). DropAll before+after test suite. Compiles clean. **NOT YET verified against live Dgraph** (test was started but hadn't completed when interrupted). |

---

## b) PARTIALLY DONE

| # | Item                         | What's done                                                                                                                              | What's missing                                                                                                                                                                                                               |
| - | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **Per-test data cleanup**    | `main_test.go` written, compiles, has correct logic (DropAll before + after)                                                             | Not verified against live Dgraph yet (ephemeral-dgraph test was started in background but never completed)                                                                                                                   |
| 2 | **`nix fmt`**                | Not run at all                                                                                                                           | All new files (`main_test.go`, `multimap_log.go`, `counter.go` rewrite, `injection_adversarial_test.go`, `stress_test.go`, `graphrag_test.go`, `mixed_bench_test.go`, `bench_test.go` changes) are unverified for formatting |
| 3 | **`nix run .#lint`**         | Not run on new code                                                                                                                      | golangci-lint status unknown for the 6+ new/modified files                                                                                                                                                                   |
| 4 | **Full `-race` test suite**  | Individual components were race-tested in prior sessions                                                                                 | Full suite run after Multimap/Log addition never happened — JSON marshal/unmarshal paths in new backends are untested under `-race` in a full suite context                                                                  |
| 5 | **b029.go "compile errors"** | Investigated — confirmed as **phantom gopls cache** (file is 53 lines, errors reference line 91+). `go build` and `go vet` pass cleanly. | gopls restarted but may re-report. The actual code is fine.                                                                                                                                                                  |

---

## c) NOT STARTED

| # | Item                                     | Effort | Notes                                                                                                                             |
| - | ---------------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **Add Dgraph to `test-all-backends.sh`** | S      | Script lists SQLite/Pebble/bbolt/DuckDB/PG/MySQL. Needs `pkgs.dgraph` in flake `runtimeInputs` + a Phase 4 section in the script. |
| 2 | **Dgraph VM test** (`nix/vm/dgraph.nix`) | M      | NixOS VM test for CI reproducibility. Pattern: postgres-vm/mysql-vm/duckdb-vm. Dgraph needs Zero + Alpha processes.               |
| 3 | **Tag `dgraphengine/v4.0.2`**            | S      | Blocked on: (a) verify gate pass, (b) answering the 3 design questions.                                                           |
| 4 | **Dgraph retry logic**                   | S      | Transient `"Please retry again"` RAFT errors need exponential backoff.                                                            |
| 5 | **Dgraph connection pool tuning**        | S      | gRPC `MaxCallRecvMsgSize` for large result sets.                                                                                  |
| 6 | **CounterGetOne optional interface**     | M      | New optional interface in `metaengine/` package — affects all engines. Blocked on design decision.                                |
| 7 | **LogBackend sequence improvement**      | S      | Currently uses `time.Now().UnixNano()` — same-nanosecond collision possible. Blocked on design decision.                          |

---

## d) TOTALLY FUCKED UP / Concerning

### 1. LogTail uses `fmt.Sprintf` for query construction

**File:** `multimap_log.go:127-131`
**Code:**

```go
q := fmt.Sprintf(`query log($col: string) {
    entries(func: eq(cqrs.log_collection, $col), orderdesc: cqrs.log_seq%s) {
        cqrs.log_value
    }
}`, firstClause)
```

While `firstClause` is derived from an `int` (not user input), this violates the **pattern invariant** that ALL Dgraph queries use `QueryWithVars` with `$variable` placeholders — never `fmt.Sprintf` with query text. The `injection_test.go` static pattern scanner checks for `fmt.Sprintf` in query construction and may or may not catch this case (it's interpolating a `first: N` clause, not a user value, but the pattern is wrong).

**Severity:** Low risk (int input), high smell (pattern violation). Should be refactored to a hardcoded query or use a variable for the limit.

### 2. CounterIncrement over-reads entire collection

**File:** `counter.go:55-60`

`counterIncrementBatch` queries ALL counters in a collection, even when the Delta has only 1-2 keys. For a collection with 10,000 counters, this reads 10,000 nodes to update 2. The tradeoff was explicitly chosen to avoid DQL injection from key interpolation, but it creates a performance cliff at scale.

**Mitigation:** The batched approach is still 3.3x faster than N sequential commits for small deltas. But for `len(deltas) << len(collection)`, a per-key query approach (via `QueryWithVars` with a `$keys` variable) would be better.

### 3. LogBackend same-nanosecond collision

**File:** `multimap_log.go:138`

`time.Now().UnixNano()` can produce the same value for two concurrent `LogAppend` calls. Under the stress test (16 goroutines), entries with identical timestamps would have nondeterministic ordering. This is a correctness issue for append-only logs where order matters.

### 4. Test coverage gaps in the new backends

`MultimapBackend` and `LogBackend` are only tested via `adttest.RunMatrix` (the generic ADT matrix). There are NO dedicated unit tests for edge cases:

- MultiAdd with the same key twice (deduplication behavior?)
- MultiGet on a nonexistent key
- LogTail with limit=0 (returns all?)
- LogTail with limit > entries (returns what exists?)
- LogTail on empty collection

### 5. No per-test isolation — DropAll is suite-level only

The `main_test.go` I created does DropAll before and after the entire suite. But if tests run in parallel (they do — every test calls `t.Parallel()`), test data from one test can interfere with another. For example, `TestAdversarialDQLInjection` inserts into `"injection-test-map"` while `TestDgraphADTMatrix` creates its own collections. Currently they use different collection names, but there's no guarantee this will stay that way. Per-test cleanup (via `t.Cleanup`) would be safer.

### 6. The ephemeral-dgraph background test never completed

I started `nix run .#ephemeral-dgraph -- go test ...` in the background (shell 05F) and it was still running with no output when the status report was requested. This could mean:

- Dgraph is slow to start
- The test is hanging
- There's a deadlock

The test was never verified to complete.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **Add a `DropAll` method to the engine** — expose a test helper so consumers can reset their Dgraph instances without importing `dgo` directly. The `main_test.go` currently duplicates the Dgraph client creation logic.

2. **Consider a Dgraph-specific `FlushSchema` method** — `engine.init()` applies schema on every `New()` call. With DropAll in TestMain, every test's `New()` re-applies schema. This is correct but wasteful. A `SchemaApplied` flag on the engine struct (reset by DropAll) would optimize this.

3. **Consistent error wrapping** — the engine uses `fmt.Errorf("dgraphengine.X: %w", err)` consistently, but `multimap_log.go` and `counter.go` don't always include the operation context (which key/collection failed). Adding `"for collection %q key %q"` context would improve debuggability.

4. **LogBackend should use a Dgraph-assigned sequence** — instead of `time.Now().UnixNano()`, Dgraph can assign a monotonically increasing ID via a counter node. This eliminates the same-nanosecond collision and makes ordering deterministic.

5. **CounterIncrement should query only the needed keys** — use `@filter(eq(cqrs.counter_key, $key1)) OR eq(cqrs.counter_key, $key2)` or a regex/in approach. This avoids the full-collection scan for small deltas.

### Testing

6. **Add dedicated unit tests for MultimapBackend** — edge cases beyond the ADT matrix: empty key, nil value, duplicate add, large values.

7. **Add dedicated unit tests for LogBackend** — edge cases: empty collection, limit=0, limit>entries, concurrent append ordering.

8. **Add a test that verifies DropAll actually clears data** — `TestMain` calls DropAll but no test asserts that the database is empty afterward. A simple `TestDropAllClearsData` would catch DropAll failures.

9. **Run the full verify gate** — `nix run .#verify` has never been run since the Multimap/Log addition. This is the ONLY source of truth for build/lint/test/race status.

10. **Add benchmarks for Multimap and Log backends** — the new backends have zero benchmark coverage. MultiAdd, MultiGet, LogAppend, LogTail should be benchmarked to validate the OLogN complexity claims.

### Code Quality

11. **Run `nix fmt` on all new files** — formatting is completely unverified. The `goexperiment.jsonv2` build tag is set but `gofumpt`/`goimports` haven't been applied.

12. **Run `golangci-lint` on dgraphengine** — the module has never been linted. Potential issues: error strings, naming, unused code.

13. **The `sanitizeKey` function in `counter.go`** duplicates logic that could be shared with `sanitizePredicate` in `engine.go:302-321`. They do slightly different things (blank-node labels vs predicate names) but the character filtering is identical.

14. **The `appliedSchemas` map in `engine.go:333`** is not protected against concurrent access from the schemaMu. Wait — it IS protected (schemaMu.Lock at line 330). But the `init()` method at line 95 applies schema without the lock. This is fine because init() runs before the engine is returned to the caller, but it's worth documenting.

### Operations

15. **Add Dgraph to `test-all-backends.sh`** — Dgraph is the only backend NOT in the cross-backend test suite. This is a CI gap.

16. **Add a Dgraph VM test** — `nix/vm/dgraph.nix` would catch regressions in CI without requiring Docker.

17. **Document the Dgraph version requirement** — tests use Dgraph 25.4.0 from nixpkgs. The DQL features used (`@filter`, `@recurse`, `DropAll`, `Cond` in mutations) may not work on older versions. The README should state the minimum version.

18. **Add retry logic for transient RAFT errors** — Dgraph returns `"Please retry again"` when the RAFT group is mid-election. The engine should retry with backoff.

### Security

19. **LogTail `fmt.Sprintf` should be eliminated** — even though the input is an int, the pattern is wrong. Use a hardcoded query for limit>0 and a separate query for unlimited.

20. **Add a fuzz test for DQL injection** — the adversarial test covers 10 vectors but a fuzz test would cover the space more thoroughly. Go 1.26 supports `testing.F`.

---

## f) Up to 50 Things to Get Done Next

### Immediate (blocks tagging v4.0.2)

1. **Verify `main_test.go` works against live Dgraph** — run `nix run .#ephemeral-dgraph -- go test -v -count=1 ./metaengine/dgraphengine/...`
2. **Run `nix fmt`** on all dgraphengine files
3. **Run `golangci-lint`** on dgraphengine (`cd metaengine/dgraphengine && GOWORK=off go run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...`)
4. **Run full `-race` test suite** against live Dgraph
5. **Fix LogTail `fmt.Sprintf`** — refactor to avoid query string interpolation
6. **Add dedicated MultimapBackend unit tests** (edge cases)
7. **Add dedicated LogBackend unit tests** (edge cases)
8. **Verify the full `nix run .#verify` gate** passes (or at minimum `nix run .#verify-fast`)

### Short-term (this week)

9. **Add Dgraph to `test-all-backends.sh`** — Phase 4 section + `pkgs.dgraph` in flake `runtimeInputs`
10. **Add Multimap + Log benchmarks** — validate OLogN claims
11. **Add retry logic for transient RAFT errors** — exponential backoff on `"Please retry again"`
12. **Improve CounterIncrement** — query only needed keys instead of full collection scan
13. **Fix LogBackend sequence** — use Dgraph counter node instead of timestamp
14. **Run `cmd/api-stability` golden regen** if any exported symbols changed
15. **Tag `dgraphengine/v4.0.2`** after all above pass

### Medium-term

16. **Add Dgraph VM test** (`nix/vm/dgraph.nix`) — NixOS VM with Zero + Alpha
17. **Add `CounterGetOne` optional interface** in `metaengine/` — O(1) single-counter reads
18. **Add gRPC connection pool tuning** — `MaxCallRecvMsgSize` for large result sets
19. **Add fuzz test for DQL injection** — `testing.F` with random key/value inputs
20. **Document minimum Dgraph version** in README and go.mod comments
21. **Add `DropAll` test helper** to the engine package (exported for consumers)
22. **Add a test that verifies schema re-application after DropAll** — ensures `init()` runs correctly on every `New()` call post-cleanup
23. **Benchmark GraphNeighbors at depth 5+** — stress test `@recurse` scaling
24. **Add GraphRAG + Vector similarity benchmark** — combine all three backends
25. **Extract shared character-sanitization logic** — `sanitizeKey` + `sanitizePredicate` share the same filter loop
26. **Add error context (collection/key) to multimap_log.go error wrapping**
27. **Consider a `HealthCheck` that verifies schema is applied** — not just gRPC connectivity
28. **Add a benchmark for CounterGet** — currently unmeasured at 3.4ms with 1,143 allocs (highest allocation in the suite)
29. **Optimize CounterGet allocations** — 1,143 allocs for a point-query-like operation is excessive. JSON unmarshal into a struct slice creates garbage.
30. **Add a `DropCollection(col)` method** — per-collection cleanup instead of nuclear DropAll. Enables parallel tests that each own their collection.

### Longer-term (ROADMAP candidates)

31. **Dgraph ACL/TLS support** — `New()` currently hardcodes `insecure.NewCredentials()`. Production deployments need TLS.
32. **Dgraph cluster mode testing** — all tests use single-node. Multi-Alpha cluster behavior (RAFT, rebalancing) is untested.
33. **Dgraph bulk loader integration** — for initial data ingestion in large GraphRAG deployments.
34. **Metaengine planner integration** — connect Dgraph to the cost-based planner with real calibration data from `CalibrateEngine`.
35. **Streaming GraphRAG** — `GraphNeighbors` returns a slice. For very large graphs, an iterator/channel API would avoid OOM.
36. **Dgraph backup/restore** — operational story for persistent deployments.
37. **Connection metrics** — expose gRPC connection stats (reconnects, latency) via the `HealthChecker` interface.
38. **Add Dgraph to `stack/` presets** — `stack/dgraph` one-call bundle (like `stack/postgres`, `stack/duckdb`).
39. **Multi-tenant Dgraph** — namespace isolation via `dgraph.type` prefixes or separate Dgraph namespaces.
40. **Dgraph GraphQL layer** — Dgraph has a native GraphQL API. Consider a `transport/graphql` adapter for direct frontend integration.
41. **Add a `Calibrate` test** — run `metaengine.CalibrateEngine` against Dgraph and verify the calibration constants match.
42. **Port the QUIC transport pattern** — Dgraph's gRPC transport could optionally use QUIC for lower latency.
43. **Add a Dgraph-specific `ExplainPlan`** — show the DQL query that will be generated for a given metaengine query.
44. **Integrate with `projectionadapter`** — Dgraph as a projection backend for event-sourced systems.
45. **Add OpenTelemetry tracing** — span per Dgraph query, attribute the DQL string.
46. **Add Prometheus metrics** — query count, latency histogram, error rate per backend.
47. **Add a `Doctor()` diagnostic** — Dgraph-specific health check (RAFT status, disk usage, memory).
48. **Cross-engine GraphRAG comparison benchmark** — Memory vs Dgraph for the same GraphRAG workload.
49. **Add a GraphRAG example** in `example/` — end-to-end pipeline with a real knowledge graph.
50. **Evaluate Dgraph v25's vector search** — Dgraph added vector similarity search. If it works, Dgraph could implement `VectorBackend` too (making it the ONLY engine with all 4 GraphRAG-relevant backends).

---

## g) Questions (3 — Cannot Answer Myself)

### 1. Should the `TestMain` DropAll strategy be suite-level or per-test?

I implemented suite-level DropAll (before + after all tests). But all tests call `t.Parallel()`, meaning they share the same Dgraph instance concurrently. Currently they use different collection names, but this is fragile. Two options:

- **A) Suite-level DropAll (current):** Fast (one DropAll), but no isolation between tests. If two tests accidentally use the same collection, they interfere.
- **B) Per-test DropAll via `t.Cleanup`:** Each test drops its collections after running. Slower (N DropAlls), but full isolation.

I lean toward B for correctness, but it requires tracking which collections each test created (or just nuking everything per test, which is slow). Which approach do you prefer?

### 2. Should I tag `dgraphengine/v4.0.2` now or after the remaining infrastructure items?

The code changes (calibration, batching, Multimap, Log, injection test) are all done and individually tested. The remaining items (test-all-backends, VM test, retry logic) are infrastructure that doesn't change the module's API or code. Tagging now captures the 6 code changes. But `nix fmt` + `nix run .#lint` haven't been run — if they surface issues, there'd need to be a v4.0.3. Should I tag after fmt+lint, or after everything?

### 3. The LogBackend `LogTail` uses `fmt.Sprintf` to inject the `first: N` limit clause into the DQL query. The input is an `int` (not user-controlled), so it's not a DQL injection risk. But it violates the "all queries use QueryWithVars" pattern. Should I:

- **A)** Refactor to two hardcoded queries (one with limit, one without) — clean but duplicates the query body
- **B)** Use a `$limit` variable in the DQL — Dgraph doesn't support variables in `first:` clauses, so this won't work
- **C)** Leave it as-is with a `//nolint` comment — the input is a validated int, the risk is zero
- **D)** Use `fmt.Sprintf` but add a test that verifies no string input can reach this code path

I lean toward A (two queries) for pattern consistency, but it's a judgment call.
