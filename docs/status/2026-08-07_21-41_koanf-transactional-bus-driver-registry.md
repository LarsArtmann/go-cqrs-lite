# Status Report: Session 8 — koanf Config, DuckDB/PG Transactional, Bus Driver Registry

**Date:** 2026-08-07 21:41
**Session goal:** Complete 3 open TODO items from the metaengine/system hardening plan

---

## a) FULLY DONE

### Task 1: koanf YAML Config — COMPLETE

- Rewrote `system/config_loader.go` to use `knadh/koanf/v2` with `file.Provider` (YAML) + `env.Provider` (env merge)
- Eliminated 4 duplicated intermediate structs (`yamlConfig`, `yamlEngine`, `yamlBus`, `yamlInstance`)
- Added `koanf:"..."` struct tags to all config types in `config_types.go` (`DeploymentConfig`, `EngineConfig`, `BusConfig`, `CacheConfig`, `InstanceConfig`)
- Added structured env var overrides: `CQRS_ENGINES__PRIMARY__DRIVER=sqlite` maps to `engines.primary.driver`
- Maintained backward compat: `CQRS_DEFAULT_DRIVER`/`CQRS_DEFAULT_DSN` still work as legacy fallbacks
- Added 2 new tests: `TestLoadConfig_StructuredEnvOverride`, `TestLoadConfig_EnvOverridesYAML`
- All 5 config loader tests pass
- Added koanf to `.golangci.yml` depguard allow list (line 193)
- Raised `DEP_BUDGET[system]` from 13 to 17 in `scripts/check-module-layers.sh`
- Removed duplicate koanf depguard entry (daemon added one, I added one)

### Task 2: DuckDB/PG Transactional — COMPLETE

- Created `metaengine/duckdbengine/transaction.go`: `dbExec` interface, `conn()` routing method, `RunInTx()` implementation
- Created `metaengine/pgengine/transaction.go`: same pattern + `inTx()` helper for StreamAppend/StreamAppendExpected
- Added `activeTx atomic.Pointer[sql.Tx]` field to both `duckdbEngine` and `pgEngine` structs
- Routed ALL 28 SQL call sites through `conn()` instead of `e.db` (14 DuckDB, 14 PG)
- Stream operations skip mutex lock when inside active tx (avoids deadlock)
- Added compile-time assertions: `_ metaengine.Transactional = (*duckdbEngine)(nil)` and same for pgEngine
- Added `RunTransactionalTest` to `enginetest/enginetest.go` (tests commit, rollback, in-tx visibility)
- Wired into `duckdbengine/stream_log_cgo_test.go` and `pgengine/stream_log_test.go`
- DuckDB test passes with `-race` (CGo)
- PG test passes with real Postgres via testcontainers (commit + rollback verified atomically)

### Task 3: Bus Driver Registry — COMPLETE

- Fixed latent `RLock()`/`Unlock()` mismatch bug in `lookupBusDriver` (`driver_registry.go:79`) — would have caused `sync: Unlock of unlocked RWMutex` panic at runtime (was dead code before this session, never caught because `buildEventBus` skipped the registry)
- Removed gochannel special-case in `buildEventBus` — all configured drivers now go through the registry
- Unknown drivers (`nats`, `redis`, etc.) now return a clear error instead of silently falling back to `simpleBus`
- Changed `buildEventBus` return type from `event.Bus` to `(event.Bus, error)` — propagated through `constructor.go`
- Auto-commit daemon also added `ErrBusDriverNotEventBus` sentinel error
- Added 2 tests: `TestBusDriverRegistry_GochannelRegistered`, `TestBusDriverRegistry_UnknownDriverErrors`
- All system tests pass with `-race`

### Fixups completed this session

- `.golangci.yml`: Added `github.com/knadh/koanf` to depguard allow list (line 193)
- `scripts/check-module-layers.sh`: Raised `DEP_BUDGET[system]` from 13 to 17
- `.golangci.yml`: Removed duplicate koanf depguard entry (daemon had added one at line 212)
- `TODO_LIST.md`: Marked all 3 items as `[x]` with completion notes

