# CQRS-Lite Implementation Plan

> A lightweight, production-ready CQRS library extracted from 4 battle-tested Go projects.

**Total Estimated Time:** 16-20 hours
**Last Updated:** 2026-03-15

---

## Prioritization Framework

Tasks sorted by: **Importance × Impact / Effort = Priority Score**

| Factor         | Description                                     |
| -------------- | ----------------------------------------------- |
| **Importance** | 1-5 (How critical is this for the core value?)  |
| **Impact**     | 1-5 (How much value does this deliver?)         |
| **Effort**     | 1-5 (How long does it take? 1=quick, 5=lengthy) |

---

## Phase 1: Foundation Layer (Est. 4-5 hours)

Core abstractions that everything else builds upon.

### 1.1 Core Types & Interfaces

| #     | Task                                                                                              | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 1.1.1 | Create `go.mod` with minimal dependencies (google/uuid, cockroachdb/errors)                       | 5          | 5      | 1      | 25    | 5min  |
| 1.1.2 | Create `event/event.go` - Event interface, BaseEvent struct, EventType, AggregateType             | 5          | 5      | 2      | 12.5  | 10min |
| 1.1.3 | Create `event/metadata.go` - EventMetadata struct (correlationId, causationId, userId, timestamp) | 5          | 4      | 2      | 10    | 10min |
| 1.1.4 | Create `event/errors.go` - Typed errors (ErrEventNotFound, ErrVersionConflict, etc.)              | 4          | 3      | 1      | 12    | 5min  |
| 1.1.5 | Create `command/command.go` - Command interface, BaseCommand struct                               | 5          | 5      | 2      | 12.5  | 10min |
| 1.1.6 | Create `command/errors.go` - Typed errors (ErrHandlerNotFound, ErrValidation, etc.)               | 4          | 3      | 1      | 12    | 5min  |
| 1.1.7 | Create `query/query.go` - Query interface, BaseQuery struct, Pagination                           | 5          | 5      | 2      | 12.5  | 10min |
| 1.1.8 | Create `query/errors.go` - Typed errors (ErrQueryNotFound, etc.)                                  | 4          | 3      | 1      | 12    | 5min  |

### 1.2 Aggregate Pattern

| #     | Task                                                                        | Importance | Impact | Effort | Score | Est.  |
| ----- | --------------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 1.2.1 | Create `aggregate/aggregate.go` - Aggregate interface, BaseAggregate struct | 5          | 4      | 2      | 10    | 10min |
| 1.2.2 | Create `aggregate/repository.go` - Repository interface                     | 5          | 4      | 2      | 10    | 10min |

---

## Phase 2: Event Layer (Est. 3-4 hours)

Event sourcing and pub/sub infrastructure.

### 2.1 Event Store Interface

| #     | Task                                                                                | Importance | Impact | Effort | Score | Est.  |
| ----- | ----------------------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 2.1.1 | Create `event/store.go` - Store interface (Append, GetByAggregate, GetGlobalStream) | 5          | 5      | 2      | 12.5  | 12min |
| 2.1.2 | Create `event/store.go` - Add GetByID, GetLatestVersion, Count methods              | 4          | 4      | 1      | 16    | 8min  |
| 2.1.3 | Create `event/snapshot.go` - Snapshot struct, SnapshotStore interface               | 4          | 4      | 2      | 8     | 10min |
| 2.1.4 | Create `event/store.go` - AppendBatch method for bulk imports                       | 3          | 4      | 2      | 6     | 10min |

### 2.2 Event Bus

