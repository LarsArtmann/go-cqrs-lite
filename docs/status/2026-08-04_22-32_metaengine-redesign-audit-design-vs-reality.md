# Status Report: Metaengine Redesign Audit — Design vs Reality

> **Date:** 2026-08-04 22:32
> **Session scope:** Read `docs/planning/metaengine-redesign.md` (1987 lines, 12 decisions D1-D12, 10 goals G1-G10), then audited `system/` (14 files, 2190 lines) and `metaengine/` core against every claim.
> **Method:** Line-by-line verification of every interface, struct, method, and wiring claim against actual source code.
> **Prior session:** Commits `958e78b3` through `8638a6b1` implemented the initial system package. This session was an audit of that work.

---

## a) FULLY DONE — built, working, tested

These are genuinely functional with compile-time assertions and tests.

### metaengine core (pre-existing, verified this session)

| Item                                                                                                               | Evidence                                                        | Tests                                    |
| ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------- | ---------------------------------------- |
| `StreamLogBackend` interface (5 methods: StreamAppend, StreamRead, StreamVersion, JournalReadAll, JournalReadFrom) | `metaengine/engine.go:415-442`                                  | ✅                                       |
| `AtomicAppender` interface (StreamAppendExpected)                                                                  | `metaengine/engine.go:451-458`                                  | ✅                                       |
| `ErrVersionConflict` sentinel                                                                                      | `metaengine/errors.go`                                          | ✅                                       |
| Memory engine implements StreamLogBackend + AtomicAppender                                                         | `metaengine/engine.go:482-483` (compile-time assert)            | ✅                                       |
| SQLite engine implements StreamLogBackend + AtomicAppender                                                         | `metaengine/sqlite_stream_log.go:135-138` (compile-time assert) | ✅ `sqlite_stream_log_test.go` (3 tests) |
| `VersionedStorage` interface (MapGetAsOf, MapExistsAsOf)                                                           | `metaengine/temporal.go:19-25`                                  | ✅                                       |
| `ExecuteAsOf` (point-in-time reads)                                                                                | `metaengine/temporal.go:48`                                     | ✅                                       |
| Memory engine implements VersionedStorage                                                                          | `metaengine/memory_versioned.go:117` (compile-time assert)      | ✅                                       |
| `SerializablePlan` (Serialize, DeserializePlan, MarshalJSON)                                                       | `metaengine/serializable.go:19-140`                             | ✅                                       |
| `Export`/`Import` (JSON dump/restore)                                                                              | `metaengine/export_import.go:12,79`                             | ✅                                       |
| `Verify` (cross-engine consistency)                                                                                | `metaengine/consistency.go:53`                                  | ✅                                       |
| Replication model (Replication, ReplicationLag, NetworkRTT)                                                        | `metaengine/replication.go`, `engine.go:54-71`                  | ✅                                       |
| 7 plan rules (schema, layout, writeamp, durability, replication, degraded, mapupdate)                              | `metaengine/rule_*.go`, `rules.go`                              | ✅                                       |

### system package (built in prior session, verified this session)

