// Package main demonstrates a complete CQRS + Event Sourcing flow
// using go-cqrs-lite with the in-memory implementations,
// followed by automatic EventCatalog generation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

// --- Typed event payloads (with struct tags for schema generation) ---

type UserCreatedPayload struct {
	Email string `description:"The user's email address" json:"email"`
}

type UserNameChangedPayload struct {
	Name string `description:"The new display name" json:"name"`
}

// --- User aggregate root ---

type User struct {
	*aggregate.Core

	email string
	name  string
}

func newUser(email string) *User {
	aggID := id.NewAggregateID()

	return &User{
		Core:  aggregate.NewCore(aggID, event.AggregateType("User")),
		email: email,
	}
}

func (u *User) Apply(evt event.Event) error {
	switch evt.Type() {
	case "UserCreated":
		var p UserCreatedPayload
		err := json.Unmarshal(evt.Payload(), &p)
		if err != nil {
			return fmt.Errorf("unmarshal UserCreated: %w", err)
		}

		u.email = p.Email
	case "UserNameChanged":
		var p UserNameChangedPayload
		err := json.Unmarshal(evt.Payload(), &p)
		if err != nil {
			return fmt.Errorf("unmarshal UserNameChanged: %w", err)
		}

		u.name = p.Name
	}

	return nil
}

func (u *User) ApplySnapshot(_ []byte) error { return nil }

func (u *User) LoadEvents(events []event.Event) error {
	return u.LoadFromHistory(u, events)
}

// ChangeName records a UserNameChanged event.
func (u *User) ChangeName(ctx context.Context, name string) error {
	payload, err := json.Marshal(UserNameChangedPayload{Name: name})
	if err != nil {
		return fmt.Errorf("marshal name payload: %w", err)
	}

	evt, err := event.NewEvent(
		"UserNameChanged",
		u.ID(),
		u.Type(),
		u.Version()+1,
		payload,
	)
	if err != nil {
		return fmt.Errorf("create UserNameChanged event: %w", err)
	}

	u.RecordEvent(ctx, evt)

	u.name = name

	return nil
}

// --- Main ---

func main() {
	ctx := context.Background()

	fmt.Println("=== CQRS + Event Sourcing Demo ===")
	fmt.Println()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	repo := aggregate.NewRepository(store, bus)

	user := newUser("alice@example.com")

	createdPayload, err := json.Marshal(UserCreatedPayload{Email: user.email})
	if err != nil {
		log.Fatalf("marshal created payload: %v", err)
	}

	created, err := event.NewEvent(
		"UserCreated",
		user.ID(),
		user.Type(),
		1,
		createdPayload,
	)
	if err != nil {
		log.Fatalf("create event: %v", err)
	}

	user.RecordEvent(ctx, created)

	if err := repo.Save(ctx, user); err != nil {
		log.Fatalf("save user: %v", err)
	}

	fmt.Printf("Created user %s (%s)\n", user.ID(), user.email)

	loaded := &User{Core: aggregate.NewCore(user.ID(), event.AggregateType("User"))}

	if err := repo.Load(ctx, loaded); err != nil {
		log.Fatalf("load user: %v", err)
	}

	fmt.Printf("Loaded user %s, version %d\n", loaded.ID(), loaded.Version())

	if err := loaded.ChangeName(ctx, "Alice Smith"); err != nil {
		log.Fatalf("change name: %v", err)
	}

	if err := repo.Save(ctx, loaded); err != nil {
		log.Fatalf("save name change: %v", err)
	}

	fmt.Printf("Changed name to %q, version %d\n", loaded.name, loaded.Version())

	fmt.Println()
	fmt.Println("CQRS + Event Sourcing flow completed successfully!")

	// --- EventCatalog Generation ---

	fmt.Println()
	fmt.Println("=== EventCatalog Generation ===")
	fmt.Println()

	outputDir := filepath.Join(".", "eventcatalog-output")

	if err := generateEventCatalog(outputDir); err != nil {
		log.Fatalf("generate event catalog: %v", err)
	}

	fmt.Printf("EventCatalog written to %s\n", outputDir)
	fmt.Println()
	fmt.Println(
		"Open it with EventCatalog (https://www.eventcatalog.dev/) to visualize your event-driven architecture.",
	)
}
