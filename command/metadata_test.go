package command_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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

func TestCommand_MetadataIsolation(t *testing.T) {
	t.Parallel()

	cmd, err := command.New(
		"CreateUser", id.NewAggregateID(),
		command.WithCorrelationID(id.NewCorrelationID()),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	m1 := cmd.Metadata()
	command.EnsureCustom(&m1)
	m1.Custom["key"] = "value"

	m2 := cmd.Metadata()
	if _, ok := m2.Custom["key"]; ok {
		t.Error("mutating Metadata() return value should not affect internal state")
	}
}

func TestCommand_MetadataMerge(t *testing.T) {
	t.Parallel()

	base := command.NewMetadata()
	base.CorrelationID = id.NewCorrelationID()
	command.EnsureCustom(&base)
	base.Custom["tenant"] = "acme"

	overlay := command.NewMetadata()
	overlay.UserID = id.NewUserID()
	command.EnsureCustom(&overlay)
	overlay.Custom["region"] = "us-east-1"

	merged := base.Merge(overlay)

	if merged.CorrelationID != base.CorrelationID {
		t.Errorf("CorrelationID not preserved: got %v, want %v",
			merged.CorrelationID, base.CorrelationID)
	}

	if merged.UserID != overlay.UserID {
		t.Errorf("UserID not overlaid: got %v, want %v", merged.UserID, overlay.UserID)
	}

	if merged.Custom["tenant"] != "acme" {
		t.Errorf("base Custom lost: tenant = %q", merged.Custom["tenant"])
	}

	if merged.Custom["region"] != "us-east-1" {
		t.Errorf("overlay Custom not copied: region = %q", merged.Custom["region"])
	}

	if _, ok := base.Custom["region"]; ok {
		t.Error("merge mutated the base Custom map")
	}
}

func TestCommand_AutoMintsID(t *testing.T) {
	t.Parallel()

	cmd, err := command.New("CreateUser", id.NewAggregateID())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if cmd.ID().IsZero() {
		t.Fatal("expected auto-minted CommandID, got zero")
	}
}

func TestCommand_WithCommandID(t *testing.T) {
	t.Parallel()

	customID := id.NewCommandID()
	cmd, err := command.New("CreateUser", id.NewAggregateID(), command.WithCommandID(customID))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if cmd.ID() != customID {
		t.Errorf("ID() = %v, want %v", cmd.ID(), customID)
	}
}

func TestCommand_TwoInstancesHaveDifferentIDs(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()

	cmd1, _ := command.New("CreateUser", aggID)
	cmd2, _ := command.New("CreateUser", aggID)

	if cmd1.ID() == cmd2.ID() {
		t.Fatal("two command instances should have different auto-minted IDs")
	}
}