| Item                                                                                                                | Evidence                                                | Tests |
| ------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- | ----- |
| `DomainConfig` / `DeploymentConfig` type separation (D11)                                                           | `system/system.go` — consumer config vs operator config | ✅    |
| `System` struct with all infra fields (D6: eventStore, cmdStore, queryStore, dispatchers, projHost, bus, projStore) | `system/system.go:212-240`                              | ✅    |
| `Op[State]` declarative routing (D10) with `Execute()`, `StreamID()`, `StreamType()`                                | `system/system.go`                                      | ✅    |
| `EventAdapter` with `WithSerialization()`, AtomicAppender fast path, RunInTx fallback, seq cache for O(1) ReadFrom  | `system/adapter_event.go` (372 lines)                   | ✅    |
| `CommandAdapter` (command.Store + SeekableCommandJournal)                                                           | `system/adapter_command.go` (147 lines)                 | ✅    |
| `QueryAdapter` (query.QueryStore + SeekableQueryJournal)                                                            | `system/adapter_query.go` (121 lines)                   | ✅    |
| `simpleBus` (in-process event bus: Publisher + Subscriber + middleware)                                             | `system/bus.go` (130 lines)                             | ✅    |
| `MultiBus` (fan-out to N publishers, first-error semantics)                                                         | `system/multi_bus.go` (55 lines)                        | ✅    |
| `CachedEventStore` (otter v2 W-TinyLFU read-through cache)                                                          | `system/cache.go` (102 lines)                           | ✅    |
| `SnapshotBackend` interface + `memorySnapshotBackend` (system-local)                                                | `system/snapshot.go` (118 lines)                        | ✅    |
| `Topology` types (Topology, InstanceTopology, BusTopology, CacheTierInfo, etc.)                                     | `system/introspection.go` (147 lines)                   | ✅    |
| Scream store types (ScreamTier, ScreamDiagnostic, ScreamReport, ErrUnsafeChange)                                    | `system/scream_store.go` (124 lines)                    | ✅    |
| Driver registry types (RegisterDriver, RegisterBusDriver, lookupDriver, createEngineFromDriver)                     | `system/driver_registry.go` (130 lines)                 | ✅    |
| Durability tiers (Strict, Normal, Relaxed)                                                                          | `system/system.go`                                      | ✅    |
| Instance roles (RoleSourceOfTruth, RoleEvents, RoleCommands, RoleQueries, RoleProjections)                          | `system/system.go`                                      | ✅    |
| 12 tests in `system_extended_test.go` (concurrent dispatch, bus pub/sub, MultiBus, snapshots, etc.)                 | All pass with `-race`                                   | ✅    |

---

## b) PARTIALLY DONE — types exist but wiring is broken or incomplete

### CRITICAL: SQLite is completely unusable through System

| Item                           | What exists                                                                                | What's broken                                                                                                                              |
| ------------------------------ | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| **Driver registry**            | `createEngineFromDriver()` exists in `driver_registry.go:118`                              | Constructor calls `createEngine()` (hardcoded `switch` at `constructor.go:219`) which only supports `"memory"`. The registry is dead code. |
| **SQLite StreamLogBackend**    | Full implementation in `metaengine/sqlite_stream_log.go` with DDL + AtomicAppender + tests | SQLite driver never registered in `init()` (only `"memory"`). `Driver: "sqlite"` fails.                                                    |
| **EventAdapter serialization** | `WithSerialization()` option + `serializedEvent` JSON envelope                             | Constructor never passes `WithSerialization()` for SQL engines. SQL-persisted events lose typed reconstruction.                            |

### Scream store — 2 rules out of ~12

| Item               | What exists                                                                      | What's missing                                                                                                                                                                    |
| ------------------ | -------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Basic safety check | `CheckSafety()` with 2 rules: `volatile-source-of-truth`, `durability-downgrade` | No `PlanDiff`, no `PlanFingerprint`, no `Manifest`, no `SCREAM` severity on Diagnostics, no cross-deploy comparison, no ADT/key-type pinning, no SQLite synchronous=OFF detection |
| `ScreamReport`     | Types defined, `HasErrors()`/`HasWarnings()` work                                | `ScreamReport()` re-runs `CheckSafety` against the _config_, not the _running plan_. Can't detect runtime drift.                                                                  |

### Config loader — stub

| Item               | What exists                 | What's missing                                                                                                                  |
| ------------------ | --------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `LoadConfig(path)` | Exported function signature | `parseYAML` is a no-op (`return nil`). koanf is not a dependency. Only reads `CQRS_DEFAULT_DRIVER`/`CQRS_DEFAULT_DSN` env vars. |

### Introspection — hardcoded values

