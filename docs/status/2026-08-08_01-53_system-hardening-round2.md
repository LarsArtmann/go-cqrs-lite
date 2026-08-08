# Status Report: System Module Hardening — Round 2

**Date:** 2026-08-08 01:53
**Session scope:** Execute the follow-up items from the prior session's status report (`docs/status/2026-08-08_01-25_system-p2-test-depth-p3-code-quality.md`). The prior session shipped 11 tasks (5 P2 + 6 P3) but left 3 open questions, 10 unstarted improvement items, and several architectural gaps. This session addressed those.

---

## A) FULLY DONE

### Code Quality (5 items)

1. **Extracted shutdown logic to `system/shutdown.go`** — `Drainer` interface, `shutdownEdge` type, `orderedEngines()` method (Kahn's algorithm), and `RegisterDrainer()` method moved from `system.go` to a new `shutdown.go` file. `system.go` was at the 350-line CI limit; now 268 lines. The `appended` dead map in the cycle-fallback path was also removed (it was computed but never read).

2. **Removed `ProjectionHostResource` constant** — The constant was declared and documented as usable in `ShutdownDependency` edges, but `orderedEngines()` only looked up engine names in `nameToIdx` — the projection host was never in that map. Edges referencing `"projection-host"` were silently ignored. The constant was a lie. Removed it, updated doc comments on `ShutdownDependency` and `shutdownEdge` to state "projection host always closes first and cannot participate in dependency edges." Updated `config_types.go` doc comments and README DomainConfig Fields table.

3. **Replaced parallel `engines`/`engineNames` slices with `namedEngine` struct** — The `System` struct had `engines []metaengine.Engine` and `engineNames []string` as parallel slices. Any code appending to one without the other caused silent misalignment. Replaced with `engines []namedEngine` where `namedEngine{engine metaengine.Engine, name string}` pairs them. Added `engineSlice()` helper for callers that need `[]metaengine.Engine` (HealthCheck, Explain, Serialize, Verify). Updated `constructor.go` (3 append sites), `introspection.go` (3 use sites), `shutdown.go` (all references), and `system_internal_test.go` (mock injection).

4. **Added `HealthCheck` to DuckDB engine** — `metaengine/duckdbengine/engine.go`. Uses `db.PingContext(ctx)` (DuckDB uses `*sql.DB`). Implements `metaengine.HealthChecker`.

5. **Added `HealthCheck` to Postgres engine** — `metaengine/pgengine/engine.go`. Uses `db.PingContext(ctx)` (Postgres uses `*sql.DB` via pgx). Implements `metaengine.HealthChecker`.

6. **Added `HealthCheck` to Pebble engine** — `metaengine/pebbleengine/engine.go`. Pebble's `*pebble.DB` does not have `PingContext` (it's an embedded LSM, not a `database/sql` connection). HealthCheck does a lightweight point-read of a non-existent key (`__health_check__`). If the DB returns `pebble.ErrNotFound`, it's healthy. Any other error (e.g., "database closed") indicates a problem. Implements `metaengine.HealthChecker`.

### Tests (10 new tests)

7. **`TestOrderedEngines_NoDeps`** — 3 engines, no shutdown deps. Verifies creation order is preserved.

8. **`TestOrderedEngines_BasicOrdering`** — 2 engines, edge `{before: "b", after: "a"}`. Verifies b closes before a.

9. **`TestOrderedEngines_CycleFallback`** — 3 engines, a↔b cycle, c not in cycle. Verifies c is processed first (inDegree 0), then a and b fall back to creation order.

10. **`TestOrderedEngines_UnknownNames`** — Edges referencing nonexistent engine names are silently ignored. Verifies creation order preserved.

11. **`TestSystem_Close_ErrorJoining`** — 2 engines both return errors from Close. Verifies `errors.Join` wraps both (not just the first). Also verifies double-close is a no-op.

12. **`TestSystem_Close_OrderMatchesOrderedEngines`** — 3 mock engines recording close order into a shared slice. Shutdown deps: c before a, c before b. Verifies c closes first, then a and b in creation order.

13. **`TestSystem_RegisterDrainer_CalledBeforeClose`** — Registers a mock drainer, calls `GracefulClose`, verifies `Drain()` was called.

14. **`TestSystem_RegisterDrainer_ErrorPropagation`** — Registers a mock drainer returning an error, verifies `GracefulClose` propagates it.

