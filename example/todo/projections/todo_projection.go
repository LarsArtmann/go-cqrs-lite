package projections

import (
	"context"
	"fmt"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

type TodoProjection struct {
	store domain.TodoReadModel
}

func NewTodoProjection(store domain.TodoReadModel) *TodoProjection {
	return &TodoProjection{store: store}
}

func (p *TodoProjection) Handle(ctx context.Context, evt event.Event) error {
	switch evt.Type() {
	case aggregate.EventDeleted:
		todoID, err := domain.ParseTodoID(evt.AggregateID().String())
		if err != nil {
			return fmt.Errorf("failed to parse aggregate ID %s: %w", evt.AggregateID(), err)
		}

		return p.store.Delete(todoID)
	case aggregate.EventCreated, aggregate.EventUpdated,
		aggregate.EventStatusChanged, aggregate.EventCompleted:
		return p.handleUpsert(evt)
	}

	return nil
}

func (p *TodoProjection) handleUpsert(evt event.Event) error {
	todo, err := payloadToTodo(evt)
	if err != nil {
		return fmt.Errorf("handle %s event %s: %w", evt.Type(), evt.AggregateID(), err)
	}

	return p.store.Put(todo)
}

var codec = codecpkg.JSONCodec{}

func payloadToTodo(evt event.Event) (*domain.Todo, error) {
	todoID, err := domain.ParseTodoID(evt.AggregateID().String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse aggregate ID %s: %w", evt.AggregateID(), err)
	}

	var payload aggregate.TodoPayload
	if err := codec.Decode(evt.Payload(), &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode todo payload for event %s: %w",
			evt.AggregateID(),
			err,
		)
	}

	return &domain.Todo{
		ID:          todoID,
		Title:       payload.Title,
		Description: payload.Description,
		Status:      payload.Status,
		Priority:    payload.Priority,
		Tags:        payload.Tags,
		CreatedAt:   payload.CreatedAt,
		UpdatedAt:   payload.UpdatedAt,
		CompletedAt: payload.CompletedAt,
		Version:     int64(evt.Version()),
	}, nil
}
