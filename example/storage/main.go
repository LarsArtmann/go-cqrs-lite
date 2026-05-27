// Package main demonstrates using the storage module with a SQLite event store.
//
// It shows:
//  1. Opening an in-memory SQLite database
//  2. Initializing the event store schema
//  3. Creating events and saving them
//  4. Loading events back from the store
//
// Run with: go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/storage"
)

type UserCreated struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	ctx := context.Background()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if err := storage.SQLiteInitSchema(ctx, db); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	backend, err := storage.NewSQLiteBackend(db)
	if err != nil {
		log.Fatalf("create backend: %v", err)
	}

	eventStore := backend.EventStore()

	userID := id.NewAggregateID()
	payload, _ := json.Marshal(UserCreated{Name: "Alice", Email: "alice@example.com"})

	evt, err := event.NewEvent("user.created", userID, "User", event.Version(1), payload)
	if err != nil {
		log.Fatalf("create event: %v", err)
	}

	fmt.Printf(
		"Saving event: type=%s aggregate=%s version=%d\n",
		evt.Type(),
		evt.AggregateID(),
		evt.Version(),
	)

	if err := eventStore.Save(
		ctx,
		"User",
		userID,
		[]event.Event{evt},
		event.Version(0),
	); err != nil {
		log.Fatalf("save events: %v", err)
	}

	loaded, err := eventStore.Load(ctx, "User", userID)
	if err != nil {
		log.Fatalf("load events: %v", err)
	}

	fmt.Printf("\nLoaded %d event(s):\n", len(loaded))
	for _, e := range loaded {
		var user UserCreated
		_ = json.Unmarshal(e.Payload(), &user)
		fmt.Printf("  type=%s user=%s email=%s\n", e.Type(), user.Name, user.Email)
	}

	fmt.Println("\nDone.")
}
