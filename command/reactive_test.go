package command_test

import (
	"context"
	"errors"
	"testing"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestNewCommandBus_BroadcastsToSubscribers(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()
	cmd := makeTestCommand("user.create")

	var got []command.Type
	bus.Subscribe(ro.OnNext(func(c command.Command) {
		got = append(got, c.Type())
	}))

	bus.Next(cmd)
	bus.Complete()

	if len(got) != 1 || got[0] != cmd.Type() {
		t.Fatalf("expected [%s], got %v", cmd.Type(), got)
	}
}

func TestFilterCommandType_DropsOtherTypes(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()
	filtered := ro.Pipe1(bus, command.FilterCommandType("user.create"))

	var got []command.Type
	filtered.Subscribe(ro.OnNext(func(c command.Command) {
		got = append(got, c.Type())
	}))

	bus.Next(makeTestCommand("user.create"))
	bus.Next(makeTestCommand("user.update"))
	bus.Next(makeTestCommand("user.create"))
	bus.Complete()

	want := []command.Type{"user.create", "user.create"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestFilterCommandTypes_AllowsMatchingTypes(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()
	filtered := ro.Pipe1(bus, command.FilterCommandTypes("user.create", "user.delete"))

	var got []command.Type
	filtered.Subscribe(ro.OnNext(func(c command.Command) {
		got = append(got, c.Type())
	}))

	bus.Next(makeTestCommand("user.create"))
	bus.Next(makeTestCommand("user.update"))
	bus.Next(makeTestCommand("user.delete"))
	bus.Complete()

	want := []command.Type{"user.create", "user.delete"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFilterCommandTypes_EmptyAllowsAll(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()
	filtered := ro.Pipe1(bus, command.FilterCommandTypes())

	var got int
	filtered.Subscribe(ro.OnNext(func(_ command.Command) {
		got++
	}))

	bus.Next(makeTestCommand("user.create"))
	bus.Next(makeTestCommand("user.update"))
	bus.Complete()

	if got != 2 {
		t.Fatalf("expected 2 commands, got %d", got)
	}
}

func TestHandlerToObserver_RoutesCommandsAndForwardsErrors(t *testing.T) {
	t.Parallel()

	bus := command.NewCommandBus()
	wantErr := errors.New("boom")

	handler := func(_ context.Context, c command.Command) error {
		if c.Type() == "user.fail" {
			return wantErr
		}

		return nil
	}

	obs := command.HandlerToObserver(handler)
	bus.Subscribe(obs)

	bus.Next(makeTestCommand("user.ok"))
	bus.Next(makeTestCommand("user.fail"))
	bus.Complete()
}

func TestNewReplayCommandBus_ReplaysToLateSubscribers(t *testing.T) {
	t.Parallel()

	bus := command.NewReplayCommandBus(2)
	cmd1 := makeTestCommand("user.first")
	cmd2 := makeTestCommand("user.second")
	cmd3 := makeTestCommand("user.third")

	bus.Next(cmd1)
	bus.Next(cmd2)
	bus.Next(cmd3)

	var got []command.Type
	bus.Subscribe(ro.OnNext(func(c command.Command) {
		got = append(got, c.Type())
	}))

	if len(got) != 2 {
		t.Fatalf("expected 2 replayed commands, got %d", len(got))
	}
	if got[0] != cmd2.Type() || got[1] != cmd3.Type() {
		t.Fatalf("expected [%s %s], got %v", cmd2.Type(), cmd3.Type(), got)
	}
}

func TestNewBehaviorCommandBus_ReplaysLatest(t *testing.T) {
	t.Parallel()

	initial := makeTestCommand("user.initial")
	bus := command.NewBehaviorCommandBus(initial)
	latest := makeTestCommand("user.latest")

	bus.Next(latest)

	var got []command.Type
	bus.Subscribe(ro.OnNext(func(c command.Command) {
		got = append(got, c.Type())
	}))

	if len(got) != 1 || got[0] != latest.Type() {
		t.Fatalf("expected [%s], got %v", latest.Type(), got)
	}
}

func makeTestCommand(cmdType command.Type) command.Command {
	cmd, err := command.New(cmdType, id.NewAggregateID())
	if err != nil {
		panic(err)
	}

	return cmd
}
