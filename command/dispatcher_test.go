package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

func TestNew_EmptyType(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")

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

func TestNew_EmptyType_Rejected(t *testing.T) {
	t.Parallel()

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	_, err := command.New("", aggID)
	if err == nil {
		t.Fatal("expected error for empty command type")
	}
}

func TestNew_ZeroAggregateID_Rejected(t *testing.T) {
	t.Parallel()

	_, err := command.New("CreateUser", id.AggregateID{})
	if err == nil {
		t.Fatal("expected error for zero aggregate ID")
	}
}

func TestDispatcher_Dispatch_HandlerError(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	ctx := context.Background()

	handlerErr := errors.New("handler failed")
	_ = d.Register("FailCommand", func(_ context.Context, _ command.Command) error {
		return handlerErr
	})

	cmd := newCmd(t, "FailCommand", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"))

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

	cmd := newCmd(t, "UnknownCmd", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"))

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

	err := d.Register("CreateUser", noopCommandHandler())
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err = d.Register("CreateUser", noopCommandHandler())
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestDispatcher_Closed_DispatchErrorChain(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	_ = d.Close()

	cmd := newCmd(t, "TestCmd", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"))

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

	err := d.Register("TestCmd", noopCommandHandler())
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

	_ = d.Register("TestCmd", noopCommandHandler())

	cmd := newCmd(t, "TestCmd", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"))

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

	err := d.Register("Cmd", noopCommandHandler())
	if err == nil {
		t.Fatal("expected error registering on closed dispatcher")
	}
}

func TestDispatcher_Dispatch_Success(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	called := false

	_ = d.Register("TestCmd", callbackCommandHandler(&called))

	cmd := newCmd(t, "TestCmd", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"))

	err := d.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected handler to be called")
	}
}

func TestDispatcher_Dispatch_WrappedClosedError(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	_ = d.Register("TestCmd", noopCommandHandler())
	_ = d.Close()

	cmd := newCmd(t, "TestCmd", idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"))

	err := d.Dispatch(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error on closed dispatcher")
	}

	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("error should wrap ErrDispatcherClosed, got: %v", err)
	}
}
