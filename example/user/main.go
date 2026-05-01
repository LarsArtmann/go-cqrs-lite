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

	"github.com/larsartmann/go-cqrs-lite/catalog"
	catalogadapters "github.com/larsartmann/go-cqrs-lite/catalog/adapters"
	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

// --- Typed event payloads (with struct tags for schema generation) ---

type UserCreatedPayload struct {
	Email string `json:"email" description:"The user's email address"`
}

type UserNameChangedPayload struct {
	Name string `json:"name" description:"The new display name"`
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
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal UserCreated: %w", err)
		}
		u.email = p.Email
	case "UserNameChanged":
		var p UserNameChangedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
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
	fmt.Println("Open it with EventCatalog (https://www.eventcatalog.dev/) to visualize your event-driven architecture.")
}

func generateEventCatalog(outputDir string) error {
	builder := catalogadapters.NewBuilder("User Service", "1.0.0")
	builder.AddService("user-svc", "User Service", "1.0.0", "Manages user accounts")

	builder.AddEvent("user-svc", newUserCreatedEvent())
	builder.AddEvent("user-svc", newUserNameChangedEvent())

	builder.AddDomain("identity", "Identity", "User identity and account management", []string{"user-svc"})

	cat := builder.Build()

	printCatalogSummary(cat)

	return builder.ExportEventCatalog(outputDir)
}

func newUserCreatedEvent() event.Catalogable {
	aggID := id.NewAggregateID()

	core, err := event.NewCatalogCore(
		"UserCreated",
		aggID,
		"User",
		1,
		nil,
		event.CatalogMeta{
			Name:          "User Created",
			Version:       "1.0.0",
			Summary:       "Fired when a new user account is created",
			AggregateType: "User",
		},
	)
	if err != nil {
		log.Fatalf("create UserCreated catalog core: %v", err)
	}

	return &userCreatedEvent{CatalogCore: core}
}

func newUserNameChangedEvent() event.Catalogable {
	aggID := id.NewAggregateID()

	core, err := event.NewCatalogCore(
		"UserNameChanged",
		aggID,
		"User",
		1,
		nil,
		event.CatalogMeta{
			Name:          "User Name Changed",
			Version:       "1.0.0",
			Summary:       "Fired when a user changes their display name",
			AggregateType: "User",
		},
	)
	if err != nil {
		log.Fatalf("create UserNameChanged catalog core: %v", err)
	}

	return &userNameChangedEvent{CatalogCore: core}
}

// --- Catalogable event types (with struct tags for schema reflection) ---

type userCreatedEvent struct {
	*event.CatalogCore

	Email string `json:"email" description:"The user's email address"`
}

type userNameChangedEvent struct {
	*event.CatalogCore

	Name string `json:"name" description:"The new display name"`
}

func printCatalogSummary(cat *catalog.Catalog) {
	fmt.Printf("Catalog: %s v%s\n", cat.Title, cat.Version)

	for _, svc := range cat.Services {
		fmt.Printf("  Service: %s (%s)\n", svc.Name, svc.ID)

		for _, evt := range svc.Events {
			schemaInfo := "(no schema)"
			if evt.Schema != nil {
				schemaInfo = fmt.Sprintf("%d properties", len(evt.Schema.Properties))
			}

			fmt.Printf("    Event: %s — %s %s\n", evt.Name, evt.Summary, schemaInfo)
		}
	}

	for _, domain := range cat.Domains {
		fmt.Printf("  Domain: %s (%s)\n", domain.Name, domain.ID)
	}
}