| #     | Task                                                                     | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------------------ | ---------- | ------ | ------ | ----- | ----- |
| 2.2.1 | Create `event/bus.go` - Handler type, EventBus struct                    | 5          | 5      | 2      | 12.5  | 10min |
| 2.2.2 | Create `event/bus.go` - Subscribe, Publish methods                       | 5          | 5      | 2      | 12.5  | 10min |
| 2.2.3 | Create `event/bus.go` - PublishSync, PublishAsync (concurrent execution) | 4          | 4      | 2      | 8     | 10min |
| 2.2.4 | Create `event/bus.go` - Unsubscribe, HasSubscribers, SubscriberCount     | 3          | 3      | 1      | 9     | 8min  |

---

## Phase 3: Command Layer (Est. 2-3 hours)

Command dispatch and handling.

### 3.1 Command Dispatcher

| #     | Task                                                                              | Importance | Impact | Effort | Score | Est.  |
| ----- | --------------------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 3.1.1 | Create `command/handler.go` - Handler interface                                   | 5          | 5      | 1      | 25    | 5min  |
| 3.1.2 | Create `command/dispatcher.go` - Dispatcher interface, DispatcherImpl struct      | 5          | 5      | 2      | 12.5  | 10min |
| 3.1.3 | Create `command/dispatcher.go` - Register method (type-safe handler registration) | 5          | 5      | 2      | 12.5  | 12min |
| 3.1.4 | Create `command/dispatcher.go` - Dispatch method with context                     | 5          | 5      | 2      | 12.5  | 10min |
| 3.1.5 | Create `command/dispatcher.go` - HasHandler, RegisteredCommands helpers           | 3          | 3      | 1      | 9     | 5min  |

---

## Phase 4: Query Layer (Est. 2-3 hours)

Query dispatch and handling.

### 4.1 Query Dispatcher

| #     | Task                                                                       | Importance | Impact | Effort | Score | Est.  |
| ----- | -------------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 4.1.1 | Create `query/handler.go` - Handler interface                              | 5          | 5      | 1      | 25    | 5min  |
| 4.1.2 | Create `query/dispatcher.go` - Dispatcher interface, DispatcherImpl struct | 5          | 5      | 2      | 12.5  | 10min |
| 4.1.3 | Create `query/dispatcher.go` - Register method                             | 5          | 5      | 2      | 12.5  | 12min |
| 4.1.4 | Create `query/dispatcher.go` - Dispatch method with context and pagination | 5          | 5      | 2      | 12.5  | 10min |
| 4.1.5 | Create `query/pagination.go` - Pagination, PageResult types                | 4          | 4      | 1      | 16    | 8min  |

---

## Phase 5: Middleware (Est. 3-4 hours)

Cross-cutting concerns.

### 5.1 Command Middleware

| #     | Task                                                                     | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------------------ | ---------- | ------ | ------ | ----- | ----- |
| 5.1.1 | Create `command/middleware.go` - Middleware type (func(Handler) Handler) | 5          | 4      | 1      | 20    | 5min  |
| 5.1.2 | Create `middleware/logging.go` - LoggingMiddleware for commands          | 4          | 4      | 2      | 8     | 10min |
| 5.1.3 | Create `middleware/recovery.go` - RecoveryMiddleware for panic recovery  | 5          | 4      | 2      | 10    | 10min |
| 5.1.4 | Create `middleware/validation.go` - ValidationMiddleware interface       | 4          | 4      | 2      | 8     | 10min |
| 5.1.5 | Create `middleware/retry.go` - RetryMiddleware with exponential backoff  | 3          | 4      | 2      | 6     | 12min |

### 5.2 Event Middleware

| #     | Task                                                                | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 5.2.1 | Create `event/middleware.go` - Middleware type for event handlers   | 4          | 4      | 1      | 16    | 5min  |
| 5.2.2 | Create `middleware/event_logging.go` - LoggingMiddleware for events | 4          | 4      | 2      | 8     | 10min |
| 5.2.3 | Create `middleware/metrics.go` - Basic metrics collection           | 3          | 4      | 2      | 6     | 12min |

---

## Phase 6: In-Memory Implementations (Est. 2 hours)

For testing and simple use cases.