| Item            | What exists                                 | What's missing                                                                                                                                       |
| --------------- | ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Snapshot(ctx)` | Returns a `Topology` with instances + buses | `HealthStatus` hardcoded `"ok"` (line 77). `Handlers: 0` hardcoded (line 92). No actual health checks (no `db.PingContext`, no `projHost.Status()`). |
| `Health(ctx)`   | Returns a string                            | Just checks `s.started` boolean. No real health aggregation.                                                                                         |
| `Explain(ctx)`  | Returns a string                            | Not implemented beyond placeholder.                                                                                                                  |

### Implemented but NOT WIRED into `New()`

| Component                                   | Status                                                                                                                           |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `MultiBus`                                  | Exists, tested in isolation. Constructor always creates single `simpleBus`. Fan-out to multiple buses (D9) doesn't work.         |
| `SnapshotBackend` + `memorySnapshotBackend` | Exists, tested. Not connected to the System lifecycle. Snapshots don't persist.                                                  |
| Bus driver registry                         | Types exist. Zero bus drivers registered. `BusConfig{Driver: "gochannel"}` does nothing — constructor always builds `simpleBus`. |

---

## c) NOT STARTED

### Design decisions with zero implementation

| #   | Decision/Feature                                                  | Design ref  | Status                                                 |
| --- | ----------------------------------------------------------------- | ----------- | ------------------------------------------------------ |
| 1   | `StreamReadAsOf` / `StreamReadAsOfVersion` on StreamLogBackend    | §5.6, §10.1 | Zero matches in entire codebase                        |
| 2   | `SnapshotBackend` in metaengine (engines implement it)            | D12         | Only in `system/snapshot.go`. No engine implements it. |
| 3   | `PlanDiff(prev, current)` engine                                  | §9.5        | Not started                                            |
| 4   | `PlanFingerprint(plan)` canonical hash                            | §9.5        | Not started                                            |
| 5   | `Manifest` type (pinned golden plan persistence)                  | §9.3        | Not started                                            |
| 6   | `SCREAM` severity level on Diagnostics                            | §9.4        | Not started (only WARN+OVERRIDE + ADVISORY used)       |
| 7   | Cross-deploy comparison (prior plan awareness)                    | §9.4        | Not started                                            |
| 8   | HTTP admin runtime config (hot-reload + graceful restart)         | §7.3        | Not started                                            |
| 9   | koanf integration (YAML + env merge)                              | §4.7, §7.2  | Not started (stub)                                     |
| 10  | Bus driver registrations (gochannel, nats, redis)                 | §7.5        | Not started (registry empty)                           |
| 11  | Pebble StreamLogBackend implementation                            | §10.1       | Not started (0 of 5 methods)                           |
| 12  | DuckDB StreamLogBackend implementation                            | §10.1       | Not started (0 of 5 methods)                           |
| 13  | Postgres StreamLogBackend implementation                          | §10.1       | Not started (0 of 5 methods)                           |
| 14  | Iroh StreamLogBackend implementation                              | §10.1       | Not started (0 of 5 methods)                           |
| 15  | samber/do DI scopes (per-instance lifecycle isolation)            | §3.7, §10.5 | Not started                                            |
| 16  | `EngineProfile.NativeTemporal` flag                               | §5.6        | Not started                                            |
| 17  | Planner auto-detection of temporal query intent (AsOf field)      | §5.6        | Not started                                            |
| 18  | `ExecuteAsOf` on SQLite, Postgres, DuckDB, Pebble engines         | §5.6        | Memory-only                                            |
| 19  | System.Verify() method (cross-scope consistency)                  | §8.3        | Not started                                            |
| 20  | System.Plan() method (combined plan across instances)             | §8.3        | Not started                                            |
| 21  | System.Explain() method (human-readable explanation)              | §8.3        | Placeholder only                                       |
| 22  | Projection E2E test (command → host.Start → projection updated)   | —           | Not started                                            |
| 23  | SQLite-through-System integration test                            | —           | Not started                                            |
| 24  | Codec defaults (CBOR default in system)                           | §10.8       | Not started                                            |
| 25  | Named engine sharing (connection pool as samber/do named service) | §10.10      | Not started                                            |

---

## d) TOTALLY FUCKED UP

### 1. The constructor bypass is the #1 critical failure

`constructor.go:219` calls `createEngine()` — a hardcoded switch supporting **only `"memory"`**. The entire driver registry (`RegisterDriver`, `lookupDriver`, `createEngineFromDriver`) is **dead code**. The SQLite StreamLogBackend implementation (fully built, fully tested at metaengine level) is **completely unreachable** through `system.New()`.

This means G4 ("run with SQLite + Memory") — the most basic goal — is **unmet**. The system runs on Memory-only.

### 2. The design doc's claim "all 5 engines implement StreamLogBackend" is FALSE

The design doc states (§5.2, §10.1): _"All engines implement it."_ Reality: only Memory and SQLite have the compile-time assertions. Pebble, DuckDB, Postgres, and Iroh have **zero** StreamLogBackend methods. Even if the constructor wiring is fixed, those engines cannot serve as source-of-truth stores.

### 3. Two files exceed the 350-line CI limit

- `constructor.go` — 369 lines (limit 350)
- `adapter_event.go` — 372 lines (limit 350)

The `nix run .#verify` gate will fail on file size.

