# Status Report: System Package Test Coverage Expansion

**Date:** 2026-08-09 00:49
**Session scope:** Add lifecycle edge-case tests, DuckDB/Postgres source-of-truth integration tests, and ShutdownDependency integration test to the `system/` package.

---

## a) FULLY DONE

### 1. Five lifecycle edge-case tests (`system_lifecycle_edge_test.go`)

White-box tests (`package system`) using mock engines and mock projection hosts:

| Test                                                  | What it verifies                                                                                                               |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `TestSystem_Close_Idempotent`                         | Double `Close()` returns nil; engine closed exactly once                                                                       |
| `TestSystem_GracefulClose_Idempotent`                 | Double `GracefulClose()` returns nil both times; engine closed once                                                            |
| `TestSystem_Start_ProjectionHostError`                | `Start()` propagates projection host errors; `started` flag prevents re-attempt (second `Start()` returns `ErrAlreadyStarted`) |
| `TestSystem_RegisterCloser_AfterClose`                | Closer registered after `Close()` is never invoked (dead resource safety)                                                      |
| `TestSystem_HealthCheckDetailed_WithFailedProjection` | Detailed health includes failed projection entries; `HealthCheck` (non-detailed) also surfaces the failure                     |

All pass with `-race` and `-count=1`.

### 2. DuckDB source-of-truth integration test (`integration_duckdb_test.go`)

- `//go:build cgo` — CGo-gated, follows DuckDB isolation pattern
- Registers `"duckdb"` driver via `duckdbengine.New("")` (in-memory)
- Full CQRS roundtrip: dispatch `task.create` → persist event → load back → verify payload → `HealthCheck` → `HealthCheckDetailed` → `Close`
- Passes with `-race` and CGo

### 3. Postgres source-of-truth integration test (`integration_postgres_test.go`)

- Registers `"postgres"` driver via `pgengine.New(dsn)`
- Same CQRS roundtrip pattern as DuckDB test
- Skips gracefully when `POSTGRES_TEST_DSN` or `DATABASE_URL` not set
- **Not yet executed against a live Postgres** — only verifies skip logic and compilation

### 4. ShutdownDependency integration tests (`integration_shutdown_test.go`)

Two tests using real engines (SQLite + Memory):

| Test                                               | What it verifies                                                                                                                                                              |
| -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TestIntegration_ShutdownDependency`               | Declares `{Before: "projections", After: "event-store"}`, dispatches a real command, verifies `ShutdownOrder()` respects the edge, clean `Close()` + double-close idempotency |
| `TestIntegration_ShutdownDependency_CycleFallback` | Cyclic dependency (`alpha→beta` + `beta→alpha`) falls back to creation order without deadlock                                                                                 |

Both pass with `-race`.

### 5. go.mod updated

- Added `metaengine/duckdbengine/v4 v4.0.1` and `metaengine/pgengine/v4 v4.0.1` as direct deps
- `go mod tidy -e` run (expected warnings about nested go.mod packages — documented pattern)
- `go build`, `go vet`, `gofmt` all clean

### 6. Full test suite verification

- Full `system/` test suite passes: non-CGo, CGo, and CGo+race modes all green
- No regressions in existing tests

---

## b) PARTIALLY DONE

### Postgres integration test — compiled but not run against live PG

The test is written, registered, and compiles cleanly. It skips when no DSN is available. **It has never been run against a real Postgres instance.** The DuckDB equivalent was tested live and passes. The Postgres test follows the same pattern so it should work, but "should" is not "verified."

**To verify:** `nix run .#integration-pg -- go test -tags "goexperiment.jsonv2" -run TestIntegration_PostgresSource ./system/...`

---

## c) NOT STARTED

Nothing from the task list is unstarted — all 4 checklist items were addressed.

---

## d) TOTALLY FUCKED UP

Nothing. No failures, no reverts, no broken builds.

---

## e) WHAT WE SHOULD IMPROVE

### 1. ShutdownOrder returns Profile().Name, not config keys — confusing API gap

