package command_test

import (
	"testing"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func TestNewCommandBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()

	var received command.Command

	bus.Subscribe(ro.OnNext(func(c command.Command) {
		received = c
	}))

	aggregateID := id.NewAggregateID()
	cmd, err := command.New("create.user", aggregateID)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	bus.Next(cmd)
	bus.Complete()

	if received == nil {
		t.Fatal("expected to receive a command")
	}

	if received.Type() != command.Type("create.user") {
		t.Errorf("expected create.user, got %s", received.Type())
	}
}

func TestNewCommandBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()

	var count int

	bus.Subscribe(ro.OnNext(func(c command.Command) {
		_ = c
	}))

	bus.Subscribe(ro.OnNext(func(c command.Command) {
		_ = c
	}))

	aggregateID := id.NewAggregateID()
	cmd, err := command.New("create.user", aggregateID)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	bus.Next(cmd)

	count = bus.CountObservers()

	if count != 2 {
		t.Errorf("expected 2 observers, got %d", count)
	}

	bus.Complete()
}

func TestFilterCommandType(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()

	filtered := ro.Pipe1(bus, command.FilterCommandType("create.user"))

	var received []command.Command

	filtered.Subscribe(ro.OnNext(func(c command.Command) {
		received = append(received, c)
	}))

	aggregateID := id.NewAggregateID()

	cmd1, _ := command.New("create.user", aggregateID)
	cmd2, _ := command.New("delete.user", aggregateID)
	cmd3, _ := command.New("create.user", aggregateID)

	bus.Next(cmd1)
	bus.Next(cmd2)
	bus.Next(cmd3)
	bus.Complete()

	if len(received) != 2 {
		t.Fatalf("expected 2 filtered commands, got %d", len(received))
	}

	for _, c := range received {
		if c.Type() != command.Type("create.user") {
			t.Errorf("expected create.user, got %s", c.Type())
		}
	}
}
