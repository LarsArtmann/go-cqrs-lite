// Package main demonstrates saga-style orchestration using only projection + command dispatch.
//
// No dedicated "saga" module is needed. A saga is just a projection that:
//  1. Tracks saga state (current step, status) as a projected view
//  2. Reacts to events by deciding the next step and dispatching commands
//  3. Handles failures by dispatching compensating (undo) commands
//
// This example simulates an order-processing saga with three steps:
//
//	Reserve Inventory → Charge Payment → Confirm Order
//
// If any step fails, compensating actions run in reverse to undo prior work.
//
// Run with: go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

const (
	contextTimeout = 5 * time.Second
	settleDelay    = 200 * time.Millisecond
)

// --- Domain events ---

type (
	InventoryReserved struct{ OrderID string }
	InventoryReleased struct{ OrderID string }
	PaymentCharged    struct{ OrderID string }
	PaymentRefunded   struct{ OrderID string }
	OrderConfirmed    struct{ OrderID string }
	StepFailed        struct {
		OrderID string
		Step    string
		Reason  string
	}
)

// --- Saga state (the "projected view") ---

type sagaStatus string

const (
	statusRunning      sagaStatus = "running"
	statusCompensating sagaStatus = "compensating"
	statusCompleted    sagaStatus = "completed"
	statusFailed       sagaStatus = "failed"
)

type sagaState struct {
	mu           sync.Mutex
	OrderID      string
	CurrentStep  int
	Status       sagaStatus
	Compensating bool
}

const totalSteps = 3

func (s *sagaState) advance(stepName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CurrentStep++

	if s.CurrentStep >= totalSteps {
		s.Status = statusCompleted
		fmt.Printf("  [saga] SAGA COMPLETED: %s\n", s.OrderID)
	} else {
		fmt.Printf("  [saga] step %d (%s) done, now at step %d\n", s.CurrentStep, stepName, s.CurrentStep+1)
	}
}

func (s *sagaState) fail(step string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = statusCompensating
	s.Compensating = true

	fmt.Printf("  [saga] FAILED at step %q, compensating...\n", step)
}

func (s *sagaState) finishCompensation() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = statusFailed
	fmt.Printf("  [saga] compensation finished, saga status: %s\n", s.Status)
}

// --- Command dispatcher (simulated) ---

type commandDispatcher struct {
	bus event.Publisher
}

func (d *commandDispatcher) dispatch(
	ctx context.Context,
	eventType event.Type,
	aggregateID id.AggregateID,
	payload any,
) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	evt, err := event.NewEvent(eventType, aggregateID, "SagaCommand", 1, bytes)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	fmt.Printf("  [dispatch] %s\n", eventType)

	return d.bus.Publish(ctx, evt)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpointStore := memory.NewMemoryCheckpointStore()

	dispatcher := &commandDispatcher{bus: bus}

	orderID := id.NewAggregateID()
	state := &sagaState{
		OrderID:     orderID.String(),
		CurrentStep: 0,
		Status:      statusRunning,
	}

	// Build the saga projection — this IS the saga orchestrator.
	builder := projection.NewBuilder("order-saga")

	// Step 1 success: InventoryReserved → dispatch charge-payment
	if err := projection.On(
		builder,
		"inventory.reserved",
		codec.JSONCodec{},
		func(_ context.Context, p InventoryReserved) error {
			state.advance("reserve-inventory")

			return dispatcher.dispatch(ctx, "charge-payment", orderID,
				PaymentCharged(p))
		},
	); err != nil {
		return fmt.Errorf("register inventory.reserved: %w", err)
	}

	// Step 2 success: PaymentCharged → dispatch confirm-order
	if err := projection.On(
		builder,
		"payment.charged",
		codec.JSONCodec{},
		func(_ context.Context, p PaymentCharged) error {
			state.advance("charge-payment")

			return dispatcher.dispatch(ctx, "confirm-order", orderID,
				OrderConfirmed(p))
		},
	); err != nil {
		return fmt.Errorf("register payment.charged: %w", err)
	}

	// Step 3 success: OrderConfirmed → saga complete
	if err := projection.On(
		builder,
		"order.confirmed",
		codec.JSONCodec{},
		func(_ context.Context, _ OrderConfirmed) error {
			state.advance("confirm-order")

			return nil
		},
	); err != nil {
		return fmt.Errorf("register order.confirmed: %w", err)
	}

	// Failure handler: any step failure triggers compensation
	if err := projection.On(
		builder,
		"saga.step_failed",
		codec.JSONCodec{},
		func(_ context.Context, p StepFailed) error {
			state.fail(p.Step)

			// Compensate in reverse order based on which step failed
			if p.Step == "charge-payment" && state.CurrentStep >= 1 {
				_ = dispatcher.dispatch(ctx, "release-inventory", orderID,
					InventoryReleased{OrderID: p.OrderID})
			}

			if p.Step == "confirm-payment" && state.CurrentStep >= 2 {
				_ = dispatcher.dispatch(ctx, "refund-payment", orderID,
					PaymentRefunded{OrderID: p.OrderID})
				_ = dispatcher.dispatch(ctx, "release-inventory", orderID,
					InventoryReleased{OrderID: p.OrderID})
			}

			state.finishCompensation()

			return nil
		},
	); err != nil {
		return fmt.Errorf("register saga.step_failed: %w", err)
	}

	sagaProjection := builder.Build()

	runner, err := projection.NewRunner(store, bus, checkpointStore)
	if err != nil {
		return fmt.Errorf("create runner: %w", err)
	}

	if err := runner.Register(sagaProjection); err != nil {
		return fmt.Errorf("register projection: %w", err)
	}

	go func() {
		if runErr := runner.Run(ctx); runErr != nil {
			log.Printf("runner stopped: %v", runErr)
		}
	}()

	// Kick off the saga by publishing the first event
	fmt.Println("Starting order saga...")
	fmt.Printf("Order ID: %s\n\n", orderID)

	if err := dispatcher.dispatch(ctx, "reserve-inventory", orderID,
		InventoryReserved{OrderID: orderID.String()}); err != nil {
		return fmt.Errorf("dispatch reserve-inventory: %w", err)
	}

	time.Sleep(settleDelay)

	fmt.Printf("\nFinal state: %s (step %d/%d)\n", state.Status, state.CurrentStep+1, totalSteps)

	_ = runner.Close()

	fmt.Println("\nDone.")

	return nil
}
