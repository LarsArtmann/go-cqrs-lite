package command_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/id"
)

func TestCommand_Metadata_Defaults(t *testing.T) {
	t.Parallel()

	cmd, err := command.New("CreateUser", id.NewAggregateID())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	m := cmd.Metadata()
	if !m.CorrelationID.IsZero() {
		t.Error("expected zero CorrelationID by default")
	}

	if !m.CausationID.IsZero() {
		t.Error("expected zero CausationID by default")
	}

	if !m.UserID.IsZero() {
		t.Error("expected zero UserID by default")
	}

	if !m.RequestID.IsZero() {
		t.Error("expected zero RequestID by default")
	}
}

func TestCommand_WithCorrelationID(t *testing.T) {
	t.Parallel()

	cid := id.NewCorrelationID()
	cmd, err := command.New("CreateUser", id.NewAggregateID(), command.WithCorrelationID(cid))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if cmd.Metadata().CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", cmd.Metadata().CorrelationID, cid)
	}
}

func TestCommand_WithCausationID(t *testing.T) {
	t.Parallel()

	caid := id.NewCausationID()
	cmd, err := command.New("CreateUser", id.NewAggregateID(), command.WithCausationID(caid))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if cmd.Metadata().CausationID != caid {
		t.Errorf("CausationID = %v, want %v", cmd.Metadata().CausationID, caid)
	}
}

func TestCommand_WithUserID(t *testing.T) {
	t.Parallel()

	uid := id.NewUserID()
	cmd, err := command.New("CreateUser", id.NewAggregateID(), command.WithUserID(uid))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if cmd.Metadata().UserID != uid {
		t.Errorf("UserID = %v, want %v", cmd.Metadata().UserID, uid)
	}
}

func TestCommand_WithRequestID(t *testing.T) {
	t.Parallel()

	rid := id.NewRequestID()
	cmd, err := command.New("CreateUser", id.NewAggregateID(), command.WithRequestID(rid))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if cmd.Metadata().RequestID != rid {
		t.Errorf("RequestID = %v, want %v", cmd.Metadata().RequestID, rid)
	}
}

func TestCommand_AllMetadata(t *testing.T) {
	t.Parallel()

	cid := id.NewCorrelationID()
	caid := id.NewCausationID()
	uid := id.NewUserID()
	rid := id.NewRequestID()

	cmd, err := command.New(
		"CreateUser", id.NewAggregateID(),
		command.WithCorrelationID(cid),
		command.WithCausationID(caid),
		command.WithUserID(uid),
		command.WithRequestID(rid),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	m := cmd.Metadata()
	if m.CorrelationID != cid {
		t.Errorf("CorrelationID = %v, want %v", m.CorrelationID, cid)
	}

	if m.CausationID != caid {
		t.Errorf("CausationID = %v, want %v", m.CausationID, caid)
	}

	if m.UserID != uid {
		t.Errorf("UserID = %v, want %v", m.UserID, uid)
	}

	if m.RequestID != rid {
		t.Errorf("RequestID = %v, want %v", m.RequestID, rid)
	}
}
