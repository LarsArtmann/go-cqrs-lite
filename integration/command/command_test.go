package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
	catdispatcher "github.com/larsartmann/go-cqrs-lite/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestNewCommand(t *testing.T) {
	t.Parallel()

	cmd := command.MustNew("CreateUser", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

	if cmd.Type() != "CreateUser" {
		t.Errorf("expected type CreateUser, got %s", cmd.Type())
	}

	if cmd.AggregateID() != id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95") {
		t.Errorf("expected aggregate ID user-123, got %s", cmd.AggregateID())
	}
}

func TestBaseCommand_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ command.Command = command.MustNew("TestCommand", id.MustParseAggregateID("01HK1549P84T9XF8R94E960633"))
}

func TestDispatcher_Register(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()

	err := dispatcher.Register("CreateUser", testhelpers.NoopCommandHandler())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	ctx := context.Background()

	executed := false
	handler := testhelpers.CallbackCommandHandler(&executed)

	_ = dispatcher.Register("CreateUser", handler)

	cmd := command.MustNew("CreateUser", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

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

	cmd := command.MustNew("UnknownCommand", id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95"))

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
		testhelpers.CommandMiddleware(&callOrder, "middleware1"),
		testhelpers.CommandMiddleware(&callOrder, "middleware2"),
	)

	dispatcher.Register("TestCommand", testhelpers.AppendCommandHandler(&callOrder))

	cmd := command.MustNew("TestCommand", id.MustParseAggregateID("01HK154ANGZHV2ZW0X3SKSNEN2"))
	_ = dispatcher.Dispatch(ctx, cmd)

	testhelpers.AssertCallOrder(t, callOrder, []string{"middleware1", "middleware2", "handler"})
}

func TestDispatcher_Closed(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	_ = dispatcher.Close()

	err := dispatcher.Register("TestCommand", testhelpers.NoopCommandHandler())
	if err == nil {
		t.Error("expected dispatcher closed error on Register")
	}

	cmd := command.MustNew("TestCommand", id.MustParseAggregateID("01HK154ANGZHV2ZW0X3SKSNEN2"))

	err = dispatcher.Dispatch(context.Background(), cmd)
	if err == nil {
		t.Error("expected dispatcher closed error on Dispatch")
	}
}

func TestDispatcher_RegisterHandlerMeta(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	meta := catdispatcher.HandlerMeta{
		Name:    "CreateUser",
		Version: "1.0.0",
		Summary: "Creates a new user",
	}

	dispatcher.RegisterHandlerMeta("user.create", meta)

	entries := dispatcher.CatalogEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got, ok := entries["user.create"]
	if !ok {
		t.Fatal("missing user.create entry")
	}

	if got.Name != "CreateUser" {
		t.Errorf("name = %q, want CreateUser", got.Name)
	}
}

func TestDispatcher_CatalogEntries_ReturnsCopy(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	dispatcher.RegisterHandlerMeta("cmd.a", catdispatcher.HandlerMeta{Name: "A"})

	entries := dispatcher.CatalogEntries()
	entries["cmd.b"] = catdispatcher.HandlerMeta{Name: "B"}

	if len(dispatcher.CatalogEntries()) != 1 {
		t.Error("CatalogEntries should return a copy, not a reference")
	}
}
