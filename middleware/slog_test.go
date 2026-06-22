package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/middleware/v3"
)

type slogTestCommand struct {
	*command.BasicCommand
}

func TestSlogAdapter_CommandLogging(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(
		&buf,
		&slog.HandlerOptions{Level: slog.LevelInfo},
	))

	mw := middleware.CommandLogging(logger)
	handler := mw(noopCommandHandler())

	core, err := command.New("test.cmd", id.NewAggregateID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := &slogTestCommand{BasicCommand: core}

	err = handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected log output, got empty")
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"command dispatching"`)) {
		t.Errorf("expected 'command dispatching' log, got: %s", output)
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"command succeeded"`)) {
		t.Errorf("expected 'command succeeded' log, got: %s", output)
	}
}

func TestSlogAdapter_CommandLogging_Error(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(
		&buf,
		&slog.HandlerOptions{Level: slog.LevelInfo},
	))

	mw := middleware.CommandLogging(logger)
	handler := mw(func(_ context.Context, _ command.Command) error {
		return context.Canceled
	})

	core, err := command.New("test.cmd", id.NewAggregateID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := &slogTestCommand{BasicCommand: core}

	err = handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"command failed"`)) {
		t.Errorf("expected 'command failed' log, got: %s", output)
	}
}

func noopCommandHandler() func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}
