package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func mustNewCmd(commandType command.Type, aggregateID id.AggregateID, opts ...command.Option) *command.BasicCommand {
	cmd, err := command.New(commandType, aggregateID, opts...)
	if err != nil {
		panic(err)
	}
	return s
}


var errTestFailure = errors.New("test failure")

type testTypedCmd struct {
	*command.BasicCommand

	Payload string
}

func TestRegisterTyped_Success(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()

	err := command.RegisterTyped(d, "test.cmd", func(_ context.Context, cmd *testTypedCmd) error {
		if cmd.Payload != "hello" {
			t.Errorf("expected payload 'hello', got %q", cmd.Payload)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	cmd := &testTypedCmd{
		BasicCommand: mustNewCmd("test.cmd", id.NewAggregateID()),
		Payload:      "hello",
	}

	if err := d.Dispatch(context.Background(), cmd); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
}

func TestRegisterTyped_HandlerError(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()

	err := command.RegisterTyped(d, "fail.cmd", func(_ context.Context, _ *testTypedCmd) error {
		return errTestFailure
	})
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	cmd := &testTypedCmd{BasicCommand: mustNewCmd("fail.cmd", id.NewAggregateID())}

	if err := d.Dispatch(context.Background(), cmd); !errors.Is(err, errTestFailure) {
		t.Fatalf("expected errTestFailure, got %v", err)
	}
}

func TestRegisterTyped_Duplicate(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	handler := func(_ context.Context, _ *testTypedCmd) error { return nil }

	err := command.RegisterTyped(d, "dup.cmd", handler)
	if err != nil {
		t.Fatalf("first RegisterTyped() error = %v", err)
	}

	err = command.RegisterTyped(d, "dup.cmd", handler)
	if err == nil {
		t.Fatal("expected duplicate registration error, got nil")
	}
}

func TestRegisterTyped_WorksWithMiddleware(t *testing.T) {
	t.Parallel()

	d := command.NewDispatcher()
	called := false

	d.Use(func(h command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			called = true

			return h(ctx, cmd)
		}
	})

	err := command.RegisterTyped(d, "mw.cmd", func(_ context.Context, _ *testTypedCmd) error {
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	cmd := &testTypedCmd{BasicCommand: mustNewCmd("mw.cmd", id.NewAggregateID())}

	if err := d.Dispatch(context.Background(), cmd); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if !called {
		t.Fatal("middleware was not called")
	}
}
