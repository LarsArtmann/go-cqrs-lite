package command_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestAsRecord_Nil(t *testing.T) {
	t.Parallel()

	got := command.AsRecord(nil)
	if got.Type != "" || got.StreamID != "" {
		t.Fatalf("nil command should return zero Record, got %+v", got)
	}
}

func TestAsRecord_BasicMapping(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	corrID := id.NewCorrelationID()
	causationID := id.NewCausationID()
	userID := id.NewUserID()

	cmd, err := command.New("user.create", streamID,
		command.WithCorrelationID(corrID),
		command.WithCausationID(causationID),
		command.WithUserID(userID),
	)
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	rec := command.AsRecord(cmd)

	if rec.Type != "user.create" {
		t.Errorf("Type: got %q, want %q", rec.Type, "user.create")
	}

	wantStreamID := record.NewStreamRef("", streamID.String())
	if rec.StreamID != wantStreamID {
		t.Errorf("StreamID: got %+v, want %+v", rec.StreamID, wantStreamID)
	}

	if rec.StreamType != "" {
		t.Errorf("StreamType: got %q, want empty (commands have no stream type)", rec.StreamType)
	}

	if rec.Version != 0 {
		t.Errorf("Version: got %d, want 0 (commands have no version)", rec.Version)
	}

	if rec.MetaData.CorrelationID != corrID.String() {
		t.Errorf("CorrelationID: got %q, want %q", rec.MetaData.CorrelationID, corrID.String())
	}

	if rec.MetaData.CausationID != causationID.String() {
		t.Errorf("CausationID: got %q, want %q", rec.MetaData.CausationID, causationID.String())
	}

	wantCause := record.Cause{Kind: record.CauseUnknown, ID: causationID.String()}
	if rec.MetaData.Cause != wantCause {
		t.Errorf("Cause: got %+v, want %+v (kind unknown: tracing does not discriminate)",
			rec.MetaData.Cause, wantCause)
	}

	if rec.MetaData.ActorID != userID.String() {
		t.Errorf("ActorID: got %q, want %q", rec.MetaData.ActorID, userID.String())
	}

	if rec.MetaData.Actor != (record.Actor{Kind: record.ActorUser, Raw: userID.String()}) {
		t.Errorf("Actor: got %+v, want structural user actor", rec.MetaData.Actor)
	}

	if rec.Payload != nil {
		t.Errorf("Payload: got %v, want nil (commands have no blob payload)", rec.Payload)
	}
}

func TestAsRecord_ActorPrecedence(t *testing.T) {
	t.Parallel()

	userID := id.NewUserID()
	actor := id.NewSystemActor("migration")

	cmd, err := command.New("user.create", id.NewStreamID(),
		command.WithUserID(userID),
		command.WithActor(actor),
	)
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	rec := command.AsRecord(cmd)

	if rec.MetaData.ActorID != "system:migration" {
		t.Errorf("ActorID: got %q, want %q (kind-discriminated actor must win)",
			rec.MetaData.ActorID, "system:migration")
	}
}

func TestAsRecord_ZeroMetadata(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	cmd, err := command.New("user.create", streamID)
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	rec := command.AsRecord(cmd)

	if rec.MetaData.CorrelationID != "" {
		t.Errorf("CorrelationID: got %q, want empty for zero metadata", rec.MetaData.CorrelationID)
	}

	if rec.MetaData.CausationID != "" {
		t.Errorf("CausationID: got %q, want empty for zero metadata", rec.MetaData.CausationID)
	}

	if !rec.MetaData.Cause.IsZero() {
		t.Errorf("Cause: got %+v, want zero for zero metadata", rec.MetaData.Cause)
	}

	if rec.MetaData.ActorID != "" {
		t.Errorf("ActorID: got %q, want empty for zero metadata", rec.MetaData.ActorID)
	}

	if rec.Type != "user.create" {
		t.Errorf("Type should still be set: got %q", rec.Type)
	}
}

func TestAsRecord_StreamRefInvariant(t *testing.T) {
	t.Parallel()

	cmd, err := command.New("user.create", id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	rec := command.AsRecord(cmd)
	if err := rec.StreamID.Validate(); err != nil {
		t.Fatalf("populated StreamID must pass Validate, got %v (%q)", err, rec.StreamID)
	}

	if _, entityID := rec.StreamID.Split(); entityID != cmd.StreamID().String() {
		t.Errorf(
			"Split entityID = %q, want the command's stream ID %q",
			entityID, cmd.StreamID().String(),
		)
	}
}
