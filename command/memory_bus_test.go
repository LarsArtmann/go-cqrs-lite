package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestMemoryBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := command.NewMemoryBus()
	ctx := context.Background()

	var received []string

	err := bus.Subscribe("user.create", func(_ context.Context, cmd command.Command) error {
		received = append(received, "typed:"+string(cmd.Type()))

		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	err = bus.SubscribeAll(func(_ context.Context, cmd command.Command) error {
		received = append(received, "all:"+string(cmd.Type()))

		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	aggID := id.NewAggregateID()
	cmd, err := command.New("user.create", aggID)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = bus.Publish(ctx, cmd)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("received %d, want 2", len(received))
	}

	if received[0] != "typed:user.create" || received[1] != "all:user.create" {
		t.Fatalf("order: got %v, want [typed:user.create, all:user.create]", received)
	}
}

func TestMemoryBus_HandlerError(t *testing.T) {
	t.Parallel()

	bus := command.NewMemoryBus()
	ctx := context.Background()

	busError := errors.New("handler failed")
	_ = bus.Subscribe("test.cmd", func(_ context.Context, _ command.Command) error {
		return busError
	})

	var allCalled bool

	_ = bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		allCalled = true

		return nil
	})

	aggID := id.NewAggregateID()
	cmd, _ := command.New("test.cmd", aggID)

	err := bus.Publish(ctx, cmd)
	if !errors.Is(err, busError) {
		t.Fatalf("Publish error: got %v, want %v", err, busError)
	}

	if allCalled {
		t.Fatal("catch-all should not be called after typed handler error")
	}
}

func TestMemoryBus_Middleware(t *testing.T) {
	t.Parallel()

	bus := command.NewMemoryBus()
	ctx := context.Background()

	var order []string

	_ = bus.Use(func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			order = append(order, "mw-before")
			err := next(ctx, cmd)
			order = append(order, "mw-after")

			return err
		}
	})

	_ = bus.Subscribe("test.cmd", func(_ context.Context, _ command.Command) error {
		order = append(order, "handler")

		return nil
	})

	aggID := id.NewAggregateID()
	cmd, _ := command.New("test.cmd", aggID)
	_ = bus.Publish(ctx, cmd)

	want := []string{"mw-before", "handler", "mw-after"}
	if len(order) != len(want) {
		t.Fatalf("order: got %v, want %v", order, want)
	}
}

func TestMemoryBus_NilHandler(t *testing.T) {
	t.Parallel()

	bus := command.NewMemoryBus()

	err := bus.Subscribe("test", nil)
	if err == nil {
		t.Fatal("Subscribe nil: expected error")
	}

	err = bus.SubscribeAll(nil)
	if err == nil {
		t.Fatal("SubscribeAll nil: expected error")
	}
}
