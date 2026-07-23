package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
)

// TestIdempotencyDemo verifies that the taskmanager's command dispatcher
// rejects duplicate command dispatches via the CommandIdempotency middleware.
// This exercises the full production stack: dispatcher → middleware chain
// (recovery + logging + retry + idempotency) → decider → repository → store.
func TestIdempotencyDemo(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	srv, err := NewServer(DefaultConfig(), logger)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	ctx := context.Background()
	taskID := id.NewStreamID()

	createCmd, err := command.New(cmdCreateTask, taskID)
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	wrapped := CreateTaskCmd{
		BasicCommand: createCmd,
		Title:        "Idempotency Test",
		Priority:     PriorityMedium,
	}

	if err := srv.CmdDisp.Dispatch(ctx, wrapped); err != nil {
		t.Fatalf("first dispatch: %v", err)
	}

	err = srv.CmdDisp.Dispatch(ctx, wrapped)
	if !errors.Is(err, idempotency.ErrDuplicate) {
		t.Fatalf("second dispatch of same command: want ErrDuplicate, got %v", err)
	}

	events, err := srv.Bundle.EventSource.Load(
		ctx,
		id.NewStreamRef(id.StreamType("Task"), taskID),
	)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}
