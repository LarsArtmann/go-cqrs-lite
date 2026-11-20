package command_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/v5/command"
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

func TestAsRecord_Nil(t *testing.T) {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	t.Parallel()

	got := command.AsRecord(nil)
	if got.Type != "" || !got.Stream.IsZero() {
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

	if rec.ID != record.InstanceID(cmd.ID().String()) {
		t.Errorf("ID: got %q, want command ID %q (identity must survive the bridge)",
			rec.ID, cmd.ID().String())
	}

	if !rec.Payload.IsZero() || rec.Payload.Enc != record.EncodingUnknown {
		t.Errorf(
			"Payload: got %v, want the absent state (commands have no payload to stamp)",
			rec.Payload,
		)
	}

	if !rec.Stream.IsZero() {
		t.Errorf(
			"Stream: got %v, want zero (BasicCommand has no stream TYPE; ADR-0166 forbids empty-type refs)",
			rec.Stream,
		)
	}

	if rec.Version != 0 {
		t.Errorf("Version: got %d, want 0 (commands have no version)", rec.Version)
	}

	if rec.MetaData.CorrelationID != record.CorrelationID(corrID.String()) {
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

	if !rec.Payload.IsZero() {
		t.Errorf("Payload: got %v, want absent (commands have no blob payload)", rec.Payload)
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

	// The thin BasicCommand bridge leaves Stream zero (no stream TYPE to
	// pair with the ID; ADR-0166 forbids empty-type refs). The command's
	// stream ID is still reachable through cmd.StreamID() itself.
	rec := command.AsRecord(cmd)
	if !rec.Stream.IsZero() {
		t.Fatalf("Stream = %v, want zero for the thin BasicCommand bridge", rec.Stream)
	}
}

func TestAsRecord_ZeroStructTreatedLikeNil(t *testing.T) {
	t.Parallel()

	// R10: a non-nil zero STRUCT (no ID, no type) must behave exactly like
	// nil — an identity-less source is not a Record.
	rec := command.AsRecord(&command.BasicCommand{})
	if !rec.Stream.IsZero() || rec.Type != "" || !rec.Payload.IsZero() {
		t.Errorf("AsRecord(zero struct) = %+v, want the zero Record (R10)", rec)
	}
}
