package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
)

func TestNewCommand(t *testing.T) {
	t.Parallel()

	cmd := command.New("CreateUser", "user-123")

	if cmd.Type() != "CreateUser" {
		t.Errorf("expected type CreateUser, got %s", cmd.Type())
	}

	if cmd.AggregateID() != "user-123" {
		t.Errorf("expected aggregate ID user-123, got %s", cmd.AggregateID())
	}
}

func TestBaseCommand_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ command.Command = command.New("TestCommand", "test-id")
}

func TestDispatcher_Register(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()

	handler := func(_ context.Context, _ command.Command) error {
		return nil
	}

	err := dispatcher.Register("CreateUser", handler)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	ctx := context.Background()

	executed := false
	handler := func(_ context.Context, _ command.Command) error {
		executed = true
		return nil
	}

	_ = dispatcher.Register("CreateUser", handler)

	cmd := command.New("CreateUser", "user-123")
	err := dispatcher.Dispatch(ctx, cmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !executed {
		t.Error("expected handler to be executed")
	}
}

func TestDispatcher_Dispatch_HandlerNotFound(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	ctx := context.Background()

	cmd := command.New("UnknownCommand", "user-123")
	err := dispatcher.Dispatch(ctx, cmd)
	if err == nil {
		t.Error("expected handler not found error")
	}
}

// testMiddleware creates a middleware that records its name in callOrder.
func testMiddleware(callOrder *[]string, name string) func(h command.Handler) command.Handler {
	return func(h command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			*callOrder = append(*callOrder, name)
			return h(ctx, cmd)
		}
	}
}

func TestDispatcher_Middleware(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	ctx := context.Background()

	var callOrder []string

	dispatcher.Use(
		testMiddleware(&callOrder, "middleware1"),
		testMiddleware(&callOrder, "middleware2"),
	)

	_ = dispatcher.Register("TestCommand", func(_ context.Context, _ command.Command) error {
		callOrder = append(callOrder, "handler")
		return nil
	})

	cmd := command.New("TestCommand", "test-123")
	_ = dispatcher.Dispatch(ctx, cmd)

	expected := []string{"middleware1", "middleware2", "handler"}
	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)
			break
		}
	}
}

func TestDispatcher_Closed(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	_ = dispatcher.Close()

	handler := func(_ context.Context, _ command.Command) error { return nil }

	err := dispatcher.Register("TestCommand", handler)
	if err == nil {
		t.Error("expected dispatcher closed error on Register")
	}

	cmd := command.New("TestCommand", "test-123")
	err = dispatcher.Dispatch(context.Background(), cmd)
	if err == nil {
		t.Error("expected dispatcher closed error on Dispatch")
	}
}
