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

type CreateTodoCommand struct {
	command.BasicCommand

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
		return nil, event.Newf(
			event.Infrastructure, "todo.commands.create_todo.1",
			"new create todo command for todo %s (title=%q, description=%q, priority=%d): %v",
			todoID,
			title,
			description,
			priority,
			err,
		)
	}

	return &CreateTodoCommand{
		BasicCommand: *core,
		Title:        title,
		Description:  description,
		Priority:     priority,
		Tags:         tags,
	}, nil
}

type CreateTodoHandler struct{ CommandHandler }

func NewCreateTodoHandler(
	events event.Store,
	eventBus event.Publisher,
) (*CreateTodoHandler, error) {
	ch, err := NewHandler(events, eventBus)
	if err != nil {
		return nil, err
	}

	return &CreateTodoHandler{CommandHandler: ch}, nil
}

func (h *CreateTodoHandler) Handle(ctx context.Context, cmd command.Command) error {
	createCmd, err := requireCommandType[*CreateTodoCommand](cmd, "*CreateTodoCommand")
	if err != nil {
		return err
	}

	return h.execute(
		ctx, createCmd.AggregateID(),
		aggregate.DecideCreate(
			createCmd.AggregateID(),
			domain.Title(createCmd.Title),
			domain.Description(createCmd.Description),
			domain.Priority(createCmd.Priority),
			createCmd.Tags,
		),
	)
}

func (c *CreateTodoCommand) MarshalJSON() ([]byte, error) {
	type Alias CreateTodoCommand

	return json.Marshal(&struct {
		*Alias

		Type string `json:"type"`
	}{
		Type:  string(CommandTypeCreate),
		Alias: (*Alias)(c),
	})
}
