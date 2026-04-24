package main

import (
	"context"
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func main() {
	ctx := context.Background()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	cmdDispatcher := command.NewDispatcher()
	repo := NewRepository(store, bus)

	bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		fmt.Printf("  [Event] %s (%s) v%d\n", evt.Type(), evt.AggregateID(), evt.Version())
		return nil
	})

	RegisterHandlers(cmdDispatcher, repo)

	userID := id.NewAggregateID()
	fmt.Println("=== Creating User ===")
	if err := cmdDispatcher.Dispatch(
		ctx,
		NewCreateUser(userID, "Alice", "alice@example.com"),
	); err != nil {
		log.Fatalf("create user: %v", err)
	}

	fmt.Println("\n=== Changing Email ===")
	if err := cmdDispatcher.Dispatch(
		ctx,
		NewChangeUserEmail(userID, "alice.new@example.com"),
	); err != nil {
		log.Fatalf("change email: %v", err)
	}

	fmt.Println("\n=== Rebuilding from Events ===")
	user, err := repo.Load(ctx, userID)
	if err != nil {
		log.Fatalf("load user: %v", err)
	}
	fmt.Printf("  ID: %s\n", user.ID())
	fmt.Printf("  Name: %s\n", user.Name())
	fmt.Printf("  Email: %s\n", user.Email())
	fmt.Printf("  Version: %d\n", user.Version())

	cmdDispatcher.Close()
	bus.Close()
	store.Close()

	fmt.Println("\n=== Generating Catalog ===")
	builder := adapters.NewBuilder("User Service API", "1.0.0")
	builder.AddService("user-service", "User Service", "1.0.0", "Manages user accounts")

	builder.AddCommand("user-service", NewCreateUser(userID, "Alice", "alice@example.com"))
	builder.AddCommand("user-service", NewChangeUserEmail(userID, "alice.new@example.com"))

	cat := builder.Build()
	fmt.Printf(
		"  Catalog: %d services, %d total commands\n",
		len(cat.Services),
		len(cat.Services[0].Commands),
	)
}
