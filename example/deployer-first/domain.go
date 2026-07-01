package main

import (
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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

// Event payloads — auto-marshaled via event.New(), using event.DefaultCodec
// (set to CBOR in main.go via stack.WithEventCodec).
// Decoding uses event.DecodePayloadAuto, which dispatches based on each
// event's encoding stamp — so mixed JSON+CBOR streams decode correctly.

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

func applyTodo(state TodoState, evt event.Event) (TodoState, error) {
	switch evt.Type() {
	case eventTodoCreated:
		p, err := event.DecodePayloadAuto[TodoCreatedPayload](evt)
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

func decideCreate(aggID id.AggregateID, title string) DecideFunc {
	return func(state TodoState, version event.Version) ([]event.Event, error) {
		if state.Exists {
			return nil, event.NewConflict("todo.create.exists", "todo already exists")
		}

		if title == "" {
			return nil, event.NewRejection("todo.create.title_required", "title is required")
		}

		evt, err := event.New(
			eventTodoCreated, aggID, aggregateType, version.Increment(),
			TodoCreatedPayload{Title: title},
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

		evt, err := event.New(
			eventTodoCompleted, aggID, aggregateType, version.Increment(),
			TodoCompletedPayload{},
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

		evt, err := event.New(
			eventTodoDeleted, aggID, aggregateType, version.Increment(),
			TodoDeletedPayload{Reason: reason},
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
