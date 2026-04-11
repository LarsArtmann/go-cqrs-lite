package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/internal/testhelpers"
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

	testhelpers.AssertCallOrder(t, callOrder, []string{"middleware1", "middleware2", "handler"})
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

func TestDispatcher_RegisterCatalogEntry(t *testing.T) {
	t.Parallel()

	dispatcher := command.NewDispatcher()
	meta := command.CatalogMeta{
		Name:    "CreateUser",
		Version: "1.0.0",
		Summary: "Creates a new user",
	}

	dispatcher.RegisterCatalogEntry("user.create", meta)

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
	dispatcher.RegisterCatalogEntry("cmd.a", command.CatalogMeta{Name: "A"})

	entries := dispatcher.CatalogEntries()
	entries["cmd.b"] = command.CatalogMeta{Name: "B"}

	if len(dispatcher.CatalogEntries()) != 1 {
		t.Error("CatalogEntries should return a copy, not a reference")
	}
}
