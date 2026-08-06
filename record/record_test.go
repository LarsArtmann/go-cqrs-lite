package record_test

import (
	"encoding/json"
	"testing"
	"time"

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
			CorrelationID:    "corr-123",
			CausationID:      "cmd-456",
			ActorID:          "user:alice",
			ClientCreatedAt:  time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
			ServerReceivedAt: time.Date(2026, 8, 6, 12, 0, 1, 0, time.UTC),
			ServerStoredAt:   time.Date(2026, 8, 6, 12, 0, 2, 0, time.UTC),
			SchemaVersion:    2,
		},
	}

	data, err := json.Marshal(original) //nolint:musttag // Record's JSON shape is intentionally untagged (ADR-0111)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded record.Record
	if err := json.Unmarshal(data, &decoded); err != nil { //nolint:musttag // Record's JSON shape is intentionally untagged (ADR-0111)
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
	if decoded.MetaData.CorrelationID != original.MetaData.CorrelationID {
		t.Errorf("CorrelationID = %q, want %q", decoded.MetaData.CorrelationID, original.MetaData.CorrelationID)
	}
	if decoded.MetaData.SchemaVersion != original.MetaData.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", decoded.MetaData.SchemaVersion, original.MetaData.SchemaVersion)
	}
	if !decoded.MetaData.ClientCreatedAt.Equal(original.MetaData.ClientCreatedAt) {
		t.Errorf("ClientCreatedAt = %v, want %v", decoded.MetaData.ClientCreatedAt, original.MetaData.ClientCreatedAt)
	}
}

func TestCommonMetadata_ZeroValue(t *testing.T) {
	t.Parallel()

	var md record.CommonMetadata
	if md.CorrelationID != "" {
		t.Error("zero-value CorrelationID should be empty")
	}
	if md.SchemaVersion != 0 {
		t.Error("zero-value SchemaVersion should be 0")
	}
	if !md.ClientCreatedAt.IsZero() {
		t.Error("zero-value ClientCreatedAt should be zero time")
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
