package commands

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
)

type CreateTodoCommand struct {
	command.Core
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
}

func NewCreateTodoCommand(
	todoID id.AggregateID,
	title, description string,
	priority int,
	tags []string,
) (*CreateTodoCommand, error) {
	core, err := command.New(aggregate.CommandCreate, todoID)
	if err != nil {
		return nil, fmt.Errorf("new create todo command for todo %s: %w", todoID, err)
	}
	return &CreateTodoCommand{
		Core:        *core,
		Title:       title,
		Description: description,
		Priority:    priority,
		Tags:        tags,
	}, nil
}

type CreateTodoHandler struct{ CommandHandler }

func NewCreateTodoHandler(events event.Store, eventBus event.Publisher) *CreateTodoHandler {
	return &CreateTodoHandler{CommandHandler: NewHandler(events, eventBus)}
}

func (h *CreateTodoHandler) Handle(ctx context.Context, cmd command.Command) error {
	createCmd, err := requireCommandType[*CreateTodoCommand](cmd, "*CreateTodoCommand")
	if err != nil {
		return err
	}
	return h.execute(ctx, createCmd.AggregateID(),
		aggregate.DecideCreate(
			createCmd.AggregateID(),
			createCmd.Title,
			createCmd.Description,
			createCmd.Priority,
			createCmd.Tags,
		),
	)
}

func (c *CreateTodoCommand) MarshalJSON() ([]byte, error) {
	return MarshalCommandJSON(c, CommandTypeCreate)
}
