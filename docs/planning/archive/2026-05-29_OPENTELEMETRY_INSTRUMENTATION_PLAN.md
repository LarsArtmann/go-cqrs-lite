# OpenTelemetry Instrumentation Plan

> **Status**: Planning
> **Date**: 2026-05-29
> **Goal**: Add superb, production-grade OpenTelemetry instrumentation across all go-cqrs-lite modules.

---

## Current State

| Module         | Tracing                                                      | Metrics                                       | Notes                                                              |
| -------------- | ------------------------------------------------------------ | --------------------------------------------- | ------------------------------------------------------------------ |
| `middleware/`  | Basic `CommandTracing`, `EventTracing`, `QueryTracing` spans | Custom `MetricsRecorder` interface (not OTel) | Only module with OTel deps (`go.opentelemetry.io/otel v1.43.0`)    |
| `core/`        | None                                                         | None                                          | No OTel deps                                                       |
| `memory/`      | None                                                         | None                                          | Test implementations — will stay clean                             |
| `storage/`     | None                                                         | None                                          | SQL/Pebble stores — **highest-value** instrumentation target       |
| `projection/`  | None                                                         | None                                          | Runner has replay+live loop — no spans                             |
| `saga/`        | None                                                         | None                                          | Runner, compensation, state machine — no spans                     |
| `stream/`      | None                                                         | None                                          | Aggregate reader — no telemetry                                    |
| `decider/`     | None                                                         | None                                          | Repository.Execute (load→fold→decide→save→publish) — critical path |
| `signing/`     | None                                                         | None                                          | Crypto operations — measurable latency                             |
| `watermill/`   | None                                                         | None                                          | Protocol adapter — no spans                                        |
| `testhelpers/` | None                                                         | None                                          | Test helpers — stays clean                                         |

---

## Architecture Decisions

### Where does OTel code live?

**Two-level approach — shared instrumentation library + per-module telemetry.**

- **Level 1 (New `otel/` module)**: Shared OTel utilities — tracer/meter helpers, semantic attribute constants, span naming conventions. Avoids duplicating OTel setup across modules.
- **Level 2 (Per-module)**: Each module with real I/O or measurable latency adds its own spans using `otel/`. Pure test helpers (`memory/`, `testhelpers/`) stay clean.

**Why a new `otel/` module?** Currently `middleware/` is the only module with OTel deps. If `storage/`, `projection/`, `saga/`, `decider/` all need OTel, they'd duplicate instrumentation names, attribute constants, and tracer/meter creation. A shared `otel/` module keeps things DRY.

### Tracer vs Meter

- **Tracing**: Every I/O-bound operation (store Save/Load, bus Publish, outbox Poll/Ack, projection replay, saga step execution, decider Execute).
- **Metrics**: Histograms for latency, counters for operations. Replace custom `MetricsRecorder` with native OTel `metric.Float64Histogram` + `metric.Int64Counter`.

### Semantic Conventions

Follow OTel messaging semantic conventions where applicable:

| Attribute                | Values                                  | Used By                    |
| ------------------------ | --------------------------------------- | -------------------------- |
| `messaging.system`       | `"go-cqrs-lite"`                        | All messaging operations   |
| `messaging.operation`    | `"publish"`, `"subscribe"`, `"process"` | Bus, projection            |
| `messaging.destination`  | Event type or aggregate type            | Bus, projection            |
| `cqrs.message.kind`      | `"command"`, `"event"`, `"query"`       | All handlers               |
| `cqrs.command.type`      | Command type string                     | Command middleware         |
| `cqrs.event.type`        | Event type string                       | Event middleware           |
| `cqrs.query.type`        | Query type string                       | Query middleware           |
| `cqrs.aggregate.type`    | Aggregate type string                   | Store, decider, projection |
| `cqrs.aggregate.id`      | Aggregate ULID                          | Store, decider             |
| `cqrs.aggregate.version` | Version int                             | Store, decider             |
| `cqrs.event.count`       | Number of events                        | Store, decider, projection |
| `cqrs.projection.name`   | Projection name                         | Projection runner          |
| `cqrs.saga.type`         | Saga type string                        | Saga runner                |
| `cqrs.saga.step`         | Step index                              | Saga runner                |

