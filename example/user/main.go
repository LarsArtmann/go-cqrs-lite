// Package main demonstrates a complete CQRS + Event Sourcing integration
// using go-cqrs-lite with the Decider pattern — pure functions for state
// reconstruction and command decisions, no mutable aggregate required.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/middleware"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== go-cqrs-lite: Full CQRS + Event Sourcing Demo ===")
	fmt.Println()

	bus, readModel, deciderRepo := setupInfrastructure()
	var publishedEvents []event.Event

	registerBusHandlers(bus, readModel, &publishedEvents)

	cmdDisp, qryDisp := setupDispatchers(deciderRepo, readModel)

	userID := runDemoSteps(ctx, cmdDisp, qryDisp, &publishedEvents)
	runErrorDemo(ctx, cmdDisp)
	runEventCatalog()

	fmt.Println("=== Demo Complete ===")
	fmt.Printf("  Events published: %d\n", len(publishedEvents))
	fmt.Printf("  Read model users: %d\n", len(readModel.List()))
	fmt.Printf("  User ID: %s\n", userID)
}

func setupInfrastructure() (
	*memory.MemoryBus,
	*ReadModelStore,
	*decider.Repository[UserState],
) {
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	readModel := NewReadModelStore()

	userDecider := decider.Decider[UserState]{
		Initial: UserState{},
		Fold:    foldUser,
	}

	deciderRepo, err := decider.NewRepository(store, bus, userDecider)
	if err != nil {
		log.Fatalf("create decider repo: %v", err)
	}

	fmt.Println("[infra] Store: MemoryStore | Bus: MemoryBus | Decider Repository: ready")
	fmt.Println("[infra] Read model projection: subscribed to bus")
	fmt.Println()

	return bus, readModel, deciderRepo
}

func setupDispatchers(
	deciderRepo *decider.Repository[UserState],
	readModel *ReadModelStore,
) (*command.Dispatcher, *query.Dispatcher) {
	cmdDisp := command.NewDispatcher()

	cmdDisp.Use(
		middleware.CommandRecovery(),
		middleware.CommandLogging(newLogger()),
		middleware.CommandMetrics(&printMetricsRecorder{}),
		middleware.CommandRetry(middleware.DefaultRetryConfig()),
	)

	registerCommandHandlers(cmdDisp, deciderRepo)

	qryDisp := query.NewDispatcher()
	registerQueryHandlers(qryDisp, readModel)

	fmt.Println("[infra] Middleware: Recovery → Logging → Metrics → Retry")
	fmt.Println()

	return cmdDisp, qryDisp
}

func runDemoSteps(
	ctx context.Context,
	cmdDisp *command.Dispatcher,
	qryDisp *query.Dispatcher,
	publishedEvents *[]event.Event,
) id.AggregateID {
	fmt.Println("--- Step 1: Create User ---")
	userID := id.NewAggregateID()

	err := cmdDisp.Dispatch(ctx, newUserCmd(userID, "alice@example.com", "Alice Smith"))
	if err != nil {
		log.Fatalf("create user: %v", err)
	}

	fmt.Printf("→ Created user %s (alice@example.com)\n\n", userID)

	fmt.Println("--- Step 2: Change Name ---")
	err = cmdDisp.Dispatch(ctx, &ChangeUserNameCmd{
		aggregateID: userID,
		name:        "Alice Johnson",
		idempotency: "change-name-" + userID.String(),
	})
	if err != nil {
		log.Fatalf("change name: %v", err)
	}

	fmt.Printf("→ Changed name to %q, version %d\n\n", "Alice Johnson", len(*publishedEvents))

	fmt.Println("--- Step 3: Query User ---")
	rm, err := query.DispatchTyped[ReadModel](ctx, qryDisp, &GetUserQuery{aggregateID: userID})
	if err != nil {
		log.Fatalf("get user: %v", err)
	}

	fmt.Printf("→ User{Email: %q, Name: %q}\n\n", rm.Email, rm.Name)

	return userID
}

func runErrorDemo(ctx context.Context, cmdDisp *command.Dispatcher) {
	fmt.Println("--- Step 5: Validation Error (empty email) ---")

	err := cmdDisp.Dispatch(ctx, newUserCmd(id.NewAggregateID(), "", "No Email"))
	if err != nil {
		evtErr, ok := errors.AsType[*event.Error](err)
		if ok {
			fmt.Printf("→ Rejected [%s]: %s\n\n", evtErr.Family(), evtErr.Message())
		} else {
			fmt.Printf("→ Error: %v\n\n", err)
		}
	}

	fmt.Println("--- Step 6: Error Classification ---")
	family := event.Classify(err)
	fmt.Printf("→ classify(create-no-email) = %s\n", family)
	fmt.Printf("→ isRetryable(create-no-email) = %v\n\n", event.IsRetryable(err))
}

func runEventCatalog() {
	fmt.Println("--- Step 7: EventCatalog ---")
	outputDir := filepath.Join(".", "eventcatalog-output")

	if err := generateEventCatalog(outputDir); err != nil {
		log.Fatalf("generate event catalog: %v", err)
	}

	fmt.Printf("→ EventCatalog written to %s\n\n", outputDir)
}
