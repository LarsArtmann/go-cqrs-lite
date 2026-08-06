// Example: metaengine-quickstart
//
// Demonstrates the full Record-aware ES-native pipeline:
// 1. Define event types following the Created/Updated/Deleted convention
// 2. Use AutoCRUDByConvention to auto-generate folds (zero boilerplate)
// 3. Wire through projectionadapter (event.Event → record.Record bridge)
// 4. Apply events and query results — Record metadata is auto-stamped
//
// This is the simplest possible metaengine consumer: 3 event types, 1 view,
// and the convention-based API eliminates all manual fold code.
package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ── Domain Events ──

type TaskCreated struct {
	ID     string
	Title  string
	Status string
}

type TaskUpdated struct {
	ID     string
	Title  string
	Status string
}

type TaskDeleted struct {
	ID string
}

// ── Read Model ──

type TaskView struct {
	ID            string
	Title         string
	Status        string
	StreamID      string // auto-stamped from Record
	Version       int64  // auto-stamped from Record
	CorrelationID string // auto-stamped from Record
}

type TaskQuery struct {
	ID string
}

func main() {
	ctx := context.Background()

	// 1. Auto-generate CRUD folds by naming convention — zero boilerplate.
	folds, err := metaengine.AutoCRUDByConvention[TaskView]("ID",
		TaskCreated{}, TaskUpdated{}, TaskDeleted{},
	)
	if err != nil {
		log.Fatalf("AutoCRUDByConvention: %v", err)
	}

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	query := metaengine.Query[TaskQuery, TaskView]("tasks", foldArgs...)

	// 2. Plan the store with the Memory engine.
	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		query,
	)
	if err != nil {
		log.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// 3. Wire the projection adapter — this bridges event.Event → record.Record.
	decoder := func(eventType string, payload []byte) (any, error) {
		switch eventType {
		case "taskCreated":
			var e TaskCreated
			return e, json.Unmarshal(payload, &e)
		case "taskUpdated":
			var e TaskUpdated
			return e, json.Unmarshal(payload, &e)
		case "taskDeleted":
			var e TaskDeleted
			return e, json.Unmarshal(payload, &e)
		default:
			return nil, fmt.Errorf("unknown event type: %s", eventType)
		}
	}

	adapter := projectionadapter.New("tasks-projection", store, decoder)

	// 4. Create events and process them through the adapter.
	streamID := id.NewStreamID()
	correlationID := id.NewCorrelationID()

	// Create
	createPayload, _ := json.Marshal(TaskCreated{ID: "task-1", Title: "Build metaengine app", Status: "open"})
	createEvt, _ := event.NewEvent("taskCreated", streamID, "Task", event.Version(1), createPayload,
		event.WithCorrelationID(correlationID),
	)
	if err := adapter.Handle(ctx, createEvt); err != nil {
		log.Fatalf("Handle create: %v", err)
	}

	// Update
	updatePayload, _ := json.Marshal(TaskUpdated{ID: "task-1", Title: "Build metaengine app", Status: "in_progress"})
	updateEvt, _ := event.NewEvent("taskUpdated", streamID, "Task", event.Version(2), updatePayload,
		event.WithCorrelationID(correlationID),
	)
	if err := adapter.Handle(ctx, updateEvt); err != nil {
		log.Fatalf("Handle update: %v", err)
	}

	// 5. Query the result — Record metadata is auto-stamped.
	result, err := metaengine.ExecuteTyped[TaskQuery, TaskView](ctx, store, TaskQuery{ID: "task-1"})
	if err != nil {
		log.Fatalf("ExecuteTyped: %v", err)
	}

	fmt.Printf("Task: %+v\n", result)
	fmt.Printf("  StreamID:      %s (auto-stamped from Record)\n", result.StreamID)
	fmt.Printf("  Version:       %d (auto-stamped from Record)\n", result.Version)
	fmt.Printf("  CorrelationID: %s (auto-stamped from Record)\n", result.CorrelationID)

	// 6. Delete
	deletePayload, _ := json.Marshal(TaskDeleted{ID: "task-1"})
	deleteEvt, _ := event.NewEvent("taskDeleted", streamID, "Task", event.Version(3), deletePayload)
	if err := adapter.Handle(ctx, deleteEvt); err != nil {
		log.Fatalf("Handle delete: %v", err)
	}

	_, err = metaengine.ExecuteTyped[TaskQuery, TaskView](ctx, store, TaskQuery{ID: "task-1"})
	fmt.Printf("\nAfter delete: %v (expected not-found error)\n", err)
}