### Span Naming Convention

| Operation         | Span Name           | SpanKind   | Module     |
| ----------------- | ------------------- | ---------- | ---------- |
| Command dispatch  | `command.handle`    | `Server`   | middleware |
| Query dispatch    | `query.handle`      | `Server`   | middleware |
| Event publish     | `event.publish`     | `Producer` | middleware |
| Event handle      | `event.handle`      | `Consumer` | middleware |
| Event store Save  | `event.store.save`  | `Client`   | storage    |
| Event store Load  | `event.store.load`  | `Client`   | storage    |
| Outbox Append     | `outbox.append`     | `Client`   | storage    |
| Outbox Poll       | `outbox.poll`       | `Client`   | storage    |
| Outbox Ack        | `outbox.ack`        | `Client`   | storage    |
| Snapshot Save     | `snapshot.save`     | `Client`   | storage    |
| Snapshot Load     | `snapshot.load`     | `Client`   | storage    |
| Checkpoint Save   | `checkpoint.save`   | `Client`   | storage    |
| Checkpoint Load   | `checkpoint.load`   | `Client`   | storage    |
| Projection run    | `projection.run`    | `Client`   | projection |
| Projection replay | `projection.replay` | `Client`   | projection |
| Projection handle | `projection.handle` | `Consumer` | projection |
| Saga start        | `saga.start`        | `Client`   | saga       |
| Saga step execute | `saga.step.execute` | `Client`   | saga       |
| Saga compensate   | `saga.compensate`   | `Client`   | saga       |
| Decider execute   | `decider.execute`   | `Internal` | decider    |
| Decider load      | `decider.load`      | `Internal` | decider    |

### No Breaking Changes

- All OTel instrumentation is **opt-in**. If no TracerProvider is configured, `otel.Tracer("...")` returns a no-op tracer. Zero overhead when disabled.
- The `WithTracer()` option pattern (already used in middleware) is the extension point.
- Existing `MetricsRecorder` interface stays for backward compat; new OTel metrics added alongside.

---

## Execution Plan (8 Steps, Pareto-ordered)

### Phase 1: Foundation (80% of value — 20% of effort)

#### Step 1: Create `otel/` shared module

**What**: New Go module with shared OTel utilities.

**Files**:

- `otel/go.mod` — depends on `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/trace`, `go.opentelemetry.io/otel/metric`
- `otel/tracer.go` — `NewTracer(name string) trace.Tracer` helper
- `otel/meter.go` — `NewMeter(name string) metric.Meter` helper
- `otel/attributes.go` — Semantic attribute constants (`AttrCommandType`, `AttrEventType`, `AttrAggregateType`, `AttrAggregateID`, `AttrAggregateVersion`, `AttrEventCount`, etc.)
- `otel/spans.go` — Common span helpers (start span, record error with status, etc.)
- `otel/doc.go` — Package documentation
- `otel/instrumentation.go` — Instrumentation name constant, version info

**Why first**: Every subsequent step depends on this. No duplication of instrumentation names or attribute keys.

**Estimated LOC**: ~150

---

#### Step 2: Enhance existing `middleware/tracing.go`

**What**: Upgrade the existing tracing middleware with richer attributes and publish-side tracing.

**Changes**:

- Add `cqrs.aggregate.id`, `cqrs.aggregate.type` attributes to existing command/event/query spans (extracted from metadata)
- Add `cqrs.aggregate.version` for events
- Use semantic constants from `otel/` module
- Add `EventPublishTracing(tracer)` — new `PublishMiddleware` that creates `event.publish` spans (`Producer` kind) with event count and event types
- Add `WithTracer()` option to middleware config (optional override, defaults to global provider)

**Files modified**:

- `middleware/tracing.go`
- `middleware/options.go`
- `middleware/go.mod` (add `otel/` dependency)

**Estimated LOC**: ~40 new, ~30 modified

---

