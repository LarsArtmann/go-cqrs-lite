package commands_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrsMemory "github.com/larsartmann/go-cqrs-lite/storage/memory/v2"
)

type commandHandlers struct {
	create *commands.CreateTodoHandler
	update *commands.UpdateTodoHandler
	delete *commands.DeleteTodoHandler
	status *commands.ChangeStatusHandler
}

func setupCommandHandlers(t *testing.T) commandHandlers {
	t.Helper()
	store := cqrsMemory.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	create, err := commands.NewCreateTodoHandler(store, bus)
	if err != nil {
		t.Fatalf("NewCreateTodoHandler: %v", err)
	}

	update, err := commands.NewUpdateTodoHandler(store, bus)
	if err != nil {
		t.Fatalf("NewUpdateTodoHandler: %v", err)
	}

	delete, err := commands.NewDeleteTodoHandler(store, bus)
	if err != nil {
		t.Fatalf("NewDeleteTodoHandler: %v", err)
	}

	status, err := commands.NewChangeStatusHandler(store, bus)
	if err != nil {
		t.Fatalf("NewChangeStatusHandler: %v", err)
	}

	return commandHandlers{create: create, update: update, delete: delete, status: status}
}

func TestCreateTodoHandler_Handle(t *testing.T) {
	t.Parallel()

	h := setupCommandHandlers(t)

	cmd, err := commands.NewCreateTodoCommand(
		id.NewAggregateID(),
		"Test Todo",
		"desc",
		1,
		[]string{"a"},
	)
	if err != nil {
		t.Fatalf("NewCreateTodoCommand() error = %v", err)
	}

	if err := h.create.Handle(context.Background(), cmd); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestCreateTodoHandler_InvalidCommandType(t *testing.T) {
	t.Parallel()

	h := setupCommandHandlers(t)

	cmd, _ := commands.NewUpdateTodoCommand(id.NewAggregateID(), "t", "d")

	err := h.create.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Handle() should error on wrong command type")
	}
}

func TestUpdateTodoHandler_Handle(t *testing.T) {
	t.Parallel()

	h := setupCommandHandlers(t)
	aggID := id.NewAggregateID()

	createCmd, _ := commands.NewCreateTodoCommand(aggID, "Original", "old", 1, nil)
	_ = h.create.Handle(context.Background(), createCmd)

	updateCmd, _ := commands.NewUpdateTodoCommand(aggID, "Updated", "new")
	if err := h.update.Handle(context.Background(), updateCmd); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestDeleteTodoHandler_Handle(t *testing.T) {
	t.Parallel()

	h := setupCommandHandlers(t)
	aggID := id.NewAggregateID()

	createCmd, _ := commands.NewCreateTodoCommand(aggID, "To Delete", "", 1, nil)
	_ = h.create.Handle(context.Background(), createCmd)

	deleteCmd, _ := commands.NewDeleteTodoCommand(aggID)
	if err := h.delete.Handle(context.Background(), deleteCmd); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestChangeStatusHandler_Handle(t *testing.T) {
	t.Parallel()

	h := setupCommandHandlers(t)
	aggID := id.NewAggregateID()

	createCmd, _ := commands.NewCreateTodoCommand(aggID, "Status Test", "", 1, nil)
	_ = h.create.Handle(context.Background(), createCmd)

	statusCmd, _ := commands.NewChangeStatusCommand(aggID, domain.StatusCompleted)
	if err := h.status.Handle(context.Background(), statusCmd); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}

func TestFullTodoLifecycle(t *testing.T) {
	h := setupCommandHandlers(t)
	aggID := id.NewAggregateID()

	createCmd, err := commands.NewCreateTodoCommand(aggID, "Lifecycle", "test", 2, []string{"tag"})
	if err != nil {
		t.Fatalf("create command: %v", err)
	}
	if err := h.create.Handle(context.Background(), createCmd); err != nil {
		t.Fatalf("create: %v", err)
	}

	updateCmd, err := commands.NewUpdateTodoCommand(aggID, "Updated", "updated desc")
	if err != nil {
		t.Fatalf("update command: %v", err)
	}
	if err := h.update.Handle(context.Background(), updateCmd); err != nil {
		t.Fatalf("update: %v", err)
	}

	statusCmd, err := commands.NewChangeStatusCommand(aggID, domain.StatusInProgress)
	if err != nil {
		t.Fatalf("status command: %v", err)
	}
	if err := h.status.Handle(context.Background(), statusCmd); err != nil {
		t.Fatalf("change status: %v", err)
	}

	statusCmd2, err := commands.NewChangeStatusCommand(aggID, domain.StatusCompleted)
	if err != nil {
		t.Fatalf("status command 2: %v", err)
	}
	if err := h.status.Handle(context.Background(), statusCmd2); err != nil {
		t.Fatalf("complete: %v", err)
	}

	deleteCmd, err := commands.NewDeleteTodoCommand(aggID)
	if err != nil {
		t.Fatalf("delete command: %v", err)
	}
	if err := h.delete.Handle(context.Background(), deleteCmd); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestCreateTodoCommand_Constructor(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		cmd, err := commands.NewCreateTodoCommand(id.NewAggregateID(), "Title", "desc", 1, nil)
		if err != nil {
			t.Fatalf("NewCreateTodoCommand() error = %v", err)
		}
		if cmd.Title != "Title" {
			t.Errorf("Title = %q, want %q", cmd.Title, "Title")
		}
	})

	t.Run("empty title creates but aggregate rejects", func(t *testing.T) {
		t.Parallel()
		h := setupCommandHandlers(t)
		cmd, _ := commands.NewCreateTodoCommand(id.NewAggregateID(), "", "", 1, nil)
		err := h.create.Handle(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error for empty title")
		}
	})
}
