package command_test

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestCommandErrors_Classification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want errorfamily.Family
	}{
		{"ErrHandlerNotFound", command.ErrHandlerNotFound, errorfamily.Rejection},
		{"ErrDispatcherClosed", command.ErrDispatcherClosed, errorfamily.Infrastructure},
		{"ErrEmptyCommandType", command.ErrEmptyCommandType, errorfamily.Rejection},
		{"ErrNilStreamID", command.ErrNilStreamID, errorfamily.Rejection},
		{"ErrTypeAssertion", command.ErrTypeAssertion, errorfamily.Rejection},
		{"ErrEmptyStreamType", command.ErrEmptyStreamType, errorfamily.Rejection},
		{"ErrDuplicateCommand", command.ErrDuplicateCommand, errorfamily.Conflict},
		{"ErrCommandNotFound", command.ErrCommandNotFound, errorfamily.Rejection},
		{"ErrStoreClosed", command.ErrStoreClosed, errorfamily.Infrastructure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := errorfamily.Classify(tc.err); got != tc.want {
				t.Fatalf("Classify(%s) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestWithCommandMetadata(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	ref := command.NewAggregateRef("User", aggID)

	md := command.Metadata{}
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

	if errorfamily.Classify(err) != errorfamily.Infrastructure {
		t.Errorf("expected Infrastructure, got %v", errorfamily.Classify(err))
	}
}
