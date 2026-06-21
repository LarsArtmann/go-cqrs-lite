package commands

import (
	"context"
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type UpdateTodoCommand struct {
	command.BasicCommand

	Title       string `json:"title"`
	Description string `json:"description"`
}

func NewUpdateTodoCommand(
	todoID id.AggregateID,
	title, description string,
) (*UpdateTodoCommand, error) {
	core, err := command.New(aggregate.CommandUpdate, todoID)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure, "todo.commands.update_todo.1",
			"new update todo command for todo %s (title=%q, description=%q): %v",
			todoID,
			title,
			description,
			err,
		)
	}

	return &UpdateTodoCommand{
		BasicCommand: *core,
		Title:        title,
		Description:  description,
	}, nil
}

type UpdateTodoHandler struct{ CommandHandler }

func NewUpdateTodoHandler(events event.Store, eventBus event.Publisher) (*UpdateTodoHandler, error) {
	ch, err := NewHandler(events, eventBus)
	if err != nil {
		return nil, err
	}

	return &UpdateTodoHandler{CommandHandler: ch}, nil
}

func (h *UpdateTodoHandler) Handle(ctx context.Context, cmd command.Command) error {
	typed, err := requireCommandType[*UpdateTodoCommand](cmd, "*UpdateTodoCommand")
	if err != nil {
		return err
	}

	return h.execute(
		ctx,
		typed.AggregateID(),
		aggregate.DecideUpdate(
			typed.AggregateID(),
			domain.Title(typed.Title),
			domain.Description(typed.Description),
		),
	)
}

func (c *UpdateTodoCommand) MarshalJSON() ([]byte, error) {
	type Alias UpdateTodoCommand

	return json.Marshal(&struct {
		*Alias

		Type string `json:"type"`
	}{
		Type:  string(CommandTypeUpdate),
		Alias: (*Alias)(c),
	})
}
