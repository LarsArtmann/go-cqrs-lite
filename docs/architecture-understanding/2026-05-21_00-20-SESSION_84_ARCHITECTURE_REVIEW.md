# Architecture Review — Session 84

**Date:** 2026-05-21
**Reviewer:** AI Architecture Partner
**Scope:** Full codebase — 12 modules, 10,067 production LOC, 21,442 test LOC

---

## Executive Summary

go-cqrs-lite is a well-designed CQRS/ES library with strong type safety, clean module isolation, and 91.6% test coverage. The architecture follows a clear layered pattern with `core` as the zero-dependency foundation and 8 satellite modules. The `decider` package is the best-designed module — a textbook example of deep module design.

**Maturity: 8.5/10** — Production-quality library with clear improvement path.

### Key Metrics

| Metric               | Value                             |
| -------------------- | --------------------------------- |
| Modules              | 12 (9 production, 3 test/example) |
| Production LOC       | 10,067                            |
| Test LOC             | 21,442 (2.1:1 ratio)              |
| Test packages        | 24                                |
| Test coverage        | 91.6%                             |
| Lint issues          | 0                                 |
| Files > 250 lines    | 0                                 |
| Functions > 30 lines | ~2 (acceptable asyncapi Export)   |
| Deprecated exports   | 21                                |
| ADRs                 | 3 (all Accepted)                  |

---

## 1. Scalability & Modularity

### Strengths

1. **Multi-module monorepo (ADR-0003)** — Each module has its own `go.mod`. `core` has zero internal deps. Consumers import only what they need. This is the correct architecture for a library/SDK.

2. **Interface-first design** — All core abstractions are interfaces (`Store`, `Bus`, `Publisher`, `Subscriber`, `SnapshotStore`, etc.). The ISP split (Publisher/Subscriber composed into Bus) enables precise dependency injection.

3. **Generic dispatcher** — `dispatcher.Dispatcher[H, M]` eliminates duplication between command, query, and event dispatch. Middleware chaining, lifecycle management, and catalog integration are shared.

4. **Branded IDs** — Compile-time type safety prevents mixing `EventID` with `UserID`. The `id.Of[T]` type alias to `go-branded-id` eliminates delegation boilerplate.

5. **Module DAG is clean**:
   ```
   core → (no deps)
   memory → core
   middleware → core
   catalog → core
   projection → core
   testhelpers → core
   integration → core + memory + testhelpers
   storage → core
   sync → (no deps — fully isolated)
   ```

### Issues

1. **`sync` shadows stdlib** — The package name `sync` requires import aliases when using both. Minor but annoying for consumers.

2. **Middleware OTEL coupling** — `middleware/tracing.go` depends on `go.opentelemetry.io/otel`, forcing the transitive dependency on all middleware consumers. Should be a separate module (`middleware/tracing/`).

3. **No dependency injection for error classification** — Every package calls `event.RegisterClassification()` in `init()`. This creates hidden global side effects with no conflict detection.

---

## 2. Service Orientation & Composability

### Strengths

1. **Library, not framework** — No opinionated transport, message broker, or SQL driver. Consumers compose their own stack. This is the correct design for a reusable library.

2. **Decider pattern** — `Decider[State]` with pure functions (`Fold`, `DecideFunc`) is composable by nature. No mutable state, no infrastructure coupling. Testing is `Fold(initialState, event)` — zero setup.

3. **Time-travel queries** — `LoadAtVersion`, `LoadAtTimestamp`, `PositionalLoader` enable temporal queries and efficient catch-up. These compose naturally with the projection runner.

4. **Catalog auto-documentation** — Reflection-based schema generation + multi-format export (AsyncAPI, OpenAPI, D2, EventCatalog) is genuinely deep: small API surface, large behavioral complexity hidden.

### Issues

1. **`aggregate` package duplication** — 70% structural overlap with `decider`. Both handle: snapshot loading, outbox detection, transactional store, event persistence, snapshot strategy. ADR-0001 already recommends decider. The aggregate package should be formally deprecated.

2. **Serialization triplication** — Three near-identical event reconstruction paths in `storage/`:
   - `helpers.go:reconstructEvent` (SQL column values)
   - `outbox_helpers.go:reconstructOutboxEvent` (outbox DTO)
   - `pebble_serialization.go:deserializeEvent` (Pebble DTO)
     All three parse IDs, build options, call `event.NewEvent()`. One shared function could replace all three.

3. **HandlerRegistry duplicates MemoryBus** — Both independently implement type-specific + wildcard handler dispatch. The pattern should be shared.

