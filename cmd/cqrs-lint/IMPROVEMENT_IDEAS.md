# cqrs-lint Improvement Ideas

> Generated from a deep analysis of **45 consumer projects** (21 analyzed from source code on disk).
> Each idea is grounded in a real anti-pattern observed in one or more consumer codebases.
> The current linter has **65 rules** (C001-C016, A001-A019, B001-B015, D001-D006, E001-E007, S001-S003).
>
> **170 ideas** organized by category. Each idea links to the consumer project(s) where the pattern was observed.

---

## Table of Contents

1. [Correctness Bug Detection (C-series)](#1-correctness-bug-detection-c-series)
2. [API Misuse & Feature Gaps (A-series)](#2-api-misuse--feature-gaps-a-series)
3. [Boilerplate & Productivity (B-series)](#3-boilerplate--productivity-b-series)
4. [Architecture & Design (E-series)](#4-architecture--design-e-series)
5. [Consistency & Style (D-series)](#5-consistency--style-d-series)
6. [Security & Compliance (S-series)](#6-security--compliance-s-series)
7. [Performance & Efficiency (P-series)](#7-performance--efficiency-p-series)
8. [Version & Migration Health (V-series)](#8-version--migration-health-v-series)
9. [Testing & Quality (T-series)](#9-testing--quality-t-series)
10. [Feature Adoption Coaching (F-series)](#10-feature-adoption-coaching-f-series)
11. [Linter DX & Infrastructure](#11-linter-dx--infrastructure)

---

## Consumer Project Reference

| Tier | Projects | Description |
|------|----------|-------------|
| **Power users** | Kernovia, Standup-Killer, DiscordSync, SwettySwipperWeb | Use 10-15+ modules, deep integration |
| **Framework** | cqrs-htmx (+ usermgmt, dashboardui) | Wraps go-cqrs-lite for ~20 consumers |
| **Medium direct** | bank-sync, browser-history, github-local-sync, go-localsync, InboxClean, crush-daily, timesheets | Direct imports of 5-15 modules |
| **Light/indirect** | Cyberdom, CV, KeyHolderAI, accountability-system, go-plugin-mvp, overview, storbi, standard-bug-tracking-schema, KeyCountdown | Minimal or transitive usage |

---

## 1. Correctness Bug Detection (C-series)

### Existing rule gaps (improvements to current rules)

1. **C006 should catch `ver+1` and `version.Int()+1` directly in event creation** — Kernovia uses `ver+1` in `event.NewEvent(..., ver+1, data)` (stack.go:469), Standup-Killer uses `event.Version(version.Int()+1)` (decide.go:29). Verify C006 covers both forms and also catches `event.Version(ver+1)`.

2. **C017: In-memory snapshot store with persistent event store** — Kernovia pairs a SQLite event store with `memory.NewMemorySnapshotStore()` — snapshots are lost on restart, making the snapshot optimization useless and potentially causing consistency issues on recovery. Detect: snapshot store type is memory when event store is persistent (SQLite/Postgres/Pebble).

3. **C018: Silent journal fallback to empty store** — cqrs-htmx's `journalFromStore` falls back to `memory.NewMemoryStore()` when the store doesn't implement `event.Journal`, meaning projections replay from an empty journal with NO error or warning. Detect: `memory.NewMemoryStore()` used as a journal fallback in a type switch / type assertion chain.

4. **C019: Multiple Repository instances for the same aggregate** — browser-history creates 3 separate `decider.NewRepository` instances for the same aggregate state type (handlers.go:87,143,189). This defeats load coalescing (singleflight) and wastes memory. Detect: >1 `NewRepository` call with the same state type parameter.

5. **C020: Panic in read model / projection handler** — Standup-Killer's `readmodel.go:184` calls `panic` on ID parse failure inside a SubscribeAll handler. A panic in a projection handler can crash the entire projection host or bus. Detect: `panic()` call inside a function that implements `projection.Projection.Handle` or is used as a bus subscriber.

6. **C021: Mutex held during payload decode** — crush-daily's ReadModelStore holds `sync.Mutex` during `DecodePayloadAuto` (setup.go:135-188). This serializes all event processing unnecessarily. Detect: `Lock()`/`RLock()` call followed by `DecodePayloadAuto` or `json.Unmarshal` before `Unlock()`.

7. **C022: Context ignored in handler (`_ = ctx`)** — crush-daily's Handle method ignores the context (setup.go:185). C016 exists for `context.Background()` but not for `_ = ctx` — the context is explicitly discarded. Detect: `_ = ctx` or `ctx` assigned to `_` inside a function with a `context.Context` parameter.

8. **C023: Shutdown error ignored** — go-appkit's `Shutdown` ignores `host.Stop()` error (`eventservice.go:117`: `_ = es.host.Stop()`). The projection host may have pending events that are lost. Detect: `_ = ` assignment on `.Close()`, `.Stop()`, `.Shutdown()` return values. Extends C015 (unchecked Close) to lifecycle methods.

9. **C024: Dual-write read model without rollback** — cqrs-htmx's SQL read model mutates in-memory state before `syncToSQL` (sql_readmodel.go:98-103). If SQL write fails, in-memory and SQL diverge with no rollback. Detect: in-memory mutation followed by `syncToSQL`/`WriteToSQL`/DB write in same handler without transaction/rollback.

10. **C025: `fmt.Errorf` without `%w` in CQRS error paths** — D006 catches this in general, but several projects (browser-history otel.go:50,58) use bare `fmt.Errorf` in CQRS-adjacent code. Tighten D006 to flag `fmt.Errorf` without `%w` in files that import go-cqrs-lite modules.

11. **C026: Idempotency TTL mismatch** — bank-sync defines `idempotencyTTL = 5 * time.Minute` (infrastructure.go:38) but passes `24*time.Hour` to the middleware (infrastructure.go:270) — the constant is dead/misleading code. Detect: named TTL constant that differs from the value actually passed to `idempotency.NewMemoryStore` or `middleware.CommandIdempotency`.

12. **C027: Bus subscription started alongside projectionhost** — InboxClean starts a projection host AND has separate bus subscriptions active. If both subscribe to the same events, duplicate processing occurs. Detect: both `projectionhost.New`/`host.Start` and `bus.Subscribe`/`bus.SubscribeAll` for overlapping event types in the same codebase.

---

## 2. API Misuse & Feature Gaps (A-series)

### Existing rule improvements

13. **A002 should detect `marshalPayload` helper pattern** — github-local-sync (events.go:95-103) and InboxClean (decide.go:107-115) both have a `marshalPayload` function that calls `json.Marshal` and passes the result to `event.NewEvent`. The current A002 detector may miss this indirection. Add detection of the two-step pattern.

14. **A014 should flag ALL `event.NewEvent` calls, not just some** — Many projects (Standup-Killer, github-local-sync, InboxClean, Kernovia) use `event.NewEvent` instead of `event.New`. Verify the rule catches every call site, not just those in certain function name patterns.

15. **A016 should be context-aware about idempotency** — The rule currently checks for idempotency middleware. Extend: if the project uses `idempotency.NewMemoryStore` directly (like Kernovia's custom store), note that the library module exists. If the project defines its OWN idempotency store interface, flag it.

16. **A017 should check for snapshot strategy, not just store** — Standup-Killer creates repos with `WithSnapshotStore` but no `WithSnapshotStrategy`. The store is useless without a strategy. Detect: `WithSnapshotStore` without `WithSnapshotStrategy`.

### New rules

17. **A020: Custom event.Bus reimplementation** — Kernovia (memory_bus.go:19) and timesheets (NewSyncBus) reimplement `event.Bus` instead of using `watermill.NewEventBus()` or the library's memory bus. Detect: a struct implementing `Subscribe`, `SubscribeAll`, `Use`, `UsePublish`, `Close` that is NOT from go-cqrs-lite or watermill.

18. **A021: Custom event.Store reimplementation** — accountability-system (memory.go) reimplements `event.Store` instead of using `storage/memory.MemoryStore`. Detect: a struct implementing `Save`, `Load`, `LoadFromVersion` that is not from a go-cqrs-lite storage module.

19. **A022: Raw `otel.Tracer()` instead of `cqrsotel`** — standard-bug-tracking-schema uses `otel.Tracer("event-sourcing/adapter")` instead of `cqrsotel.NewTracer("app")`. The library's `otel/` module provides re-exports with CQRS-specific span names and views. Detect: direct `otel.Tracer()`/`otel.Meter()` call in files that import go-cqrs-lite modules.

20. **A023: Custom in-memory snapshot store** — cqrs-htmx usermgmt reimplements MemorySnapshotStore (snapshot.go:89-173, ~80 LOC) instead of using `storage/memory.NewMemorySnapshotStore()`. Detect: a struct implementing `SnapshotSink`/`SnapshotSource` methods that is not from go-cqrs-lite.

21. **A024: Decorative event sourcing (decider shape, no wiring)** — storbi has state folding and event types but never creates an event store, bus, or repository. The decider pattern is imported but the pipeline is never wired. Detect: imports `event/`, `decider/` but no `event.New`/`event.NewEvent` calls and no `decider.NewRepository` / `decider.NewTypedRepository`.

22. **A025: Command/query only, no events** — KeyHolderAI imports `command` and `query` but has no event sourcing, no decider, no event store. It uses CQRS dispatchers as a thin service layer. Flag: this may be intentional (CQRS without ES), but suggest event sourcing for audit trail.

23. **A026: Event bus only, no CQRS pipeline** — Cyberdom uses only `event` and `watermill` with no command dispatcher, decider, or query handler. It's using go-cqrs-lite as a bare event bus. Flag: suggest adding command/query separation for a fuller CQRS architecture.

24. **A027: `event.WithCodec` repeated on every event** — crush-daily calls `event.WithCodec(codec.JSONCodec{})` on every single `event.New` call (decider.go:33,71,101,126). The codec should be set once via `event.DefaultCodec` or at the repository/bundle level. Detect: `event.WithCodec` appearing 3+ times in the same file.

25. **A028: cqrs-htmx used only for HTTP middleware** — CV and overview import cqrs-htmx but use it only for `RequestIDFromContext`, `CSRFMiddleware`, `SecurityHeadersMiddleware`. Zero CQRS types. This isn't using go-cqrs-lite at all — cqrs-htmx is an HTTP framework that happens to depend on go-cqrs-lite. Flag: no CQRS-related types used despite cqrs-htmx import.

26. **A029: `bus.UsePublish` is a stub returning nil** — accountability-system's custom bus has `UsePublish` that returns `nil` (no middleware chain). Detect: `func ... UsePublish(...) error { return nil }` in a struct implementing `event.Bus`.

27. **A030: In-memory checkpoint store with persistent event store** — Similar to C017 but for checkpoints. cqrs-htmx defaults to `memory.NewMemoryCheckpointStore()` even with SQLite event stores. Projections replay from zero on every restart. Detect: `MemoryCheckpointStore` paired with a persistent event store.

28. **A031: In-memory DLQ with persistent event store** — cqrs-htmx hardcodes `projectionhost.NewMemoryDeadLetterStore()` (es_projection_setup.go:83). Dead letters are lost on restart. Detect: `NewMemoryDeadLetterStore` paired with a persistent event store.

---

## 3. Boilerplate & Productivity (B-series)

### New rules

29. **B016: Manual checkpoint replay table** — browser-history manually creates a checkpoint table and replay logic (server.go:351-387) that duplicates `projectionhost.Host`. Detect: SQL table named `checkpoint`/`projection_offset` + manual `ReadFrom`/`ReadAll` loop.

30. **B017: Manual read model rebuild from scratch** — crush-daily rebuilds the entire read model from scratch on every startup (setup.go:43-62). No checkpoint, no incremental catch-up. Detect: `Rehydrate`/`Rebuild`/`Replay` method that loads ALL events from the store on startup.

31. **B018: Repeated bus.Subscribe boilerplate** — bank-sync has 5 projections each subscribing with the same error-wrapping pattern (projections.go). Detect: 3+ `bus.Subscribe` calls with identical error-handling structure in the same file.

32. **B019: O(n^2) read model (repo.Load per event)** — timesheets' read model calls `repo.Load` on every event in the SubscribeAll handler (cqrs.go:82). For N events this is O(N^2) because Load re-reads all prior events each time. Detect: `repo.Load` or `repository.Load` call inside a `bus.SubscribeAll` handler.

33. **B020: Manual legacy field upcasting** — go-localsync manually upcasts legacy fields at decode time (item_adapter.go:57-115) instead of using `schema.Upcaster`/`schema.VersionedStore`. Detect: field-renaming or field-defaulting logic in a decode/unmarshal function that is NOT inside a `schema.NewUpcaster` callback.

34. **B021: Missing `StrictApply` / `decider.StrictApply`** — 6 of 8 medium consumers use plain `fold` or `apply` instead of `decider.StrictApply`. Without it, unknown event types are silently ignored (related to C003, but for the recommendation side). Detect: `decider.Decider` with `Fold:` set to a function that does NOT call `decider.StrictApply`.

35. **B022: Manual correlation enricher instead of `CommandCausalityEnricher`** — crush-daily uses a custom `correlation.ContextEnricher()` (setup.go:225) instead of `decider.CommandCausalityEnricher`. Detect: custom enricher function passed to `decider.NewRepository` that is not `decider.CommandCausalityEnricher`.

36. **B023: Missing command middleware entirely** — timesheets, go-localsync, and storbi have ZERO command middleware (no recovery, no logging, no retry). A panic in any command handler crashes the process. Detect: `command.Dispatcher` with no `.Use()` calls.

37. **B024: Missing event bus recovery middleware** — Several projects don't use `middleware.EventRecovery`. A panic in a bus handler can crash the entire bus. Detect: `event.Bus` or `watermill.NewEventBus` with no `middleware.EventRecovery` / `middleware.NewRecovery` in the middleware chain.

38. **B025: Missing state cache on repository** — Most projects don't use `decider.WithStateCache`. For hot streams, incremental loads are 7.4x faster. Detect: `decider.NewRepository` without `WithStateCache` option, especially for aggregates with high event counts.

39. **B026: Manual event type registration instead of catalog** — Many projects define event types as string constants but never register them with `catalog.NewBuilder`. Event documentation and OpenAPI/AsyncAPI generation is unavailable. Detect: 3+ event type string constants with no `catalog` import.

---

## 4. Architecture & Design (E-series)

### New rules

40. **E008: cqrs-htmx primary path bypasses stack presets** — cqrs-htmx's `NewService` uses `buildDeciderRepositories` (manual) instead of `buildStackRepositories` (stack presets). The most common consumer path doesn't benefit from stack's opinionated defaults. Detect: `decider.NewRepository` called directly when `stack.Bundle` is available in the same module.

41. **E009: No HTTP integration for CQRS** — standard-bug-tracking-schema has a full ES setup but no HTTP transport layer (no cqrs-htmx, no transport/http). Commands/queries can only be dispatched programmatically. Detect: `command.Dispatcher` + `query.Dispatcher` with no HTTP handler registration.

42. **E010: Event capture without domain validation** — DiscordSync captures Discord events directly into the store without command/decider validation. There's no domain rule enforcement before persistence. Flag: this is a valid pattern for external event ingestion, but suggest wrapping in a command for validation.

43. **E011: Adapter layer between decider and command handlers** — standard-bug-tracking-schema has an `EventSourcingAdapter` that bridges decider output to command handlers. KeyCountdown has a `BusAdapter` with double event conversion (211+ LOC). Excessive adapter layers add indirection. Detect: >2 layers between `command.Handler` and `decider.Repository.Execute`.

44. **E012: Dual-write migration bus without completion criteria** — Kernovia has a `dual_write_bus.go` that publishes to both legacy and new systems, but no mechanism to detect when the migration is complete or disable the dual-write. Detect: dual-write pattern (publishing to 2+ buses) without a feature flag or completion check.

45. **E013: Signing configured but disabled by default** — Kernovia has full signing infrastructure (signing.go) but `DefaultSignerConfig()` sets `Enabled: false`. The security infrastructure is present but inert. Detect: signing/encryption setup code present but disabled via a config flag defaulting to false.

46. **E014: No read-your-writes consistency** — Several projects (crush-daily, timesheets) don't wait for projection drain before responding to commands. The read model may be stale when the command handler returns. Detect: `host.Start` / projection setup without `waitForDrain` or blocking call.

47. **E015: Watermill EventBus without ordered delivery** — Projects using `watermill.NewEventBus()` should verify that ordered delivery is configured for projections. The library's EventBus uses `BlockPublishUntilSubscriberAck=true` by default, but custom configurations may break this. Detect: watermill EventBus config with `BlockPublishUntilSubscriberAck=false`.

---

## 5. Consistency & Style (D-series)

### New rules

48. **D007: Inconsistent event creation API (`event.New` vs `event.NewEvent`)** — Kernovia uses both `event.New` and `event.NewEvent` in the same codebase. Detect: both APIs used in the same project; recommend standardizing on `event.New`.

49. **D008: Inconsistent codec usage** — github-local-sync uses `event.DecodePayload[T](evt, p.codec)` (explicit codec) in projections but `event.DecodePayloadAuto` in folds. Mixing explicit and auto decode is inconsistent. Detect: both `DecodePayload[` and `DecodePayloadAuto[` in the same project.

50. **D009: Inconsistent Close detection pattern** — cqrs-htmx uses `io.Closer` in one place (es_setup.go:117) and anonymous `interface{ Close() error }` in another (service_core.go:430) for the same operation. Detect: multiple patterns for the same interface check.

51. **D010: Generic error code `"internal"`** — cqrs-htmx uses `errorfamily.WrapTransient(err, "internal", ...)` instead of a descriptive namespaced code. Detect: `"internal"` used as an error code in `errorfamily.Wrap*` calls.

52. **D011: Nil payload events** — InboxClean passes `nil` as event payload for toggle events (decide.go:25). This creates events with empty payloads that can't be decoded. Detect: `event.NewEvent`/`event.New` with `nil` as the payload argument.

53. **D012: Schema version not stamped on events** — Most projects don't use `event.WithSchemaVersion()`. Without it, schema evolution (upcasting) is impossible to implement retroactively. Detect: `event.New`/`event.NewEvent` calls without `event.WithSchemaVersion` in the options.

---

## 6. Security & Compliance (S-series)

### New rules

54. **S004: PII data without encryption (field-level)** — browser-history stores browsing history (URLs, timestamps) without encryption. Detect: event payload structs with fields named `url`, `email`, `phone`, `address`, `ssn`, `token`, `password`, `apikey` when no encryption module is imported.

55. **S005: Event signing available but disabled** — Kernovia has `signing.go` with `DefaultSignerConfig()` setting `Enabled: false`. Detect: signing module imported but signer construction guarded by a boolean flag that defaults to false.

56. **S006: Financial data without encryption** — bank-sync uses AES-256-GCM encryption (gold standard), but other financial projects (timesheets) store financial data without encryption. Detect: monetary field names (amount, price, balance, salary) without encryption module import.

57. **S007: In-memory session/token store** — cqrs-htmx defaults to `NewInMemorySessionStore()`. Session tokens are lost on restart, forcing re-authentication. Detect: in-memory session/token store used in production server context.

---

## 7. Performance & Efficiency (P-series)

> New category — the linter has no performance rules today.

58. **P001: O(N^2) read model via repo.Load in projection** — timesheets' projection calls `repo.Load` on every event (cqrs.go:82). For N events, this is O(N^2) because each Load re-reads all prior events. Suggest projecting directly from event payloads.

59. **P002: Full read model rebuild on every startup** — crush-daily rebuilds the entire read model from scratch on every startup (setup.go:43-62). With 10K events, this adds seconds to startup time. Suggest checkpoint-based incremental catch-up via projectionhost.

60. **P003: Mutex held during payload decode** — crush-daily holds `sync.Mutex` during `DecodePayloadAuto` (setup.go:135-188). Decode is CPU-bound and doesn't need lock protection. Suggest decoding outside the lock, then acquiring the lock only for the map mutation.

61. **P004: Multiple repository instances for same aggregate** — browser-history creates 3 `decider.NewRepository` instances for one aggregate type (handlers.go:87,143,189). Each instance has its own singleflight group and state cache. Suggest sharing one repository.

62. **P005: No state cache on hot aggregate** — For aggregates with high event counts (>100 events/stream), the lack of `decider.WithStateCache` means every command triggers a full stream load. Suggest `WithStateCache` + snapshot strategy.

63. **P006: Polling loop for drain check** — cqrs-htmx's `waitForDrain` polls every 10ms (es_projection_setup.go:150-192). A channel/callback would be zero-latency. Flag: polling loop with <100ms interval for state-change detection.

64. **P007: Manual retry loop with bit-shift backoff** — DiscordSync has a hand-rolled `appendWithRetry` (storage.go:207-241) with `baseBackoff << time.Duration(attempt-1)` bitshift. This has a subtle bug: left-shifting a Duration shifts the nanoseconds representation. Suggest `retry.Do` from the library.

65. **P008: Projection host WithBatchSize not set** — InboxClean sets batch size 500 but most projects don't set it at all, defaulting to whatever the library default is. For large event streams, the batch size significantly affects throughput. Flag: `projectionhost.New` without `WithBatchSize`.

66. **P009: JSON codec for large payloads** — Projects with large event payloads (DiscordSync's message events) should use CBOR for ~35% smaller payloads. Detect: event payload structs with many fields (>10) or `[]byte` fields using JSON codec.

67. **P010: No snapshot strategy on large aggregates** — Projects with aggregates that could accumulate >100 events should configure snapshot strategy. Detect: aggregate with >50 events in the stream without `WithSnapshotStrategy`.

---

## 8. Version & Migration Health (V-series)

> New category — critical for the multi-version ecosystem.

68. **V001: v3 and v4 modules mixed in the same project** — go-plugin-mvp imports v3 modules directly while cqrs-htmx is v4. go-appkit is entirely v3. Detect: both `go-cqrs-lite/.../v3` and `go-cqrs-lite/.../v4` import paths in the same go.mod.

69. **V002: Unpinned go-cqrs-lite version** — Several projects (sec, SwettySwipperWeb for some modules) use `(unpinned)` versions. Detect: go-cqrs-lite dependency without a specific version tag.

70. **V003: Version lag behind latest** — Many projects are on v4.0.x or v4.1.x while latest is v4.2.0+. Detect: go-cqrs-lite version more than 2 minor versions behind the latest known release.

71. **V004: Vendored copy of go-cqrs-lite** — go-plugin-mvp has a vendored `third_party/go-cqrs-lite-eventtest` workaround. A019 exists for vendored copies; extend to detect `third_party` directory copies.

72. **V005: eventtest pseudo-version mismatch** — go-plugin-mvp vendors eventtest because the published version doesn't match the stack version requirement. Detect: vendored eventtest package alongside go-cqrs-lite imports.

73. **V006: Mixed direct/indirect version pins** — Some projects pin event/v4 at v4.2.0 directly but get v4.1.0 transitively through cqrs-htmx. Detect: direct require with different version than the indirect require for the same module.

---

## 9. Testing & Quality (T-series)

> New category — the linter has B015 for test utilities but no testing-specific rules.

74. **T001: No scenario tests for deciders** — Only Standup-Killer, DiscordSync, SwettySwipperWeb, and KeyCountdown use `scenario/v4`. Most projects with deciders have no BDD tests. Detect: `decider.Decider` defined but no `scenario.Given` calls in test files.

75. **T002: No scenario tests for projections** — Only DiscordSync uses `scenario.GivenProjection`. Detect: `projection.Projection` implementations with no `scenario.GivenProjection` in test files.

76. **T003: No eventtest imports** — Projects with event stores should use `event/v4/eventtest` for fake stores/buses in tests. Only 9 projects import testutil, 20 import eventtest. Detect: event store usage with no eventtest import.

77. **T004: No golden/snapshot tests** — Projects with catalog/documentation generation should snapshot-test their output. Detect: `catalog` usage with no `go-snaps` / `snaps` import.

78. **T005: Projection without error-handling test** — Projections should have tests for malformed payloads and unknown event types. Detect: `projection.Projection` with no test for `ThenError` or error-path assertions.

79. **T006: Decider test without conflict-path test** — Standup-Killer tests both success AND conflict paths (ThenError). Most projects only test happy paths. Detect: `scenario.Given` with `Then` but no `ThenError` for the same decider.

80. **T007: No integration test for event round-trip** — Projects with event stores should test save→load→publish round-trips. Detect: event store usage with no test that calls both Save and Load on the same stream.

81. **T008: Test files import production event store** — Tests should use `eventtest.FakeStore` or `storage/memory.MemoryStore`, not the production store. Detect: test files importing `storage` or `storage/turso` (production stores) instead of test utilities.

---

## 10. Feature Adoption Coaching (F-series)

> New category — proactively suggest features consumers are missing.

82. **F001: No tombstone soft-delete** — Only github-local-sync uses `event.MarkTombstone`. Projects with delete operations should use tombstone metadata for soft-delete. Detect: function/method named `Delete*` in a project with events but no `MarkTombstone` usage.

83. **F002: No catalog/documentation** — Only DiscordSync, bank-sync, github-local-sync, crush-daily, Kernovia (stub) use catalog. Most projects have no event documentation. Detect: 3+ event types with no `catalog.NewBuilder` usage.

84. **F003: No OTel tracing** — Only 6 projects import `otel/v4`. The rest have zero distributed tracing. Detect: server-mode project (feature profile) with no OTel import.

85. **F004: No Prometheus metrics** — Only DiscordSync, crush-daily, SwettySwipperWeb use Prometheus. Detect: server-mode project with no Prometheus import.

86. **F005: No schema upcasters** — Only bank-sync, Kernovia, DiscordSync use schema upcasters. Projects with evolving event schemas should plan for upcasting. Detect: event payloads with version >1 but no `schema.NewUpcaster`.

87. **F006: No encryption for sensitive data** — Only bank-sync and Kernovia (disabled) use encryption. Detect: event payloads with PII field names and no encryption import.

88. **F007: No idempotency middleware** — Many projects with command dispatchers don't use idempotency. Detect: `command.Dispatcher` with no idempotency middleware in server/async mode.

89. **F008: No CBOR codec** — Most projects default to JSON. For event-heavy systems, CBOR is ~35% smaller. Detect: high event volume (many event types) with JSON codec default.

90. **F009: No scheduling module** — No project uses `scheduling/v4` despite it being available for deadline timers ("cancel order after 30 min"). Detect: domain with time-based business rules (deadlines, expirations, timeouts) with no scheduling import.

91. **F010: No graph projections** — Only Kernovia and Standup-Killer import `graph/v4`. Projects with relationship-heavy domains (social networks, org charts) could benefit. Detect: domain with recursive traversal queries and no graph import.

92. **F011: No relational projections** — Only DiscordSync and github-local-sync use relational projections. Projects with multi-table writes per event should use `storage.RelationalProjection`. Detect: multiple SQL INSERT/UPDATE statements in a single projection handler.

93. **F012: No deriver module** — No project uses `deriver/v4`. Projects that derive commands from events (saga-like patterns) should use it. Detect: bus.SubscribeAll handler that dispatches commands based on events.

94. **F013: No transport/http or transport/grpc** — Most projects don't use the transport modules. Projects that need remote dispatch should use them instead of hand-rolled HTTP/gRPC handlers. Detect: manual HTTP handler for command/query dispatch.

95. **F014: No kv.Cache** — Projects using `kv.TypedStore` without `kv.Cache` miss the caching layer. Detect: `kv.NewTypedStore` without `kv.NewCache`.

96. **F015: No metaengine** — Only Kernovia uses metaengine. Projects with complex query patterns could benefit from the cost-based planner. (Low priority — metaengine is early stage.)

97. **F016: No listing module for stream management** — Projects with many event streams should use `listing.StreamListing` for stream status tracking. Detect: >5 aggregate types with no listing import.

98. **F017: No dedup module** — The `dedup` ring buffer is imported by many projects indirectly but rarely used directly. Projects with at-least-once delivery should use it explicitly. Detect: bus subscription without dedup module.

---

## 11. Linter DX & Infrastructure

### Analysis accuracy improvements

99. **Feature profile should detect adapter/bridge patterns** — KeyCountdown wraps its own event system in adapters conforming to go-cqrs-lite interfaces. The linter's feature detection should recognize adapter types and adjust rules accordingly.

100. **Feature profile should detect cqrs-htmx usage** — Projects going through cqrs-htmx get different defaults than projects using go-cqrs-lite directly. The linter should detect cqrs-htmx imports and apply different rule sets.

101. **Feature profile should detect event-capture architecture** — DiscordSync has no command/decider/query. The linter shouldn't flag "missing decider" when the architecture is intentionally event-capture-only.

102. **Severity calibration based on aggregate type** — Financial aggregates (bank-sync, timesheets) should have stricter rules (encryption, idempotency) than internal tools (Standup-Killer, overview).

### Rule quality improvements

103. **Rules should provide migration paths, not just complaints** — Instead of "use event.New", provide the exact code transformation. The C001 auto-fix is a good model.

104. **Rules should link to documentation** — Each finding should include a URL to the relevant SKILL.md or ADR for the recommended pattern.

105. **Rules should respect project conventions** — If a project consistently uses manual wiring (like Kernovia), don't flag every manual wiring instance; flag only the ones with actual bugs.

106. **Rules should understand the stack preset boundary** — If a project uses `stack/sqlite`, the linter should know that the store, bus, snapshot, and checkpoint are configured correctly and skip related rules.

107. **Confidence scoring should consider codebase maturity** — A 500-line prototype shouldn't get the same severity as a 50K-line production system. Add a `--maturity` flag or auto-detect.

### New rule categories

108. **Documentation rules (DOC-series)** — Rules that check for missing docs, stale catalog entries, undocumented events.

109. **Observability rules (OBS-series)** — Rules that check for tracing spans, metrics, structured logging in CQRS handlers.

110. **Resilience rules (RES-series)** — Rules that check for retry, circuit breaker, dead-letter queue, graceful shutdown patterns.

111. **Data integrity rules (DI-series)** — Rules that check for optimistic concurrency, idempotency, transactional consistency.

### Output and reporting

112. **Group findings by aggregate/domain** — Instead of a flat list, group findings by the aggregate or domain they affect. "Your User aggregate has 5 issues" is more actionable than 5 scattered findings.

113. **Show feature adoption scorecard** — Beyond the health score, show which features the project uses vs misses. "You use 8/15 modules. Consider: scheduling, encryption, catalog."

114. **Diff mode** — `cqrs-lint --diff` shows only NEW findings since the last run. Useful for CI to prevent regressions without overwhelming developers.

115. **Fix-all command** — `cqrs-lint --fix-all` applies all auto-fixable findings in dependency order (version arithmetic before missing commit, etc.).

116. **Interactive suppression** — `cqrs-lint --suppress` walks through each finding and lets the developer accept (suppress) or fix it interactively.

117. **SARIF rule metadata** — Include rule documentation URL, severity guidance, and remediation steps in SARIF output for GitHub Code Scanning.

### CLI improvements

118. **`cqrs-lint profile <project>` command** — Analyzes a project and prints a detailed usage profile (modules used, features enabled, anti-patterns found, feature adoption scorecard).

119. **`cqrs-lint compare <project1> <project2>` command** — Compares two projects' CQRS usage and highlights differences. Useful for teams standardizing across services.

120. **`cqrs-lint upgrade-check`** — Checks if the project's go-cqrs-lite version has breaking changes or new features since the pinned version.

121. **Config inheritance** — `.cqrs-lint.json` in a parent directory should be inherited by subdirectories, with local overrides. Useful for monorepos.

### Performance

122. **Incremental analysis** — Cache the AST scan results and only re-scan changed files. Currently, every run re-parses all Go files.

123. **Parallel rule execution** — The pipeline already supports `ParallelDetectors: true`, but verify all rules are thread-safe. Add a benchmark suite for the linter itself.

124. **Memory-bounded analysis** — For very large codebases (>1000 files), the linter should stream files rather than loading all ASTs into memory.

---

## Extended Ideas (125-170)

### Deep pattern detection

125. **Detect custom retry loops more accurately** — DiscordSync's `appendWithRetry` (storage.go:207-241) has a bitshift backoff bug (`baseBackoff << time.Duration(attempt-1)` shifts Duration's nanosecond representation). The current B008 rule should catch this but may miss the bitshift variant.

126. **Detect event type string typos** — If a fold function handles "UserCreated" but the emit code uses "user.created", the event is silently ignored. Cross-reference fold switch cases with `event.New` type strings.

127. **Detect orphaned event types** — Events emitted but never handled by any fold or projection. E006 exists but may miss events emitted through adapters/bridges (like KeyCountdown's BusAdapter).

128. **Detect orphaned commands** — Commands defined but never dispatched. E005 exists but may miss commands dispatched through cqrs-htmx's HTTP layer.

129. **Detect missing error family classification in domain logic** — Several projects use `fmt.Errorf` or `errors.New` in domain logic instead of `errorfamily` constructors. D006 exists but should be stricter in files that import `event` or `decider`.

130. **Detect context propagation gaps in event handlers** — If a projection handler receives `ctx` but calls a function that creates a new context, tracing is broken. Detect: `context.Background()`, `context.TODO()`, or `context.WithCancel(context.Background())` inside a handler function body.

131. **Detect unbounded in-memory growth** — In-memory read models that use `map[string]T` without eviction grow unboundedly. Detect: `map[` in a projection handler that subscribes to `SubscribeAll` without any size limit or eviction.

132. **Detect goroutine leaks in event handlers** — If a handler starts a goroutine (`go func()`) that is not tracked or cancelled, it may outlive the projection host. Detect: `go func()` inside a `Handle` method without context cancellation.

### Cross-module rules

133. **Detect encryption/signing mismatch** — If the event bus has encryption middleware but the event store doesn't (or vice versa), events are stored in cleartext but transmitted encrypted (or vice versa). Cross-check bus middleware vs store wrapper.

134. **Detect snapshot codec / event codec mismatch** — If events use CBOR but snapshots use JSON (or vice versa), there's an inconsistency that could cause decode failures on recovery. Detect: `codec.CBORCodec{}` for events but `codec.JSONCodec{}` for snapshots.

135. **Detect checkpoint store / event store backend mismatch** — SQLite event store with memory checkpoint store means checkpoints are lost on restart, causing full projection replay every time.

136. **Detect idempotency store / event store backend mismatch** — Memory idempotency store with persistent event store means dedup state is lost on restart, allowing duplicate processing after restart.

### cqrs-htmx specific rules

137. **Detect `journalFromStore` silent fallback** — cqrs-htmx's journal detection falls back to empty memory store without error. The linter should detect this pattern in cqrs-htmx consumers specifically.

138. **Detect hardcoded memory DLQ in cqrs-htmx** — `projectionhost.NewMemoryDeadLetterStore()` with `0` retry threshold in server-mode projects means poison events are immediately dropped.

139. **Detect `waitForDrain` polling overhead** — cqrs-htmx's 10ms polling loop adds latency. Suggest using a channel-based notification from projectionhost.

140. **Detect ProjectionStatusEntry field duplication** — cqrs-htmx's `ProjectionStatusEntry` manually mirrors `projectionhost.WorkerState`. Changes to `WorkerState` won't be reflected.

### Domain-specific rules

141. **Detect money as float64 (extend C008)** — C008 exists but should also check for `float32`, `float64` fields with names like `amount`, `balance`, `price`, `cost`, `fee`, `tax`, `salary`, `rate`, `total`.

142. **Detect timestamp without timezone** — C013 exists for `time.Time` in event payloads. Extend to also flag `time.Time` fields without explicit timezone documentation in projections.

143. **Detect PII in event payloads without redaction** — Event payloads containing email addresses, phone numbers, or SSNs should be encrypted or redacted before persistence.

144. **Detect event payload struct size** — Event payloads with >20 fields are hard to evolve and should be split. Suggest smaller, more focused event types.

### Migration and upgrade rules

145. **Detect `event.NewEvent` deprecation migration progress** — Track how many `event.NewEvent` calls remain vs `event.New`. Show progress: "15/20 event creation calls migrated to event.New (75%)."

146. **Detect `Register` (deprecated) vs `RegisterTyped` migration** — A014 flags deprecated `Register`. Track migration progress across the codebase.

147. **Detect v3-to-v4 migration blockers** — For projects still on v3, identify specific API differences that block migration (removed types, renamed functions).

148. **Detect feature flag cleanup needed** — Dual-write buses and migration code should have feature flags that are eventually cleaned up. Detect: dual-write patterns without flag cleanup in the same codebase.

### Educational and coaching rules

149. **Suggest event storming documentation** — If a project has >10 event types, suggest creating an event catalog with `catalog.NewBuilder` for documentation.

150. **Suggest CQRS diagram generation** — If a project has commands, events, and projections, suggest generating a D2 architecture diagram with the catalog module.

151. **Suggest read model tier upgrade** — If a project uses in-memory read models with SubscribeAll, suggest upgrading to SQLViewStore or RelationalProjection for persistence.

152. **Suggest snapshot strategy** — If aggregate event count is high (detect from test data or schema), suggest EveryNEvents or ReadPressure snapshot strategy.

153. **Suggest StrictApply adoption** — If a fold function uses a plain switch with a default that returns nil, suggest `decider.StrictApply` for compile-time safety.

154. **Suggest BDD scenario tests for critical aggregates** — If an aggregate handles financial or security operations, strongly suggest scenario tests.

### Integration rules

155. **Detect missing health checks** — `stack.Bundle.HealthCheck` exists for Kubernetes probes. Detect: server-mode project without health check endpoint.

156. **Detect missing graceful shutdown** — `bundle.GracefulClose` and `projectionhost.Stop` should be called on SIGTERM. Detect: `signal.Notify` without Close/Stop calls.

157. **Detect missing WAL mode for SQLite** — `storage.SQLiteEnableWAL` should be called for all SQLite-backed stores in production. Detect: SQLite store without WAL pragma.

158. **Detect missing busy_timeout for SQLite** — `storage.SQLiteEnableWAL` includes busy_timeout=5000. Detect: `database/sql` open with SQLite DSN without busy_timeout.

### Error handling rules

159. **Detect error swallowing in command handlers** — Command handlers that return nil after an error silently lose failures. Detect: `if err != nil { return nil }` in a function registered with `RegisterTyped`.

160. **Detect error swallowing in projection handlers** — Projection handlers that ignore errors from `DecodePayloadAuto` or SQL operations. C010 exists but should also cover SQL errors.

161. **Detect panic in marshal/encode paths** — Functions with `panic()` in marshal/encode paths (B011 exists for must*-prefixed, extend to all marshal helpers).

162. **Detect missing error wrapping** — Errors returned from library calls should be wrapped with context. Detect: `return err` (bare) for errors from go-cqrs-lite function calls.

### Concurrency rules

163. **Detect race condition in read model** — In-memory read models accessed from multiple goroutines without proper synchronization. Detect: `map` field in a read model struct without `sync.RWMutex` or `sync.Map`.

164. **Detect shared mutable state in event handler** — Global or package-level variables modified inside event handlers. A015 exists but should also detect `var x = map[...]` modified in handlers.

165. **Detect goroutine without context cancellation** — `go func()` without a derived context can outlive the parent. Detect: `go func()` in handler code without `ctx` propagation.

### Data model rules

166. **Detect branded ID misuse** — Using `id.StreamID` where `id.UserID` is intended (or vice versa). Type-safe branded IDs prevent mixing aggregate types.

167. **Detect string IDs instead of branded IDs** — Plain `string` or `int` used as IDs instead of `id.Of[T]`. Detect: struct fields named `*ID` or `*Id` with type `string`.

168. **Detect event payload without json tags** — Event payload structs without `json:"..."` tags use Go field names in JSON encoding, which is inconsistent with typical JSON conventions.

169. **Detect event payload with embedded `time.Time`** — Embedded `time.Time` in event payloads can cause timezone issues via CBOR encoding. C013 exists; extend to embedded fields.

170. **Detect nullable fields in event payloads** — `*string`, `*int` pointer fields in event payloads can cause nil-dereference on decode. Suggest value types with `omitempty` or sentinel values.

---

## Summary Statistics

| Category | Ideas | Existing rules in category |
|----------|-------|---------------------------|
| Correctness (C) | 27 (C001-C016 existing + C017-C027 new) | 16 |
| API Misuse (A) | 31 (A001-A019 existing + A020-A031 new) | 19 |
| Boilerplate (B) | 26 (B001-B015 existing + B016-B026 new) | 15 |
| Architecture (E) | 15 (E001-E007 existing + E008-E015 new) | 7 |
| Consistency (D) | 12 (D001-D006 existing + D007-D012 new) | 5 |
| Security (S) | 7 (S001-S003 existing + S004-S007 new) | 3 |
| Performance (P) | 13 (new category) | 0 |
| Version/Migration (V) | 6 (new category) | 0 |
| Testing (T) | 8 (new category) | 0 |
| Feature Adoption (F) | 17 (new category) | 0 |
| DX & Infrastructure | 24 | N/A |
| **Total** | **170** | **65 existing** |

---

## Priority Recommendations

### Immediate (highest impact for real consumers)

1. **C006 version arithmetic** — observed in Kernovia, Standup-Killer (verify current detection)
2. **A014 event.NewEvent → event.New** — observed in most projects
3. **B021 StrictApply recommendation** — 6/8 projects miss this
4. **B023 missing command middleware** — several projects have zero protection
5. **P001 O(N^2) read model** — timesheets has a critical performance bug
6. **V001 v3/v4 mixing** — go-plugin-mvp, go-appkit need upgrading
7. **C017 in-memory snapshot with persistent store** — Kernovia loses snapshots
8. **B019 manual read model rebuild** — crush-daily adds seconds to every startup
9. **C019 multiple repos for same aggregate** — browser-history wastes resources
10. **F001 tombstone soft-delete** — delete operations without audit trail

### Short-term (high value, moderate effort)

11. **B016 manual checkpoint replay** — browser-history reinvents projectionhost
12. **A020/A021 custom bus/store reimplementation** — accountability-system, Kernovia
13. **P007 bit-shift retry bug** — DiscordSync has a real bug
14. **D012 missing schema version stamping** — most projects
15. **B025 missing state cache** — most projects
16. **E010 event capture without validation** — DiscordSync pattern
17. **F003/F004 missing OTel/Prometheus** — most server projects
18. **T001 no scenario tests** — most projects with deciders
19. **C021 mutex held during decode** — crush-daily
20. **C022 context ignored in handler** — crush-daily

### Long-term (nice to have, ecosystem-wide)

21. **Feature adoption scorecard** — show what each project uses vs misses
22. **Profile command** — detailed usage analysis per project
23. **Incremental analysis** — cache AST results for faster re-runs
24. **cqrs-htmx-aware rules** — different defaults for framework consumers
25. **Domain-specific severity calibration** — stricter rules for financial/security domains
