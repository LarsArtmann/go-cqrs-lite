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
//   - Type            ← q.Type()
//   - Payload         ← q.Payload() (cloned, safe to modify)
//   - StreamID        ← record.NewStreamRef("", q.ID().String())
//   - StreamType      ← "" (queries do not carry a stream type)
//   - Version         ← 0 (queries have no version)
//   - CorrelationID   ← q.Metadata().Tracing.CorrelationID
//   - CausationID     ← q.Metadata().Tracing.CausationID
//   - ActorID         ← Tracing.ActorID ("kind:raw") when set, else Tracing.UserID
//   - ClientCreatedAt ← q.ReceivedAt()
//   - SchemaVersion   ← 0 (queries have no schema version)
//
// A nil query returns a zero-valued Record.
func AsRecord(q *PersistedQuery) record.Record {
	if q == nil {
		return record.Record{}
	}

	md := q.Metadata()
	tracing := md.Tracing

	return record.Record{
		Type:       string(q.Type()),
		Payload:    q.Payload(),
		StreamID:   record.NewStreamRef("", q.ID().String()),
		StreamType: "",
		MetaData: record.CommonMetadata{
			CorrelationID:   brandedString(tracing.CorrelationID),
			CausationID:     brandedString(tracing.CausationID),
			ActorID:         actorString(tracing),
			ClientCreatedAt: q.ReceivedAt(),
		},
	}
}

// brandedString returns the string form of a branded ID, or "" if it is zero.
// Prevents zero-value ULIDs from leaking as "0000..." into Record metadata.
func brandedString[T interface {
	String() string
	IsZero() bool
}](v T) string {
	if v.IsZero() {
		return ""
	}

	return v.String()
}

// actorString resolves the Record's ActorID: the kind-discriminated
// Tracing.ActorID in its self-describing "kind:raw" form when set, falling
// back to the bare Tracing.UserID for records that predate ActorID.
func actorString(tracing metadata.Tracing) string {
	if !tracing.ActorID.IsZero() {
		return tracing.ActorID.PrefixedString()
	}

	return brandedString(tracing.UserID)
}
