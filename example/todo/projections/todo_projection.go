package projections

import (
	"context"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

type TodoProjection struct {
	store domain.TodoReadModel
}

func NewTodoProjection(store domain.TodoReadModel) *TodoProjection {
	return &TodoProjection{store: store}
}

func (p *TodoProjection) Name() string { return "todo-read-model" }

func (p *TodoProjection) EventTypes() []event.Type {
	return []event.Type{
		aggregate.EventCreated,
		aggregate.EventUpdated,
		aggregate.EventStatusChanged,
		aggregate.EventCompleted,
		aggregate.EventDeleted,
	}
}

func (p *TodoProjection) Handle(_ context.Context, evt event.Event) error {
	switch evt.Type() {
	case aggregate.EventDeleted:
		todoID, err := domain.ParseTodoID(evt.AggregateID().String())
		if err != nil {
			return event.Newf(event.Infrastructure, "todo.projections.todo_projection.1", "failed to parse aggregate ID %s: %v", evt.AggregateID(), err)
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
		return event.Newf(event.Infrastructure, "todo.projections.todo_projection.2", "handle %s event %s: %v", evt.Type(), evt.AggregateID(), err)
	}

	return p.store.Put(todo)
}

func payloadToTodo(evt event.Event) (*domain.Todo, error) {
	todoID, err := domain.ParseTodoID(evt.AggregateID().String())
	if err != nil {
		return nil, event.Newf(event.Infrastructure, "todo.projections.todo_projection.3", "failed to parse aggregate ID %s: %v", evt.AggregateID(), err)
	}

	var payload aggregate.TodoPayload

	codec := codecpkg.JSONCodec{}
	if err := codec.Decode(evt.Payload(), &payload); err != nil {
		return nil, event.Newf(event.Infrastructure, "todo.projections.todo_projection.4", 
			"failed to decode todo payload for event %s: %v",
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

var _ event.Projection = (*TodoProjection)(nil)
