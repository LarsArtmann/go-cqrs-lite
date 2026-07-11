package watermill_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func TestCommandBusPublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus()
	defer bus.Close()

	var received atomic.Int32

	err := bus.Subscribe("user.create", func(_ context.Context, _ command.Command) error {
		received.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	aggID := id.NewAggregateID()
	cmd, err := command.New("user.create", aggID)
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	err = bus.Publish(context.Background(), cmd)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	waitFor(t, func() bool { return received.Load() > 0 }, 2*time.Second)
	if received.Load() != 1 {
		t.Fatalf("expected 1 command received, got %d", received.Load())
	}
}

func TestCommandBusSubscribeAll(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus()
	defer bus.Close()

	var received atomic.Int32

	err := bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		received.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("subscribeAll: %v", err)
	}

	aggID := id.NewAggregateID()
	for _, ct := range []command.Type{"a.b", "c.d"} {
		cmd, _ := command.New(ct, aggID)
		_ = bus.Publish(context.Background(), cmd)
	}

	waitFor(t, func() bool { return received.Load() >= 2 }, 2*time.Second)
	if received.Load() != 2 {
		t.Fatalf("expected 2 commands, got %d", received.Load())
	}
}

func TestCommandBusPublishEmpty(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus()
	defer bus.Close()

	err := bus.Publish(context.Background())
	if err != nil {
		t.Fatalf("publish empty: %v", err)
	}
}

func TestCommandBusCloseIdempotent(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus()

	if err := bus.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestCommandBusPublishAfterClose(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus()
	_ = bus.Close()

	aggID := id.NewAggregateID()
	cmd, _ := command.New("user.create", aggID)
	err := bus.Publish(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error publishing after close")
	}
}

func TestCommandBusMiddleware(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus()
	defer bus.Close()

	var order []string
	var mu sync.Mutex

	appendOrder := func(s string) {
		mu.Lock()
		order = append(order, s)
		mu.Unlock()
	}

	_ = bus.Use(
		func(next command.Handler) command.Handler {
			return func(ctx context.Context, cmd command.Command) error {
				appendOrder("mw1-before")
				err := next(ctx, cmd)
				appendOrder("mw1-after")

				return err
			}
		},
		func(next command.Handler) command.Handler {
			return func(ctx context.Context, cmd command.Command) error {
				appendOrder("mw2-before")
				err := next(ctx, cmd)
				appendOrder("mw2-after")

				return err
			}
		},
	)

	var received atomic.Int32
	_ = bus.Subscribe("test.cmd", func(_ context.Context, _ command.Command) error {
		appendOrder("handler")
		received.Add(1)

		return nil
	})

	aggID := id.NewAggregateID()
	cmd, _ := command.New("test.cmd", aggID)
	_ = bus.Publish(context.Background(), cmd)

	waitFor(t, func() bool { return received.Load() > 0 }, 2*time.Second)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("order length: got %d, want %d (%v)", len(order), len(expected), order)
	}
	for i, s := range expected {
		if order[i] != s {
			t.Fatalf("order[%d]: got %q, want %q (full: %v)", i, order[i], s, order)
		}
	}
}

func TestCommandBusMetadataRoundTrip(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus()
	defer bus.Close()

	correlationID := id.NewCorrelationID()
	userID := id.NewUserID()

	var receivedCmd *command.BasicCommand
	var done atomic.Bool

	_ = bus.Subscribe("user.create", func(_ context.Context, cmd command.Command) error {
		if bc, ok := cmd.(*command.BasicCommand); ok {
			receivedCmd = bc
		}
		done.Store(true)

		return nil
	})

	aggID := id.NewAggregateID()
	original, _ := command.New(
		"user.create", aggID,
		command.WithCorrelationID(correlationID),
		command.WithUserID(userID),
		command.WithCustomMetadata("tenant", "acme"),
	)
	_ = bus.Publish(context.Background(), original)

	waitFor(t, done.Load, 2*time.Second)

	if receivedCmd == nil {
		t.Fatal("command not received")
	}
	md := receivedCmd.Metadata()
	if md.CorrelationID != correlationID {
		t.Fatalf("correlation_id mismatch: got %s, want %s",
			md.CorrelationID, correlationID)
	}
	if md.UserID != userID {
		t.Fatalf("user_id mismatch")
	}
	if md.Custom["tenant"] != "acme" {
		t.Fatalf("custom.tenant mismatch")
	}
}

func TestCommandBusCustomTopic(t *testing.T) {
	t.Parallel()

	bus := cqrswatermill.NewCommandBus(
		cqrswatermill.WithCommandBusTopic("my-commands"),
	)
	defer bus.Close()

	var received atomic.Int32

	_ = bus.Subscribe("test.cmd", func(_ context.Context, _ command.Command) error {
		received.Add(1)

		return nil
	})

	aggID := id.NewAggregateID()
	cmd, _ := command.New("test.cmd", aggID)
	_ = bus.Publish(context.Background(), cmd)

	waitFor(t, func() bool { return received.Load() > 0 }, 2*time.Second)
	if received.Load() != 1 {
		t.Fatalf("expected 1 command, got %d", received.Load())
	}
}