#### Step 3: Add `middleware/metrics_otel.go` — Native OTel Metrics

**What**: OTel-native metrics implementation alongside existing `MetricsRecorder`.

**Files**:

- `middleware/metrics_otel.go` — New file with:
  - `OTelMetricsRecorder` struct implementing `MetricsRecorder` using `metric.Float64Histogram`
  - `NewOTelMetricsRecorder(meter metric.Meter) (*OTelMetricsRecorder, error)` constructor
  - `CommandOTelMetrics(meter)`, `EventOTelMetrics(meter)`, `QueryOTelMetrics(meter)` middleware factories
  - Instruments: `cqrs.command.duration`, `cqrs.event.duration`, `cqrs.query.duration`
  - Attributes: `type`, `status` (success/error)

**Backward compatibility**: Keep existing `MetricsRecorder` interface + `CommandMetrics`/`EventMetrics`/`QueryMetrics`. Consumers can choose.

**Estimated LOC**: ~120

---

### Phase 2: Storage Layer (High value — visible latency)

#### Step 4: Instrument `storage/` module

**What**: Add spans for all EventStore, Outbox, Snapshot, and Checkpoint operations.

**Changes per store**:

`SQLEventStore`:

- `Save` → `event.store.save` span with `cqrs.aggregate.*` attributes + `cqrs.event.count`
- `Load` / `LoadFromVersion` / `LoadToVersion` / `LoadToTimestamp` → `event.store.load` span
- `AppendBatch` → `event.store.append_batch` span
- `LoadBackwards` → `event.store.load_backwards` span
- `ReadAll` / `ReadFrom` (Journal) → `event.store.read_all` / `event.store.read_from` span

`PebbleEventStore`:

- Same span pattern as SQL store

`SQLOutbox`:

- `Append` → `outbox.append` span
- `PollPending` → `outbox.poll` span with `cqrs.outbox.entries_count`
- `Ack` → `outbox.ack` span with `cqrs.outbox.entries_count`

`SQLSnapshotStore`:

- `Save` → `snapshot.save` span
- `Load` → `snapshot.load` span

`SQLCheckpointStore`:

- `Save` → `checkpoint.save` span
- `Load` → `checkpoint.load` span

**Wiring**: Add `WithTracer(trace.Tracer)` option to store constructors (optional, defaults to global tracer).

**Files modified**: ~6 store files, `options.go`, `go.mod`
**Files added**: `otel.go` (tracer setup helper for storage)

**Estimated LOC**: ~200

---

### Phase 3: Domain Orchestration (Critical business paths)

#### Step 5: Instrument `decider/` Repository

**What**: Add spans to `Execute` and `Load` methods.

**Changes**:

- `Execute` → `decider.execute` span (Internal kind) with child spans:
  - `decider.load` — load + fold state
  - `decider.decide` — call decide function (links to parent command span)
  - `decider.save` — persist events
  - `decider.publish` — publish events
- `Load` → `decider.load` span
- Attributes: `cqrs.aggregate.type`, `cqrs.aggregate.id`, `cqrs.event.count` (produced events)
- Tracer injected via `WithTracer()` `RepositoryOption`

**Files modified**: `decider.go`, `options.go`, `go.mod`
**Files added**: `otel.go`

**Estimated LOC**: ~100

---

#### Step 6: Instrument `projection/` Runner

**What**: Add spans for replay and live event handling.

**Changes**:

- `Run` → `projection.run` span (long-lived, may span the entire application lifecycle)
- `replay()` → `projection.replay` span per projection with `cqrs.projection.name`, `cqrs.event.count`
- `handleAndCheckpoint` → `projection.handle` span (Consumer kind) with `cqrs.event.type`, `cqrs.projection.name`
- `subscribeLive` → `projection.subscribe` span
- Add `WithTracer()` `RunnerOption`

**Files modified**: `runner.go`, `runner_live.go`, `options.go`, `go.mod`
**Files added**: `otel.go`

**Estimated LOC**: ~120

---

#### Step 7: Instrument `saga/` Runner

**What**: Add spans for saga lifecycle operations.

