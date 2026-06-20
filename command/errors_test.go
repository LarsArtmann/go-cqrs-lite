package command_test

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestErrorFamilyAliases(t *testing.T) {
	t.Parallel()

	t.Run("Classify returns correct family", func(t *testing.T) {
		t.Parallel()

		rej := command.NewRejection("test.code", "rejection msg")
		if command.Classify(rej) != command.Rejection {
			t.Error("expected Rejection family")
		}

		con := command.NewConflict("test.code", "conflict msg")
		if command.Classify(con) != command.Conflict {
			t.Error("expected Conflict family")
		}

		tra := command.NewTransient("test.code", "transient msg")
		if command.Classify(tra) != command.Transient {
			t.Error("expected Transient family")
		}

		cor := command.NewCorruption("test.code", "corruption msg")
		if command.Classify(cor) != command.Corruption {
			t.Error("expected Corruption family")
		}

		inf := command.NewInfrastructure("test.code", "infra msg")
		if command.Classify(inf) != command.Infrastructure {
			t.Error("expected Infrastructure family")
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		t.Parallel()

		tra := command.NewTransient("test.code", "transient")
		if !command.IsRetryable(tra) {
			t.Error("transient should be retryable")
		}

		rej := command.NewRejection("test.code", "rejection")
		if command.IsRetryable(rej) {
			t.Error("rejection should not be retryable")
		}
	})

	t.Run("Wrap variants preserve family", func(t *testing.T) {
		t.Parallel()

		inner := errors.New("inner")

		wr := command.WrapRejection(inner, "wrap.rej", "msg")
		if command.Classify(wr) != command.Rejection {
			t.Error("WrapRejection should classify as Rejection")
		}

		wc := command.WrapConflict(inner, "wrap.con", "msg")
		if command.Classify(wc) != command.Conflict {
			t.Error("WrapConflict should classify as Conflict")
		}

		wt := errorfamily.WrapTransient(inner, "wrap.tra", "msg")
		if errorfamily.Classify(wt) != errorfamily.Transient {
			t.Error("WrapTransient should classify as Transient")
		}

		wco := command.WrapCorruption(inner, "wrap.cor", "msg")
		if command.Classify(wco) != command.Corruption {
			t.Error("WrapCorruption should classify as Corruption")
		}

		wi := command.WrapInfrastructure(inner, "wrap.inf", "msg")
		if command.Classify(wi) != command.Infrastructure {
			t.Error("WrapInfrastructure should classify as Infrastructure")
		}
	})

	t.Run("Wrap with explicit family", func(t *testing.T) {
		t.Parallel()

		inner := errors.New("inner")
		wrapped := command.Wrap(inner, command.Rejection, "wrap.family", "msg")
		if command.Classify(wrapped) != command.Rejection {
			t.Error("Wrap should use provided family")
		}
	})

	t.Run("Wrapf and Newf", func(t *testing.T) {
		t.Parallel()

		inner := errors.New("inner")
		wf := command.Wrapf(inner, command.Conflict, "wrapf.code", "value=%d", 42)
		if command.Classify(wf) != command.Conflict {
			t.Error("Wrapf should use provided family")
		}

		nf := command.Newf(command.Transient, "newf.code", "count=%d", 7)
		if command.Classify(nf) != command.Transient {
			t.Error("Newf should use provided family")
		}
	})

	t.Run("ExitCode", func(t *testing.T) {
		t.Parallel()

		rej := command.NewRejection("test.code", "msg")
		if command.ExitCode(rej) == 0 {
			t.Error("rejection should have non-zero exit code")
		}

		if command.ExitCode(errors.New("plain")) == 0 {
			t.Error("plain error should have non-zero exit code")
		}
	})
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

	if command.Classify(err) != command.Infrastructure {
		t.Errorf("expected Infrastructure, got %v", command.Classify(err))
	}
}
