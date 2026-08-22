package event

import (
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestAsRecord_NilEvent(t *testing.T) {
	rec := AsRecord(nil)
	if rec.Type != "" || rec.Payload != nil || rec.StreamID != "" {
		t.Errorf("AsRecord(nil) = %+v, want zero Record", rec)
	}
}

func TestAsRecord_BasicMapping(t *testing.T) {
	streamID := id.NewStreamID()
	occurredAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	correlationID := id.NewCorrelationID()
	userID := id.NewUserID()

	payload := UserCreated{Name: "Alice"}

	evt, err := New(
		"user.created",
		streamID,
		"User",
		Version(3),
		payload,
		WithOccurredAt(occurredAt),
		WithCorrelationID(correlationID),
		WithUserID(userID),
		WithSchemaVersion(2),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := AsRecord(evt)

	if rec.Type != "user.created" {
		t.Errorf("Type = %q, want %q", rec.Type, "user.created")
	}

	if rec.StreamType != "User" {
		t.Errorf("StreamType = %q, want %q", rec.StreamType, "User")
	}

	wantStreamID := record.NewStreamRef("User", streamID.String())
	if rec.StreamID != wantStreamID {
		t.Errorf("StreamID = %q, want %q", rec.StreamID, wantStreamID)
	}

	st, id2 := rec.StreamID.Split()
	if st != "User" || id2 != streamID.String() {
		t.Errorf("StreamID.Split = (%q, %q), want (%q, %q)", st, id2, "User", streamID.String())
	}

	if rec.Version != 3 {
		t.Errorf("Version = %d, want 3", rec.Version)
	}

	if rec.MetaData.CorrelationID != correlationID.String() {
		t.Errorf("CorrelationID = %q, want %q", rec.MetaData.CorrelationID, correlationID.String())
	}

	if rec.MetaData.ActorID != userID.String() {
		t.Errorf("ActorID = %q, want %q", rec.MetaData.ActorID, userID.String())
	}

	if rec.MetaData.Actor != (record.Actor{Kind: record.ActorUser, Raw: userID.String()}) {
		t.Errorf("Actor = %+v, want structural user actor (legacy UserID fallback)",
			rec.MetaData.Actor)
	}

	t.Run("kind-discriminated ActorID wins over UserID", func(t *testing.T) {
		t.Parallel()

		actor := id.NewServiceActor("api-gateway")
		evtWithActor, err := New(
			"user.created", streamID, "User", Version(3), payload,
			WithUserID(userID),
			WithActor(actor),
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		got := AsRecord(evtWithActor).MetaData.ActorID
		if got != "service:api-gateway" {
			t.Errorf("ActorID = %q, want %q", got, "service:api-gateway")
		}
	})

	if rec.MetaData.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", rec.MetaData.SchemaVersion)
	}

	if !rec.MetaData.ClientCreatedAt.Equal(occurredAt) {
		t.Errorf("ClientCreatedAt = %v, want %v", rec.MetaData.ClientCreatedAt, occurredAt)
	}

	if rec.MetaData.Created.IsZero() || !rec.MetaData.Created.Time().Equal(occurredAt) {
		t.Errorf("Created = %v, want known stamp at %v", rec.MetaData.Created, occurredAt)
	}

	if !rec.MetaData.Received.IsZero() || !rec.MetaData.Stored.IsZero() {
		t.Error("Received/Stored must stay unknown at the event layer (store's job)")
	}

	if len(rec.Payload) == 0 {
		t.Error("Payload is empty")
	}
}

func TestAsRecord_CausationID(t *testing.T) {
	t.Run("typed Causation takes precedence", func(t *testing.T) {
		causationID := id.NewCausationID()
		commandID := id.NewCommandID()

		evt, err := New(
			"user.created",
			id.NewStreamID(),
			"User",
			Version(1),
			UserCreated{Name: "Bob"},
			WithCausationID(causationID),
			WithCausation("create_user", commandID),
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		rec := AsRecord(evt)

		if rec.MetaData.CausationID != commandID.String() {
			t.Errorf("CausationID = %q, want command ID %q",
				rec.MetaData.CausationID, commandID.String())
		}

		wantCause := record.Cause{Kind: record.CauseCommand, ID: commandID.String()}
		if rec.MetaData.Cause != wantCause {
			t.Errorf("Cause = %+v, want %+v", rec.MetaData.Cause, wantCause)
		}
	})

	t.Run("falls back to Tracing.CausationID when no typed Causation", func(t *testing.T) {
		causationID := id.NewCausationID()

		evt, err := New(
			"user.created",
			id.NewStreamID(),
			"User",
			Version(1),
			UserCreated{Name: "Bob"},
			WithCausationID(causationID),
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		rec := AsRecord(evt)

		if rec.MetaData.CausationID != causationID.String() {
			t.Errorf("CausationID = %q, want tracing causation %q",
				rec.MetaData.CausationID, causationID.String())
		}

		wantCause := record.Cause{Kind: record.CauseUnknown, ID: causationID.String()}
		if rec.MetaData.Cause != wantCause {
			t.Errorf("Cause = %+v, want %+v (kind unknown: tracing does not discriminate)",
				rec.MetaData.Cause, wantCause)
		}
	})

	t.Run("empty when neither set", func(t *testing.T) {
		evt, err := New(
			"user.created",
			id.NewStreamID(),
			"User",
			Version(1),
			UserCreated{Name: "Bob"},
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		rec := AsRecord(evt)

		if rec.MetaData.CausationID != "" {
			t.Errorf("CausationID = %q, want empty", rec.MetaData.CausationID)
		}

		if !rec.MetaData.Cause.IsZero() {
			t.Errorf("Cause = %+v, want zero (no cause recorded)", rec.MetaData.Cause)
		}
	})
}

func TestAsRecord_IDAndEncoding(t *testing.T) {
	t.Parallel()

	newEvent := func(t *testing.T, opts ...Option) Event {
		t.Helper()
		evt, err := New(
			"user.created", id.NewStreamID(), "User", Version(1),
			UserCreated{Name: "Codec"}, opts...,
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		return evt
	}

	t.Run("default codec stamps json or cbor per event encoding", func(t *testing.T) {
		t.Parallel()

		evt := newEvent(t)
		rec := AsRecord(evt)

		if rec.ID != evt.ID().String() {
			t.Errorf("ID = %q, want event ID %q", rec.ID, evt.ID().String())
		}

		if rec.Encoding != string(evt.Encoding()) {
			t.Errorf("Encoding = %q, want %q", rec.Encoding, evt.Encoding())
		}

		if rec.Encoding != "json" && rec.Encoding != "cbor" {
			t.Errorf("Encoding = %q, want a known codec stamp", rec.Encoding)
		}
	})

	t.Run("explicit CBOR event survives with its stamp", func(t *testing.T) {
		t.Parallel()

		evt := newEvent(t, WithCodec(codec.CBORCodec{}))
		if evt.Encoding() != codec.EncodingCBOR {
			t.Fatalf("test setup: event encoding = %q, want cbor", evt.Encoding())
		}

		if got := AsRecord(evt).Encoding; got != "cbor" {
			t.Errorf("Encoding = %q, want %q (mixed streams stay self-describing)", got, "cbor")
		}
	})

	t.Run("explicit JSON event survives with its stamp", func(t *testing.T) {
		t.Parallel()

		evt := newEvent(t, WithCodec(codec.JSONCodec{}))
		if evt.Encoding() != codec.EncodingJSON {
			t.Fatalf("test setup: event encoding = %q, want json", evt.Encoding())
		}

		if got := AsRecord(evt).Encoding; got != "json" {
			t.Errorf("Encoding = %q, want %q", got, "json")
		}
	})
}

func TestAsRecord_ZeroValueMetadata(t *testing.T) {
	evt, err := New(
		"thing.happened",
		id.NewStreamID(),
		"Thing",
		Version(1),
		UserCreated{Name: "Test"},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := AsRecord(evt)

	if rec.MetaData.CorrelationID != "" {
		t.Errorf("CorrelationID = %q, want empty", rec.MetaData.CorrelationID)
	}

	if rec.MetaData.CausationID != "" {
		t.Errorf("CausationID = %q, want empty", rec.MetaData.CausationID)
	}

	if rec.MetaData.ActorID != "" {
		t.Errorf("ActorID = %q, want empty", rec.MetaData.ActorID)
	}

	if rec.MetaData.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1 (default)", rec.MetaData.SchemaVersion)
	}

	if !rec.MetaData.ServerReceivedAt.IsZero() {
		t.Error("ServerReceivedAt should be zero (unknown at event layer)")
	}

	if !rec.MetaData.ServerStoredAt.IsZero() {
		t.Error("ServerStoredAt should be zero (unknown at event layer)")
	}
}

// UserCreated is a test payload type.
type UserCreated struct {
	Name string
}

func TestAsRecord_StreamRefInvariant(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	evt, err := New("user.created", streamID, "User", Version(3), UserCreated{Name: "Alice"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := AsRecord(evt)
	if err := rec.StreamID.Validate(); err != nil {
		t.Fatalf("populated StreamID must pass Validate, got %v (%q)", err, rec.StreamID)
	}

	if streamType, entityID := rec.StreamID.Split(); streamType != "User" ||
		entityID != streamID.String() {
		t.Errorf(
			"Split = (%q, %q), want (%q, %q)",
			streamType, entityID, "User", streamID.String(),
		)
	}
}
