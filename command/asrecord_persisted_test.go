package command_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestAsRecordPersisted_Nil(t *testing.T) {
	t.Parallel()

	got := command.AsRecordPersisted(nil)
	if got.Type != "" || got.StreamID != "" {
		t.Fatalf("nil command should return zero Record, got %+v", got)
	}
}

func TestAsRecordPersisted_FullFidelity(t *testing.T) {
	t.Parallel()

	ref := command.NewStreamRef("user", id.NewStreamID())
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

	if rec.ID != cmd.ID().String() {
		t.Errorf("ID: got %q, want command ID (identity must survive the bridge)", rec.ID)
	}

	if rec.Type != "user.create" {
		t.Errorf("Type: got %q, want %q", rec.Type, "user.create")
	}

	if string(rec.Payload) != string(payload) {
		t.Errorf("Payload: got %q, want persisted payload carried (thin BasicCommand bridge leaves it empty)", rec.Payload)
	}

	if rec.Encoding != record.EncodingUnknown {
		t.Errorf("Encoding: got %v, want EncodingUnknown (ADR-0044 envelope self-describes)", rec.Encoding)
	}

	if rec.StreamType != "user" {
		t.Errorf("StreamType: got %q, want persisted stream type carried (thin bridge leaves it empty)", rec.StreamType)
	}

	if rec.StreamID != record.NewStreamRefOrZero("user", ref.ID.String()) {
		t.Errorf("StreamID: got %+v, want full-fidelity stream ref", rec.StreamID)
	}

	if rec.Version != 0 {
		t.Errorf("Version: got %d, want 0 (commands have no version)", rec.Version)
	}

	if !rec.MetaData.Received.Time().Equal(receivedAt) {
		t.Errorf("Received: got %v, want receive stamp from ReceivedAt", rec.MetaData.Received)
	}

	if !rec.MetaData.ClientCreatedAt.Equal(receivedAt) {
		t.Errorf("ClientCreatedAt: got %v, want lockstep with Received until the v5 cut", rec.MetaData.ClientCreatedAt)
	}

	if rec.MetaData.CorrelationID != tracing.CorrelationID.String() {
		t.Errorf("CorrelationID: got %q, want %q", rec.MetaData.CorrelationID, tracing.CorrelationID.String())
	}

	if rec.MetaData.CausationID != tracing.CausationID.String() {
		t.Errorf("CausationID: got %q, want %q", rec.MetaData.CausationID, tracing.CausationID.String())
	}

	wantCause := record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	if rec.MetaData.Cause != wantCause {
		t.Errorf("Cause: got %+v, want %+v (tracing chain is honestly CauseUnknown)", rec.MetaData.Cause, wantCause)
	}

	if rec.MetaData.ActorID != tracing.UserID.String() {
		t.Errorf("ActorID: got %q, want UserID fallback %q", rec.MetaData.ActorID, tracing.UserID.String())
	}

	if rec.MetaData.Created.IsZero() != true {
		t.Errorf("Created: got %v, want zero (persisted commands carry no client clock)", rec.MetaData.Created)
	}

	if rec.MetaData.SchemaVersion != 0 {
		t.Errorf("SchemaVersion: got %d, want 0", rec.MetaData.SchemaVersion)
	}
}

func TestAsRecordPersisted_ZeroTracing(t *testing.T) {
	t.Parallel()

	ref := command.NewStreamRef("user", id.NewStreamID())

	cmd, err := command.NewPersistedCommand("user.create", ref, nil)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	rec := command.AsRecordPersisted(cmd)

	if !rec.MetaData.Cause.IsZero() {
		t.Errorf("Cause: got %+v, want zero (no causation recorded)", rec.MetaData.Cause)
	}

	if rec.Payload != nil {
		t.Errorf("Payload: got %q, want nil passthrough for nil payload", rec.Payload)
	}
}

func TestAsRecordPersisted_ActorKindWins(t *testing.T) {
	t.Parallel()

	ref := command.NewStreamRef("user", id.NewStreamID())
	actor := id.NewUserActor(id.NewUserID())

	cmd, err := command.NewPersistedCommand("user.create", ref, nil,
		command.WithCommandMetadata(command.Metadata{
			Tracing: metadata.Tracing{ActorID: actor},
		}),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	rec := command.AsRecordPersisted(cmd)

	if rec.MetaData.ActorID != actor.PrefixedString() {
		t.Errorf("ActorID: got %q, want kind-discriminated %q", rec.MetaData.ActorID, actor.PrefixedString())
	}
}