The `ShutdownOrder()` method returns engine profile names (e.g., `"memory"`, `"sqlite"`), not the config map keys the consumer declares in `DeploymentConfig.Engines` (e.g., `"event-store"`, `"projections"`). But `ShutdownDependency.Before`/`After` fields reference config keys. So the consumer declares dependencies using config keys, but `ShutdownOrder()` returns different names. This is a real usability trap — I hit it during testing and had to adjust my test to check profile names instead. **This should be documented or fixed.**

### 2. The `init()` driver registration pattern leaks state across test files

Each integration test file (`integration_lifecycle_test.go`, `integration_duckdb_test.go`, `integration_postgres_test.go`) registers drivers in `init()`. Since they're all in `package system_test`, they share the same `system.RegisterDriver` global map. If two files register the same driver name, the last one wins silently. Currently no conflict exists (pebble, duckdb, postgres are all distinct), but this is fragile. A `TestMain` that registers all drivers would be cleaner.

### 3. DuckDB adds ~20 indirect dependencies to system/go.mod

Adding `duckdbengine` pulled in `apache/arrow-go`, `duckdb-go-bindings` (6 platform packages), `flatbuffers`, `goccy/go-json`, etc. This inflates the system module's dependency tree for what is a test-only import. The `//go:build cgo` tag means the DuckDB test file won't compile without CGo, but the go.mod entries are always present. If this is a concern, the DuckDB test could be moved to a separate `system/integration/` sub-module (like `testutil/pgtestcontainer`).

### 4. No test for `GracefulClose` drain-error-then-close interaction

`GracefulClose` returns immediately if a drainer fails, without calling `Close()`. The existing `lifecycle_drain_test.go` tests drain timeout but not the "drain error → Close not called → resources leak" scenario. My `GracefulClose_Idempotent` test covers the happy path but not this failure path.

### 5. The Postgres test uses the same DSN for all parallel tests

If multiple `TestIntegration_PostgresSource_*` tests run in parallel against the same DSN, they'll collide on table names (`cqrs_stream_log`, etc.). The DuckDB test avoids this by using `:memory:` (isolated per connection). The Pebble test uses no DSN. The Postgres test should create a per-test database (the `pgtestcontainer` pattern does this).

---

## f) Up to 50 things to get done next

### System package — additional test coverage

1. Run Postgres integration test against live PG via `nix run .#integration-pg`
2. Add per-test database isolation for Postgres integration test (create/drop DB per test)
3. Add `TestSystem_GracefulClose_DrainError_NoClose` — verify Close is NOT called when drain fails
4. Add `TestSystem_Start_Idempotent` — verify second Start returns `ErrAlreadyStarted` (without projection host)
5. Add `TestSystem_Start_NoProjectionHost` — Start with nil projHost returns nil
6. Add `TestSystem_Close_ProjectionHostError_ThenEngineError` — verify both errors are joined
7. Add `TestSystem_Close_EngineError_ContinuesClosing` — verify engine close errors don't short-circuit
8. Add `TestSystem_HealthCheck_Stopped` — HealthCheck returns `ErrSystemStopped` after Close
9. Add `TestSystem_ConcurrentClose` — concurrent Close() calls from multiple goroutines
10. Add `TestSystem_ConcurrentGracefulClose` — concurrent GracefulClose calls
11. Add `TestSystem_RegisterCloser_Concurrent` — concurrent RegisterCloser + Close
12. Add `TestSystem_RegisterDrainer_AfterClose` — RegisterDrainer after Close (should it error?)
13. Add `TestSystem_Drain_Standalone` integration — Drain without Close (rolling deploy scenario)
14. Add `TestSystem_ShutdownDependency_UnknownEngine` — dependency referencing non-existent engine name
15. Add `TestSystem_HealthCheckDetailed_NoEngines` — empty system returns empty slice, not nil
16. Add `TestSystem_LagPerProjection_NoProjHost` — returns nil
17. Add `TestSystem_WorkerStatus_NoProjHost` — returns nil

### System package — fix the ShutdownOrder naming gap

18. Document that `ShutdownOrder()` returns `Profile().Name`, not config keys
19. OR: change `ShutdownOrder()` to return config keys (breaking change, needs ADR)
20. OR: add `ShutdownOrderDetailed()` that returns both config key + profile name

### System package — broader improvements

