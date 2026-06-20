# Saga / Process Manager Design

**Status:** Planning | **Date:** 2026-05-01

## Problem

Long-running business processes (order fulfillment, user onboarding, payment flows)
span multiple aggregates and require coordination with compensation (rollback) logic.

## Design Options

### Option A: Choreography (Event-Driven)

Each service reacts to events independently. No central coordinator.

```
OrderCreated → PaymentProcessed → OrderConfirmed
                                    ↓ (on failure)
                              PaymentFailed → OrderCancelled
```

**Pros:** Loose coupling, naturally distributed
**Cons:** Hard to visualize flow, no single source of truth, compensation is implicit

### Option B: Orchestration (Saga Coordinator)

A central saga instance drives the process step by step.

```
SagaInstance
  ├── Step 1: CreateOrder → on success → Step 2
  ├── Step 2: ProcessPayment → on success → Step 3
  ├── Step 3: ConfirmOrder → on success → Complete
  └── (on any failure) → Compensate backwards
```

**Pros:** Clear flow, explicit compensation, single source of truth
**Cons:** Tighter coupling to coordinator, single point of failure

### Recommendation: Orchestration

Aligns with the library's explicit, type-safe approach.

## Proposed API

```go
// Define a saga
type OrderSaga struct {
    saga.Core
}

func (s *OrderSaga) Steps() []saga.Step {
    return []saga.Step{
        {Name: "create-order", Command: &CreateOrder{}, Compensate: &CancelOrder{}},
        {Name: "process-payment", Command: &ProcessPayment{}, Compensate: &RefundPayment{}},
        {Name: "confirm-order", Command: &ConfirmOrder{}},
    }
}

// Runner manages saga lifecycle
runner := saga.NewInMemoryRunner(store, commandDispatcher)
runner.Register(&OrderSaga{})
```

## Core Types

| Type            | Purpose                                              |
| --------------- | ---------------------------------------------------- |
| `saga.Core`     | Base saga with ID, state, current step               |
| `saga.Step`     | Command + optional compensation command              |
| `saga.Instance` | Persistent saga state (which step, status)           |
| `saga.Store`    | Persist saga instances                               |
| `saga.Runner`   | Execute steps, handle failures, trigger compensation |

## Persistence

- `saga.Instance` stored in SQL: `saga_instances` table
- State machine: `Pending → Running → Completed \| Compensating \| Failed`
- Each state transition is an event (audit trail)

## Open Questions

- Timeout handling for long-running steps? → `saga.Step.Timeout time.Duration` (default: none)
- Retry policy per step? → Reuse `event.IsRetryable` + exponential backoff from middleware
- Saga interleaving (same saga type, different instances)? → Each instance has unique `saga.InstanceID` (branded)
- Correlation across services? → Reuse `event.Metadata.CorrelationID` from existing event system

## Answers to Open Questions

### Timeout

Each `Step` gets an optional `Timeout time.Duration`. The runner uses `context.WithTimeout` when dispatching a step's command. If timeout fires, the step fails and compensation begins.

```go
type Step struct {
    Name       string
    Command    func(instanceID id.AggregateID) command.Command
    Compensate func(instanceID id.AggregateID) command.Command // nil = no compensation
    Timeout    time.Duration                                   // 0 = no timeout
}
```

### Retry

Reuse the library's existing `event.IsRetryable` classification. The runner checks `IsRetryable(err)` and applies exponential backoff with configurable `MaxRetries` per step. Non-retryable errors trigger compensation immediately.

### Correlation

The saga runner injects `event.WithCorrelationID(sagaID)` into all commands dispatched during the saga. This creates a natural audit trail using the existing metadata system — no new concepts needed.

## Integration with go-cqrs-lite Types

The saga module builds on existing library types rather than introducing parallel concepts:

| Saga Concept         | Uses Existing Type                          |
| -------------------- | ------------------------------------------- |
| Saga Instance ID     | `id.AggregateID` (branded with saga marker) |
| Step Commands        | `command.Command` interface                 |
| Correlation          | `event.Metadata.CorrelationID`              |
| Error Classification | `event.IsRetryable`, `event.Family`         |
| Persistence          | `event.Store` (saga state as events)        |
| Checkpointing        | `event.CheckpointStore`                     |
| Runner pattern       | Same as `projection.Runner`                 |

## Implementation Phases

### Phase 1: Core Types (New Module: `saga/`)

**Files:**

```
saga/
├── go.mod
├── saga.go           — Core, Step, Instance types
├── runner.go         — Runner with Register, Run
├── store.go          — Store interface
├── errors.go         — Sentinel errors
└── options.go        — RunnerOption functional options
```

**Types (~80 lines):**

```go
type Instance struct {
    ID         id.AggregateID
    SagaType   string
    Status     Status    // Pending, Running, StepCompleted, Compensating, Completed, Failed
    CurrentStep int
    Steps      []Step
    Err        error
}

type Status string

const (
    StatusPending       Status = "pending"
    StatusRunning       Status = "running"
    StatusStepCompleted Status = "step_completed"
    StatusCompensating  Status = "compensating"
    StatusCompleted     Status = "completed"
    StatusFailed        Status = "failed"
)
```

**Estimated effort:** 4 hours

### Phase 2: In-Memory Runner

- `InMemoryRunner` with `Register(saga)` and `Start(ctx)`
- Dispatches commands via `command.Dispatcher`
- Listens to events via `event.Subscriber`
- Updates instance state on each step completion
- Triggers compensation on failure
- Full test suite with table-driven tests

**Estimated effort:** 6 hours

### Phase 3: Persistence

- `SagaStore` interface: `Save`, `Load`, `Update`
- SQL implementation in `storage/saga_store.go`
- Instance state serialized as JSON
- Recovery on restart: load all `StatusRunning` instances

**Estimated effort:** 4 hours

### Phase 4: Compensation Engine

- Backward compensation: iterate completed steps in reverse
- Idempotent compensation: track which steps have been compensated
- Compensation timeout and failure handling
- Dead-letter for permanently failed sagas

**Estimated effort:** 4 hours

## Total Estimated Effort

| Phase            | Effort  | Depends On |
| ---------------- | ------- | ---------- |
| Core Types       | 4h      | Nothing    |
| In-Memory Runner | 6h      | Phase 1    |
| Persistence      | 4h      | Phase 2    |
| Compensation     | 4h      | Phase 2    |
| **Total**        | **18h** |            |

## Recommended Start

Phase 1 (Core Types) can begin immediately. It only depends on `core/` types that are already stable. The `saga` module would be a new Go module in the monorepo: `github.com/larsartmann/go-cqrs-lite/saga`.

## References

- [Axon Framework Sagas](https://docs.axoniq.io/reference-guide/v/4.10/sagas)
- [Temporal.io](https://temporal.io/)
- [Eventuate Tram Saga](https://eventuate.io/)
