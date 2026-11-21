// Package core_test holds the cross-module contract tests: invariants
// that must hold across the event, command, and query twins at once (the
// "trinity" — one Record shape, ADR-0111).
package core_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/v5/command"
	"github.com/larsartmann/go-cqrs-lite/core/v5/event"
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
	"github.com/larsartmann/go-cqrs-lite/core/v5/query"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

// trinityFixture builds one event, one persisted command, and one persisted
// query sharing the same tracing metadata, so the contract test can assert
// the bridges resolve metadata IDENTICALLY.
func trinityFixture(t *testing.T) (event.Event, *command.PersistedCommand, *query.PersistedQuery, metadata.Tracing) {
	t.Helper()

	tracing := metadata.Tracing{
		CorrelationID: id.NewCorrelationID(),
		CausationID:   id.NewCausationID(),
		Actor:         record.Actor{Kind: record.ActorService, Raw: "api-gateway"},
	}

	streamID := id.NewStreamID()
	streamRef, err := id.NewStreamRef("User", streamID)
	if err != nil {
		t.Fatalf("NewStreamRef: %v", err)
	}

	evtMeta := event.NewMetadata()
	evtMeta.Tracing = tracing
	evt, err := event.NewEvent(
		"user.created", streamID, "User", event.Version(3), []byte(`{"n":"1"}`),
		event.WithMetadata(evtMeta))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	receivedAt := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

	cmdMeta := command.Metadata{
		Tracing: tracing,
	}
	cmd, err := command.NewPersistedCommand(
		"create_user",
		streamRef,
		[]byte(`{"name":"alice"}`),
		command.WithReceivedAt(receivedAt),
		command.WithCommandMetadata(cmdMeta),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	qMeta := query.Metadata{
		Tracing: tracing,
	}
	q, err := query.NewPersistedQuery(
		"get_user",
		[]byte(`{"id":"42"}`),
		query.WithQueryReceivedAt(receivedAt),
		query.WithQueryMetadata(qMeta),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	return evt, cmd, q, tracing
}

// TestTrinityContract_AsRecord pins the ONE Record shape across the three
// twins (M09): identity and type always survive; Stream membership follows
// the entity (event/command persisted carry it, queries never do); Version
// is a fact only events have; and identical tracing resolves to identical
// CorrelationID, Cause, and Actor on every bridge.
func TestTrinityContract_AsRecord(t *testing.T) {
	t.Parallel()

	evt, cmd, q, tracing := trinityFixture(t)

	records := map[string]record.Record{
		"event":            event.AsRecord(evt),
		"commandPersisted": command.AsRecordPersisted(cmd),
		"query":            query.AsRecord(q),
	}

	for name, rec := range records {
		if rec.ID == "" {
			t.Errorf("%s: instance identity must survive the bridge", name)
		}

		if rec.Type == "" {
			t.Errorf("%s: type must survive the bridge", name)
		}

		if rec.MetaData.CorrelationID != record.CorrelationID(tracing.CorrelationID.String()) {
			t.Errorf("%s: CorrelationID = %q, want the shared tracing value", name, rec.MetaData.CorrelationID)
		}

		wantCause := record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
		if rec.MetaData.Cause != wantCause {
			t.Errorf("%s: Cause = %v, want the shared tracing fallback %v", name, rec.MetaData.Cause, wantCause)
		}

		wantActor := tracing.Actor
		if rec.MetaData.Actor != wantActor {
			t.Errorf("%s: Actor = %v, want the structural field %v", name, rec.MetaData.Actor, wantActor)
		}
	}

	if got := records["event"].Stream; got.Validate() != nil {
		t.Errorf("event Stream = %v, must be a valid member ref", got)
	}

	if got := records["commandPersisted"].Stream; got.Validate() != nil || got != cmd.StreamRef() {
		t.Errorf("persisted-command Stream = %v, must carry the full ref %v", got, cmd.StreamRef())
	}

	if !records["query"].Stream.IsZero() {
		t.Errorf("query Stream = %v, must be zero (a query is not a stream member, R9)", records["query"].Stream)
	}

	if records["event"].Version != 3 {
		t.Errorf("event Version = %d, want the stream position", records["event"].Version)
	}

	if records["commandPersisted"].Version != 0 || records["query"].Version != 0 {
		t.Error("commands and queries are unversioned — Version must be 0")
	}

	if records["event"].Payload.Enc != record.EncodingJSON {
		t.Errorf("event payload must be self-describing, got %v", records["event"].Payload.Enc)
	}

	if records["commandPersisted"].Payload.IsZero() || records["query"].Payload.IsZero() {
		t.Error("persisted commands and queries carry payload bytes")
	}

	if records["commandPersisted"].Payload.Enc != record.EncodingUnknown ||
		records["query"].Payload.Enc != record.EncodingUnknown {
		t.Error("command/query payloads are opaque at the record layer (bridge cannot know the envelope)")
	}

	assertTrinityStamps(t, records, evt, cmd, q)
}

// assertTrinityStamps pins the M06 stamp contract: every bridge stamps a
// KNOWN Received; Created stays zero when no client clock exists; Stored is
// never bridge-stamped; and each twin's Received time equals its source
// clock (command/query received-at, event mint).
func assertTrinityStamps(
	t *testing.T,
	records map[string]record.Record,
	evt event.Event,
	cmd *command.PersistedCommand,
	q *query.PersistedQuery,
) {
	t.Helper()

	for name, rec := range records {
		if rec.MetaData.Received.IsZero() {
			t.Errorf("%s: Received stamp must be known after the bridge (M06: every bridge stamps intake)", name)
		}

		if !rec.MetaData.Created.IsZero() {
			t.Errorf("%s: Created = %v, want zero — the fixture records no client clock", name, rec.MetaData.Created)
		}

		if !rec.MetaData.Stored.IsZero() {
			t.Errorf(
				"%s: Stored = %v, want zero — only the store stamps Stored, never the bridge",
				name,
				rec.MetaData.Stored,
			)
		}
	}

	if got := records["commandPersisted"].MetaData.Received.Time(); !got.Equal(cmd.ReceivedAt()) {
		t.Errorf("persisted-command Received = %v, want the explicit received clock %v", got, cmd.ReceivedAt())
	}

	if got := records["query"].MetaData.Received.Time(); !got.Equal(q.ReceivedAt()) {
		t.Errorf("query Received = %v, want the explicit received clock %v", got, q.ReceivedAt())
	}

	if got := records["event"].MetaData.Received.Time(); !got.Equal(evt.OccurredAt()) {
		t.Errorf(
			"event Received = %v, want the mint clock %v (the event bridge's honest home for intake)",
			got,
			evt.OccurredAt(),
		)
	}
}
