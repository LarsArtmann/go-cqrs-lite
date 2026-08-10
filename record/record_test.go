package record_test

import (
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestStreamRef_ConstructAndSplit(t *testing.T) {
	t.Parallel()

	ref := record.NewStreamRef("User", "01JTEST")
	if got := ref.String(); got != "User/01JTEST" {
		t.Errorf("String() = %q, want %q", got, "User/01JTEST")
	}

	streamType, entityID := ref.Split()
	if streamType != "User" {
		t.Errorf("streamType = %q, want %q", streamType, "User")
	}

	if entityID != "01JTEST" {
		t.Errorf("entityID = %q, want %q", entityID, "01JTEST")
	}
}

func TestStreamRef_SplitInvalid(t *testing.T) {
	t.Parallel()

	ref := record.StreamRef("no-slash")
	streamType, entityID := ref.Split()
	if streamType != "" || entityID != "" {
		t.Errorf("Split() on invalid ref = (%q, %q), want (\"\", \"\")", streamType, entityID)
	}
}

func TestRecord_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := record.Record{
		Type:       "user.created",
		Payload:    []byte(`{"name":"alice"}`),
		StreamID:   record.NewStreamRef("User", "01JTEST"),
		StreamType: "User",
		Version:    1,
		MetaData: record.CommonMetadata{
			CorrelationID:    id.NewCorrelationID(),
			CausationID:      id.NewCausationID(),
			ActorID:          id.NewUserActor(id.NewUserID()),
			RequestID:        id.NewRequestID(),
			ClientCreatedAt:  time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
			ServerReceivedAt: time.Date(2026, 8, 6, 12, 0, 1, 0, time.UTC),
			ServerStoredAt:   time.Date(2026, 8, 6, 12, 0, 2, 0, time.UTC),
			SchemaVersion:    2,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded record.Record

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, original.Type)
	}

	if string(decoded.Payload) != string(original.Payload) {
		t.Errorf("Payload = %q, want %q", decoded.Payload, original.Payload)
	}

	if decoded.StreamID != original.StreamID {
		t.Errorf("StreamID = %q, want %q", decoded.StreamID, original.StreamID)
	}

	if decoded.StreamType != original.StreamType {
		t.Errorf("StreamType = %q, want %q", decoded.StreamType, original.StreamType)
	}

	if decoded.Version != original.Version {
		t.Errorf("Version = %d, want %d", decoded.Version, original.Version)
	}

	if !decoded.MetaData.CorrelationID.Equal(original.MetaData.CorrelationID) {
		t.Errorf(
			"CorrelationID = %v, want %v",
			decoded.MetaData.CorrelationID,
			original.MetaData.CorrelationID,
		)
	}

	if !decoded.MetaData.ActorID.Equal(original.MetaData.ActorID) {
		t.Errorf("ActorID = %v, want %v", decoded.MetaData.ActorID, original.MetaData.ActorID)
	}

	if decoded.MetaData.SchemaVersion != original.MetaData.SchemaVersion {
		t.Errorf(
			"SchemaVersion = %d, want %d",
			decoded.MetaData.SchemaVersion,
			original.MetaData.SchemaVersion,
		)
	}

	if !decoded.MetaData.ClientCreatedAt.Equal(original.MetaData.ClientCreatedAt) {
		t.Errorf(
			"ClientCreatedAt = %v, want %v",
			decoded.MetaData.ClientCreatedAt,
			original.MetaData.ClientCreatedAt,
		)
	}
}

func TestCommonMetadata_ZeroValue(t *testing.T) {
	t.Parallel()

	var md record.CommonMetadata
	if !md.CorrelationID.IsZero() {
		t.Error("zero-value CorrelationID should be zero")
	}

	if md.SchemaVersion != 0 {
		t.Error("zero-value SchemaVersion should be 0")
	}

	if !md.ClientCreatedAt.IsZero() {
		t.Error("zero-value ClientCreatedAt should be zero time")
	}

	if !md.ActorID.IsZero() {
		t.Error("zero-value ActorID should be zero")
	}
}

func TestCommonMetadata_Merge(t *testing.T) {
	t.Parallel()

	base := record.CommonMetadata{
		CorrelationID: id.NewCorrelationID(),
		ActorID:       id.NewSystemActor("scheduler"),
		SchemaVersion: 1,
	}

	overlay := record.CommonMetadata{
		CausationID: id.NewCausationID(),
		ActorID:     id.NewUserActor(id.NewUserID()),
	}

	result := base.Merge(overlay)

	if !result.CorrelationID.Equal(base.CorrelationID) {
		t.Error("Merge should preserve base CorrelationID")
	}

	if result.CausationID != overlay.CausationID {
		t.Error("Merge should overlay CausationID")
	}

	if result.ActorID.Equal(base.ActorID) {
		t.Error("Merge should overlay ActorID")
	}

	if result.SchemaVersion != base.SchemaVersion {
		t.Error("Merge should preserve base SchemaVersion")
	}
}

func TestRecord_EmbeddingWorks(t *testing.T) {
	t.Parallel()

	type EventRecord struct {
		record.Record

		Encoding string
	}

	er := EventRecord{
		Record: record.Record{
			Type:       "test.event",
			StreamType: "Test",
			Version:    1,
		},
		Encoding: "json",
	}

	if er.Type != "test.event" {
		t.Errorf("embedded Type = %q, want %q", er.Type, "test.event")
	}

	if er.Encoding != "json" {
		t.Errorf("Encoding = %q, want %q", er.Encoding, "json")
	}
}
