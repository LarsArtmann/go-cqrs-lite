// Package scheduling provides durable deadline timers for event-sourced systems.
//
// A [TimerStore] records scheduled commands that should fire at a future time.
// A [Scheduler] polls the store for due timers and invokes a callback — typically
// dispatching a command to the CQRS command bus.
//
// The payload type is generic: pick a concrete command type to get compile-time
// safety instead of an untyped `any` at the boundary.
//
// Common use cases: "cancel order after 30 minutes unpaid", "send reminder
// email 24 hours after signup", "expire session after 15 minutes idle".
//
// # Quick Start
//
//	store := scheduling.NewMemoryTimerStore[CancelOrderCmd]()
//	sched := scheduling.New[CancelOrderCmd](store, func(ctx context.Context, t scheduling.Timer[CancelOrderCmd]) error {
//	    return commandBus.Dispatch(ctx, t.Payload)
//	})
//	store.Schedule(ctx, scheduling.Timer[CancelOrderCmd]{
//	    ID:        "order-cancel-123",
//	    FireAt:    time.Now().Add(30 * time.Minute),
//	    Payload:   CancelOrderCmd{OrderID: "123"},
//	})
//	go sched.Start(ctx)
package scheduling
