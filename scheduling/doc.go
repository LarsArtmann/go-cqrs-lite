// Package scheduling provides durable deadline timers for event-sourced systems.
//
// A TimerStore records scheduled commands that should fire at a future time.
// A Dispatcher polls the store for due timers and invokes a callback — typically
// dispatching a command to the CQRS command bus.
//
// Common use cases: "cancel order after 30 minutes unpaid", "send reminder
// email 24 hours after signup", "expire session after 15 minutes idle".
//
// # Quick Start
//
//	store := scheduling.NewMemoryTimerStore()
//	sched := scheduling.New(store, func(ctx context.Context, t scheduling.Timer) error {
//	    return commandBus.Dispatch(ctx, t.Payload)
//	})
//	store.Schedule(ctx, scheduling.Timer{
//	    ID:        "order-cancel-123",
//	    FireAt:    time.Now().Add(30 * time.Minute),
//	    Payload:   "cancel_order_command_json",
//	})
//	go sched.Start(ctx)
package scheduling
