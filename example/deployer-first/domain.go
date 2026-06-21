package main

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// DecideFunc is the typed decision signature for the Todo aggregate.
type DecideFunc = decider.DecideFunc[TodoState]

// This file is CONSUMER code: the domain model. It knows nothing about where
// events are stored or how read models are built — that is the deployer's job
// (see main.go).

const (
	eventTodoCreated   = event.Type("todo.created")
	eventTodoCompleted = event.Type("todo.completed")
	eventTodoDeleted   = event.Type("todo.deleted")

	aggregateType = event.AggregateType("Todo")
)

// Event payloads — auto-marshaled to JSON.

type TodoCreatedPayload struct {
	Title string `json:"title"`
}

type TodoCompletedPayload struct{}

type TodoDeletedPayload struct {
	Reason string `json:"reason"`
}

// TodoState is the event-sourced aggregate state. Apply rebuilds it from events.

type TodoState struct {
	Title     string
	Completed bool
	Exists    bool
	Deleted   bool
}

var jsonCodec = codec.JSONCodec{}

func applyTodo(state TodoState, evt event.Event) (TodoState, error) {
	switch evt.Type() {
	case eventTodoCreated:
		p, err := event.DecodePayload[TodoCreatedPayload](evt, jsonCodec)
		if err != nil {
			return state, err
		}

		state.Title = p.Title
		state.Exists = true
	case eventTodoCompleted:
		state.Completed = true
	case eventTodoDeleted:
		state.Deleted = true
	}

	return state, nil
}

// Command decision functions. Each returns a DecideFunc that the
// decider.Repository executes against the current (replayed) state.

// marshalPayload serializes v to JSON, returning an Infrastructure error on failure.
func marshalPayload(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return b, nil
}

func decideCreate(aggID id.AggregateID, title string) DecideFunc {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if state.Exists {
			return nil, event.NewConflict("todo.create.exists", "todo already exists")
		}

		if title == "" {
			return nil, event.NewRejection("todo.create.title_required", "title is required")
		}

		payload, err := marshalPayload(TodoCreatedPayload{Title: title})
		if err != nil {
			return nil, event.Newf(event.Infrastructure, "todo.create.payload", "%v", err)
		}

		evt, err := event.NewEvent(
			eventTodoCreated, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(event.Infrastructure, "todo.create.1", "build event: %v", err)
		}

		return []event.Event{evt}, nil
	}
}

func decideComplete(aggID id.AggregateID) DecideFunc {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if !state.Exists {
			return nil, event.NewRejection("todo.complete.not_found", "todo does not exist")
		}

		if state.Completed {
			return nil, event.NewConflict("todo.complete.done", "todo already completed")
		}

		payload, err := marshalPayload(TodoCompletedPayload{})
		if err != nil {
			return nil, event.Newf(event.Infrastructure, "todo.complete.payload", "%v", err)
		}

		evt, err := event.NewEvent(
			eventTodoCompleted, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(event.Infrastructure, "todo.complete.1", "build event: %v", err)
		}

		return []event.Event{evt}, nil
	}
}

func decideDelete(aggID id.AggregateID, reason string) DecideFunc {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if !state.Exists {
			return nil, event.NewRejection("todo.delete.not_found", "todo does not exist")
		}

		if state.Deleted {
			return nil, event.NewConflict("todo.delete.deleted", "todo already deleted")
		}

		payload, err := marshalPayload(TodoDeletedPayload{Reason: reason})
		if err != nil {
			return nil, event.Newf(event.Infrastructure, "todo.delete.payload", "%v", err)
		}

		evt, err := event.NewEvent(
			eventTodoDeleted, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(event.Infrastructure, "todo.delete.1", "build event: %v", err)
		}

		// Soft-delete via tombstone metadata (ADR-0006). Materialize's
		// OnTombstone handles this — no hard delete.
		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, event.Newf(
				event.Infrastructure,
				"todo.delete.2",
				"mark tombstone: %v",
				markErr,
			)
		}

		return []event.Event{marked}, nil
	}
}
