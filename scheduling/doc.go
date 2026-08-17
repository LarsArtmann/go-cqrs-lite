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
//	    opts := []command.Option{command.WithActor(id.NewSystemActor("scheduler"))}
//	    if actor, err := id.ParseActorID(t.Actor); err == nil && !actor.IsZero() {
//	        opts = append(opts, command.WithActor(actor))
//	    }
//	    return commandBus.Dispatch(ctx, t.Payload, opts...)
//	})
//	store.Schedule(ctx, scheduling.Timer[CancelOrderCmd]{
//	    ID:        "order-cancel-123",
//	    FireAt:    time.Now().Add(30 * time.Minute),
//	    Payload:   CancelOrderCmd{OrderID: "123"},
//	    Actor:     "user:01JXYZ...",
//	})
//	go sched.Start(ctx)
package scheduling
