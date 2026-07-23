package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

func TestNewCommand(t *testing.T) {
	t.Parallel()

	cmd := testutil.NewCmd(
		t,
		"CreateUser",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

	if cmd.Type() != "CreateUser" {
		t.Errorf("expected type CreateUser, got %s", cmd.Type())
	}

	if cmd.StreamID() != idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95") {
		t.Errorf("expected aggregate ID user-123, got %s", cmd.StreamID())
	}
}

func TestBaseCommand_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ command.Command = testutil.NewCmd(t, "TestCommand", idtest.ParseAggregateID(t, "01HK1549P84T9XF8R94E960633"))
}

func TestDispatcher_Register(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()

	err := dispatcher.Register("CreateUser", noopCommandHandler())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	ctx := context.Background()

	executed := false
	handler := callbackCommandHandler(&executed)

	_ = dispatcher.Register("CreateUser", handler)

	cmd := testutil.NewCmd(
		t,
		"CreateUser",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

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

	cmd := testutil.NewCmd(
		t,
		"UnknownCommand",
		idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95"),
	)

	err := dispatcher.Dispatch(ctx, cmd)
	if err == nil {
		t.Error("expected handler not found error")
	}
}

func TestDispatcher_Middleware(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	ctx := context.Background()

	var callOrder []string

	dispatcher.Use(
		commandMiddleware(&callOrder, "middleware1"),
		commandMiddleware(&callOrder, "middleware2"),
	)

	_ = dispatcher.Register("TestCommand", appendCommandHandler(&callOrder))

	cmd := testutil.NewCmd(
		t,
		"TestCommand",
		idtest.ParseAggregateID(t, "01HK154ANGZHV2ZW0X3SKSNEN2"),
	)
	_ = dispatcher.Dispatch(ctx, cmd)

	eventtest.AssertCallOrder(t, callOrder, []string{"middleware1", "middleware2", "handler"})
}

func TestDispatcher_Closed(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	_ = dispatcher.Close()

	err := dispatcher.Register("TestCommand", noopCommandHandler())
	if err == nil {
		t.Error("expected dispatcher closed error on Register")
	}

	cmd := testutil.NewCmd(
		t,
		"TestCommand",
		idtest.ParseAggregateID(t, "01HK154ANGZHV2ZW0X3SKSNEN2"),
	)

	err = dispatcher.Dispatch(context.Background(), cmd)
	if err == nil {
		t.Error("expected dispatcher closed error on Dispatch")
	}
}