### 4. `system/` not in api-stability modules list

`cmd/api-stability/main.go` does not include `"system"`. `TestEveryGoModDirIsInModulesList` will fail. The api-stability golden file is not regenerated for all new exported symbols.

### 5. Introspection returns lies

`HealthStatus: "ok"` is hardcoded — no actual health check runs. `Handlers: 0` is hardcoded — the dispatcher may have handlers registered. An admin UI consuming this data would display false information.

### 6. simpleBus handler independence

`simpleBus.dispatch` chains handlers — if one handler fails, subsequent handlers may not execute. Standard `event.Bus` semantics expect each handler to be called independently. This is a behavioral correctness issue for projections and side-effects.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Verify claims against source code, not summaries.** This session initially trusted a sub-agent's summary that StreamLogBackend was "implemented by all 5 engines." It took a direct prompt ("Other metaengine files you can double check with?") to discover only 2 of 5 engines implement it. ALWAYS grep the engine implementation files directly.

2. **The design doc makes claims the code doesn't back.** Every instance of "all engines" or "all backends" in the design doc should be verified with `grep -rn` against the actual engine directories. The doc is aspirational, not factual.

3. **Wire-then-test, not build-then-defer.** The prior session built MultiBus, SnapshotBackend, and the driver registry but never wired them into `New()`. Each of these should have been wired AND tested in an E2E flow before moving to the next feature. Building disconnected components is deferred integration debt.

4. **The constructor is the single point of failure.** It's 369 lines, does too much (engine creation, adapter wiring, projection host setup, command/query registration, cache wrapping), and bypasses the registry. It should be decomposed into smaller functions, each testable independently.

### Architecture improvements

5. **Auto-detect serialization need.** The constructor should check whether the engine is Memory (store pointers directly) or SQL-backed (serialize). This is a one-line type assertion or a flag on EngineProfile. It's inexcusable that this is missing.

6. **Scream store needs the plan diff before it's useful.** The current 2 rules check the _config_, not the _plan_. The scream store's value proposition is detecting unsafe _runtime_ changes by diffing the current SerializablePlan against a pinned manifest. Without PlanDiff/Fingerprint/Manifest, it's a config validator, not a scream store.

7. **Engine coverage gap is structural.** Only 2 of 5 engines implement StreamLogBackend. The remaining 3 (Pebble, DuckDB, Postgres) each need 5 stream-keyed SQL/KV methods. This is mechanical work but it's a prerequisite for G7 (add/remove backends).

---

## f) Up to 50 things to get done next

### P0 — Critical (blocks production use)

1. Replace `createEngine()` with `createEngineFromDriver()` in `constructor.go:219`
2. Register SQLite driver in `init()` or separate driver package
3. Auto-detect serialization: pass `WithSerialization()` for non-Memory engines
4. Write SQLite-through-System integration test (full CQRS roundtrip)
5. Write projection E2E test (command → host.Start → verify projection updated)
6. Split `constructor.go` (369→<350 lines) — extract projection wiring
7. Split `adapter_event.go` (372→<350 lines) — extract serialization
8. Add `system/` to api-stability modules list in `cmd/api-stability/main.go`
9. Regenerate api-stability golden for all new exported symbols

### P1 — High value (makes the design actually work)

