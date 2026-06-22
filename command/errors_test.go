package command_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestCommandErrors_Classification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want event.Family
	}{
		{"ErrHandlerNotFound", command.ErrHandlerNotFound, event.Rejection},
		{"ErrDispatcherClosed", command.ErrDispatcherClosed, event.Infrastructure},
		{"ErrEmptyCommandType", command.ErrEmptyCommandType, event.Rejection},
		{"ErrNilAggregateID", command.ErrNilAggregateID, event.Rejection},
		{"ErrTypeAssertion", command.ErrTypeAssertion, event.Rejection},
		{"ErrEmptyAggregateType", command.ErrEmptyAggregateType, event.Rejection},
		{"ErrDuplicateCommand", command.ErrDuplicateCommand, event.Conflict},
		{"ErrCommandNotFound", command.ErrCommandNotFound, event.Rejection},
		{"ErrStoreClosed", command.ErrStoreClosed, event.Infrastructure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := event.Classify(tc.err); got != tc.want {
				t.Fatalf("Classify(%s) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestWithCommandMetadata(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	md := command.NewMetadata()
	md.CorrelationID = id.NewCorrelationID()

	cmd, err := command.NewPersistedCommand(
		"CreateUser", ref, []byte(`{}`),
		command.WithCommandMetadata(md),
	)
	if err != nil {
		t.Fatalf("create persisted command: %v", err)
	}

	if cmd.Metadata().CorrelationID != md.CorrelationID {
		t.Errorf("expected correlation %q, got %q", md.CorrelationID, cmd.Metadata().CorrelationID)
	}
}

func TestDispatchOnClosed(t *testing.T) {
	d := command.NewDispatcher()

	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	cmd, _ := command.New("CreateUser", id.NewAggregateID())
	err := d.Dispatch(t.Context(), cmd)
	if err == nil {
		t.Fatal("expected error dispatching on closed dispatcher")
	}

	if event.Classify(err) != event.Infrastructure {
		t.Errorf("expected Infrastructure, got %v", event.Classify(err))
	}
}
