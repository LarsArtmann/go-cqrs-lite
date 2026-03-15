package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
)

func TestNewCommand(t *testing.T) {
	cmd := command.New("CreateUser", "user-123")
	if cmd.Type() != "CreateUser" {
		t.Errorf("expected CreateUser, got %s", cmd.Type())
	}
	if cmd.AggregateID() != "user-123" {
		t.Errorf("expected aggregateID user-123, got %s", cmd.AggregateID())
	}
}

func TestDispatcher(t *testing.T) {
	d := command.NewDispatcher()
	ctx := context.Background()

	executed := make([]command.Command, 0)

	handler := func(ctx context.Context, cmd command.Command) error {
		executed = append(executed, cmd)
		return nil
	}

	err := d.Register("CreateUser", handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := command.New("CreateUser", "user-123")
	err = d.Dispatch(ctx, cmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(executed) != 1 {
		t.Errorf("expected 1 command, got %d", len(executed))
	}

	if _, ok := d.Dispatch(ctx, command.New("UpdateUser", "user-456")); err == nil {
		t.Errorf("expected handler not found error for unregistered command")
	}
}

func TestDispatcherClosed(t *testing.T) {
	d := command.NewDispatcher()
	_ = d.Close()

	cmd := command.New("CreateUser", "user-123")
	err := d.Dispatch(context.Background(), cmd)
	if err == nil {
		t.Error("expected dispatcher closed error")
	}

	if _, ok := d.Register("CreateUser", nil); err == nil {
		t.Error("expected nil error for registering on closed dispatcher")
	}
}

}
