package event

import (
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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
//   - Type             ← evt.Type()
//   - Payload          ← evt.Payload() (cloned, safe to modify)
//   - StreamID         ← record.NewStreamRef(streamType, streamID)
//   - StreamType       ← evt.StreamType()
//   - Version          ← evt.Version()
//   - CorrelationID    ← evt.Metadata().Tracing.CorrelationID
//   - CausationID      ← see precedence rule below
//   - ActorID          ← evt.Metadata().Tracing.UserID (the "who")
//   - ClientCreatedAt  ← evt.OccurredAt() (best available creation timestamp)
//   - ServerReceivedAt ← zero (unknown at the event layer; set by the store)
//   - ServerStoredAt   ← zero (unknown at the event layer; set by the store)
//   - SchemaVersion    ← evt.SchemaVersion()
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
// A nil Event returns a zero-valued Record.
func AsRecord(evt Event) record.Record {
	if evt == nil {
		return record.Record{}
	}

	md := evt.Metadata()
	tracing := md.Tracing

	causationID := brandedString(tracing.CausationID)
	if md.Causation != nil && !md.Causation.CommandID.IsZero() {
		causationID = md.Causation.CommandID.String()
	}

	streamType := string(evt.StreamType())

	return record.Record{
		Type:       string(evt.Type()),
		Payload:    evt.Payload(),
		StreamID:   record.NewStreamRef(streamType, evt.StreamID().String()),
		StreamType: streamType,
		Version:    int64(evt.Version()),
		MetaData: record.CommonMetadata{
			CorrelationID:   brandedString(tracing.CorrelationID),
			CausationID:     causationID,
			ActorID:         brandedString(tracing.UserID),
			ClientCreatedAt: evt.OccurredAt(),
			SchemaVersion:   int(evt.SchemaVersion()),
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

// Compile-time: verify the branded ID types satisfy the constraint.
var _ = brandedString[id.CorrelationID]