15. **`TestSystem_ResetProjection_RestartAndReplay`** — Two-phase test with SQLite persistence: (1) produce event, start, process, verify projection data, stop, reset, close; (2) new System with same SQLite DSN, start, verify replay from zero checkpoint, verify projection data matches. Uses `metaengine.ExecuteTyped` for typed result extraction (SQLite returns `JSONValue`, not `TaskView` directly).

### Documentation (4 files updated)

16. **`system/README.md`** — Updated ShutdownDependencies description in DomainConfig Fields table to reflect projection host always closes first.

17. **`CHANGELOG.md`** — Updated System package section: removed `ProjectionHostResource` mention, added all 4 engine HealthChecks, expanded test count from 5 to 7+ new tests, updated API surface note (3 system symbols, 4 engine symbols).

18. **`FEATURES.md`** — Updated Shutdown dependencies row (removed ProjectionHostResource, added "projection host always closes first"), updated test count 33→43+, updated status line to mention all 4 engine HealthChecks.

19. **`TODO_LIST.md`** — Updated P3 section: added HealthCheck on all engines, shutdown.go extraction, namedEngine struct, all new tests.

### Verification

- `go build -tags "goexperiment.jsonv2" ./...` — PASS (entire workspace)
- `go test -tags "goexperiment.jsonv2" -count=1 -race ./system/... ./metaengine/...` — PASS
- `cmd/api-stability` — 3781 exports verified, golden up to date
- `cmd/doc-check` — 580 references valid across 5 packages
- `wc -l system/system.go` — 268 lines (under 350-line CI limit)

### Files Changed

| File                                | Change                                                                                                                                                                                                        |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `system/system.go`                  | Removed Drainer, shutdownEdge, ProjectionHostResource, orderedEngines, RegisterDrainer. Added `namedEngine` type + `engineSlice()` helper. Changed `engines`/`engineNames` fields to `engines []namedEngine`. |
| `system/shutdown.go`                | NEW. Contains Drainer, shutdownEdge, orderedEngines, RegisterDrainer. Uses `namedEngine` struct.                                                                                                              |
| `system/config_types.go`            | Updated doc comments on ShutdownDependency + ShutdownDependencies field (removed ProjectionHostResource references).                                                                                          |
| `system/constructor.go`             | Updated 3 engine append sites to use `namedEngine{engine, name}`.                                                                                                                                             |
| `system/introspection.go`           | Updated HealthCheck loop, Serialize, Verify to use `engineSlice()` / iterate `namedEngine`.                                                                                                                   |
| `system/system_internal_test.go`    | Updated mock injection to use `namedEngine`. Added 6 new internal tests (orderedEngines × 4, Close × 2).                                                                                                      |
| `system/system_hardening_test.go`   | Added 3 new black-box tests (RegisterDrainer × 2, ResetProjection restart-and-replay).                                                                                                                        |
| `metaengine/duckdbengine/engine.go` | Added `HealthCheck(ctx)` method.                                                                                                                                                                              |
| `metaengine/pgengine/engine.go`     | Added `HealthCheck(ctx)` method.                                                                                                                                                                              |
| `metaengine/pebbleengine/engine.go` | Added `HealthCheck(ctx)` method.                                                                                                                                                                              |
| `system/README.md`                  | Updated ShutdownDependencies description.                                                                                                                                                                     |
| `CHANGELOG.md`                      | Updated System package section.                                                                                                                                                                               |
| `FEATURES.md`                       | Updated lifecycle row, shutdown deps row, test count.                                                                                                                                                         |
| `TODO_LIST.md`                      | Updated P3 section with all new work.                                                                                                                                                                         |
| `docs/api_surface.txt`              | Regenerated (3781 exports).                                                                                                                                                                                   |

---

## B) PARTIALLY DONE

### Badger and Dgraph engines still lack HealthCheck

The prior session's status report mentioned Badger as a candidate for HealthChecker. This session added HealthCheck to DuckDB, Postgres, and Pebble, but did not add it to Badger (`metaengine/badgerengine/`) or Dgraph (`metaengine/dgraphengine/`). Both build successfully. Badger has external state (LSM on disk) and would benefit from a health check. Dgraph is a distributed graph database accessed via gRPC — a health check would need to verify the gRPC connection.

### No tests for the new engine HealthChecks

