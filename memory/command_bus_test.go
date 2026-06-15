package memory_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
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

func TestMemoryCommandBus_Middleware(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var calls []string

	trackingMW := func(name string) command.Middleware {
		return func(next command.Handler) command.Handler {
			return func(ctx context.Context, cmd command.Command) error {
				calls = append(calls, name+":before")
				err := next(ctx, cmd)
				calls = append(calls, name+":after")

				return err
			}
		}
	}

	if err := bus.Use(trackingMW("outer"), trackingMW("inner")); err != nil {
		t.Fatalf("Use: %v", err)
	}

	var handlerCalled atomic.Bool
	if err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		handlerCalled.Store(true)

		return nil
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if err := bus.Publish(ctx, newTestCommand(t, "CreateUser")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if !handlerCalled.Load() {
		t.Error("handler was not called")
	}

	// Outer wraps inner; inner wraps the handler. Execution order:
	// outer:before → inner:before → handler → inner:after → outer:after
	want := []string{"outer:before", "inner:before", "inner:after", "outer:after"}
	if len(calls) != len(want) {
		t.Fatalf("middleware call sequence = %v, want %v", calls, want)
	}

	for i, w := range want {
		if calls[i] != w {
			t.Errorf("calls[%d] = %q, want %q (full: %v)", i, calls[i], w, calls)
		}
	}
}

func TestMemoryCommandBus_Middleware_AfterSubscribe(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var mwSeen atomic.Bool
	if err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		return nil
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Adding middleware AFTER subscription must rebuild the chain to include
	// the new middleware for existing handlers.
	if err := bus.Use(func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			mwSeen.Store(true)

			return next(ctx, cmd)
		}
	}); err != nil {
		t.Fatalf("Use: %v", err)
	}

	if err := bus.Publish(ctx, newTestCommand(t, "CreateUser")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if !mwSeen.Load() {
		t.Error("middleware added after Subscribe was not invoked")
	}
}

func TestMemoryCommandBus_Publish_Variadic_MultipleCommands(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var count atomic.Int32
	if err := bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		count.Add(1)

		return nil
	}); err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	if err := bus.Publish(
		ctx,
		newTestCommand(t, "CreateUser"),
		newTestCommand(t, "UpdateUser"),
	); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if got := count.Load(); got != 2 {
		t.Errorf("handler invocations = %d, want 2", got)
	}
}

func TestMemoryCommandBus_Publish_HandlerError_StopsChain(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	sentinel := errors.New("handler failure")

	err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		return sentinel
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	err = bus.Publish(ctx, newTestCommand(t, "CreateUser"))
	if err == nil {
		t.Fatal("expected error from failing handler, got nil")
	}

	if !errors.Is(err, sentinel) {
		t.Errorf("expected error chain to wrap sentinel, got: %v", err)
	}
}

func TestMemoryCommandBus_Publish_Closed(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()

	if err := bus.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := bus.Publish(context.Background(), newTestCommand(t, "CreateUser"))
	if err == nil {
		t.Fatal("expected error on Publish after Close")
	}

	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("expected ErrDispatcherClosed, got: %v", err)
	}
}

func TestMemoryCommandBus_Subscribe_Closed(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()

	if err := bus.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error on Subscribe after Close")
	}

	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("expected ErrDispatcherClosed, got: %v", err)
	}
}

func TestMemoryCommandBus_SubscribeAll_Closed(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()

	if err := bus.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error on SubscribeAll after Close")
	}

	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("expected ErrDispatcherClosed, got: %v", err)
	}
}

func TestMemoryCommandBus_Use_Closed(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()

	if err := bus.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := bus.Use(func(next command.Handler) command.Handler { return next })
	if err == nil {
		t.Fatal("expected error on Use after Close")
	}

	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("expected ErrDispatcherClosed, got: %v", err)
	}
}

func TestMemoryCommandBus_Concurrent(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryCommandBus()
	ctx := context.Background()

	var count atomic.Int32

	if err := bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		count.Add(1)

		return nil
	}); err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	const goroutines = 20
	const perGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			aggID := id.NewAggregateID()
			for range perGoroutine {
				cmd, err := command.New("CreateUser", aggID)
				if err != nil {
					t.Errorf("New: %v", err)

					return
				}

				if err := bus.Publish(ctx, cmd); err != nil {
					t.Errorf("Publish: %v", err)

					return
				}
			}
		}()
	}

	wg.Wait()

	want := int32(goroutines * perGoroutine)
	if got := count.Load(); got != want {
		t.Errorf("total handler invocations = %d, want %d", got, want)
	}
}