21. Consolidate driver registration into a `TestMain` instead of scattered `init()` functions
22. Consider moving CGo-dependent integration tests to a sub-module to avoid dep bloat
23. Add a `system.DriverFor(name)` test helper that returns a factory for explicit registration
24. Add Badger engine source-of-truth integration test (`metaengine/badgerengine`)
25. Add bbolt engine source-of-truth integration test (if it implements StreamLogBackend)
26. Add multi-engine projection integration test: SQLite SOT + DuckDB projections
27. Add multi-engine projection integration test: Postgres SOT + Memory projections
28. Add `TestIntegration_GracefulClose_RealProjectionHost` — GracefulClose with a live projection host
29. Add `TestIntegration_ResetProjection` — Reset a real projection after handler bug fix
30. Add `TestIntegration_HealthCheck_AfterClose` — HealthCheck on stopped system

### Project-wide — observed issues during this session

31. **gopls shows 200+ phantom errors** — all `go mod tidy` noise from workspace modules. Documented in AGENTS.md but still overwhelming. Consider a `.goplsignore` or workspace filtering.
32. **LSP diagnostics in tool output are noise** — every View/edit shows 200+ project errors unrelated to the file being edited. These should be filtered to the current file only.
33. **go.sum entries missing for pebbleengine** — `GOWORK=off go test` fails; workspace mode works. This is documented but means per-module testing requires workspace mode.
34. **`encoding/json/v2` warnings from gopls** — gopls runs without `goexperiment.jsonv2` tag so it flags `json.Marshal`/`json.Unmarshal` as requiring go1.27. Documented but noisy.
35. **No `TestMain` in system package** — driver registration via `init()` is fragile and order-dependent

### CI / verification

36. Add CGo-enabled system test job to CI (currently CI may not test the DuckDB integration test)
37. Add Postgres integration test to CI via ephemeral PG (`nix run .#integration-pg`)
38. Run `nix run .#verify` to verify the full gate passes with the new tests
39. Check if api-stability golden needs regen (new test files only, no new exports — probably not)
40. Check if `cmd/doc-check` needs updating (new test files, no doc changes — probably not)

### Metaengine test parity

41. Verify DuckDB engine passes `enginetest.RunMatrix` (cross-engine parity)
42. Verify Postgres engine passes `enginetest.RunMatrix`
43. Add `TestIntegration_DuckDBSource_OptimisticConcurrency` — version check on concurrent writes
44. Add `TestIntegration_PostgresSource_OptimisticConcurrency` — same for PG
45. Add `TestIntegration_DuckDBSource_Journal` — `JournalReadAll` / `JournalReadFrom` on DuckDB
46. Add `TestIntegration_PostgresSource_Journal` — same for PG

### Documentation

47. Document the driver registration pattern in system README (init-time, database/sql model)
48. Add a "Testing Integration" section to system README covering DSN env vars
49. Update AGENTS.md test command to include CGo variant for system tests
50. Consider adding `system/INTEGRATION_TESTING.md` with platform-specific instructions

---

## g) Questions

### 1. Should DuckDB/Postgres integration tests live in the `system/` module or a separate sub-module?

Adding `duckdbengine` to `system/go.mod` pulls ~20 indirect deps (Arrow, FlatBuffers, DuckDB bindings for 6 platforms). A separate `system/integration/` sub-module would keep the system module lean, following the `testutil/pgtestcontainer` precedent. But it adds a `go.mod` maintenance burden. Which direction do you prefer?

### 2. Should `ShutdownOrder()` return config keys instead of Profile().Name?

Currently `ShutdownDependency.Before/After` reference config keys, but `ShutdownOrder()` returns internal profile names. This inconsistency caused a test failure during development. Changing `ShutdownOrder()` to return config keys would be a (technically) breaking change to an exported API. Should I fix this, or just document the discrepancy?

### 3. Should the Postgres integration test create a per-test database for isolation?

The DuckDB test uses `:memory:` (isolated by nature). The Postgres test currently uses a shared DSN, meaning parallel tests would collide on table names. The `pgtestcontainer` pattern creates a per-test database. Should I wire that in, or is single-test-per-run acceptable for now?