### 6.1 In-Memory Store & Bus

| #     | Task                                                                      | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 6.1.1 | Create `event/memory_store.go` - In-memory EventStore implementation      | 4          | 4      | 2      | 8     | 12min |
| 6.1.2 | Create `event/memory_store.go` - Thread-safe operations with sync.RWMutex | 4          | 4      | 2      | 8     | 10min |
| 6.1.3 | Create `event/memory_store_test.go` - Unit tests for memory store         | 4          | 3      | 2      | 6     | 12min |
| 6.1.4 | Create `event/bus_test.go` - Unit tests for event bus                     | 4          | 3      | 2      | 6     | 12min |

---

## Phase 7: Tests & Examples (Est. 3-4 hours)

Comprehensive test coverage and usage examples.

### 7.1 Core Tests

| #     | Task                                                             | Importance | Impact | Effort | Score | Est.  |
| ----- | ---------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 7.1.1 | Create `command/dispatcher_test.go` - Dispatcher unit tests      | 5          | 4      | 2      | 10    | 12min |
| 7.1.2 | Create `query/dispatcher_test.go` - Dispatcher unit tests        | 5          | 4      | 2      | 10    | 12min |
| 7.1.3 | Create `aggregate/aggregate_test.go` - Aggregate tests           | 4          | 4      | 2      | 8     | 10min |
| 7.1.4 | Create `middleware/logging_test.go` - Logging middleware tests   | 3          | 3      | 2      | 4.5   | 10min |
| 7.1.5 | Create `middleware/recovery_test.go` - Recovery middleware tests | 4          | 3      | 2      | 6     | 10min |

### 7.2 Integration Tests

| #     | Task                                                         | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------ | ---------- | ------ | ------ | ----- | ----- |
| 7.2.1 | Create `integration_test.go` - Full CQRS flow test           | 5          | 5      | 3      | 8.3   | 15min |
| 7.2.2 | Create `integration_test.go` - Event sourcing roundtrip test | 5          | 4      | 3      | 6.7   | 12min |
| 7.2.3 | Create `integration_test.go` - Middleware chain test         | 4          | 4      | 2      | 8     | 10min |

### 7.3 Examples

| #     | Task                                                                | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 7.3.1 | Create `example/user/` - User aggregate example                     | 4          | 5      | 3      | 6.7   | 15min |
| 7.3.2 | Create `example/user/commands.go` - CreateUser, UpdateUser commands | 4          | 4      | 2      | 8     | 10min |
| 7.3.3 | Create `example/user/queries.go` - GetUser, ListUsers queries       | 4          | 4      | 2      | 8     | 10min |
| 7.3.4 | Create `example/user/events.go` - UserCreated, UserUpdated events   | 4          | 4      | 2      | 8     | 10min |
| 7.3.5 | Create `example/user/handlers.go` - Command/Query handlers          | 4          | 4      | 2      | 8     | 12min |
| 7.3.6 | Create `example/main.go` - Complete working example                 | 4          | 5      | 2      | 10    | 12min |

---

## Phase 8: Documentation & Polish (Est. 1-2 hours)

### 8.1 Documentation

| #     | Task                                                          | Importance | Impact | Effort | Score | Est.  |
| ----- | ------------------------------------------------------------- | ---------- | ------ | ------ | ----- | ----- |
| 8.1.1 | Add GoDoc comments to all exported types                      | 4          | 4      | 2      | 8     | 15min |
| 8.1.2 | Create `docs/architecture.md` - Architecture decision records | 3          | 4      | 2      | 6     | 12min |
| 8.1.3 | Update README.md with complete API reference                  | 4          | 4      | 2      | 8     | 10min |
| 8.1.4 | Create `CODE_OF_CONDUCT.md`                                   | 2          | 2      | 1      | 4     | 3min  |
| 8.1.5 | Create `CONTRIBUTING.md`                                      | 3          | 3      | 2      | 4.5   | 8min  |

