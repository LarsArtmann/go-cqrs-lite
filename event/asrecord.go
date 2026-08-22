package event

import (
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// AsRecord converts an Event into a [record.Record], mapping all structural
// fields. This is the adapter that bridges the event-sourcing pipeline into the
// metaengine's Record-aware fold system (ADR-0111, ADR-0112).
//
// Record-aware folds (created via metaengine.OnRecord) receive the full Record
// — StreamID, Version, MetaData — alongside the decoded payload. Without this
// adapter, OnRecord folds get zero-valued Records through the real pipeline.
//
// Field mapping:
//
//   - ID              ← evt.ID() — the event's instance identity, no longer
//     dropped by the bridge (review P5)
//   - Type             ← evt.Type()
//   - Encoding        ← evt.Encoding() — the codec stamp ("json"/"cbor")
//     survives the bridge, so mixed-codec streams stay self-describing
//   - Payload          ← evt.Payload() (cloned, safe to modify)
//   - StreamID         ← record.NewStreamRefOrZero(streamType, streamID)
//     (zero when the event's stream ID is empty — no identity rather than a
//     malformed identity; the populated form always passes Validate)
//   - StreamType       ← evt.StreamType()
//   - Version          ← evt.Version()
//   - CorrelationID    ← evt.Metadata().Tracing.CorrelationID
//   - CausationID      ← see precedence rule below (Deprecated: removed in v5)
//   - Cause            ← see precedence rule below
//   - ActorID          ← see precedence rule below (Deprecated: removed in v5)
//   - Actor            ← metadata.RecordActor(tracing): the same precedence
//     as ActorID, resolved structurally (kind explicit, no parse tax)
//   - ClientCreatedAt  ← evt.OccurredAt() (Deprecated: removed in v5 —
//     populated in lockstep with Created until the cut)
//   - Created          ← NewStamp(evt.OccurredAt()) — same source, explicit
//     presence
//   - Received/Stored  ← zero Stamps (unknown at the event layer; stamped by
//     the store)
//   - SchemaVersion    ← evt.SchemaVersion()
//
// Cause precedence — the same resolution order as CausationID, but with the
// causer's kind stated explicitly:
//  1. If evt.Metadata().Causation is non-nil and Causation.CommandID is
//     non-zero, Cause is {CauseCommand, commandID}. This is the strongest
//     signal — the causation was typed at the source.
//  2. Otherwise, when Tracing.CausationID is set, Cause is {CauseUnknown,
//     causationID}: the ID survives, and the kind is honestly "unknown"
//     because the tracing chain does not discriminate it.
//  3. If neither is set, Cause is zero (no cause recorded).
//
// CausationID precedence: the Record's CausationID is resolved with the
// following priority:
//  1. If evt.Metadata().Causation is non-nil and Causation.CommandID is
//     non-zero, the typed command ID wins. This is the strongest signal —
//     it means the event was produced by a specific command whose ID is known.
//  2. Otherwise, Tracing.CausationID is used. This is the generic tracing-level
//     causation chain, set by middleware.
//  3. If both are zero, CausationID is empty.
//
// ActorID precedence: the kind-discriminated Tracing.ActorID wins when set,
// serialized in its self-describing "kind:raw" form (e.g. "user:01ARZ...").
// When no actor was recorded, Tracing.UserID is used as the legacy fallback.
//
// A nil Event returns a zero-valued Record.
func AsRecord(evt Event) record.Record {
	if evt == nil {
		return record.Record{}
	}

	md := evt.Metadata()
	tracing := md.Tracing

	causationID := metadata.BrandedString(tracing.CausationID)
	cause := record.Cause{}
	if md.Causation != nil && !md.Causation.CommandID.IsZero() {
		causationID = md.Causation.CommandID.String()
		cause = record.Cause{Kind: record.CauseCommand, ID: md.Causation.CommandID.String()}
	} else if !tracing.CausationID.IsZero() {
		cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	}

	streamType := string(evt.StreamType())

	return record.Record{
		ID:         evt.ID().String(),
		Type:       string(evt.Type()),
		Encoding:   string(evt.Encoding()),
		Payload:    evt.Payload(),
		StreamID:   record.NewStreamRefOrZero(streamType, evt.StreamID().String()),
		StreamType: streamType,
		Version:    int64(evt.Version()),
		MetaData: record.CommonMetadata{
			CorrelationID:   metadata.BrandedString(tracing.CorrelationID),
			CausationID:     causationID,
			Cause:           cause,
			ActorID:         metadata.ActorString(tracing),
			Actor:           metadata.RecordActor(tracing),
			ClientCreatedAt: evt.OccurredAt(),
			Created:         record.NewStamp(evt.OccurredAt()),
			SchemaVersion:   int(evt.SchemaVersion()),
		},
	}
}

// Compile-time: verify the branded ID types satisfy the constraint.
var _ = metadata.BrandedString[id.CorrelationID]
