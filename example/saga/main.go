// Package main demonstrates a simple order-processing saga using the saga module.
//
// The saga orchestrates three steps:
//  1. Reserve inventory
//  2. Charge payment
//  3. Confirm order
//
// If any step fails, compensating actions run in reverse order to undo prior work.
// Run with: go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

func orderSagaSteps() []saga.Step {
	return []saga.Step{
		{
			Name: "reserve-inventory",
			Action: func(_ context.Context, sagaID id.AggregateID) command.Command {
				return command.MustNew("reserve-inventory", sagaID)
			},
			Compensate: func(_ context.Context, sagaID id.AggregateID) command.Command {
				return command.MustNew("release-inventory", sagaID)
			},
			Timeout: 5 * time.Second,
		},
		{
			Name: "charge-payment",
			Action: func(_ context.Context, sagaID id.AggregateID) command.Command {
				return command.MustNew("charge-payment", sagaID)
			},
			Compensate: func(_ context.Context, sagaID id.AggregateID) command.Command {
				return command.MustNew("refund-payment", sagaID)
			},
			Timeout: 10 * time.Second,
		},
		{
			Name:   "confirm-order",
			Action: func(_ context.Context, sagaID id.AggregateID) command.Command { return command.MustNew("confirm-order", sagaID) },
			Timeout: 5 * time.Second,
		},
	}
}

type orderSaga struct{}

func (orderSaga) SagaType() string      { return "order" }
func (orderSaga) Steps() []saga.Step    { return orderSagaSteps() }

type loggingDispatcher struct {
	dispatched []string
}

func (d *loggingDispatcher) Dispatch(_ context.Context, cmd command.Command) error {
	d.dispatched = append(d.dispatched, string(cmd.Type()))
	fmt.Printf("  -> dispatched: %s\n", cmd.Type())
	return nil
}

func main() {
	ctx := context.Background()
	store := saga.NewMemoryStore()
	dispatcher := &loggingDispatcher{}

	runner := saga.NewRunner(store, dispatcher,
		saga.WithRetryPolicy(3, 100*time.Millisecond),
	)

	if err := runner.Register(orderSaga{}); err != nil {
		log.Fatalf("register saga: %v", err)
	}

	fmt.Println("Starting order saga...")
	instance, err := runner.Start(ctx, "order", nil)
	if err != nil {
		log.Fatalf("start saga: %v", err)
	}
	fmt.Printf("Saga started: id=%s status=%s\n", instance.ID, instance.Status)

	for instance.Status == saga.StatusRunning {
		fmt.Printf("\nExecuting step %d...\n", instance.CurrentStep+1)
		if err := runner.ExecuteStep(ctx, instance.ID); err != nil {
			log.Printf("step failed: %v", err)
			break
		}

		loaded, loadErr := store.Load(ctx, instance.ID)
		if loadErr != nil {
			log.Fatalf("load state: %v", loadErr)
		}
		instance.State = *loaded
		fmt.Printf("Status: %s (step %d/%d)\n", instance.Status, instance.CurrentStep, len(instance.Steps))
	}

	fmt.Printf("\nFinal status: %s\n", instance.Status)
	fmt.Printf("Dispatched commands: %v\n", dispatcher.dispatched)
}
