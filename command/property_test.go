package command_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestCommandCreation_ValidType(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := command.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		aggID := id.NewAggregateID()

		cmd, err := command.New(typ, aggID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Type() != typ {
			t.Fatalf("type mismatch: got %q, want %q", cmd.Type(), typ)
		}
		if cmd.AggregateID() != aggID {
			t.Fatalf("aggregateID mismatch")
		}
	})
}

func TestCommandCreation_EmptyTypeRejected(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	_, err := command.New("", aggID)
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestCommandMetadata_Roundtrip(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		corrID := id.NewCorrelationID()
		causID := id.NewCausationID()

		typ := command.Type("TestCommand")
		aggID := id.NewAggregateID()

		cmd, err := command.New(
			typ, aggID,
			command.WithCorrelationID(corrID),
			command.WithCausationID(causID),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		meta := cmd.Metadata()
		if meta.CorrelationID != corrID {
			t.Fatalf("correlationID mismatch")
		}
		if meta.CausationID != causID {
			t.Fatalf("causationID mismatch")
		}
	})
}

func TestCommandDispatch_UnknownTypeRejected(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := command.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		aggID := id.NewAggregateID()

		d := command.NewDispatcher()

		cmd, err := command.New(typ, aggID)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		err = d.Dispatch(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error for unregistered command")
		}
	})
}

func TestCommandDispatch_RegisterAndDispatch(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := command.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		aggID := id.NewAggregateID()

		d := command.NewDispatcher()

		called := false
		err := d.Register(typ, func(_ context.Context, cmd command.Command) error {
			called = true
			if cmd.Type() != typ {
				t.Fatalf("handler received wrong type: %q", cmd.Type())
			}

			return nil
		})
		if err != nil {
			t.Fatalf("register: %v", err)
		}

		cmd, err := command.New(typ, aggID)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		err = d.Dispatch(context.Background(), cmd)
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if !called {
			t.Fatal("handler was not called")
		}
	})
}
