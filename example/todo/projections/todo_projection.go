package projections

import (
	"context"
	"fmt"

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
	case aggregate.EventCreated:
		return p.handleCreated(evt)
	case aggregate.EventUpdated:
		return p.handleUpdated(evt)
	case aggregate.EventStatusChanged:
		return p.handleStatusChanged(evt)
	case aggregate.EventCompleted:
		return p.handleCompleted(evt)
	case aggregate.EventDeleted:
		return p.handleDeleted(evt)
	}
	return nil
}

func (p *TodoProjection) handleCreated(evt event.Event) error {
	todo, err := payloadToTodo(evt)
	if err != nil {
		return fmt.Errorf("handle created event %s: %w", evt.AggregateID(), err)
	}
	return p.store.Put(todo)
}

func (p *TodoProjection) handleUpdated(evt event.Event) error {
	todo, err := payloadToTodo(evt)
	if err != nil {
		return fmt.Errorf("handle updated event %s: %w", evt.AggregateID(), err)
	}
	return p.store.Put(todo)
}

func (p *TodoProjection) handleStatusChanged(evt event.Event) error {
	todo, err := payloadToTodo(evt)
	if err != nil {
		return fmt.Errorf("handle status changed event %s: %w", evt.AggregateID(), err)
	}
	return p.store.Put(todo)
}

func (p *TodoProjection) handleCompleted(evt event.Event) error {
	todo, err := payloadToTodo(evt)
	if err != nil {
		return fmt.Errorf("handle completed event %s: %w", evt.AggregateID(), err)
	}
	return p.store.Put(todo)
}

func (p *TodoProjection) handleDeleted(evt event.Event) error {
	todoID, err := domain.ParseTodoID(evt.AggregateID().String())
	if err != nil {
		return fmt.Errorf("failed to parse aggregate ID %s: %w", evt.AggregateID(), err)
	}
	return p.store.Delete(todoID)
}

var codec = event.JSONCodec{}

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
