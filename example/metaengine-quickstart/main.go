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

var errUnknownEventType = errors.New("unknown event type")

// ── Domain Events ──
//
// This quickstart's point is the smallest possible declaration surface:
// plain structs. The task identity is a branded id.StreamID in the events,
// the read model, and the query alike; on the wire it is a bare string, so
// branding costs the demo nothing. The two stamp columns mirror the Record's
// own string fields, which is what the file-level strong-id directive covers.
//branching-flow:ignore file

type TaskCreated struct {
	ID     id.StreamID
	Title  string
	Status string
}

type TaskUpdated struct {
	ID     id.StreamID
	Title  string
	Status string
}

type TaskDeleted struct {
	ID id.StreamID
}

// ── Read Model ──

type TaskView struct {
	ID     id.StreamID
	Title  string
	Status string

	//cqrs-lint:ignore(A032) the Record stamps this column with its own string field
	StreamID string // auto-stamped from Record

	Version int64 // auto-stamped from Record

	//cqrs-lint:ignore(A032) the Record stamps this column with its own string field
	CorrelationID string // auto-stamped from Record
}

type TaskQuery struct {
	ID id.StreamID
}

func main() {
	ctx := context.Background()

	sections := []struct {
		title string
		run   func(context.Context) error
	}{
		{title: "1/4 Map ADT: CRUD task view (convention folds)", run: runTaskDemo},
		{title: "2/4 Graph ADT: follow network traversal", run: runGraphDemo},
		{title: "3/4 Vector ADT: k-NN semantic search", run: runVectorDemo},
		{title: "4/4 Operator config: boot from cqrs.yaml", run: runConfigFileDemo},
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

	defer metaengine.DeferClose(store)

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
	taskID, err := id.ParseStreamID("task-1")
	if err != nil {
		return err
	}

	streamID := id.NewStreamID()
	correlationID := id.NewCorrelationID()

	// Create
	createPayload, _ := json.Marshal(
		TaskCreated{ID: taskID, Title: "Build metaengine app", Status: "open"},
	)

	createEvt, _ := event.New("TaskCreated", streamID, "Task", event.Version(1), createPayload,
		event.WithCorrelationID(correlationID),
	)
	if err := adapter.Handle(ctx, createEvt); err != nil {
		return fmt.Errorf("handle create: %w", err)
	}

	// Update
	const taskUpdatedPosition = 2

	updatePayload, _ := json.Marshal(
		TaskUpdated{ID: taskID, Title: "Build metaengine app", Status: "in_progress"},
	)

	updateEvt, _ := event.New(
		"TaskUpdated",
		streamID,
		"Task",
		event.Version(taskUpdatedPosition),
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
		TaskQuery{ID: taskID},
	)
	if err != nil {
		return fmt.Errorf("ExecuteTyped: %w", err)
	}

	fmt.Printf("Task: %+v\n", result)
	fmt.Printf("  StreamID:      %s (auto-stamped from Record)\n", result.StreamID)
	fmt.Printf("  Version:       %d (auto-stamped from Record)\n", result.Version)
	fmt.Printf("  CorrelationID: %s (auto-stamped from Record)\n", result.CorrelationID)

	// 6. Delete
	deletePayload, _ := json.Marshal(TaskDeleted{ID: taskID})

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

	_, err = metaengine.ExecuteTyped[TaskQuery, TaskView](ctx, store, TaskQuery{ID: taskID})
	if err != nil {
		fmt.Printf("\nAfter delete: %v (expected not-found error)\n", err)
	}

	return nil
}
