package aggregate

import (
	"errors"
	"time"

	cqrsCommand "github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

var (
	ErrTodoAlreadyExists = errors.New("todo already exists")
	ErrUnknownEventType  = errors.New("unknown event type")
)

const AggregateType event.AggregateType = "todo"

const (
	EventCreated       event.Type = "todo.created"
	EventUpdated       event.Type = "todo.updated"
	EventDeleted       event.Type = "todo.deleted"
	EventCompleted     event.Type = "todo.completed"
	EventStatusChanged event.Type = "todo.status_changed"
)

type TodoPayload struct {
	Title       domain.Title       `json:"title"`
	Description domain.Description `json:"description"`
	Status      domain.TodoStatus  `json:"status"`
	Priority    domain.Priority    `json:"priority"`
	Tags        []string           `json:"tags"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	CompletedAt *time.Time         `json:"completedAt,omitempty"`
}

const (
	CommandCreate       cqrsCommand.Type = "todo.create"
	CommandUpdate       cqrsCommand.Type = "todo.update"
	CommandDelete       cqrsCommand.Type = "todo.delete"
	CommandChangeStatus cqrsCommand.Type = "todo.change_status"
)