HealthCheck was added to DuckDB, Postgres, and Pebble engines but no tests were written specifically for them. The existing `TestSystem_HealthCheck_SQLite` test covers the SQLite path, but there are no equivalent tests for DuckDB/Postgres/Pebble. DuckDB requires CGo, Postgres requires a running instance — both are harder to test in CI. Pebble could be tested easily (in-memory vfs).

### No test for Pebble HealthCheck on closed DB

The Pebble HealthCheck does a point-read of a non-existent key. When the DB is closed, `Get` returns an error that is NOT `pebble.ErrNotFound` — the HealthCheck correctly propagates this. But there's no test verifying this behavior.

---

## C) NOT STARTED

1. **Add HealthCheck to Badger engine** — `metaengine/badgerengine/`. Badger has `db.View()` for read-only transactions. A health check could do a lightweight `db.View(func(txn *badger.Txn) error { return nil })`.

2. **Add HealthCheck to Dgraph engine** — `metaengine/dgraphengine/`. Dgraph is accessed via gRPC (`dgo` client). Health check could verify the connection via `ctx` cancellation or a lightweight query.

3. **Add tests for DuckDB/Postgres/Pebble HealthCheck** — Integration tests verifying each engine's `HealthCheck` method. Pebble is easy (in-memory). DuckDB needs CGo. Postgres needs a running instance.

4. **Tag releases** — `metaengine/sqliteengine/v4.0.1`, `metaengine/duckdbengine/v4.0.1`, `metaengine/pgengine/v4.0.1`, `metaengine/pebbleengine/v4.0.1` (all have new `HealthCheck` export), `system/v4.1.0` (removed `ProjectionHostResource`, internal refactor). Not done because version bumps should be batched with the next feature release.

5. **Update AGENTS.md** — The system module section in `AGENTS.md` still mentions `ProjectionHostResource` in the Modules list. Should be removed. Also, the `metaengine` section should mention that all 4 external-state engines now implement `HealthChecker`.

6. **Consider `System.Drain(ctx)` standalone method** — Currently drain is only accessible via `GracefulClose`. A standalone `Drain` method would let consumers drain without closing (e.g., for rolling deploys).

7. **Consider structured HealthCheck results** — Currently `HealthCheck` returns only the first error. A structured result (per-engine status) would be more useful for dashboards.

8. **Consider `System.EngineNames()` introspection method** — Returns engine names for diagnostics. Currently `Explain()` prints the count but not the names.

9. **Consider `System.ShutdownOrder()` introspection** — Returns the resolved close order. Useful for debugging shutdown hangs.

10. **`orderedEngines()` cycle fallback is subtly fragile** — When a cycle is detected, remaining engines are appended by checking `inDegree[i] > 0`. This works because Kahn's algorithm only adds nodes with `inDegree == 0` to `result`, so cycle nodes are never in `result`. But there's no explicit deduplication guard. The code is correct, but a future refactor could accidentally break this invariant.

---

## D) TOTALLY FUCKED UP

Nothing is fucked up. All code compiles, all tests pass with `-race`, the api-stability golden is current, doc-check passes, and all files are under the 350-line CI limit.

---

## E) WHAT WE SHOULD IMPROVE

### Architectural

1. **`ProjectionHostResource` should have been caught in the prior session.** The constant was declared, documented, and even had an example in the doc comment — but `orderedEngines()` silently ignored it. This is a documentation lie. The prior session should have either implemented it or not declared it. This session removed it, which is the right call, but the constant shipped in commit `f68412e85` and was already in the api-stability golden. Removing an exported symbol is a breaking change — consumers who referenced it would break. Since this is a pre-release module (no tagged release of `system/v4.1.0` yet), this is acceptable, but it highlights the importance of not shipping constants that don't work.

2. **The `namedEngine` refactoring is correct but could go further.** `engineSlice()` allocates a new slice on every call. HealthCheck, Explain, Serialize, and Verify all call it. For hot paths (HealthCheck in a Kubernetes liveness probe at 1Hz), this is fine. For cold paths (Explain, Serialize, Verify), it's negligible. But a `func (s *System) engines() []metaengine.Engine` method that caches the slice would be cleaner. Not worth the complexity now.

3. **The Pebble HealthCheck is a heuristic, not a true ping.** `Get("__health_check__")` returning `ErrNotFound` proves the DB can read, but doesn't verify write capability or LSM health. A more thorough check could do a `Set` + `Delete` of a health-check key, but that would add write amplification. The current approach is the right tradeoff for a liveness probe.

