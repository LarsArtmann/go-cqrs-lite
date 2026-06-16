package commands

import (
	"context"
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type DeleteTodoCommand struct{ command.BasicCommand }

func NewDeleteTodoCommand(todoID id.AggregateID) (*DeleteTodoCommand, error) {
	core, err := command.New(aggregate.CommandDelete, todoID)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure,
			"todo.commands.delete_todo.1",
			"new delete todo command for todo %s: %v",
			todoID,
			err,
		)
	}

	return &DeleteTodoCommand{BasicCommand: *core}, nil
}

type DeleteTodoHandler struct{ CommandHandler }

func NewDeleteTodoHandler(events event.Store, eventBus event.Publisher) *DeleteTodoHandler {
	return &DeleteTodoHandler{CommandHandler: NewHandler(events, eventBus)}
}

func (h *DeleteTodoHandler) Handle(ctx context.Context, cmd command.Command) error {
	typed, err := requireCommandType[*DeleteTodoCommand](cmd, "*DeleteTodoCommand")
	if err != nil {
		return err
	}

	return h.execute(
		ctx, typed.AggregateID(),
		aggregate.DecideDelete(typed.AggregateID()),
	)
}

func (c *DeleteTodoCommand) MarshalJSON() ([]byte, error) {
	type Alias DeleteTodoCommand

	return json.Marshal(&struct {
		*Alias

		Type string `json:"type"`
	}{
		Type:  string(CommandTypeDelete),
		Alias: (*Alias)(c),
	})
}
