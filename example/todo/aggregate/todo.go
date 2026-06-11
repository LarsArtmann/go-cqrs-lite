package aggregate

import (
	"errors"
	"time"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v2"
	cqrsCommand "github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
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

var codec = codecpkg.JSONCodec{}

type TodoPayload struct {
	Title       domain.Title      `json:"title"`
	Description domain.Description `json:"description"`
	Status      domain.TodoStatus `json:"status"`
	Priority    domain.Priority   `json:"priority"`
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