10. Wire MultiBus into `New()` when InstanceConfig has multiple Publish targets
11. Wire SnapshotBackend into `New()` and System lifecycle
12. Fix simpleBus handler independence (each handler called separately)
13. Fix introspection: real health checks (db.Ping, projHost.Status), real handler counts
14. Implement `PlanDiff(prev, current *SerializablePlan) (*DiffResult, error)`
15. Implement `PlanFingerprint(plan *SerializablePlan) string` (canonical hash)
16. Implement `Manifest` type (pinned plan persistence to `plan.pin.json`)
17. Wire scream store into `New()` — run CheckSafety on startup, refuse on SCREAM
18. Integrate koanf for YAML + env config loading (replace stub `parseYAML`)
19. Register gochannel bus driver (wrap watermill GoChannel or use simpleBus)
20. Implement Pebble StreamLogBackend (5 stream-keyed methods)
21. Implement DuckDB StreamLogBackend (5 stream-keyed methods)
22. Implement Postgres StreamLogBackend (5 stream-keyed methods)
23. Update AGENTS.md with system/ module entry

### P2 — Important for completeness

24. Implement `System.Verify()` (cross-instance consistency check)
25. Implement `System.Plan()` (combined plan across all instances)
26. Implement `System.Explain()` (human-readable explanation string)
27. Implement `StreamReadAsOf` / `StreamReadAsOfVersion` on StreamLogBackend interface
28. Implement StreamReadAsOf on Memory engine
29. Implement StreamReadAsOf on SQLite engine
30. Implement `SnapshotBackend` in metaengine (engines implement it)
31. Implement `ExecuteAsOf` on SQLite engine
32. Add `SCREAM` severity level to metaengine Diagnostics
33. Add durability-aware scream rules (SQLite synchronous=OFF = SCREAM)
34. Add ADT-change scream rule (detecting ADT change for existing collection)
35. Add key-type-change scream rule
36. Register NATS bus driver (for cross-service fan-out)
37. Register Redis bus driver (for cache invalidation)
38. Add codec config to InstanceConfig (CBOR default, per-instance override)

### P3 — Future polish

39. Implement samber/do DI scopes for per-instance lifecycle isolation
40. Implement HTTP admin runtime config (hot-reload for additive changes)
41. Implement HTTP admin graceful restart (for structural changes)
42. Implement `EngineProfile.NativeTemporal` flag
43. Implement planner auto-detection of temporal query intent (AsOf field in input struct)
44. Implement `ExecuteAsOf` on DuckDB engine (time-travel)
45. Implement `ExecuteAsOf` on Postgres engine (temporal tables)
46. Implement Iroh StreamLogBackend (CRDT-backed source-of-truth)
47. Named engine sharing (connection pools as samber/do named services)
48. Multi-DB SQLite preset test (G5: separate DBs for events, queries, projections)
49. Add per-instance disk usage to introspection
50. Add cache hit rate to introspection CacheTierInfo

---

## g) Questions that need user input

### Q1: Should system/ ship SQLite driver registration inside the system package, or in a separate driver package?

The design doc (§7.1) shows `import _ "github.com/larsartmann/go-cqrs-lite/drivers/sqlite"` — a separate package. But the current `init()` in `driver_registry.go` registers Memory inline. For SQLite, a separate package means `system/` doesn't depend on `modernc.org/sqlite` unless the consumer imports the driver. The tradeoff: cleaner dependency boundary (separate package) vs simpler setup (inline registration). Which model do you want?

### Q2: Should the scream store block `New()` on SCREAM-tier violations, or return a started System with a poisoned scream report?

The design doc (§9.6) says "refuse to start." But the current `New()` doesn't call `CheckSafety()` at all. Should we make `New()` return `ErrUnsafeChange` on SCREAM violations (hard fail), or should it start the system but expose the report via `ScreamReport()` (soft fail with visibility)? The hard-fail model is safer but harder to test/debug.

### Q3: Should the Pebble/DuckDB/Postgres StreamLogBackend implementations happen now, or wait until the SQLite path is proven end-to-end?

The design doc says "sqlite+memory first, add engines mechanically." But the doc also claims "all 5 engines implement it" — that's currently false for 3 of 5. Do you want to fix the doc's claim (change to "Memory + SQLite") or implement the missing 3 before the claim becomes true? The 3 implementations are mechanical (5 stream-keyed methods each) but non-trivial for Pebble (LSM key design) and DuckDB (CGo columnar).
