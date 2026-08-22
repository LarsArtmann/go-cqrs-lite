package query

import (
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// AsRecord converts a *PersistedQuery into a [record.Record], mapping all
// structural fields. This is the query-side counterpart to
// [event.AsRecord] and [command.AsRecord], completing the Record adapter
// set so the metaengine can operate on all three entity types uniformly
// (ADR-0111, WAL unification).
//
// Queries differ from events and commands in several ways that affect the
// mapping:
//   - Queries have no StreamType → left empty.
//   - Queries have no Version → left zero.
//   - Queries have no SchemaVersion → left zero.
//   - Queries carry an id.RequestID (unique per query instance).
//   - Queries DO carry a payload blob (the encoded query parameters),
//     unlike commands whose data lives in typed structs.
//
// Field mapping:
//
//   - ID              ← q.ID() — the query's instance identity, no longer
//     dropped by the bridge (review P5)
//   - Type            ← q.Type()
//   - Payload         ← q.Payload() (cloned, safe to modify)
//   - Encoding        ← "" (the payload is ADR-0044 envelope-wrapped — the
//     envelope carries its own codec stamp)
//   - StreamID        ← record.NewStreamRefOrZero("", q.ID().String())
//     (zero when the query's ID is empty — no identity rather than a
//     malformed identity; the empty stream type is legal by design)
//   - StreamType      ← "" (queries do not carry a stream type)
//   - Version         ← 0 (queries have no version)
//   - CorrelationID   ← q.Metadata().Tracing.CorrelationID
//   - CausationID     ← q.Metadata().Tracing.CausationID (Deprecated: removed in v5)
//   - Cause           ← {CauseUnknown, Tracing.CausationID} when set — the
//     tracing chain does not discriminate the causer's kind, so the Cause
//     states that honestly instead of guessing
//   - ActorID         ← Tracing.ActorID ("kind:raw") when set, else Tracing.UserID
//     (Deprecated: removed in v5)
//   - Actor           ← metadata.RecordActor(tracing): same precedence,
//     resolved structurally (kind explicit, no parse tax)
//   - ClientCreatedAt ← q.ReceivedAt() (Deprecated: removed in v5 — kept in
//     lockstep until the cut; note the source is the server-receive clock)
//   - Received        ← NewStamp(q.ReceivedAt()) — the honest home for the
//     server-receive clock. Created stays zero: PersistedQuery carries no
//     client clock.
//   - SchemaVersion   ← 0 (queries have no schema version)
//
// A nil query returns a zero-valued Record.
func AsRecord(q *PersistedQuery) record.Record {
	//art-dupl:accept dep-isolated twin of command.AsRecord; lockstep Record population is by design
	if q == nil {
		return record.Record{}
	}

	md := q.Metadata()
	tracing := md.Tracing

	var cause record.Cause
	if !tracing.CausationID.IsZero() {
		cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	}

	return record.Record{
		ID:         q.ID().String(),
		Type:       string(q.Type()),
		Payload:    q.Payload(),
		StreamID:   record.NewStreamRefOrZero("", q.ID().String()),
		StreamType: "",
		MetaData: record.CommonMetadata{
			CorrelationID:   metadata.BrandedString(tracing.CorrelationID),
			CausationID:     metadata.BrandedString(tracing.CausationID),
			Cause:           cause,
			ActorID:         metadata.ActorString(tracing),
			Actor:           metadata.RecordActor(tracing),
			ClientCreatedAt: q.ReceivedAt(),
			Received:        record.NewStamp(q.ReceivedAt()),
		},
	}
}