4. **DuckDB and Postgres HealthChecks are untested.** The code is trivial (`db.PingContext(ctx)`), but "trivial" code can still have import issues, driver registration issues, or connection pool exhaustion. Integration tests would catch these.

5. **`system_internal_test.go` is growing.** It now has 7 tests + 3 mock types (unhealthyEngine, closeOrderEngine, failingEngine). If more internal tests are added, consider splitting to `system_internal_shutdown_test.go` and `system_internal_health_test.go`.

### Testing

6. **`TestSystem_ResetProjection_RestartAndReplay` uses SQLite shared cache.** The DSN `file:TestName?mode=memory&cache=shared` means the SQLite DB persists across `System` instances in the same test process. This is a correct integration test pattern, but it's SQLite-specific — the test can't run with Memory engine (events are lost on Close). The test should be guarded with a comment explaining this.

7. **No test for `GracefulClose` with context expiring during drain.** The existing `TestSystem_GracefulClose_ContextExpired` uses an already-cancelled context, which triggers the Close-timeout path, not the drain-timeout path. A test with a slow drainer and a short context would verify the drain timeout behavior.

8. **`TestSystem_Close_OrderMatchesOrderedEngines` uses mock engines, not real ones.** This is correct for testing the ordering logic, but doesn't verify that real engines (SQLite, Pebble) actually close in the right order. A real-engine test would be an integration test.

9. **The `closeOrderEngine` mock uses a `sync.Mutex` to protect the shared close-order slice.** This is correct, but the `System.Close()` method also holds `s.mu.Lock()` — if a mock engine's `Close()` tried to acquire `s.mu`, it would deadlock. The current mock doesn't, but future mock authors should be aware of this.

### Code Quality

10. **`shutdown.go` is 108 lines — under the 30-line function limit but approaching the file-level comfort zone.** `orderedEngines()` is 60 lines. If more shutdown logic is added (e.g., `System.Drain`, `System.ShutdownOrder`), consider splitting to `shutdown_order.go` and `shutdown_drain.go`.

11. **The `engineSlice()` helper allocates on every call.** For 3-5 engines (typical), this is ~100ns. For 20 engines (extreme), it's ~500ns. Acceptable for HealthCheck at 1Hz, but if `Explain()` or `Serialize()` are called more frequently, a cached slice would be better.

12. **`config_types.go` doc comments still reference "projection host always closes first" in two places** — the `ShutdownDependencies` field comment and the `ShutdownDependency` type comment. These are consistent now, but if the behavior changes (e.g., projection host participates in ordering), both need updating.

---

## F) Up to 50 Things We Should Get Done Next

### HealthCheck completeness (high priority)

1. Add `HealthCheck` to Badger engine (`db.View(func(txn) error { return nil })`)
2. Add `HealthCheck` to Dgraph engine (gRPC connection check)
3. Add test for Pebble `HealthCheck` (in-memory vfs, healthy + closed)
4. Add test for DuckDB `HealthCheck` (CGo required)
5. Add test for Postgres `HealthCheck` (testcontainers or nix)
6. Add test for SQLite `HealthCheck` on closed DB (verify error)
7. Add `HealthChecker` to the `Store.HealthCheck` test matrix — verify it delegates to all engines
8. Consider `Engine.Status()` interface — richer than just HealthCheck (capacity, latency, errors)

### System module improvements (medium priority)

9. Add `System.Drain(ctx)` standalone method — expose drain without Close
10. Add `System.EngineNames()` — introspection for diagnostics
11. Add `System.ShutdownOrder()` — introspection for debugging shutdown hangs
12. Consider structured `HealthCheck` result (per-engine status, not just first error)
13. Add `System.LagPerProjection()` — expose projection host lag via System
14. Add `System.LagDuration()` — same
15. Add `System.WorkerStatus()` — expose projection host worker status
16. Add `System.RegisterCloser(name, closer)` — let consumers register external resources
17. Test `GracefulClose` with context expiring during drain phase (slow drainer + short ctx)
18. Test `GracefulClose` with context expiring during close phase (slow engine + short ctx)

### Shutdown ordering (medium priority)

