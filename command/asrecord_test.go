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

	if rec.MetaData.ActorID != userID.String() {
		t.Errorf("ActorID: got %q, want %q", rec.MetaData.ActorID, userID.String())
	}

	if rec.Payload != nil {
		t.Errorf("Payload: got %v, want nil (commands have no blob payload)", rec.Payload)
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

	if rec.MetaData.ActorID != "" {
		t.Errorf("ActorID: got %q, want empty for zero metadata", rec.MetaData.ActorID)
	}

	if rec.Type != "user.create" {
		t.Errorf("Type should still be set: got %q", rec.Type)
	}
}
