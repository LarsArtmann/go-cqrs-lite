// Example: metaengine-quickstart
//
// Demonstrates the Record-aware metaengine pipeline in three sections:
// 1. Maps: CRUD folds by Created/Updated/Deleted convention (zero boilerplate)
// 2. Graph: a follow network folded into edges, queried by traversal depth
// 3. Vector: document embeddings folded into vectors, queried by k-NN search
//
// All three go through the same pipeline: declare folds + queries, Plan a
// store over an engine, apply records, execute typed queries. Swap the
// Memory engine for any backend without touching the domain code.
package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

const demoTaskID = "task-1"

var errUnknownEventType = errors.New("unknown event type")

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

	sections := []struct {
		title string
		run   func(context.Context) error
	}{
		{title: "1/3 Map ADT: CRUD task view (convention folds)", run: runTaskDemo},
		{title: "2/3 Graph ADT: follow network traversal", run: runGraphDemo},
		{title: "3/3 Vector ADT: k-NN semantic search", run: runVectorDemo},
	}

	for _, section := range sections {
		fmt.Printf("\n═══ %s ═══\n", section.title)
		if err := section.run(ctx); err != nil {
			log.Fatalf("%s: %v", section.title, err)
		}
	}
}

func runTaskDemo(ctx context.Context) error {
	// 1. Auto-generate CRUD folds by naming convention — zero boilerplate.
	folds, err := metaengine.AutoCRUDByConvention[TaskView]("ID",
		TaskCreated{}, TaskUpdated{}, TaskDeleted{},
	)
	if err != nil {
		return fmt.Errorf("AutoCRUDByConvention: %w", err)
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
		return fmt.Errorf("plan: %w", err)
	}

	defer func() { _ = store.Close() }()

	// 3. Wire the projection adapter — this bridges event.Event → record.Record.
	decoder := func(eventType string, payload []byte) (any, error) {
		switch eventType {
		case "TaskCreated":
			var e TaskCreated

			return e, json.Unmarshal(payload, &e)
		case "TaskUpdated":
			var e TaskUpdated

			return e, json.Unmarshal(payload, &e)
		case "TaskDeleted":
			var e TaskDeleted

			return e, json.Unmarshal(payload, &e)
		default:
			return nil, fmt.Errorf("%w: %s", errUnknownEventType, eventType)
		}
	}

	adapter := projectionadapter.New("tasks-projection", store, decoder)

	// 4. Create events and process them through the adapter.
	streamID := id.NewStreamID()
	correlationID := id.NewCorrelationID()

	// Create
	createPayload, _ := json.Marshal(
		TaskCreated{ID: demoTaskID, Title: "Build metaengine app", Status: "open"},
	)

	createEvt, _ := event.New("TaskCreated", streamID, "Task", event.Version(1), createPayload,
		event.WithCorrelationID(correlationID),
	)
	if err := adapter.Handle(ctx, createEvt); err != nil {
		return fmt.Errorf("handle create: %w", err)
	}

	// Update
	updatePayload, _ := json.Marshal(
		TaskUpdated{ID: demoTaskID, Title: "Build metaengine app", Status: "in_progress"},
	)

	updateEvt, _ := event.New(
		"TaskUpdated",
		streamID,
		"Task",
		event.Version(2), //nolint:mnd // sequential stream position
		updatePayload,
		event.WithCorrelationID(correlationID),
	)
	if err := adapter.Handle(ctx, updateEvt); err != nil {
		return fmt.Errorf("handle update: %w", err)
	}

	// 5. Query the result — Record metadata is auto-stamped.
	result, err := metaengine.ExecuteTyped[TaskQuery, TaskView](
		ctx,
		store,
		TaskQuery{ID: demoTaskID},
	)
	if err != nil {
		return fmt.Errorf("ExecuteTyped: %w", err)
	}

	fmt.Printf("Task: %+v\n", result)
	fmt.Printf("  StreamID:      %s (auto-stamped from Record)\n", result.StreamID)
	fmt.Printf("  Version:       %d (auto-stamped from Record)\n", result.Version)
	fmt.Printf("  CorrelationID: %s (auto-stamped from Record)\n", result.CorrelationID)

	// 6. Delete
	deletePayload, _ := json.Marshal(TaskDeleted{ID: demoTaskID})

	deleteEvt, _ := event.New(
		"TaskDeleted",
		streamID,
		"Task",
		event.Version(3), //nolint:mnd // sequential stream position
		deletePayload,
	)
	if err := adapter.Handle(ctx, deleteEvt); err != nil {
		return fmt.Errorf("handle delete: %w", err)
	}

	_, err = metaengine.ExecuteTyped[TaskQuery, TaskView](ctx, store, TaskQuery{ID: demoTaskID})
	if err != nil {
		fmt.Printf("\nAfter delete: %v (expected not-found error)\n", err)
	}

	return nil
}
