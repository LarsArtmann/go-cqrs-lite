package commands

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	todoaggregate "github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

var ErrInvalidCommandType = event.NewRejection("todo.invalid_command_type", "invalid command type")

type CommandHandler struct {
	repo *decider.Repository[todoaggregate.TodoState]
}

func NewHandler(events event.Store, eventBus event.Publisher) CommandHandler {
	repo, err := decider.NewRepository(
		events, eventBus, todoaggregate.NewTodoDecider(),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create repository: %v", err))
	}

	return CommandHandler{repo: repo}
}

func (m *CommandHandler) execute(
	ctx context.Context,
	aggID id.AggregateID,
	decide decider.DecideFunc[todoaggregate.TodoState],
) error {
	return m.repo.Execute(ctx, aggID, todoaggregate.AggregateType, decide)
}

func requireCommandType[T any](cmd command.Command, expected string) (T, error) {
	var zero T
	if cmd == nil {
		return zero, event.Newf(event.Infrastructure, "todo.commands.mixin.1", "%v: expected %s, got nil", ErrInvalidCommandType, expected)
	}

	typed, ok := cmd.(T)
	if !ok {
		return zero, event.Newf(event.Infrastructure, "todo.commands.mixin.2", "%v: expected %s, got %T", ErrInvalidCommandType, expected, cmd)
	}

	return typed, nil
}
