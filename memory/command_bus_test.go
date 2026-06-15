package memory_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func newTestCommand(t *testing.T, cmdType command.Type) command.Command {
	t.Helper()

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	cmd, err := command.New(cmdType, aggID)
	if err != nil {
		t.Fatalf("create test command: %v", err)
	}

	return cmd
}

func TestMemoryCommandBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var received atomic.Int32
	err := bus.Subscribe("CreateUser", func(_ context.Context, cmd command.Command) error {
		if cmd.Type() != "CreateUser" {
			t.Errorf("Type = %q, want CreateUser", cmd.Type())
		}

		received.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	err = bus.Publish(ctx, newTestCommand(t, "CreateUser"))
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := received.Load(); got != 1 {
		t.Errorf("handler invocations = %d, want 1", got)
	}
}

func TestMemoryCommandBus_Publish_NoSubscriber(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()

	// Publish with no subscribers must be a no-op (not an error).
	err := bus.Publish(context.Background(), newTestCommand(t, "CreateUser"))
	if err != nil {
		t.Fatalf("Publish with no subscribers should not error, got: %v", err)
	}
}

func TestMemoryCommandBus_Subscribe_MultipleHandlersPerType(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var counter atomic.Int32

	for range 3 {
		err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
			counter.Add(1)

			return nil
		})
		if err != nil {
			t.Fatalf("Subscribe: %v", err)
		}
	}

	if err := bus.Publish(ctx, newTestCommand(t, "CreateUser")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := counter.Load(); got != 3 {
		t.Errorf("handler invocations = %d, want 3", got)
	}
}

func TestMemoryCommandBus_Subscribe_TypeIsolation(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var createCount, updateCount atomic.Int32

	if err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		createCount.Add(1)

		return nil
	}); err != nil {
		t.Fatalf("Subscribe Create: %v", err)
	}

	if err := bus.Subscribe("UpdateUser", func(_ context.Context, _ command.Command) error {
		updateCount.Add(1)

		return nil
	}); err != nil {
		t.Fatalf("Subscribe Update: %v", err)
	}

	if err := bus.Publish(ctx, newTestCommand(t, "CreateUser")); err != nil {
		t.Fatalf("Publish Create: %v", err)
	}

	if err := bus.Publish(ctx, newTestCommand(t, "UpdateUser")); err != nil {
		t.Fatalf("Publish Update: %v", err)
	}

	if got := createCount.Load(); got != 1 {
		t.Errorf("Create handler invocations = %d, want 1", got)
	}

	if got := updateCount.Load(); got != 1 {
		t.Errorf("Update handler invocations = %d, want 1", got)
	}
}

func TestMemoryCommandBus_SubscribeAll(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var mutex sync.Mutex
	received := make(map[command.Type]int)

	err := bus.SubscribeAll(func(_ context.Context, cmd command.Command) error {
		mutex.Lock()
		defer mutex.Unlock()
		received[cmd.Type()]++

		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	if err := bus.Publish(
		ctx,
		newTestCommand(t, "CreateUser"),
		newTestCommand(t, "UpdateUser"),
		newTestCommand(t, "DeleteUser"),
	); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	if len(received) != 3 {
		t.Fatalf("expected 3 distinct types received, got %d: %v", len(received), received)
	}

	for _, want := range []command.Type{"CreateUser", "UpdateUser", "DeleteUser"} {
		if got := received[want]; got != 1 {
			t.Errorf("SubscribeAll received %q %d times, want 1", want, got)
		}
	}
}

func TestMemoryCommandBus_SubscribeAll_FiresWithTypeHandlers(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var typeCount, allCount atomic.Int32

	if err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		typeCount.Add(1)

		return nil
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if err := bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		allCount.Add(1)

		return nil
	}); err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	if err := bus.Publish(ctx, newTestCommand(t, "CreateUser")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Both the typed handler and the catch-all handler must fire.
	if got := typeCount.Load(); got != 1 {
		t.Errorf("type handler invocations = %d, want 1", got)
	}

	if got := allCount.Load(); got != 1 {
		t.Errorf("all handler invocations = %d, want 1", got)
	}
}
