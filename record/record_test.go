package record_test

import (
	"encoding/json/v2"
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

func TestStreamRef_SplitLeadingSlash(t *testing.T) {
	t.Parallel()

	// Leading slash = empty stream type → allowed (matches Validate and the
	// command/query asrecord pattern); the entity ID is still returned.
	streamType, entityID := record.StreamRef("/01JTEST").Split()
	if streamType != "" || entityID != "01JTEST" {
		t.Errorf(
			"Split() on leading-slash ref = (%q, %q), want (\"\", \"01JTEST\")",
			streamType,
			entityID,
		)
	}
}

func TestStreamRef_SplitTrailingSlash(t *testing.T) {
	t.Parallel()

	// Trailing slash = empty entity ID → invalid.
	streamType, entityID := record.StreamRef("User/").Split()
	if streamType != "" || entityID != "" {
		t.Errorf(
			"Split() on trailing-slash ref = (%q, %q), want (\"\", \"\")",
			streamType,
			entityID,
		)
	}
}

func TestStreamRef_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref     record.StreamRef
		wantErr bool
	}{
		{record.NewStreamRef("User", "01JTEST"), false},
		{
			record.NewStreamRef("", "01JTEST"),
			false,
		}, // empty streamType allowed (command/query pattern)
		{record.StreamRef("no-slash"), true}, // missing separator
		{
			record.StreamRef("/01JTEST"),
			false,
		}, // empty streamType but entityID present → valid
		{record.StreamRef("User/"), true}, // empty entityID → invalid
		{record.StreamRef(""), true},      // completely empty → invalid
	}

	for _, tt := range tests {
		err := tt.ref.Validate()
		if tt.wantErr && err == nil {
			t.Errorf("Validate(%q) = nil, want error", tt.ref)
		}

		if !tt.wantErr && err != nil {
			t.Errorf("Validate(%q) = %v, want nil", tt.ref, err)
		}
	}
}

func TestStreamRef_ConstructSplitRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct{ streamType, entityID string }{
		{"User", "01JTEST"},
		{"", "01JTEST"}, // empty streamType (command/query asrecord pattern)
	}

	for _, tc := range cases {
		ref := record.NewStreamRef(tc.streamType, tc.entityID)

		if err := ref.Validate(); err != nil {
			t.Errorf(
				"Validate(NewStreamRef(%q, %q)) = %v, want nil",
				tc.streamType,
				tc.entityID,
				err,
			)
		}

		gotType, gotID := ref.Split()
		if gotType != tc.streamType || gotID != tc.entityID {
			t.Errorf(
				"Split(NewStreamRef(%q, %q)) = (%q, %q), want (%q, %q)",
				tc.streamType, tc.entityID,
				gotType, gotID,
				tc.streamType, tc.entityID,
			)
		}
	}
}

func TestNewStreamRefOrZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		streamType string
		entityID   string
		want       record.StreamRef
	}{
		{"typed ref", "User", "01JTEST", "User/01JTEST"},
		{"empty stream type is legal", "", "01JTEST", "/01JTEST"},
		{"empty entity ID yields zero", "User", "", ""},
		{"both empty yields zero", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := record.NewStreamRefOrZero(tc.streamType, tc.entityID)
			if got != tc.want {
				t.Fatalf(
					"NewStreamRefOrZero(%q, %q) = %q, want %q",
					tc.streamType, tc.entityID, got, tc.want,
				)
			}

			if got != "" && got.Validate() != nil {
				t.Errorf(
					"non-zero result %q must pass Validate, got %v",
					got, got.Validate(),
				)
			}
		})
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
			Created:          record.NewStamp(time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)),
			Received:         record.NewStamp(time.Date(2026, 8, 6, 12, 0, 1, 0, time.UTC)),
			SchemaVersion:    2,
		},
	}

	data, err := json.Marshal(original)
	if err != nil { // Record's JSON shape is intentionally untagged (ADR-0111)
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
	if decoded.MetaData.CorrelationID != original.MetaData.CorrelationID {
		t.Errorf(
			"CorrelationID = %q, want %q",
			decoded.MetaData.CorrelationID,
			original.MetaData.CorrelationID,
		)
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
	if !decoded.MetaData.Created.Time().Equal(original.MetaData.Created.Time()) ||
		decoded.MetaData.Created.IsZero() != original.MetaData.Created.IsZero() {
		t.Errorf("Created = %v, want %v", decoded.MetaData.Created, original.MetaData.Created)
	}
	if !decoded.MetaData.Received.Time().Equal(original.MetaData.Received.Time()) ||
		decoded.MetaData.Received.IsZero() != original.MetaData.Received.IsZero() {
		t.Errorf("Received = %v, want %v", decoded.MetaData.Received, original.MetaData.Received)
	}
	if !decoded.MetaData.Stored.IsZero() || !original.MetaData.Stored.IsZero() {
		t.Errorf("Stored round trip must keep the zero (unknown) stamp, got %v", decoded.MetaData.Stored)
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
	if !md.Created.IsZero() || !md.Received.IsZero() || !md.Stored.IsZero() {
		t.Error("zero-value Stamps must be unknown (zero Stamp)")
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