---

## b) PARTIALLY DONE

### Documentation gaps

- **system/README.md**: Does not document the new structured env var format (`CQRS_ENGINES__PRIMARY__DRIVER`). The README still shows the old API surface (`BusConfig{Driver: "memory"}` — should be `"gochannel"`). Pre-existing issue, not addressed this session.
- **AGENTS.md**: The config_loader.go section in AGENTS.md still references `yaml.v3` parsing. Not updated to mention koanf.
- **ADR-0105**: The ADR accepted koanf but was never updated to reflect the implementation. No status update committed.

### Test coverage

- `RunTransactionalTest` only exercises `MapBackend` (MapSet/MapGet). Does not test `CounterIncrement` or `StreamAppend` inside a transaction. Future enhancement.
- No test for concurrent `RunInTx` calls on DuckDB/PG (the mutex serialization). Race detector passes but no explicit concurrency test.

---

## c) NOT STARTED

- Full `nix run .#verify` gate has NOT been run this session (takes 3-4 min). Only targeted module tests were run.
- `nix run .#lint` not run (golangci-lint was run manually on system/ only — pre-existing issues found in `constructor.go` funlen and `scream_plan.go` perfsprint, both NOT my files).
- `nix run .#check-duplication` not run (baseline was auto-updated by daemon).
- system/README.md update for koanf env var format.
- AGENTS.md update for koanf integration.

---

## d) TOTALLY FUCKED UP

### The RLock/Unlock bug was hiding for who knows how long

`driver_registry.go:79` had `busDriverMu.RLock()` paired with `defer busDriverMu.Unlock()` (write-unlock after read-lock). This is a **fatal panic** in Go (`sync: Unlock of unlocked RWMutex`). It was never caught because:
1. The old `buildEventBus` had a special case: `if busCfg.Driver == "" || busCfg.Driver == "gochannel" { continue }` — it skipped the registry entirely for the only registered driver.
2. No test ever called `lookupBusDriver` directly.
3. The function was effectively dead code until I removed the gochannel special-case.

The moment my code removed the special-case and called `lookupBusDriver("gochannel")`, it panicked. This is a classic "dead code hides fatal bugs" scenario.

### Stale GREEN claim from earlier in this session

When I first ran the system tests after my bus driver changes, I saw a panic (`sync: Unlock of unlocked RWMutex`) and initially suspected a Go 1.26 concurrency issue. I ran multiple diagnostics before realizing it was a simple RLock/Unlock typo. I should have read the 5-line function first.

### Auto-commit daemon interference

The daemon committed my changes in pieces (6+ separate commits) and also:
- Added a duplicate koanf depguard entry (I had to dedup)
- Added `ErrBusDriverNotEventBus` sentinel error (reasonable enhancement)
- Made unrelated changes to benchkit, example/taskmanager, storage/bbolt (OTel tracing)
- These concurrent changes made it harder to verify which changes were mine

---

## e) WHAT WE SHOULD IMPROVE

