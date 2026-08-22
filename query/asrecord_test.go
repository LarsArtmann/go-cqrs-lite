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

	if got.ID != qWithMeta.ID().String() {
		t.Errorf("ID = %q, want request ID %q (identity must survive the bridge)",
			got.ID, qWithMeta.ID().String())
	}

	if got.Encoding != record.EncodingUnknown {
		t.Errorf(
			"Encoding = %v, want EncodingUnknown (envelope-wrapped payload carries its own stamp)",
			got.Encoding,
		)
	}

	if string(got.Payload) != `{"id":"42"}` {
		t.Errorf("Payload = %q, want encoded query", got.Payload)
	}

	if got.StreamID != record.NewStreamRef("", qWithMeta.ID().String()) {
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

	wantCause := record.Cause{Kind: record.CauseUnknown, ID: causationID.String()}
	if got.MetaData.Cause != wantCause {
		t.Errorf("Cause = %+v, want %+v (kind unknown: tracing does not discriminate)",
			got.MetaData.Cause, wantCause)
	}

	if got.MetaData.ActorID != userID.String() {
		t.Errorf("ActorID = %q, want %q", got.MetaData.ActorID, userID)
	}

	if got.MetaData.Actor != (record.Actor{Kind: record.ActorUser, Raw: userID.String()}) {
		t.Errorf("Actor = %+v, want structural user actor", got.MetaData.Actor)
	}

	if !got.MetaData.ClientCreatedAt.Equal(receivedAt) {
		t.Errorf("ClientCreatedAt = %v, want %v", got.MetaData.ClientCreatedAt, receivedAt)
	}

	if got.MetaData.Received.IsZero() || !got.MetaData.Received.Time().Equal(receivedAt) {
		t.Errorf("Received = %v, want known stamp at %v (server-receive clock)",
			got.MetaData.Received, receivedAt)
	}

	if !got.MetaData.Created.IsZero() {
		t.Error("Created must stay unknown for queries (no client clock on PersistedQuery)")
	}

	if got.MetaData.SchemaVersion != 0 {
		t.Errorf("SchemaVersion = %d, want 0", got.MetaData.SchemaVersion)
	}
}

func TestAsRecord_ActorPrecedence(t *testing.T) {
	t.Parallel()

	actor := id.NewBotActor("report-generator")

	q, err := query.New("get_user", query.WithActor(actor))
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	pq, err := query.NewPersistedQuery("get_user", []byte("x"),
		query.WithQueryMetadata(q.Metadata()),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	if got := query.AsRecord(pq).MetaData.ActorID; got != "bot:report-generator" {
		t.Errorf("ActorID = %q, want %q (kind-discriminated actor must win)",
			got, "bot:report-generator")
	}
}

func TestAsRecord_ZeroTracingYieldsEmptyStrings(t *testing.T) {
	t.Parallel()

	q, err := query.NewPersistedQuery("get_user", []byte("x"))
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	got := query.AsRecord(q)

	if got.MetaData.CorrelationID != "" || got.MetaData.CausationID != "" ||
		got.MetaData.ActorID != "" || !got.MetaData.Cause.IsZero() {
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

func TestAsRecord_StreamRefInvariant(t *testing.T) {
	t.Parallel()

	q, err := query.NewPersistedQuery("get_user", []byte(`{"id":"42"}`))
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	got := query.AsRecord(q)
	if err := got.StreamID.Validate(); err != nil {
		t.Fatalf("populated StreamID must pass Validate, got %v (%q)", err, got.StreamID)
	}

	if _, entityID := got.StreamID.Split(); entityID != q.ID().String() {
		t.Errorf(
			"Split entityID = %q, want the query's request ID %q",
			entityID, q.ID().String(),
		)
	}
}
