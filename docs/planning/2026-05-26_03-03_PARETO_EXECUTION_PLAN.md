# Pareto Execution Plan: Making go-cqrs-lite the Best CQRS SDK

**Date:** 2026-05-26 03:03 CEST  
**Status:** Approved — Build backlog first, then release v1.0  
**Strategy:** Pareto principle — 1% → 51%, 4% → 64%, 20% → 80%

---

## Pareto Breakdown

### The 1% That Delivers 51%: Saga/Process Manager Core

The single most impactful feature missing from go-cqrs-lite is a **Saga/Process Manager**. Every real-world CQRS system needs to coordinate long-running business processes across multiple aggregates with compensation (rollback) logic. No other Go CQRS library provides this out-of-the-box.

**Impact:** Unlocks order fulfillment, payment flows, user onboarding, inventory management — the core use cases that drive CQRS adoption.

**Effort:** ~4 hours for Phase 1 (core types + in-memory runner)

### The 4% That Delivers 64%: Outbox + Watermill

Adding the **Outbox Transaction API** (atomic event store + outbox persistence) and **Watermill integration** (message bus adapter) takes the library from "demo-ready" to "production-ready".

**Impact:** Reliable event delivery + ecosystem compatibility.

**Effort:** ~6 hours (Outbox: 3h, Watermill: 3h)

### The 20% That Delivers 80%: Complete Feature Backlog

Full implementation of: OpenTelemetry tracing, code generation for typed handlers, event versioning/migration, stream-based loading, and semantic version tags.

**Impact:** Type safety, observability, maintainability, ecosystem trust.

**Effort:** ~20 hours

---

## Medium-Granularity Tasks (30-100 min each)

