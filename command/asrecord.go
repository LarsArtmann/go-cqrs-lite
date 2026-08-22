package command

import (
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// AsRecord converts a *BasicCommand into a [record.Record], mapping all
// structural fields. This is the command-side counterpart to
// [event.AsRecord], bridging command-driven pipelines into the metaengine's
// Record-aware fold system (ADR-0111, ADR-0112).
//
// Commands differ from events in several ways that affect the mapping:
//   - Commands have no StreamType → left empty.
//   - Commands have no Version → left zero.
//   - Commands have no SchemaVersion → left zero.
//   - Commands carry an id.CommandID (unique per command instance).
//   - Commands do NOT carry a payload in the Record sense — the Payload field
//     is left nil. Command data lives in the concrete command struct, not in
//     a serialized blob.
//
// Field mapping:
//
//   - ID             ← cmd.ID() — the command's instance identity, no
//     longer dropped by the bridge (review P5)
//   - Type          ← cmd.Type()
//   - Payload       ← nil (commands are typed structs, not blobs)
//   - Encoding      ← "" (no payload to stamp)
//   - StreamID      ← record.NewStreamRefOrZero("", cmd.StreamID().String())
//     (zero when the command's stream ID is empty — no identity rather than
//     a malformed identity; the empty stream type is legal by design)
//   - StreamType    ← "" (commands do not carry a stream type)
//   - Version       ← 0 (commands have no version)
//   - CorrelationID ← cmd.Metadata().Tracing.CorrelationID
//   - CausationID   ← cmd.Metadata().Tracing.CausationID (Deprecated: removed in v5)
//   - Cause         ← {CauseUnknown, Tracing.CausationID} when set — the
//     tracing chain does not discriminate the causer's kind, so the Cause
//     states that honestly instead of guessing
//   - ActorID       ← Tracing.ActorID ("kind:raw") when set, else Tracing.UserID
//     (Deprecated: removed in v5)
//   - Actor         ← metadata.RecordActor(tracing): same precedence,
//     resolved structurally (kind explicit, no parse tax)
//   - SchemaVersion ← 0 (commands have no schema version)
//
// A nil command returns a zero-valued Record.
func AsRecord(cmd *BasicCommand) record.Record {
	if cmd == nil {
		return record.Record{}
	}

	md := cmd.Metadata()
	tracing := md.Tracing

	var cause record.Cause
	if !tracing.CausationID.IsZero() {
		cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	}

	return record.Record{
		ID:         cmd.ID().String(),
		Type:       string(cmd.Type()),
		StreamID:   record.NewStreamRefOrZero("", cmd.StreamID().String()),
		StreamType: "",
		MetaData: record.CommonMetadata{
			CorrelationID: metadata.BrandedString(tracing.CorrelationID),
			CausationID:   metadata.BrandedString(tracing.CausationID),
			Cause:         cause,
			ActorID:       metadata.ActorString(tracing),
			Actor:         metadata.RecordActor(tracing),
		},
	}
}
