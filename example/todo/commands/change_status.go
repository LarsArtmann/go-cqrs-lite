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

type ChangeStatusCommand struct {
	command.BasicCommand

	Status domain.TodoStatus `json:"status"`
}

func NewChangeStatusCommand(
	todoID id.AggregateID,
	status domain.TodoStatus,
) (*ChangeStatusCommand, error) {
	core, err := command.New(aggregate.CommandChangeStatus, todoID)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure,
			"todo.commands.change_status.1",
			"new change status command for todo %s: %v",
			todoID,
			err,
		)
	}

	return &ChangeStatusCommand{
		BasicCommand: *core,
		Status:       status,
	}, nil
}

type ChangeStatusHandler struct{ CommandHandler }

func NewChangeStatusHandler(events event.Store, eventBus event.Publisher) *ChangeStatusHandler {
	return &ChangeStatusHandler{CommandHandler: NewHandler(events, eventBus)}
}

func (h *ChangeStatusHandler) Handle(ctx context.Context, cmd command.Command) error {
	typed, err := requireCommandType[*ChangeStatusCommand](cmd, "*ChangeStatusCommand")
	if err != nil {
		return err
	}

	return h.execute(
		ctx, typed.AggregateID(),
		aggregate.DecideChangeStatus(typed.AggregateID(), typed.Status),
	)
}

func (c *ChangeStatusCommand) MarshalJSON() ([]byte, error) {
	type Alias ChangeStatusCommand

	return json.Marshal(&struct {
		*Alias

		Type string `json:"type"`
	}{
		Type:  string(CommandTypeChangeStatus),
		Alias: (*Alias)(c),
	})
}
