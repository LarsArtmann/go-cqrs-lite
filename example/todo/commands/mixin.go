package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	todoaggregate "github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
)

var ErrInvalidCommandType = errors.New("invalid command type")

type CommandHandler struct {
	repo *decider.Repository[todoaggregate.TodoState]
}

func NewHandler(events event.Store, eventBus event.Publisher) CommandHandler {
	repo, err := decider.NewRepository(
		events, eventBus, todoaggregate.TodoDecider,
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

type CommandTypeError struct{ Expected string }

func (e *CommandTypeError) Error() string { return "invalid command type: expected " + e.Expected }

func requireCommandType[T any](cmd command.Command, expected string) (T, error) {
	var zero T
	if cmd == nil {
		return zero, fmt.Errorf("%w: expected %s, got nil", ErrInvalidCommandType, expected)
	}
	typed, ok := cmd.(T)
	if !ok {
		return zero, fmt.Errorf("%w: expected %s, got %T", ErrInvalidCommandType, expected, cmd)
	}
	return typed, nil
}
