package command_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/v5/command"
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

func TestAsRecordPersisted_Nil(t *testing.T) {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	t.Parallel()

	got := command.AsRecordPersisted(nil)
	if got.Type != "" || !got.Stream.IsZero() {
		t.Fatalf("nil command should return zero Record, got %+v", got)
	}
}

func TestAsRecordPersisted_FullFidelity(t *testing.T) {
	t.Parallel()

	ref, err := command.NewStreamRef("user", id.NewStreamID())
	if err != nil {
		t.Fatalf("NewStreamRef: %v", err)
	}
	receivedAt := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	payload := []byte(`{"name":"Alice"}`)
	tracing := metadata.Tracing{
		CorrelationID: id.NewCorrelationID(),
		CausationID:   id.NewCausationID(),
		UserID:        id.NewUserID(),
	}

	cmd, err := command.NewPersistedCommand("user.create", ref, payload,
		command.WithReceivedAt(receivedAt),
		command.WithCommandMetadata(command.Metadata{Tracing: tracing}),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	rec := command.AsRecordPersisted(cmd)

	if rec.ID != record.InstanceID(cmd.ID().String()) {
		t.Errorf("ID: got %q, want command ID (identity must survive the bridge)", rec.ID)
	}

	if rec.Type != "user.create" {
		t.Errorf("Type: got %q, want %q", rec.Type, "user.create")
	}

	if string(rec.Payload.Data) != string(payload) {
		t.Errorf(
			"Payload.Data: got %q, want persisted payload carried (thin BasicCommand bridge leaves it absent)",
			rec.Payload.Data,
		)
	}

	if rec.Payload.Enc != record.EncodingUnknown {
		t.Errorf(
			"Payload.Enc = %v, want EncodingUnknown (the bridge cannot know the envelope)",
			rec.Payload.Enc,
		)
	}

	if rec.Stream != ref {
		t.Errorf(
			"Stream: got %v, want the persisted ref %v at full fidelity",
			rec.Stream, ref,
		)
	}

	if rec.Stream.Type != "user" {
		t.Errorf(
			"Stream.Type: got %q, want persisted stream type carried (thin bridge leaves Stream zero)",
			rec.Stream.Type,
		)
	}

	if rec.Version != 0 {
		t.Errorf("Version: got %d, want 0 (commands have no version)", rec.Version)
	}

	if !rec.MetaData.Received.Time().Equal(receivedAt) {
		t.Errorf("Received: got %v, want receive stamp from ReceivedAt", rec.MetaData.Received)
	}

	if !rec.MetaData.ClientCreatedAt.IsZero() {
		t.Errorf(
			"ClientCreatedAt: got %v, want zero (no client clock on commands; the old ReceivedAt relabel was the R8 lie)",
			rec.MetaData.ClientCreatedAt,
		)
	}

	if rec.MetaData.CorrelationID != record.CorrelationID(tracing.CorrelationID.String()) {
		t.Errorf(
			"CorrelationID: got %q, want %q",
			rec.MetaData.CorrelationID,
			tracing.CorrelationID.String(),
		)
	}

	if rec.MetaData.CausationID != tracing.CausationID.String() {
		t.Errorf(
			"CausationID: got %q, want %q",
			rec.MetaData.CausationID,
			tracing.CausationID.String(),
		)
	}

	wantCause := record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	if rec.MetaData.Cause != wantCause {
		t.Errorf(
			"Cause: got %+v, want %+v (tracing chain is honestly CauseUnknown)",
			rec.MetaData.Cause,
			wantCause,
		)
	}

	if rec.MetaData.ActorID != tracing.UserID.String() {
		t.Errorf(
			"ActorID: got %q, want UserID fallback %q",
			rec.MetaData.ActorID,
			tracing.UserID.String(),
		)
	}

	if rec.MetaData.Created.IsZero() != true {
		t.Errorf(
			"Created: got %v, want zero (persisted commands carry no client clock)",
			rec.MetaData.Created,
		)
	}

	if rec.MetaData.SchemaVersion != 0 {
		t.Errorf("SchemaVersion: got %d, want 0", rec.MetaData.SchemaVersion)
	}
}

func TestAsRecordPersisted_ZeroTracing(t *testing.T) {
	t.Parallel()

	ref, err := command.NewStreamRef("user", id.NewStreamID())
	if err != nil {
		t.Fatalf("NewStreamRef: %v", err)
	}

	cmd, err := command.NewPersistedCommand("user.create", ref, nil)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	rec := command.AsRecordPersisted(cmd)

	if !rec.MetaData.Cause.IsZero() {
		t.Errorf("Cause: got %+v, want zero (no causation recorded)", rec.MetaData.Cause)
	}

	if !rec.Payload.IsZero() {
		t.Errorf("Payload: got %v, want the absent state for nil payload", rec.Payload)
	}
}

func TestAsRecordPersisted_ActorKindWins(t *testing.T) {
	t.Parallel()

	ref, err := command.NewStreamRef("user", id.NewStreamID())
	if err != nil {
		t.Fatalf("NewStreamRef: %v", err)
	}
	actor := id.NewUserActor(id.NewUserID())

	cmd, err := command.NewPersistedCommand("user.create", ref, nil,
		command.WithCommandMetadata(command.Metadata{
			ActorID: actor,
		}),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	rec := command.AsRecordPersisted(cmd)

	if rec.MetaData.ActorID != actor.PrefixedString() {
		t.Errorf(
			"ActorID: got %q, want kind-discriminated %q",
			rec.MetaData.ActorID,
			actor.PrefixedString(),
		)
	}
}

func TestAsRecordPersisted_ZeroStructTreatedLikeNil(t *testing.T) {
	t.Parallel()

	// R10: a non-nil zero STRUCT (no ID, no type) must behave exactly like
	// nil — an identity-less source is not a Record.
	rec := command.AsRecordPersisted(&command.PersistedCommand{})
	if !rec.Stream.IsZero() || rec.Type != "" || !rec.Payload.IsZero() {
		t.Errorf("AsRecordPersisted(zero struct) = %+v, want the zero Record (R10)", rec)
	}
}
