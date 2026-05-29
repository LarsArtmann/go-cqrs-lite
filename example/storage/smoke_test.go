package main

import (
	"context"
	"encoding/json"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/storage"
)

func TestSQLite_SaveAndLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := storage.SQLiteInitSchema(ctx, db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	backend, err := storage.NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("create backend: %v", err)
	}

	eventStore := backend.EventStore()

	userID := id.NewAggregateID()
	payload, _ := json.Marshal(UserCreated{Name: "Alice", Email: "alice@example.com"})

	evt, err := event.NewEvent("user.created", userID, "User", event.Version(1), payload)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := eventStore.Save(
		ctx,
		"User",
		userID,
		[]event.Event{evt},
		event.Version(0),
	); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := eventStore.Load(ctx, "User", userID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	var user UserCreated
	_ = json.Unmarshal(loaded[0].Payload(), &user)

	if user.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user.Name)
	}
}