1. **Dead code is a liability**: `lookupBusDriver` was dead code with a fatal bug. The registry pattern is sound but was never exercised. Every exported function should have at least one test calling it.
2. **Pre-existing lint violations in system/**: `constructor.go` (funlen 114 > 100), `scream_plan.go` (7 perfsprint, 1 golines). These are NOT my files but they'd fail `nix run .#lint`.
3. **16 COVERAGE GAPs in check-module-layers.sh**: Newer modules (badgerengine, dgraphengine, graphadapter, sqliteengine, metaengine/bench, example/metaengine-quickstart, testutil/pgtestcontainer, record, etc.) were never added to the LAYER/DEP_BUDGET maps. This means the layer check always fails with `::error::16 module(s) missing from LAYER or DEP_BUDGET maps`.
4. **api-surface golden has uncommitted drift**: `docs/api_surface.txt` has uncommitted changes for `ErrInvalidTTL` (idempotency) and bbolt OTel methods — these are from daemon changes, not mine.
5. **system/README.md is stale**: References `Driver: "memory"` for BusConfig (should be `"gochannel"`), doesn't document YAML loading or env var format.
6. **Transaction test doesn't cover counters/streams**: Only MapBackend is exercised. CounterIncrement and StreamAppend inside RunInTx should be tested.
7. **No concurrency test for RunInTx**: The mutex serialization on DuckDB/PG is critical for correctness but only verified via `-race`, not an explicit concurrent-call test.
8. **DuckDB StreamAppendExpected still uses mutex, not tx**: When called inside RunInTx, it skips the mutex (correct), but the version check + append are not atomic at the DB level (unlike PG which uses the transaction). DuckDB relies on the mutex for atomicity.

---

## f) Up to 50 things to do next

### High priority (verify gate blockers)
1. Run `nix run .#verify` to get a clean GREEN state
2. Fix the 16 COVERAGE GAPs in `scripts/check-module-layers.sh` (add badgerengine, dgraphengine, graphadapter, sqliteengine, etc. to LAYER + DEP_BUDGET maps)
3. Fix pre-existing lint violations in `system/constructor.go` (funlen 114 > 100 — extract sub-functions)
4. Fix pre-existing lint violations in `system/scream_plan.go` (7 perfsprint, 1 golines)
5. Regenerate api-surface golden for daemon changes (`cd cmd/api-stability && GOWORK=off go run main.go -update`)
6. Run `nix run .#check-duplication` to verify the new transaction.go files don't create harmful clones

### Medium priority (documentation)
7. Update `system/README.md` to document structured env var format
8. Update `system/README.md` to fix `Driver: "memory"` → `Driver: "gochannel"`
9. Update `AGENTS.md` to mention koanf integration in the config loading section
10. Update `ADR-0105` to reflect the implementation status (koanf integrated, structured env vars)
11. Document the `ErrBusDriverNotEventBus` sentinel in the errors section

### Medium priority (test coverage)
12. Add `CounterIncrement` inside `RunInTx` to `RunTransactionalTest`
13. Add `StreamAppend` inside `RunInTx` to `RunTransactionalTest`
14. Add a concurrent `RunInTx` test (two goroutines, verify serialized)
15. Add a test for `RegisterBusDriver` with a custom factory (verify the full registry lifecycle)
16. Add a test for `LoadConfig` with invalid YAML (error path)
17. Add a test for `LoadConfig` with a non-existent file path (error path)
18. Add a test for structured env var override of `instances` array elements

### Medium priority (code quality)
19. Extract `dbExec` interface to a shared location (currently duplicated in duckdbengine + pgengine + sqliteengine — 3 copies of the same 5-line interface)
20. Consider whether DuckDB's `StreamAppendExpected` should use a DB transaction instead of mutex when inside `RunInTx`
21. Add `//art-dupl:accept` comments to the cross-module `dbExec` interface copies (like the existing pattern for `scanStreamValues`)
22. Consider whether `buildEventBus` should return an error if NO buses are configured (currently silently returns simpleBus — correct behavior, but could be confusing)

### Low priority (future features)
23. Register NATS bus driver (`RegisterBusDriver("nats", ...)`) in a sibling module
24. Register Redis bus driver (`RegisterBusDriver("redis", ...)`) in a sibling module
25. Add a `BusConfig.Timeout` field for external bus drivers
26. Add koanf `file.Provider` support for watching config changes (hot reload)
27. Add a `LoadConfigFromBytes(data []byte)` variant for embedded configs
28. Add JSON schema generation for `DeploymentConfig` (for editor autocomplete)
29. Add a `Validate()` method on `DeploymentConfig` that catches config errors before `New()`
30. Consider whether `RunInTx` should support nested transactions (savepoints) for SQL engines
31. Add a `Store.InTransaction` test that verifies cross-collection atomicity end-to-end
32. Add DuckDB `RunInTx` benchmark (transaction overhead vs autocommit)
33. Add PG `RunInTx` benchmark
34. Consider whether Pebble/Badger engines need `Transactional` (they have `AtomicAppender` but no `RunInTx`)
35. Add a `Doctor()` diagnostic for transaction support (report which engines implement Transactional)

### Cleanup
36. Verify the daemon's bbolt OTel changes compile and test cleanly
37. Verify the daemon's benchkit changes compile and test cleanly
38. Verify the daemon's example/taskmanager migration compiles and tests cleanly
39. Run `nix fmt` to verify formatting
40. Run `go mod tidy` on all affected modules
41. Check if `system/go.mod` needs the `gopkg.in/yaml.v3` dep anymore (now indirect — verify it's not imported anywhere)
42. Check if the koanf `providers/file` dependency pulls in `fsnotify` (file watching) — if so, verify it's not a heavy transitive dep chain
43. Add the new `ErrBusDriverNotEventBus` error to the api-surface golden
44. Verify `system/config_types.go` koanf tags don't break JSON marshaling (koanf tags are separate from json tags)
45. Run the cqrs-lint on the system module to verify no new lint findings
46. Check if the `RunTransactionalTest` function needs to be in the api-surface golden (it's exported from enginetest)
47. Consider adding `Transactional` to the `Doctor()` report output
48. Update the metaengine design docs to mention DuckDB/PG Transactional support
49. Update `FOUR-TIER-MODEL.md` if the Transactional capability changes any tier assignments
50. Consider whether `Store.InTransaction` should warn when no engine implements Transactional

---

## g) Questions I CANNOT figure out myself

1. **Should the `dbExec` interface be shared?** It's identical across duckdbengine, pgengine, and sqliteengine (3 copies). Moving it to `metaengine/` would break the "metaengine has zero engine-specific types" boundary. A shared `sqlutils` package would work but adds a module. Should I accept the duplication (cross-module pattern) or extract it?

2. **Should DuckDB's `StreamAppendExpected` use a DB-level transaction instead of a Go mutex?** Currently it relies on `e.mu.Lock()` for atomicity (version check + append). PG uses a real `BEGIN` transaction. DuckDB supports transactions but the mutex is simpler. The tradeoff: mutex is correct for single-process but doesn't protect against concurrent processes sharing the same DuckDB file.

3. **Should the dep budget for `system` be tightened back after koanf?** koanf adds 4 direct production deps (koanf/v2, parsers/yaml, providers/env, providers/file) which brings the total to 17. The alternative was hand-writing env merging (which I did initially with yaml.v3). The koanf approach is more maintainable but heavier. Is 17 acceptable or should I look for ways to reduce it?

---

## Commits this session (my work, chronologically)

| Commit | Description |
|--------|-------------|
| `39d96e8f7` | DuckDB engine: add activeTx field, route through conn(), add Transactional |
| `a84ae600d` | PG engine: add Transactional interface, refactor shared patterns |
| `8a5c106fa` | Test coverage: transactional helpers for DuckDB/PG contract tests |
| `d230e3da5` | System: surface bus driver errors instead of silent fallback |
| `b4a6fd371` | Fix bus driver registry RLock/Unlock bug + bbolt OTel (daemon mixed in) |
| `887d3bbde` | System: migrate config to koanf (daemon mixed in bbolt OTel) |
| `308eecc80` | Status report from session 7 (daemon) |
| `01c26b807` | Example: migrate taskmanager to system/v4 composition root (daemon) |

---

## Test results at session end

| Module | Result | Notes |
|--------|--------|-------|
| `system/` | PASS (0.064s, race-clean) | All config + bus driver + wiring tests |
| `metaengine/duckdbengine/` | PASS (0.163s, race-clean, CGo) | Includes `RunTransactionalTest` |
| `metaengine/pgengine/` | PASS (8.272s, testcontainers) | Includes `RunTransactionalTest` on real PG |
| `metaengine/sqliteengine/` | PASS (0.009s) | Regression check |
| `metaengine/` | PASS (6.912s) | Regression check |
| Full workspace build | PASS | `go build -tags "goexperiment.jsonv2" ./...` |
| check-layers | BUDGET OK | 16 pre-existing COVERAGE GAPs remain |
| depguard (system/) | CLEAN | No koanf violations |