**Changes**:

- `Start` → `saga.start` span with `cqrs.saga.type`
- `HandleEvent` / step execution → `saga.step.execute` with `cqrs.saga.type`, `cqrs.saga.step`
- `Compensate` → `saga.compensate` with `cqrs.saga.type`
- Add `WithTracer()` `RunnerOption`

**Files modified**: `runner.go`, `runner_execute.go`, `options.go`, `go.mod`
**Files added**: `otel.go`

**Estimated LOC**: ~100

---

### Phase 4: Polish & Testing

#### Step 8: Integration tests + documentation

**What**: Verify all instrumentation works end-to-end.

**Test plan**:

- Add OTel SDK test setup (`TracerProvider` + `SpanRecorder`) to `integration/` module
- Integration test: command dispatch → decider execute → store save → bus publish → projection handle — all linked via trace propagation
- Verify span hierarchy, attributes, and error recording
- Per-module unit tests for new instrumentation (using existing `SpanRecorder` pattern from `middleware/tracing_test.go`)

**Documentation**:

- Update `AGENTS.md` with `otel` module in module list and dependency graph
- Update `FEATURES.md` (if exists) with observability feature
- Add example showing how to wire up OTel provider with the library
- Add ADR for instrumentation architecture decisions

**Files added**: ~2 test files, ~1 doc update
**Estimated LOC**: ~200

---

## Module Dependency Impact (After)

```
otel/ (NEW)           → go.opentelemetry.io/otel, otel/trace, otel/metric
middleware/            → otel/ + core + testhelpers (already has otel deps)
storage/              → otel/ + core + saga
projection/           → otel/ + core + memory + testhelpers
saga/                 → otel/ + core
decider/              → otel/ + core
integration/          → otel/ + core + memory + testhelpers

Unchanged (no OTel):
core/                 → stays dependency-free
memory/               → stays clean (test impl)
testhelpers/          → stays clean (test helpers)
signing/              → low priority, crypto is fast
stream/               → low priority, reads
watermill/            → low priority, protocol adapter
catalog/              → no I/O
```

## Risk Mitigation

| Risk                             | Mitigation                                                                                        |
| -------------------------------- | ------------------------------------------------------------------------------------------------- |
| OTel dependency bloat in `core/` | `otel/` is a separate module; `core/` stays dependency-free. Modules opt-in by importing `otel/`. |
| Breaking changes                 | All instrumentation is additive — `WithTracer()` options, no API changes.                         |
| Performance overhead             | No-op tracer when no provider configured. Zero alloc on the happy path.                           |
| Test complexity                  | OTel SDK provides `SpanRecorder` for deterministic assertions (already used in `middleware/`).    |
| Dependency version conflicts     | Single `otel/` module controls OTel version; other modules inherit via `otel/`.                   |

## Estimated Scope

| Step                          | New Files | Modified Files | LOC (est.) |
| ----------------------------- | --------- | -------------- | ---------- |
| 1. `otel/` module             | 5         | 0              | ~150       |
| 2. Enhanced tracing           | 0         | 3              | ~70        |
| 3. OTel metrics               | 1         | 0              | ~120       |
| 4. Storage instrumentation    | 1         | 7              | ~200       |
| 5. Decider instrumentation    | 1         | 3              | ~100       |
| 6. Projection instrumentation | 1         | 4              | ~120       |
| 7. Saga instrumentation       | 1         | 3              | ~100       |
| 8. Tests + docs               | 3         | 2              | ~200       |
| **Total**                     | **13**    | **22**         | **~1,060** |

## Execution Order (Dependency Graph)

```
Step 1 (otel/ module)
├── Step 2 (enhanced middleware tracing)
├── Step 3 (OTel metrics middleware)
├── Step 4 (storage instrumentation)
├── Step 5 (decider instrumentation)
├── Step 6 (projection instrumentation)
└── Step 7 (saga instrumentation)
    └── Step 8 (integration tests + docs)
```

Steps 2-7 can proceed in parallel after Step 1. Step 8 requires all prior steps.
