package record_test

import (
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

func mustStreamRef(t *testing.T, streamType, entityID string) record.StreamRef {
	t.Helper()

	sid, err := id.ParseStreamID(entityID)
	if err != nil {
		t.Fatalf("ParseStreamID(%q): %v", entityID, err)
	}

	ref, err := id.NewStreamRef(id.StreamType(streamType), sid)
	if err != nil {
		t.Fatalf("NewStreamRef(%q, %q): %v", streamType, entityID, err)
	}

	return ref
}

func TestStreamRef_AliasLockstep(t *testing.T) {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	// ADR-0166: record.StreamRef is an alias of id.StreamRef — one canonical
	// struct, one validation rule set. These cross-type constructions only
	// compile while the alias holds (the record.Type lockstep pattern,
	// ADR-0111 contract 21a).
	t.Parallel()

	sid := id.NewStreamID()

	var asRecord record.StreamRef = id.StreamRef{Type: "User", ID: sid}
	if asRecord != (record.StreamRef{Type: id.StreamType("User"), ID: sid}) {
		t.Errorf("id.StreamRef and record.StreamRef diverged: %v", asRecord)
	}

	if got := asRecord.String(); got != "User:"+sid.String() {
		t.Errorf("String() = %q, want %q", got, "User:"+sid.String())
	}

	if got := asRecord.StreamKey(); got != "User:"+sid.Get() {
		t.Errorf("StreamKey() = %q, want %q", got, "User:"+sid.Get())
	}
}

func TestStreamRef_ZeroMeansNotAStreamMember(t *testing.T) {
	t.Parallel()

	var zero record.StreamRef
	if !zero.IsZero() {
		t.Error("the zero StreamRef must report IsZero")
	}

	// The zero ref is NOT a valid stream identity — it states "not a member
	// of any stream" (queries, type-less commands), a different fact from a
	// well-formed ref. Constructing it through the validating door fails.
	if err := zero.Validate(); err == nil {
		t.Error("zero StreamRef must fail Validate (it is non-membership, not identity)")
	}

	rec := record.Record{}
	if !rec.Stream.IsZero() {
		t.Error("a zero Record must carry a zero Stream (not a stream member)")
	}
}

func TestStreamRef_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ref     record.StreamRef
		wantErr bool
	}{
		{"typed ref", mustStreamRef(t, "User", "01JTEST"), false},
		{"empty type is illegal (ADR-0166)", record.StreamRef{ID: mustStreamRef(t, "User", "01JTEST").ID}, true},
		{"zero ID is illegal (ADR-0166)", record.StreamRef{Type: "User"}, true},
		{"zero ref", record.StreamRef{}, true},
	}

	for _, tt := range tests {
		err := tt.ref.Validate()
		if tt.wantErr && err == nil {
			t.Errorf("%s: Validate(%v) = nil, want error", tt.name, tt.ref)
		}

		if !tt.wantErr && err != nil {
			t.Errorf("%s: Validate(%v) = %v, want nil", tt.name, tt.ref, err)
		}
	}
}

func TestNewStreamRefOrZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		streamType string
		entityID   string
		wantZero   bool
	}{
		{"typed ref", "User", "01JTEST", false},
		{"empty stream type yields zero (illegal at construction, ADR-0166)", "", "01JTEST", true},
		{"empty entity ID yields zero", "User", "", true},
		{"both empty yields zero", "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sid, _ := id.ParseStreamID(tc.entityID)
			got := id.NewStreamRefOrZero(id.StreamType(tc.streamType), sid)
			if got.IsZero() != tc.wantZero {
				t.Fatalf(
					"NewStreamRefOrZero(%q, %q) zero=%v, want %v",
					tc.streamType, tc.entityID, got.IsZero(), tc.wantZero,
				)
			}

			if !got.IsZero() && got.Validate() != nil {
				t.Errorf("non-zero result %v must pass Validate, got %v", got, got.Validate())
			}
		})
	}
}

func TestStreamRef_ParseStreamKeyRoundTrip(t *testing.T) {
	t.Parallel()

	ref := mustStreamRef(t, "User", "01JTEST")

	parsed, err := id.ParseStreamRef(ref.StreamKey())
	if err != nil {
		t.Fatalf("ParseStreamRef(StreamKey()) = %v, want nil", err)
	}

	if parsed != ref {
		t.Errorf("ParseStreamRef(%q) = %v, want %v", ref.StreamKey(), parsed, ref)
	}
}

func TestRecord_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := record.Record{
		Type:    "user.created",
		Payload: record.PayloadFrom([]byte(`{"name":"alice"}`), record.EncodingJSON),
		Stream:  mustStreamRef(t, "User", "01JTEST"),
		Version: 1,
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
	if string(decoded.Payload.Data) != string(original.Payload.Data) {
		t.Errorf("Payload.Data = %q, want %q", decoded.Payload.Data, original.Payload.Data)
	}

	if decoded.Payload.Enc != original.Payload.Enc {
		t.Errorf("Payload.Enc = %v, want %v", decoded.Payload.Enc, original.Payload.Enc)
	}
	if decoded.Stream != original.Stream {
		t.Errorf("Stream = %v, want %v", decoded.Stream, original.Stream)
	}
	if got := decoded.Stream.StreamKey(); got != original.Stream.StreamKey() {
		t.Errorf("Stream.StreamKey() = %q, want %q", got, original.Stream.StreamKey())
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
		t.Errorf(
			"Stored round trip must keep the zero (unknown) stamp, got %v",
			decoded.MetaData.Stored,
		)
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
		Type:     "test.event",
		Stream:   mustStreamRef(t, "Test", "01JTEST"),
		Version:  1,
		Encoding: "json",
	}

	if er.Type != "test.event" {
		t.Errorf("embedded Type = %q, want %q", er.Type, "test.event")
	}
	if er.Encoding != "json" {
		t.Errorf("Encoding = %q, want %q", er.Encoding, "json")
	}
}