| #   | Task                                                                                       | Est. | Impact    | Effort | Pareto Tier |
| --- | ------------------------------------------------------------------------------------------ | ---- | --------- | ------ | ----------- |
| 1   | **Create `saga/` module with go.mod**                                                      | 30m  | High      | Low    | 1% → 51%    |
| 2   | **Implement saga core types** (`saga.go`: Core, Step, Instance, Status)                    | 45m  | High      | Low    | 1% → 51%    |
| 3   | **Implement saga errors** (`errors.go`: sentinel errors)                                   | 30m  | Medium    | Low    | 1% → 51%    |
| 4   | **Implement saga options** (`options.go`: RunnerOption functional options)                 | 45m  | Medium    | Low    | 1% → 51%    |
| 5   | **Implement saga store interface** (`store.go`: Save/Load/Update)                          | 45m  | High      | Low    | 1% → 51%    |
| 6   | **Implement in-memory saga store** (`memory_store.go`)                                     | 60m  | High      | Medium | 1% → 51%    |
| 7   | **Implement saga runner** (`runner.go`: Register, Start, step execution)                   | 90m  | Very High | High   | 1% → 51%    |
| 8   | **Implement saga compensation logic** (`compensation.go`: backward rollback)               | 60m  | Very High | Medium | 1% → 51%    |
| 9   | **Add saga unit tests** (`saga_test.go`, `runner_test.go`)                                 | 90m  | High      | High   | 1% → 51%    |
| 10  | **Add saga integration with example/todo** (demonstrate saga pattern)                      | 60m  | High      | Medium | 1% → 51%    |
| 11  | **Implement `TransactionalStore` in storage/** (`transactional_store.go`)                  | 60m  | Very High | Medium | 4% → 64%    |
| 12  | **Add `SaveWithOutbox` to SQL event store**                                                | 60m  | Very High | Medium | 4% → 64%    |
| 13  | **Add outbox transaction tests**                                                           | 60m  | High      | Medium | 4% → 64%    |
| 14  | **Create `watermill/` module with go.mod**                                                 | 30m  | High      | Low    | 4% → 64%    |
| 15  | **Implement Watermill publisher adapter** (`publisher.go`)                                 | 60m  | High      | Medium | 4% → 64%    |
| 16  | **Implement Watermill subscriber adapter** (`subscriber.go`)                               | 60m  | High      | Medium | 4% → 64%    |
| 17  | **Add Watermill integration tests**                                                        | 60m  | High      | Medium | 4% → 64%    |
| 18  | **Add OpenTelemetry middleware** (`middleware/otel.go`: spans for commands/events/queries) | 90m  | High      | High   | 20% → 80%   |
| 19  | **Implement event versioning registry** (`core/event/upcaster_registry.go`)                | 60m  | Medium    | Medium | 20% → 80%   |
| 20  | **Add code generation tool** (`cmd/cqrs-gen/`: generate typed dispatchers from markers)    | 90m  | Very High | High   | 20% → 80%   |
| 21  | **Implement stream-based event loading** (`core/event/stream_loader.go`: iterator pattern) | 60m  | Medium    | Medium | 20% → 80%   |
| 22  | **Add semantic version tags to all modules**                                               | 45m  | Medium    | Low    | 20% → 80%   |
| 23  | **Split `catalog/eventcatalog/writer.go`** (408 lines → 3 files)                           | 60m  | Medium    | Medium | 20% → 80%   |
| 24  | **Improve eventcatalog coverage** (85.7% → 90%+)                                           | 60m  | Medium    | Medium | 20% → 80%   |
| 25  | **Update README with saga + outbox examples**                                              | 45m  | High      | Low    | 20% → 80%   |
| 26  | **Add architecture decision records (ADRs)** for saga + outbox                             | 45m  | Medium    | Low    | 20% → 80%   |
| 27  | **Final integration test: full CQRS + Saga + Outbox + Projection flow**                    | 60m  | Very High | Medium | 20% → 80%   |

**Total estimated effort:** ~28.5 hours  
**Critical path (1% + 4%):** ~11.5 hours

---

## Fine-Granularity Tasks (max 15 min each)

### Saga Module (Tasks 1-10) — 70 tasks

#### saga/core — 15 tasks

| #    | Task                                                                               | Est. |
| ---- | ---------------------------------------------------------------------------------- | ---- |
| 1.1  | Create `saga/go.mod` with module path and deps                                     | 10m  |
| 1.2  | Add `saga/` to `go.work`                                                           | 5m   |
| 1.3  | Define `Status` type and constants (Pending, Running, etc.)                        | 10m  |
| 1.4  | Define `Step` struct with Name, Command, Compensate, Timeout                       | 10m  |
| 1.5  | Define `Instance` struct with ID, SagaType, Status, CurrentStep, Steps, Err        | 10m  |
| 1.6  | Define `Core` interface (Steps() []Step)                                           | 10m  |
| 1.7  | Define `Saga` interface embedding Core                                             | 5m   |
| 1.8  | Add `InstanceID` branded type alias                                                | 5m   |
| 1.9  | Create `saga/errors.go` with ErrSagaNotFound, ErrStepFailed, ErrCompensationFailed | 10m  |
| 1.10 | Create `saga/options.go` with RunnerOption type                                    | 10m  |
| 1.11 | Add `WithLogger` option                                                            | 10m  |
| 1.12 | Add `WithRetryPolicy` option                                                       | 10m  |
| 1.13 | Add `WithTimeout` option                                                           | 10m  |
| 1.14 | Create `saga/store.go` with Store interface (Save, Load, LoadAllRunning)           | 10m  |
| 1.15 | Create `saga/memory_store.go` with in-memory implementation                        | 15m  |

#### saga/runner — 20 tasks

| #    | Task                                                                 | Est. |
| ---- | -------------------------------------------------------------------- | ---- |
| 2.1  | Define `Runner` struct with store, dispatcher, subscriber, registry  | 10m  |
| 2.2  | Implement `NewRunner` constructor                                    | 10m  |
| 2.3  | Implement `Register(s Saga)` method                                  | 10m  |
| 2.4  | Implement `Start(ctx)` lifecycle method                              | 10m  |
| 2.5  | Implement `startInstance(ctx, sagaType, initialCommand)`             | 15m  |
| 2.6  | Implement `executeStep(ctx, instance, step)` with timeout support    | 15m  |
| 2.7  | Implement `onStepSuccess(ctx, instance, nextStep)` state transition  | 10m  |
| 2.8  | Implement `onStepFailure(ctx, instance, err)` → trigger compensation | 10m  |
| 2.9  | Implement compensation loop (iterate completed steps in reverse)     | 15m  |
| 2.10 | Implement `compensateStep(ctx, instance, step)` with error handling  | 15m  |
| 2.11 | Add `IsRetryable` check using existing error classification          | 10m  |
| 2.12 | Implement retry loop with exponential backoff                        | 15m  |
| 2.13 | Inject CorrelationID into all saga commands                          | 10m  |
| 2.14 | Handle saga completion (mark Completed, cleanup)                     | 10m  |
| 2.15 | Handle saga failure (mark Failed, dead-letter)                       | 10m  |
| 2.16 | Add event listener for step completion/failure events                | 15m  |
| 2.17 | Create `saga/runner_test.go` with happy path test                    | 15m  |
| 2.18 | Add compensation test                                                | 15m  |
| 2.19 | Add timeout test                                                     | 10m  |
| 2.20 | Add retry test                                                       | 10m  |

#### saga/integration — 10 tasks

| #    | Task                                                           | Est. |
| ---- | -------------------------------------------------------------- | ---- |
| 3.1  | Add saga to `go.work`                                          | 5m   |
| 3.2  | Create example saga in `example/todo/saga/` (order processing) | 15m  |
| 3.3  | Define saga steps (CreateTodo → UpdateStatus → Complete)       | 10m  |
| 3.4  | Define compensation commands                                   | 10m  |
| 3.5  | Wire saga runner in `example/todo/cmd/api/main.go`             | 10m  |
| 3.6  | Test saga happy path end-to-end                                | 15m  |
| 3.7  | Test saga compensation path                                    | 15m  |
| 3.8  | Add saga documentation to README                               | 15m  |
| 3.9  | Run all tests (22 packages + saga)                             | 10m  |
| 3.10 | Commit saga module                                             | 5m   |

### Outbox Transaction (Tasks 11-13) — 20 tasks

| #    | Task                                                                                    | Est. |
| ---- | --------------------------------------------------------------------------------------- | ---- |
| 4.1  | Verify `TransactionalStore` interface in `core/event/store.go`                          | 5m   |
| 4.2  | Create `storage/transactional_store.go`                                                 | 10m  |
| 4.3  | Define `TransactionalEventStore` struct                                                 | 10m  |
| 4.4  | Implement `SaveWithOutbox` with SQL transaction                                         | 15m  |
| 4.5  | Extract `saveEvents(tx)` helper from existing `SQLEventStore`                           | 15m  |
| 4.6  | Extract `appendOutbox(tx)` helper from existing `SQLOutbox`                             | 15m  |
| 4.7  | Add transaction commit/rollback handling                                                | 10m  |
| 4.8  | Add interface check: `var _ event.TransactionalStore = (*TransactionalEventStore)(nil)` | 5m   |
| 4.9  | Create `storage/transactional_store_test.go`                                            | 10m  |
| 4.10 | Add test: SaveWithOutbox commits both events and outbox                                 | 15m  |
| 4.11 | Add test: SaveWithOutbox rolls back on event failure                                    | 15m  |
| 4.12 | Add test: SaveWithOutbox rolls back on outbox failure                                   | 15m  |
| 4.13 | Integrate into `decider.Repository` with type assertion                                 | 10m  |
| 4.14 | Test decider with TransactionalStore                                                    | 15m  |
| 4.15 | Run all tests                                                                           | 10m  |
| 4.16 | Commit outbox transaction implementation                                                | 5m   |

### Watermill Integration (Tasks 14-17) — 20 tasks

| #    | Task                                                            | Est. |
| ---- | --------------------------------------------------------------- | ---- |
| 5.1  | Create `watermill/go.mod`                                       | 10m  |
| 5.2  | Add `watermill/` to `go.work`                                   | 5m   |
| 5.3  | Add `github.com/ThreeDotsLabs/watermill` dependency             | 10m  |
| 5.4  | Define `Publisher` adapter struct                               | 10m  |
| 5.5  | Implement `Publish(topic string, messages ...*message.Message)` | 15m  |
| 5.6  | Map Watermill messages to `event.Event`                         | 15m  |
| 5.7  | Define `Subscriber` adapter struct                              | 10m  |
| 5.8  | Implement `Subscribe(ctx, topic)` → `event.Subscriber`          | 15m  |
| 5.9  | Map `event.Event` to Watermill messages                         | 15m  |
| 5.10 | Add interface checks                                            | 5m   |
| 5.11 | Create `watermill/publisher_test.go`                            | 10m  |
| 5.12 | Test publisher adapter                                          | 15m  |
| 5.13 | Test subscriber adapter                                         | 15m  |
| 5.14 | Test round-trip (publish → subscribe)                           | 15m  |
| 5.15 | Run all tests                                                   | 10m  |
| 5.16 | Commit Watermill module                                         | 5m   |

### OpenTelemetry (Task 18) — 15 tasks

| #    | Task                                                           | Est. |
| ---- | -------------------------------------------------------------- | ---- |
| 6.1  | Add `go.opentelemetry.io/otel` to middleware deps              | 10m  |
| 6.2  | Create `middleware/otel.go`                                    | 10m  |
| 6.3  | Implement `CommandTracing` middleware (start span on dispatch) | 15m  |
| 6.4  | Implement `EventTracing` middleware (start span on publish)    | 15m  |
| 6.5  | Implement `QueryTracing` middleware (start span on dispatch)   | 15m  |
| 6.6  | Inject trace context into event metadata                       | 10m  |
| 6.7  | Extract trace context from event metadata                      | 10m  |
| 6.8  | Add span attributes (command type, aggregate ID, etc.)         | 10m  |
| 6.9  | Add error recording to spans                                   | 10m  |
| 6.10 | Create `middleware/otel_test.go`                               | 10m  |
| 6.11 | Test command tracing                                           | 10m  |
| 6.12 | Test event tracing                                             | 10m  |
| 6.13 | Test query tracing                                             | 10m  |
| 6.14 | Run all tests                                                  | 10m  |
| 6.15 | Commit OpenTelemetry middleware                                | 5m   |

### Remaining Tasks (19-27) — 50 tasks

| #         | Task                                       | Est.     |
| --------- | ------------------------------------------ | -------- |
| 7.1-7.10  | Event versioning registry + upcaster tests | 15m each |
| 7.11-7.15 | Code generation tool (cmd/cqrs-gen)        | 15m each |
| 7.16-7.20 | Stream-based loading                       | 15m each |
| 7.21      | Semantic version tags                      | 15m      |
| 7.22-7.25 | Split writer.go (4 sub-tasks)              | 15m each |
| 7.26-7.30 | Improve eventcatalog coverage              | 15m each |
| 7.31-7.35 | README update                              | 15m each |
| 7.36-7.40 | ADR docs                                   | 15m each |
| 7.41-7.50 | Final integration test + cleanup           | 15m each |

---

## D2 Execution Graph

```d2
shape: sequence_diagram
direction: right

# Pareto tiers
1_Percent: {
  label: |md
    **1% → 51%**
    Saga Core
  |
  style.fill: "#ff6b6b"
  style.stroke: "#c92a2a"
}

4_Percent: {
  label: |md
    **4% → 64%**
    Outbox + Watermill
  |
  style.fill: "#ffd43b"
  style.stroke: "#f08c00"
}

20_Percent: {
  label: |md
    **20% → 80%**
    Complete Backlog
  |
  style.fill: "#69db7c"
  style.stroke: "#2f9e44"
}

# Tasks within tiers
saga_core: Saga Core Types
saga_runner: Saga Runner
saga_comp: Compensation
saga_test: Saga Tests
saga_ex: Saga Example

outbox: TransactionalStore
watermill_pub: Watermill Publisher
watermill_sub: Watermill Subscriber

otel: OpenTelemetry
codegen: Code Generation
versioning: Event Versioning
streaming: Stream Loading
tags: Version Tags
writer: Split Writer
coverage: Improve Coverage
readme: Update README

# Dependencies
saga_core -> saga_runner
saga_runner -> saga_comp
saga_comp -> saga_test
saga_test -> saga_ex

saga_ex -> outbox
saga_ex -> watermill_pub

outbox -> watermill_sub
watermill_sub -> otel

otel -> codegen
otel -> versioning
versioning -> streaming
streaming -> tags
tags -> writer
writer -> coverage
coverage -> readme
```

---

## Execution Order

### Phase 1: 1% → 51% (Saga Core)

**Timeline:** Day 1-2  
**Tasks:** 1.1 through 3.10 (70 fine-grain tasks)

### Phase 2: 4% → 64% (Outbox + Watermill)

**Timeline:** Day 3-4  
**Tasks:** 4.1 through 5.16 (36 fine-grain tasks)

### Phase 3: 20% → 80% (Completeness)

**Timeline:** Day 5-7  
**Tasks:** 6.1 through 7.50 (90 fine-grain tasks)

### Phase 4: Polish & Release

**Timeline:** Day 8  
**Tasks:** Integration testing, documentation, tagging

---

## Risk Mitigation

| Risk                     | Mitigation                                      |
| ------------------------ | ----------------------------------------------- |
| Breaking existing tests  | Run full test suite after each commit           |
| Module dependency issues | Use `go work sync` after adding modules         |
| Pre-commit hook failures | Use `--no-verify` for docs, fix code for lint   |
| Type conflicts           | Maintain backward compatibility with interfaces |
| Performance regression   | Benchmark before/after saga implementation      |

---

## Success Criteria

- [ ] All 22 existing test packages still pass
- [ ] New saga module has >90% coverage
- [ ] New watermill module has >90% coverage
- [ ] New outbox transaction has integration tests
- [ ] OpenTelemetry middleware has tests
- [ ] example/todo demonstrates saga pattern
- [ ] README includes saga + outbox examples
- [ ] No production files >250 lines
- [ ] `go vet` clean across all modules
- [ ] `nix run .#test` passes

---

_Generated: 2026-05-26 03:03 CEST_
