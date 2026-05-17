package commands

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
)

type UpdateTodoCommand struct {
	command.Core
	Title       string `json:"title"`
	Description string `json:"description"`
}

func NewUpdateTodoCommand(
	todoID id.AggregateID,
	title, description string,
) (*UpdateTodoCommand, error) {
	core, err := command.New(aggregate.CommandUpdate, todoID)
	if err != nil {
		return nil, fmt.Errorf("new update todo command for todo %s: %w", todoID, err)
	}
	return &UpdateTodoCommand{
		Core:        *core,
		Title:       title,
		Description: description,
	}, nil
}

type UpdateTodoHandler struct{ CommandHandler }

func NewUpdateTodoHandler(events event.Store, eventBus event.Publisher) *UpdateTodoHandler {
	return &UpdateTodoHandler{CommandHandler: NewHandler(events, eventBus)}
}

func (h *UpdateTodoHandler) Handle(ctx context.Context, cmd command.Command) error {
	typed, err := requireCommandType[*UpdateTodoCommand](cmd, "*UpdateTodoCommand")
	if err != nil {
		return err
	}
	return h.execute(ctx, typed.AggregateID(),
		aggregate.DecideUpdate(typed.AggregateID(), typed.Title, typed.Description),
	)
}

func (c *UpdateTodoCommand) MarshalJSON() ([]byte, error) {
	return MarshalCommandJSON(c, CommandTypeUpdate)
}
