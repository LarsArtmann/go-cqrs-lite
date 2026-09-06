package command_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestCommandCreation_ValidType(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := command.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		streamID := id.NewStreamID()

		cmd, err := command.New(typ, streamID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Type() != typ {
			t.Fatalf("type mismatch: got %q, want %q", cmd.Type(), typ)
		}
		if cmd.StreamID() != streamID {
			t.Fatalf("streamID mismatch")
		}
	})
}

func TestCommandCreation_EmptyTypeRejected(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()

	_, err := command.New("", streamID)
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
		streamID := id.NewStreamID()

		cmd, err := command.New(
			typ, streamID,
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
		streamID := id.NewStreamID()

		d := command.NewDispatcher()

		cmd, err := command.New(typ, streamID)
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
		streamID := id.NewStreamID()

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

		cmd, err := command.New(typ, streamID)
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
