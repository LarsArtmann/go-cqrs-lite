# Branching-Flow Comprehensive Fix Plan

**Generated:** 2026-06-10
**Source:** `branching-flow all . --no-emoji`
**Total Issues:** 389 across 8 linters

## Priority Matrix

| #                                                  | Linter       | Issue                                                                               | Severity | Impact | Effort | Est.     |
| -------------------------------------------------- | ------------ | ----------------------------------------------------------------------------------- | -------- | ------ | ------ | -------- |
| **PANIC (1 issue)**                                |
| 1                                                  | panic        | gracefulshutdown send on closed channel                                             | medium   | high   | low    | 8min     |
| **CONTEXT — Error Context Loss (13 issues)**       |
| 2                                                  | context      | pebble/journal.go:84 — missing `limit` in error                                     | medium   | medium | low    | 5min     |
| 3                                                  | context      | storage/event_store_global.go:47 — missing `limit` in error                         | medium   | medium | low    | 5min     |
| 4                                                  | context      | storage/sql/query_engine.go:48 — missing `aggType` in error                         | medium   | high   | low    | 8min     |
| 5                                                  | context      | integration/simulation/generator.go:66 — missing `aggregates`, `eventsPerAggregate` | medium   | low    | low    | 5min     |
| 6                                                  | context      | memory/checkpoint.go:35 — missing `projectionName` in error                         | medium   | medium | low    | 5min     |
| 7                                                  | context      | memory/store_load.go:35 — missing `op` in error                                     | medium   | medium | low    | 5min     |
| 8                                                  | context      | middleware/logging.go:42 — missing `prefix`, `msgType` in error                     | medium   | medium | low    | 8min     |
| 9                                                  | context      | middleware/recovery.go:34 — missing `msgKind`, `typeName` in error                  | medium   | medium | low    | 8min     |
| **STRONG-ID — Weak ID Types (21 issues)**          |
| 10                                                 | strong-id    | catalog/d2/connections.go — `serviceDisplayID`, `eventDisplayID` as string          | medium   | medium | medium | 10min    |
| 11                                                 | strong-id    | catalog/openapi/types.go:49 — `OperationID` as string                               | medium   | medium | low    | 8min     |
| 12                                                 | strong-id    | catalog/types_helpers.go:49 — `ID` as string (RefID)                                | medium   | medium | low    | 8min     |
| 13                                                 | strong-id    | catalog/types_resources.go:31,66 — `ID` as string (FlowStepID, FlowEdgeID)          | medium   | medium | low    | 10min    |
| 14                                                 | strong-id    | catalog/internal/cattest/builders.go — `serviceID`, `messageID` as string           | low      | low    | low    | 5min     |
| 15                                                 | strong-id    | example/saga-pattern — `OrderID` as string (7 occurrences)                          | medium   | low    | low    | 10min    |
| 16                                                 | strong-id    | middleware/healthcheck.go — `ReleaseID`, `ComponentID` as string                    | medium   | medium | low    | 10min    |
| 17                                                 | strong-id    | middleware/sse.go — `id` as string (2 occurrences)                                  | high     | high   | low    | 8min     |
| **DUPE — Duplicate Types (15 issues in 6 groups)** |
| 18                                                 | dupe         | example/catalog-server vs example/user — `CreateUserPayload` + 4 others             | low      | low    | low    | 10min    |
| 19                                                 | dupe         | catalog/asyncapi vs openapi — `Info` struct                                         | medium   | medium | medium | 12min    |
| 20                                                 | dupe         | example/projection — `ItemAdded`, `ItemRemoved` near-dupes                          | low      | low    | low    | 5min     |
| 21                                                 | dupe         | example/user/commands.go — `CreateUserCmd`, `RebirthUserCmd` near-dupes             | low      | low    | low    | 5min     |
| 22                                                 | dupe         | storage — `AggregateProjection` vs `SQLAggregateReader` near-dupes                  | medium   | medium | medium | 12min    |
| 23                                                 | dupe         | projection — `Builder` vs `builtProjection` near-dupes                              | medium   | low    | medium | 10min    |
| **ANTI-PATTERNS (5 issues)**                       |
| 24                                                 | anti-pattern | catalog.Message — large struct (17 fields)                                          | medium   | high   | high   | 12min    |
| 25                                                 | anti-pattern | catalog.Service — large struct (16 fields)                                          | medium   | high   | high   | 12min    |
| 26                                                 | anti-pattern | example/todo/storage/PebbleBase — base-naming                                       | low      | low    | low    | 5min     |
| 27                                                 | anti-pattern | storage/sql.Base — base-naming                                                      | low      | medium | low    | 8min     |
| 28                                                 | anti-pattern | storage/sql.ClosableBase — base-naming                                              | low      | medium | low    | 8min     |
| **MIXINS — Composition Opportunities (19 issues)** |
| 29                                                 | mixins       | catalog/asyncapi.Info ↔ openapi.Info — shared InfoMixin                             | low      | medium | medium | 10min    |
| 30                                                 | mixins       | catalog/d2.Exporter — ExporterMixin                                                 | low      | low    | low    | 5min     |
| 31                                                 | mixins       | catalog/openapi.RequestBody — RequestBodyMixin                                      | low      | low    | low    | 5min     |
| 32                                                 | mixins       | event.builder — builderMixin                                                        | low      | low    | low    | 5min     |
| 33                                                 | mixins       | event.projectionFunc — projectionFuncMixin                                          | low      | low    | low    | 5min     |
| 34                                                 | mixins       | example/todo structs — TodoPayloadMixin etc.                                        | low      | low    | low    | 5min     |
| 35                                                 | mixins       | example/user commands — CreateUserCmdMixin etc.                                     | low      | low    | low    | 5min     |
| 36                                                 | mixins       | memory.MemoryCheckpointStore — MemoryMixin                                          | low      | low    | low    | 5min     |
| 37                                                 | mixins       | memory.MemoryCommandStore — MemoryMixin (medium confidence)                         | low      | low    | low    | 5min     |
| 38                                                 | mixins       | projection.Builder — BuilderMixin (medium confidence)                               | low      | low    | low    | 5min     |
| 39                                                 | mixins       | storage.AggregateProjection — AggregateProjectionMixin (medium confidence)          | low      | low    | low    | 5min     |
| **PHANTOM — Primitive Obsession (315 issues)**     |
| 40                                                 | phantom      | catalog — channelKey, ref, version, description, name, title etc.                   | low      | medium | high   | deferred |
| 41                                                 | phantom      | event — encoding strings, event type strings                                        | low      | medium | high   | deferred |
| 42                                                 | phantom      | example modules — string primitives throughout                                      | low      | low    | high   | deferred |
| 43                                                 | phantom      | storage — SQL dialect strings, table names                                          | low      | low    | high   | deferred |
| 44                                                 | phantom      | middleware — kind/type/name strings                                                 | low      | low    | high   | deferred |

