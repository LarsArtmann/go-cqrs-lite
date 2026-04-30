// Package main demonstrates a complete CQRS + Event Sourcing flow
// using go-cqrs-lite with the in-memory implementations.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

// User aggregate root — a real domain model using Core for event sourcing.
type User struct {
	*aggregate.Core

	email string
	name  string
}

func newUser(email string) *User {
	aggID := id.NewAggregateID()

	return &User{
		Core: aggregate.NewCore(aggID, event.AggregateType("User")),
		email: email,
	}
}

func (u *User) Apply(evt event.Event) error {
	switch evt.Type() {
	case "UserCreated":
		u.email = string(evt.Payload())
	case "UserNameChanged":
		u.name = string(evt.Payload())
	}

	return nil
}

func (u *User) ApplySnapshot(_ []byte) error {
	return nil
}

func (u *User) LoadEvents(events []event.Event) error {
	return u.LoadFromHistory(u, events)
}

// ChangeName records a UserNameChanged event.
func (u *User) ChangeName(ctx context.Context, name string) error {
	evt, err := event.NewEvent(
		"UserNameChanged",
		u.ID(),
		u.Type(),
		u.Version()+1,
		[]byte(name),
	)
	if err != nil {
		return fmt.Errorf("create UserNameChanged event: %w", err)
	}

	u.RecordEvent(ctx, evt)

	u.name = name

	return nil
}

func main() {
	ctx := context.Background()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()

	repo := aggregate.NewRepository(store, bus)

	user := newUser("alice@example.com")

	created, err := event.NewEvent(
		"UserCreated",
		user.ID(),
		user.Type(),
		1,
		[]byte(user.email),
	)
	if err != nil {
		log.Fatalf("create event: %v", err)
	}

	user.RecordEvent(ctx, created)

	err = repo.Save(ctx, user)
	if err != nil {
		log.Fatalf("save user: %v", err)
	}

	fmt.Fprintf(os.Stdout, "Created user %s (%s)\n", user.ID(), user.email)

	loaded := &User{Core: aggregate.NewCore(user.ID(), event.AggregateType("User"))}

	err = repo.Load(ctx, loaded)
	if err != nil {
		log.Fatalf("load user: %v", err)
	}

	fmt.Fprintf(os.Stdout, "Loaded user %s, version %d\n", loaded.ID(), loaded.Version())

	err = loaded.ChangeName(ctx, "Alice Smith")
	if err != nil {
		log.Fatalf("change name: %v", err)
	}

	err = repo.Save(ctx, loaded)
	if err != nil {
		log.Fatalf("save name change: %v", err)
	}

	fmt.Fprintf(os.Stdout, "Changed name to %q, version %d\n", loaded.name, loaded.Version())

	fmt.Fprintln(os.Stdout, "CQRS + Event Sourcing flow completed successfully!")
}
