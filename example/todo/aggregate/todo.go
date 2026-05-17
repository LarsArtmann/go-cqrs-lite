package aggregate

import (
	"errors"
	"time"

	cqrsCommand "github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
)

var ErrTodoAlreadyExists = errors.New("todo already exists")

const AggregateType event.AggregateType = "todo"

const (
	EventCreated       event.Type = "todo.created"
	EventUpdated       event.Type = "todo.updated"
	EventDeleted       event.Type = "todo.deleted"
	EventCompleted     event.Type = "todo.completed"
	EventStatusChanged event.Type = "todo.status_changed"
)

var codec = event.JSONCodec{}

type TodoPayload struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      domain.TodoStatus `json:"status"`
	Priority    int               `json:"priority"`
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

const (
	CommandCreate       cqrsCommand.Type = "todo.create"
	CommandUpdate       cqrsCommand.Type = "todo.update"
	CommandDelete       cqrsCommand.Type = "todo.delete"
	CommandChangeStatus cqrsCommand.Type = "todo.change_status"
)
