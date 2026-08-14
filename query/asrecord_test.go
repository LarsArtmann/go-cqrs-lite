package query_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestAsRecord_MapsStructuralFields(t *testing.T) {
	t.Parallel()

	correlationID := id.NewCorrelationID()
	causationID := id.NewCausationID()
	userID := id.NewUserID()
	receivedAt := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	q, err := query.NewPersistedQuery(
		"get_user",
		[]byte(`{"id":"42"}`),
		query.WithQueryReceivedAt(receivedAt),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	md := query.Metadata{}
	md.CorrelationID = correlationID
	md.CausationID = causationID
	md.UserID = userID
	qWithMeta, err := query.NewPersistedQuery(
		"get_user",
		[]byte(`{"id":"42"}`),
		query.WithQueryReceivedAt(receivedAt),
		query.WithQueryMetadata(md),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery with metadata: %v", err)
	}

	got := query.AsRecord(qWithMeta)

	if got.Type != "get_user" {
		t.Errorf("Type = %q, want get_user", got.Type)
	}

	if string(got.Payload) != `{"id":"42"}` {
		t.Errorf("Payload = %q, want encoded query", got.Payload)
	}

	if got.StreamID != record.NewStreamRef("", q.ID().String()) {
		t.Errorf("StreamID = %q, want request-ID based ref", got.StreamID)
	}

	if got.StreamType != "" {
		t.Errorf("StreamType = %q, want empty", got.StreamType)
	}

	if got.Version != 0 {
		t.Errorf("Version = %d, want 0", got.Version)
	}

	if got.MetaData.CorrelationID != correlationID.String() {
		t.Errorf("CorrelationID = %q, want %q", got.MetaData.CorrelationID, correlationID)
	}

	if got.MetaData.CausationID != causationID.String() {
		t.Errorf("CausationID = %q, want %q", got.MetaData.CausationID, causationID)
	}

	if got.MetaData.ActorID != userID.String() {
		t.Errorf("ActorID = %q, want %q", got.MetaData.ActorID, userID)
	}

	if !got.MetaData.ClientCreatedAt.Equal(receivedAt) {
		t.Errorf("ClientCreatedAt = %v, want %v", got.MetaData.ClientCreatedAt, receivedAt)
	}

	if got.MetaData.SchemaVersion != 0 {
		t.Errorf("SchemaVersion = %d, want 0", got.MetaData.SchemaVersion)
	}
}

func TestAsRecord_ZeroTracingYieldsEmptyStrings(t *testing.T) {
	t.Parallel()

	q, err := query.NewPersistedQuery("get_user", []byte("x"))
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	got := query.AsRecord(q)

	if got.MetaData.CorrelationID != "" || got.MetaData.CausationID != "" || got.MetaData.ActorID != "" {
		t.Errorf("zero tracing must map to empty strings, got %+v", got.MetaData)
	}
}

func TestAsRecord_NilQuery(t *testing.T) {
	t.Parallel()

	got := query.AsRecord(nil)
	if got.Type != "" || got.Payload != nil || got.StreamID != "" || got.Version != 0 {
		t.Errorf("AsRecord(nil) = %+v, want zero Record", got)
	}
}