19. Test `orderedEngines` with 5+ engines and multiple edges (complex DAG)
20. Test `orderedEngines` with self-loop edge ({before: "a", after: "a"}) — should be ignored
21. Test `orderedEngines` with duplicate edges — should not double-count inDegree
22. Add `//nolint` or refactor for the gopls nilness false positive on `system.go:180` (was on `orderedEngines`, now in `shutdown.go` — gopls may still report it)
23. Consider extracting `orderedEngines` to a standalone function (not a method) for testability

### Documentation (low priority)

24. Update `AGENTS.md` system module section — remove `ProjectionHostResource`, mention all 4 engine HealthChecks
25. Add `ShutdownDependency` example to README Quick Start
26. Add `Drainer` example to README
27. Document that `ProjectionHostResource` was removed and why
28. Add architecture diagram showing drain → close lifecycle
29. Update `metaengine` README to list which engines implement `HealthChecker`

### Testing polish (low priority)

30. Split `system_internal_test.go` if it grows past ~250 lines
31. Add `TestSystem_Close_NoEngines` — Close on a system with zero engines
32. Add `TestSystem_Close_ProjectionHostError` — projection host Stop fails, engine close still runs
33. Add `TestSystem_GracefulClose_NoDrainers` — GracefulClose with zero registered drainers (should just Close)
34. Add `TestSystem_GracefulClose_MultipleDrainers` — verify all drainers are called in order
35. Add `TestOrderedEngines_EmptyEngines` — empty engine list returns empty
36. Add `TestOrderedEngines_EmptyDeps` — no deps, no engines — returns empty
37. Add comment to `TestSystem_ResetProjection_RestartAndReplay` explaining SQLite shared-cache pattern

### Code quality (low priority)

38. Cache `engineSlice()` result if profiling shows allocation pressure
39. Consider `namedEngine` implementing `metaengine.Engine` interface (delegating Profile/Close to the inner engine) — eliminates `engineSlice()` entirely
40. Add `//nolint:funlen` to `orderedEngines` if it exceeds 30 lines after any future additions
41. Consider `shutdownEdge` using `namedEngine` names instead of raw strings for type safety

### Release (when ready)

42. Tag `metaengine/sqliteengine/v4.0.1` (new `HealthCheck`)
43. Tag `metaengine/duckdbengine/v4.0.1` (new `HealthCheck`)
44. Tag `metaengine/pgengine/v4.0.1` (new `HealthCheck`)
45. Tag `metaengine/pebbleengine/v4.0.1` (new `HealthCheck`)
46. Tag `system/v4.1.0` (removed `ProjectionHostResource`, `namedEngine` refactor, new tests)
47. Verify all module versions are monotonically increasing before tagging

### Integration (future)

48. Add integration test: system with SQLite source-of-truth + Memory projections + HealthCheck
49. Add integration test: system with Pebble source-of-truth + HealthCheck
50. Add integration test: GracefulClose with real Watermill router as Drainer

---

## G) Questions (3)

### 1. Should `namedEngine` implement `metaengine.Engine` directly (delegating `Profile()` and `Close()` to the inner engine)?

Currently `engineSlice()` allocates a new `[]metaengine.Engine` on every call to extract the raw engines. If `namedEngine` implemented `Engine` (with `Profile()` returning `engine.Profile()` and `Close()` returning `engine.Close()`), then `s.engines` could be used directly as `[]metaengine.Engine` via interface satisfaction — no `engineSlice()` needed. The `name` field would still be accessible via type assertion or a `Name()` method. This would eliminate the allocation entirely but adds an interface layer. Is the complexity worth it?

### 2. Should I tag the engine modules now, or wait for the Badger/Dgraph HealthCheck additions to batch?

Four engines now have new `HealthCheck` exports (SQLite, DuckDB, Postgres, Pebble). Two more engines (Badger, Dgraph) are candidates. Tagging now means 4 patch releases; waiting means 6. The modules are pre-release (untagged or at v4.0.0). Should I tag the 4 now, or wait for all 6?

### 3. Should the `System.HealthCheck` method return structured per-engine results instead of just the first error?

Currently `HealthCheck` returns the first unhealthy engine's error. A structured result (`[]EngineHealth{Name, Error}`) would be more useful for dashboards and debugging, but would change the return type (breaking change). Alternatively, a `HealthCheckDetailed()` method could return structured results while `HealthCheck()` stays as-is. Which approach do you prefer?
