package query

import (
	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

// AsRecord converts a *PersistedQuery into a [record.Record], mapping all
// structural fields. This is the query-side counterpart to
// [event.AsRecord] and [command.AsRecord], completing the Record adapter
// set so the metaengine can operate on all three entity types uniformly
// (ADR-0111, WAL unification).
//
// Queries differ from events and commands in several ways that affect the
// mapping:
//   - Queries belong to no stream → Stream zero (a query is not a stream
//     member; the previous fabricated empty-type ref from the query's own
//     ID is gone — review R9, ADR-0166).
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
//   - Payload         ← record.PayloadFrom(q.Payload(),
//     EncodingUnknown) — bytes and stamp fused (cloned, safe to modify);
//     the stamp stays Unknown because the bridge cannot know whether the
//     bytes are an ADR-0044 envelope (producers that do know stamp
//     EncodingEnvelope)
//   - Stream           ← zero — a query is not a stream member (R9); no
//     fabricated ref from the query's own ID
//   - Version          ← 0 (queries have no version)
//   - CorrelationID   ← q.Metadata().Tracing.CorrelationID
//   - CausationID     ← q.Metadata().Tracing.CausationID (Deprecated: removed in v5)
//   - Cause           ← {CauseUnknown, Tracing.CausationID} when set — the
//     tracing chain does not discriminate the causer's kind, so the Cause
//     states that honestly instead of guessing
//   - ActorID         ← Tracing.ActorID ("kind:raw") when set, else Tracing.UserID
//     (Deprecated: removed in v5)
//   - Actor           ← metadata.RecordActor(tracing): same precedence,
//     resolved structurally (kind explicit, no parse tax)
//   - ClientCreatedAt ← zero (Deprecated field, in lockstep with Created
//     until the cut; PersistedQuery carries NO client clock — the old
//     relabeling of ReceivedAt as a client clock was the R8 lie)
//   - Created/Received ← Created zero (no client clock); Received
//     ← NewStamp(q.ReceivedAt()) — the honest home for the server-receive
//     clock.
//   - SchemaVersion   ← 0 (queries have no schema version)
//
// A nil query — or a zero STRUCT (no ID, no type) — returns a zero-valued
// Record (review R10).
func AsRecord(q *PersistedQuery) record.Record {
	//art-dupl:accept dep-isolated twin of command.AsRecord; lockstep Record population is by design
	// A zero STRUCT is treated exactly like nil (review R10).
	if q == nil || q.ID().IsZero() || q.Type() == "" {
		return record.Record{} //nolint:exhaustruct_v5 // the zero Record IS the verdict
	}

	md := q.Metadata()
	tracing := md.Tracing

	var cause record.Cause
	if !tracing.CausationID.IsZero() {
		cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	}

	return record.Record{
		ID:      record.InstanceID(q.ID().String()),
		Type:    q.Type(),
		Payload: record.PayloadFrom(q.Payload(), record.EncodingUnknown),
		Stream:  record.StreamRef{},
		MetaData: record.CommonMetadata{
			CorrelationID: record.CorrelationID(metadata.BrandedString(tracing.CorrelationID)),
			CausationID:   metadata.BrandedString(tracing.CausationID),
			Cause:         cause,
			ActorID:       metadata.ActorString(tracing),
			Actor:         metadata.RecordActor(tracing),
			Received:      record.NewStamp(q.ReceivedAt()),
		},
	}
}
