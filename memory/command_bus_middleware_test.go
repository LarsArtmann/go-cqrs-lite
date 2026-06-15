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

	err := bus.Use(trackingMW("outer"), trackingMW("inner"))
	if err != nil {
		t.Fatalf("Use: %v", err)
	}

	var handlerCalled atomic.Bool
	err = bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		handlerCalled.Store(true)

		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	err = bus.Publish(ctx, newTestCommand(t, "CreateUser"))
	if err != nil {
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
	err := bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Adding middleware AFTER subscription must rebuild the chain to include
	// the new middleware for existing handlers.
	err = bus.Use(func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			mwSeen.Store(true)

			return next(ctx, cmd)
		}
	})
	if err != nil {
		t.Fatalf("Use: %v", err)
	}

	err = bus.Publish(ctx, newTestCommand(t, "CreateUser"))
	if err != nil {
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
	err := bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		count.Add(1)

		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeAll: %v", err)
	}

	err = bus.Publish(
		ctx,
		newTestCommand(t, "CreateUser"),
		newTestCommand(t, "UpdateUser"),
	)
	if err != nil {
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

	err := bus.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	err = bus.Publish(context.Background(), newTestCommand(t, "CreateUser"))
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

	err := bus.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	err = bus.Subscribe("CreateUser", func(_ context.Context, _ command.Command) error {
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

	err := bus.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	err = bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
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

	err := bus.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	err = bus.Use(func(next command.Handler) command.Handler { return next })
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

	err := bus.SubscribeAll(func(_ context.Context, _ command.Command) error {
		count.Add(1)

		return nil
	})
	if err != nil {
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
