package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
)

type DeleteTodoCommand struct{ command.Core }

func NewDeleteTodoCommand(todoID id.AggregateID) (*DeleteTodoCommand, error) {
	core, err := command.New(aggregate.CommandDelete, todoID)
	if err != nil {
		return nil, fmt.Errorf("new delete todo command for todo %s: %w", todoID, err)
	}
	return &DeleteTodoCommand{Core: *core}, nil
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
		Type string `json:"type"`
		*Alias
	}{
		Type:  string(CommandTypeDelete),
		Alias: (*Alias)(c),
	})
}