4. **Middleware triple repetition** — 18 near-identical functions (6 concerns × 3 CQRS dimensions). This is a Go generics limitation (can't abstract over function signatures), but the maintenance burden is real.

---

## 3. Module Depth Assessment

| Module                | Depth      | Assessment                                                                            |
| --------------------- | ---------- | ------------------------------------------------------------------------------------- |
| `core/decider`        | ⭐⭐⭐⭐⭐ | Best module. Small interface, deep behavior (load→fold→decide→save→publish→snapshot). |
| `core/event`          | ⭐⭐⭐⭐   | Central, deep. ISP split is clean. Some dead weight (Builder, Catalogable).           |
| `catalog`             | ⭐⭐⭐⭐   | Reflection-based schema + multi-format export is genuinely sophisticated.             |
| `sync`                | ⭐⭐⭐⭐   | Vector clock + LWW resolver. Fully isolated, zero deps.                               |
| `storage`             | ⭐⭐⭐⭐   | Dialect abstraction is clean. Serialization duplication is the main issue.            |
| `projection`          | ⭐⭐⭐⭐   | Position-aware replay + retry. HandlerRegistry could be deeper.                       |
| `core/pkg/dispatcher` | ⭐⭐⭐⭐   | Generic dispatch infrastructure. Hidden gem.                                          |
| `core/pkg/id`         | ⭐⭐⭐     | Branded IDs work well. `Of[T]` is too abstract a name.                                |
| `memory`              | ⭐⭐⭐     | Production-quality in-memory implementations. Inconsistent constructor naming.        |
| `middleware`          | ⭐⭐⭐     | Shallow by design (thin wrappers). OTEL coupling is a concern.                        |
| `core/aggregate`      | ⭐⭐       | 70% duplicated with decider. Wide interface (9 methods). Should be deprecated.        |
| `core/command`        | ⭐⭐       | Thin wrapper around generic dispatcher. `Core` is a 2-field struct.                   |
| `core/query`          | ⭐⭐       | Same as command. `DispatchTyped[T]` is the only genuinely deep feature.               |

---

## 4. Coupling Analysis

### Hub: `core/event` (correct, inevitable)

Every module imports `core/event`. This is expected for an event-sourcing library. The `init()` registration pattern is the only concern — it creates invisible global state.

### Fan-in: `middleware` (correct)

Middleware imports `command`, `event`, and `query` — the only module that couples all three. Correct for cross-cutting concerns.

### Isolated: `sync` (perfect)

Zero imports from any other module. Only stdlib. The most isolated package. Could be extracted to its own repo without any changes.

### Potential circular: Error classification

`aggregate`, `projection`, `storage` all register sentinels in `event`'s global classifier. No circular deps (it's `init()`-based), but it's a hidden contract.

---

## 5. Recommendations (Prioritized)

### Critical (Should Fix Next)

| #   | Recommendation                                      | Impact | Effort | Breaking |
| --- | --------------------------------------------------- | ------ | ------ | -------- |
| 1   | Unify event serialization (3→1 reconstruction path) | High   | 3h     | No       |
| 2   | Remove deprecated catalog API (21 exports)          | High   | 2h     | Yes      |
| 3   | Rename `CQRSAdapter` → `PebbleEventStore`           | Medium | 1h     | Yes      |
| 4   | Extract `middleware/tracing` to separate module     | Medium | 1h     | No       |

### High Impact (Should Plan)

| #   | Recommendation                                               | Impact | Effort | Breaking |
| --- | ------------------------------------------------------------ | ------ | ------ | -------- |
| 5   | Formally deprecate `aggregate` package                       | High   | 2h     | Yes      |
| 6   | Rename `helpers.go` files to domain-specific names           | Low    | 1h     | No       |
| 7   | Rename `Mixin` suffixes to idiomatic Go names                | Low    | 30m    | Yes      |
| 8   | Fix inconsistent `memory/` constructor naming                | Low    | 30m    | Yes      |
| 9   | Share handler dispatch between HandlerRegistry and MemoryBus | Medium | 2h     | No       |

### Future (Consider for v2)

| #   | Recommendation                                          | Impact | Effort |
| --- | ------------------------------------------------------- | ------ | ------ |
| 10  | Move `sync` to its own repository                       | Low    | 1h     |
| 11  | Consolidate `Core` naming across packages               | Low    | 4h     |
| 12  | Replace `init()` error registration with explicit setup | Medium | 3h     |

---

## 6. Architecture Quality Score

| Dimension     | Score | Notes                                                            |
| ------------- | ----- | ---------------------------------------------------------------- |
| Modularity    | 9/10  | Clean DAG, multi-module, ISP                                     |
| Depth         | 8/10  | Decider and catalog are exemplary; command/query are thin        |
| Composability | 9/10  | Library-not-framework, interface-first                           |
| Type Safety   | 10/10 | Branded IDs, typed versions, error taxonomy                      |
| Test Quality  | 9/10  | 91.6% coverage, BDD + table-driven + benchmarks                  |
| Naming        | 7/10  | Generally good; CQRSAdapter, Mixin, helpers.go are weak spots    |
| Consistency   | 7/10  | Constructor naming varies; Core stamp coupling; deprecated drift |
| Documentation | 8/10  | CONTEXT.md, 3 ADRs, godoc on most exports                        |

**Overall: 8.5/10** — A well-crafted library with clear improvement path.
