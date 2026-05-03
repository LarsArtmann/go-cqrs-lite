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

	// --- Infrastructure ---
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

	// --- Bus subscribers (projection + logging) ---
	var publishedEvents []event.Event

	registerBusHandlers(bus, readModel, &publishedEvents)

	fmt.Println("[infra] Store: MemoryStore | Bus: MemoryBus | Decider Repository: ready")
	fmt.Println("[infra] Read model projection: subscribed to bus")
	fmt.Println()

	// --- Command dispatcher with middleware ---
	cmdDispatcher := command.NewDispatcher()

	cmdDispatcher.Use(
		middleware.CommandRecovery(),
		middleware.CommandLogging(newLogger()),
		middleware.CommandMetrics(&printMetricsRecorder{}),
		middleware.CommandRetry(middleware.DefaultRetryConfig()),
	)

	registerCommandHandlers(cmdDispatcher, deciderRepo)

	// --- Query dispatcher ---
	qryDispatcher := query.NewDispatcher()
	registerQueryHandlers(qryDispatcher, readModel)

	fmt.Println("[infra] Middleware: Recovery → Logging → Metrics → Retry")
	fmt.Println()

	// --- Step 1: Create User ---
	fmt.Println("--- Step 1: Create User ---")
	userID := id.NewAggregateID()

	err = cmdDispatcher.Dispatch(ctx, newUserCmd(userID, "alice@example.com", "Alice Smith"))
	if err != nil {
		log.Fatalf("create user: %v", err)
	}

	fmt.Printf("→ Created user %s (alice@example.com)\n\n", userID)

	// --- Step 2: Change Name ---
	fmt.Println("--- Step 2: Change Name ---")
	err = cmdDispatcher.Dispatch(ctx, &ChangeUserNameCmd{
		aggregateID: userID,
		name:        "Alice Johnson",
		idempotency: "change-name-" + userID.String(),
	})
	if err != nil {
		log.Fatalf("change name: %v", err)
	}

	fmt.Printf("→ Changed name to %q, version %d\n\n", "Alice Johnson", len(publishedEvents))

	// --- Step 3: Query User (from read model) ---
	fmt.Println("--- Step 3: Query User ---")
	getResult, err := qryDispatcher.Dispatch(ctx, &GetUserQuery{aggregateID: userID})
	if err != nil {
		log.Fatalf("get user: %v", err)
	}

	if rm, ok := getResult.(ReadModel); ok {
		fmt.Printf("→ User{Email: %q, Name: %q}\n\n", rm.Email, rm.Name)
	}

	// --- Step 4: List Users ---
	fmt.Println("--- Step 4: List Users ---")
	listResult, err := qryDispatcher.Dispatch(ctx, &ListUsersQuery{})
	if err != nil {
		log.Fatalf("list users: %v", err)
	}

	if users, ok := listResult.([]ReadModel); ok {
		for i, u := range users {
			fmt.Printf("  [%d] %s (%s)\n", i, u.Email, u.Name)
		}
	}

	fmt.Println()

	// --- Step 5: Validation Error ---
	fmt.Println("--- Step 5: Validation Error (empty email) ---")
	err = cmdDispatcher.Dispatch(ctx, newUserCmd(id.NewAggregateID(), "", "No Email"))
	if err != nil {
		var evtErr *event.Error
		if errors.As(err, &evtErr) {
			fmt.Printf("→ Rejected [%s]: %s\n\n", evtErr.Family, evtErr.Message)
		} else {
			fmt.Printf("→ Error: %v\n\n", err)
		}
	}

	// --- Step 6: Error Classification ---
	fmt.Println("--- Step 6: Error Classification ---")
	family := event.Classify(err)
	fmt.Printf("→ classify(create-no-email) = %s\n", family)
	fmt.Printf("→ isRetryable(create-no-email) = %v\n\n", event.IsRetryable(err))

	// --- Step 7: EventCatalog ---
	fmt.Println("--- Step 7: EventCatalog ---")
	outputDir := filepath.Join(".", "eventcatalog-output")

	if err := generateEventCatalog(outputDir); err != nil {
		log.Fatalf("generate event catalog: %v", err)
	}

	fmt.Printf("→ EventCatalog written to %s\n\n", outputDir)

	// --- Summary ---
	fmt.Println("=== Demo Complete ===")
	fmt.Printf("  Events published: %d\n", len(publishedEvents))
	fmt.Printf("  Read model users: %d\n", len(readModel.List()))
}
