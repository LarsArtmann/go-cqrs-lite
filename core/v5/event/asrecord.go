package event

import (
	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
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
//   - Payload          ← record.PayloadFrom(evt.Payload(),
//     recordEncoding(evt.Encoding())) — bytes and codec stamp fused into
//     one value: mixed-codec streams stay self-describing, and codecs the
//     record layer does not know stamp EncodingUnknown (opaque) rather
//     than guessing (cloned, safe to modify)
//   - Stream          ← id.NewStreamRefOrZero(evt.StreamType(), evt.StreamID())
//     (zero when the identity is incomplete — no identity rather than a
//     malformed identity, ADR-0166; events validate both parts at
//     construction, so the zero arm covers reconstruct-without-identity)
//   - Version          ← evt.Version()
//   - CorrelationID    ← evt.Metadata().Tracing.CorrelationID
//   - CausationID      ← typed command ID, tracing fallback (Deprecated:
//     removed with the v4 train, T26)
//   - Cause            ← evt.Cause() — the field wins; twin and tracing
//     consulted only when it is empty
//   - ActorID          ← see precedence rule below (Deprecated: removed in v5)
//   - Actor            ← metadata.RecordActor(tracing): the same precedence
//     as ActorID, resolved structurally (kind explicit, no parse tax)
//   - ClientCreatedAt ← zero (Deprecated field, in lockstep with Created
//     until the cut; the event layer carries NO client clock — OccurredAt
//     is the server mint clock, review R8)
//   - Created         ← zero (no client clock exists at the event layer)
//   - Received        ← NewStamp(evt.OccurredAt()) — the mint clock IS the
//     server's intake clock for the fact; the bridge stays deterministic
//     (no clock of its own, replay-safe)
//   - Stored          ← zero (the store stamps it)
//   - SchemaVersion    ← evt.SchemaVersion()
//
// Cause precedence (review R6): the event's first-class cause FIELD wins —
// it was typed at birth by [WithCause] (the CommandCausalityEnricher
// path). When the field is empty, the metadata Causation twin is consulted
// (events born through legacy paths), then Tracing.CausationID as the
// honest unknown-kind fallback. The deprecated CausationID string follows
// the same chain (typed command ID, tracing fallback, empty).
//
// ActorID precedence: the kind-discriminated Tracing.ActorID wins when set,
// serialized in its self-describing "kind:raw" form (e.g. "user:01ARZ...").
// When no actor was recorded, Tracing.UserID is used as the legacy fallback.
//
// A nil Event — or a zero STRUCT (no ID, no type) — returns a zero-valued
// Record (review R10): identity-less sources are not records.
func AsRecord(evt Event) record.Record {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	// A zero STRUCT is treated exactly like nil (review R10): an event
	// without identity or type is not a record, and minting one would
	// fabricate a Record the source never carried.
	if evt == nil || evt.ID().IsZero() || evt.Type() == "" {
		return record.Record{} //nolint:exhaustruct_v5 // the zero Record IS the verdict
	}

	md := evt.Metadata()
	tracing := md.Tracing

	causationID := metadata.BrandedString(tracing.CausationID)

	if md.Causation != nil && !md.Causation.CommandID.IsZero() {
		causationID = md.Causation.CommandID.String()
	}

	cause := evt.Cause()
	if cause.IsZero() {
		if md.Causation != nil && !md.Causation.CommandID.IsZero() {
			cause = record.Cause{Kind: record.CauseCommand, ID: md.Causation.CommandID.String()}
		} else if !tracing.CausationID.IsZero() {
			cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
		}
	}

	return record.Record{
		ID:      record.InstanceID(evt.ID().String()),
		Type:    evt.Type(),
		Payload: record.PayloadFrom(evt.Payload(), recordEncoding(evt.Encoding())),
		Stream:  id.NewStreamRefOrZero(evt.StreamType(), evt.StreamID()),
		Version: record.Version(evt.Version().UInt64()),
		MetaData: record.CommonMetadata{
			CorrelationID: record.CorrelationID(metadata.BrandedString(tracing.CorrelationID)),
			CausationID:   causationID,
			Cause:         cause,
			ActorID:       metadata.ActorString(tracing),
			Actor:         metadata.RecordActor(tracing),
			Received:      record.NewStamp(evt.OccurredAt()),
			SchemaVersion: int(evt.SchemaVersion()),
		},
	}
}

// Compile-time: verify the branded ID types satisfy the constraint.
var _ = metadata.BrandedString[id.CorrelationID]

// recordEncoding maps the event's codec stamp onto the record's compact
// Encoding. Codecs the record layer does not know stamp EncodingUnknown:
// the payload is still carried, just not self-describing through the record.
func recordEncoding(enc codec.Encoding) record.Encoding {
	parsed, err := record.ParseEncoding(string(enc))
	if err != nil {
		return record.EncodingUnknown
	}

	return parsed
}
