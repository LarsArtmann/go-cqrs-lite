package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestNew_EmptyType(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	_, err := command.New("", aggID)
	if err == nil {
		t.Error("expected error for empty command type")
	}

	if !errors.Is(err, command.ErrEmptyCommandType) {
		t.Errorf("errors.Is(err, ErrEmptyCommandType) = false, got: %v", err)
	}
}

func TestNew_ZeroAggregateID(t *testing.T) {
	t.Parallel()

	_, err := command.New("CreateUser", id.AggregateID{})
	if err == nil {
		t.Error("expected error for zero aggregate ID")
	}

	if !errors.Is(err, command.ErrNilAggregateID) {
		t.Errorf("errors.Is(err, ErrNilAggregateID) = false, got: %v", err)
	}
}

func TestMustNew_PanicsOnEmptyType(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty command type")
		}
	}()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	_ = command.MustNew("", aggID)
}

func TestMustNew_PanicsOnZeroAggregateID(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for zero aggregate ID")
		}
	}()

	_ = command.MustNew("CreateUser", id.AggregateID{})
}

func TestDispatcher_Dispatch_HandlerError(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	ctx := context.Background()

	handlerErr := errors.New("handler failed")
	_ = d.Register("FailCommand", func(_ context.Context, _ command.Command) error {
		return handlerErr
	})

	cmd := command.MustNew("FailCommand", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	err := d.Dispatch(ctx, cmd)
	if err == nil {
		t.Fatal("expected error from handler")
	}

	if !errors.Is(err, handlerErr) {
		t.Errorf("error should wrap handler error, got: %v", err)
	}
}

func TestDispatcher_Dispatch_HandlerNotFound_ErrorChain(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	ctx := context.Background()

	cmd := command.MustNew("UnknownCmd", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	err := d.Dispatch(ctx, cmd)
	if err == nil {
		t.Fatal("expected error for unregistered command")
	}

	if !errors.Is(err, command.ErrHandlerNotFound) {
		t.Errorf("error should be ErrHandlerNotFound, got: %v", err)
	}
}

func TestDispatcher_Register_Duplicate(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()

	err := d.Register("CreateUser", func(_ context.Context, _ command.Command) error { return nil })
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err = d.Register("CreateUser", func(_ context.Context, _ command.Command) error { return nil })
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestDispatcher_Closed_DispatchErrorChain(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	_ = d.Close()

	cmd := command.MustNew("TestCmd", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	err := d.Dispatch(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error on closed dispatcher")
	}

	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("error should wrap ErrDispatcherClosed, got: %v", err)
	}
}

func TestDispatcher_Closed_RegisterErrorChain(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	_ = d.Close()

	err := d.Register("TestCmd", func(_ context.Context, _ command.Command) error { return nil })
	if err == nil {
		t.Fatal("expected error on closed dispatcher")
	}

	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("error should wrap ErrDispatcherClosed, got: %v", err)
	}
}

func TestDispatcher_Use(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	called := false

	d.Use(func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			called = true

			return next(ctx, cmd)
		}
	})

	_ = d.Register("TestCmd", func(_ context.Context, _ command.Command) error { return nil })

	cmd := command.MustNew("TestCmd", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	err := d.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected middleware to be called")
	}
}

func TestDispatcher_Register_ClosedDispatcher(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	_ = d.Close()

	err := d.Register("Cmd", func(_ context.Context, _ command.Command) error { return nil })
	if err == nil {
		t.Fatal("expected error registering on closed dispatcher")
	}
}
