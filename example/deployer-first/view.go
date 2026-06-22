package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// TodoView is the materialized read model — a flat, query-optimized projection
// of the Todo aggregate. It is built by Materialize from the event stream, not
// by querying the aggregate directly.
type TodoView struct {
	Title      string `json:"title"`
	Completed  bool   `json:"completed"`
	Tombstoned bool   `json:"tombstoned"`
}

// IsTombstoned reports whether this view record has been soft-deleted.
// Materialize.List consults this (via TombstonePolicy) to filter tombstoned
// records out of default results while keeping them in the store.
func (v *TodoView) IsTombstoned() bool { return v.Tombstoned }

// todoKey extracts the read-model key from an event. Every event this projection
// handles carries the aggregate ID, so the key is simply that.
func todoKey(evt event.Event) (id.AggregateID, error) {
	return evt.AggregateID(), nil
}

// configureMaterialize wires the consumer's OnCreate / OnUpdate / OnTombstone
// callbacks. This is where the consumer's business logic for building the view
// lives. The deployer already chose WHERE the view is stored (see main.go).
func configureMaterialize(mat *stack.Materialize[TodoView, id.AggregateID]) {
	mat.OnCreate = func(_ context.Context, evt event.Event) (*TodoView, error) {
		p, err := event.DecodePayload[TodoCreatedPayload](evt, jsonCodec)
		if err != nil {
			return nil, fmt.Errorf("decode todo.created: %w", err)
		}

		return &TodoView{Title: p.Title}, nil
	}

	mat.OnUpdate = func(_ context.Context, evt event.Event, existing *TodoView) (*TodoView, error) {
		if evt.Type() == eventTodoCompleted {
			updated := *existing
			updated.Completed = true

			return &updated, nil
		}

		return existing, nil
	}

	mat.OnTombstone = func(_ context.Context, _ event.Event, existing *TodoView) (*TodoView, error) {
		// Mark the record tombstoned: it stays in the store but is excluded
		// from default List results (soft-delete, no data loss).
		if existing == nil {
			return &TodoView{Tombstoned: true}, nil
		}

		updated := *existing
		updated.Tombstoned = true

		return &updated, nil
	}
}
