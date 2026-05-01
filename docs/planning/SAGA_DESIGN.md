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

- Timeout handling for long-running steps?
- Retry policy per step?
- Saga interleaving (same saga type, different instances)?
- Correlation across services?

## References

- [Axon Framework Sagas](https://docs.axoniq.io/reference-guide/v/4.10/sagas)
- [Temporal.io](https://temporal.io/)
- [Eventuate Tram Saga](https://eventuate.io/)