### 8.2 CI/CD

| #     | Task                                                    | Importance | Impact | Effort | Score | Est. |
| ----- | ------------------------------------------------------- | ---------- | ------ | ------ | ----- | ---- |
| 8.2.1 | Create `.github/workflows/test.yml` - Run tests on PR   | 4          | 4      | 1      | 16    | 5min |
| 8.2.2 | Create `.github/workflows/lint.yml` - Run golangci-lint | 4          | 4      | 1      | 16    | 5min |
| 8.2.3 | Create `Makefile` - build, test, lint, cover targets    | 4          | 3      | 1      | 12    | 8min |
| 8.2.4 | Create `.golangci.yml` - Linter configuration           | 4          | 3      | 1      | 12    | 5min |

---

## Summary Table

| Phase                         | Tasks  | Est. Time  | Dependencies |
| ----------------------------- | ------ | ---------- | ------------ |
| **Phase 1: Foundation**       | 10     | 4-5h       | None         |
| **Phase 2: Events**           | 8      | 3-4h       | Phase 1      |
| **Phase 3: Commands**         | 5      | 2-3h       | Phase 1, 2   |
| **Phase 4: Queries**          | 5      | 2-3h       | Phase 1, 2   |
| **Phase 5: Middleware**       | 8      | 3-4h       | Phase 3, 4   |
| **Phase 6: In-Memory**        | 4      | 2h         | Phase 2      |
| **Phase 7: Tests & Examples** | 14     | 3-4h       | All above    |
| **Phase 8: Documentation**    | 9      | 1-2h       | All above    |
| **TOTAL**                     | **63** | **16-20h** | -            |

---

## Dependency Graph

```
go.mod (1.1.1)
    │
    ├── event/ (1.1.2-1.1.4, 2.1.1-2.2.4)
    │       │
    │       ├── command/ (1.1.5-1.1.6, 3.1.1-3.1.5)
    │       │       │
    │       │       └── middleware/ (5.1.1-5.1.5)
    │       │
    │       ├── query/ (1.1.7-1.1.8, 4.1.1-4.1.5)
    │       │       │
    │       │       └── middleware/ (5.2.1-5.2.3)
    │       │
    │       ├── aggregate/ (1.2.1-1.2.2)
    │       │
    │       └── memory/ (6.1.1-6.1.4)
    │
    └── tests & examples (7.1.1-7.3.6)
            │
            └── docs & CI (8.1.1-8.2.4)
```

---

## Progress Tracking

### Checklist

Copy this section to track progress:

```markdown
## Progress

- [ ] Phase 1: Foundation Layer
  - [ ] 1.1.1 go.mod
  - [ ] 1.1.2 event/event.go
  - [ ] 1.1.3 event/metadata.go
  - [ ] 1.1.4 event/errors.go
  - [ ] 1.1.5 command/command.go
  - [ ] 1.1.6 command/errors.go
  - [ ] 1.1.7 query/query.go
  - [ ] 1.1.8 query/errors.go
  - [ ] 1.2.1 aggregate/aggregate.go
  - [ ] 1.2.2 aggregate/repository.go
- [ ] Phase 2: Event Layer
- [ ] Phase 3: Command Layer
- [ ] Phase 4: Query Layer
- [ ] Phase 5: Middleware
- [ ] Phase 6: In-Memory
- [ ] Phase 7: Tests & Examples
- [ ] Phase 8: Documentation & Polish
```

---

## Quality Gates

Before each phase completion:

- [ ] All code compiles (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] Coverage > 80% (`go test -cover ./...`)
- [ ] No linting errors (`golangci-lint run`)
- [ ] Files under 250 lines
- [ ] Functions under 30 lines
- [ ] No `any` types
- [ ] Context accepted as first parameter in public methods

---

_This plan is designed for incremental delivery. Each phase produces usable artifacts._