## Recommended Execution Order

### Sprint 1: Correctness & Safety (1h total)

1. Fix gracefulshutdown channel panic (#1)
2. Fix storage/sql/query_engine.go context loss (#4)
3. Fix middleware/sse.go strong-id: use `id.ClientID` (#17)
4. Fix middleware/healthcheck.go strong-id (#16)
5. Fix middleware/logging.go context loss (#8)
6. Fix middleware/recovery.go context loss (#9)

### Sprint 2: Error Context (30min total)

7. Fix pebble/journal.go context loss (#2)
8. Fix storage/event_store_global.go context loss (#3)
9. Fix memory/checkpoint.go context loss (#6)
10. Fix memory/store_load.go context loss (#7)
11. Fix integration/simulation/generator.go context loss (#5)

### Sprint 3: Strong IDs in Library Modules (45min total)

12. Fix catalog/d2/connections.go strong-id (#10)
13. Fix catalog/openapi/types.go strong-id (#11)
14. Fix catalog/types_helpers.go strong-id (#12)
15. Fix catalog/types_resources.go strong-id (#13)
16. Fix catalog/internal/cattest/builders.go strong-id (#14)

### Sprint 4: Anti-Patterns — Large Structs (30min total)

17. Split catalog.Message (17→focused structs) (#24)
18. Split catalog.Service (16→focused structs) (#25)

### Sprint 5: Anti-Patterns — Naming (20min total)

19. Rename storage/sql.Base → behavior-focused name (#27)
20. Rename storage/sql.ClosableBase → behavior-focused name (#28)
21. Rename example/todo PebbleBase (#26)

### Sprint 6: Duplicate Types — Library (30min total)

22. Consolidate catalog/asyncapi.Info ↔ openapi.Info (#19)
23. Consolidate storage AggregateProjection ↔ SQLAggregateReader (#22)
24. Consolidate projection Builder ↔ builtProjection (#23)

### Sprint 7: Duplicate Types — Examples (20min total)

25. Deduplicate example/catalog-server vs example/user (#18)
26. Fix example/projection near-dupes (#20)
27. Fix example/user commands near-dupes (#21)

### Sprint 8: Strong IDs — Examples (15min total)

28. Fix example/saga-pattern OrderID (#15)

### Sprint 9: Mixins — Medium Confidence (25min total)

29. Extract shared InfoMixin in catalog (#29)
30. Extract memory mixin pattern (#36, #37)
31. Extract projection BuilderMixin (#38)
32. Extract storage AggregateProjectionMixin (#39)

### Sprint 10: Phantom Types (DEFERRED — 315 issues)

- Too many violations for immediate action
- Strategy: incrementally add phantom types for NEW code
- High-value candidates: catalog types (channelKey, ref), middleware kind/type strings
- File follow-up plan per module
