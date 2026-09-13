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
//   - Encoding      ← EncodingUnknown (no payload to stamp)
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
//
// Deprecated: planned for removal at v5. This bridges the thin in-memory
// form only — the Record it returns has no payload and an empty stream type.
// For persisted commands use [AsRecordPersisted], which carries the payload,
// the full stream identity, and the receive stamps. Behavior is unchanged
// until the v5 cut (see the AsRecord asymmetry section of
// docs/reviews/2026-09-13_event-command-duplication-hypothesis-review.md).
func AsRecord(cmd *BasicCommand) record.Record {
	//art-dupl:accept dep-isolated twin of query.AsRecord; lockstep Record population is by design
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

// AsRecordPersisted converts a *PersistedCommand into a [record.Record],
// mapping all structural fields. This is the command-side counterpart to
// [event.AsRecord] and [query.AsRecord] for the PERSISTED command form,
// completing the Record adapter set so the metaengine can operate on all
// three entity types uniformly (ADR-0111, WAL unification). It is also the
// concrete bridge a future ADR-0112 command-sourcing layer builds on.
//
// AsRecordPersisted bridges the FULL form, unlike [AsRecord] on *BasicCommand:
//   - Payload        ← cmd.Payload() (cloned, safe to modify)
//   - StreamID/Type  ← cmd.StreamRef() at full fidelity — the persisted
//     stream type is carried, where the BasicCommand bridge leaves it empty
//   - Received       ← cmd.ReceivedAt() — the honest home for the
//     server-receive clock. Created stays zero: PersistedCommand carries no
//     client clock.
//
// Field mapping:
//
//   - ID              ← cmd.ID() — the command's instance identity
//   - Type            ← cmd.Type()
//   - Payload         ← cmd.Payload() (commands persisted through the
//     ADR-0044 envelope are self-describing; the envelope carries its own
//     codec stamp)
//   - Encoding        ← EncodingUnknown (no codec stamp on the command
//     itself — same rule as the query bridge)
//   - StreamID        ← record.NewStreamRefOrZero(streamType, streamID)
//     (zero when the stream ID is empty — no identity rather than a
//     malformed identity)
//   - StreamType      ← cmd.StreamType() (populated, unlike the
//     BasicCommand bridge)
//   - Version         ← 0 (commands have no version)
//   - CorrelationID   ← cmd.Metadata().Tracing.CorrelationID
//   - CausationID     ← cmd.Metadata().Tracing.CausationID (Deprecated: removed in v5)
//   - Cause           ← {CauseUnknown, Tracing.CausationID} when set — the
//     tracing chain does not discriminate the causer's kind, so the Cause
//     states that honestly instead of guessing
//   - ActorID         ← Tracing.ActorID ("kind:raw") when set, else Tracing.UserID
//     (Deprecated: removed in v5)
//   - Actor           ← metadata.RecordActor(tracing): same precedence,
//     resolved structurally (kind explicit, no parse tax)
//   - ClientCreatedAt ← cmd.ReceivedAt() (Deprecated: removed in v5 — kept in
//     lockstep with Received until the cut; note the source is the
//     server-receive clock)
//   - Received        ← NewStamp(cmd.ReceivedAt())
//   - SchemaVersion   ← 0 (commands have no schema version)
//
// A nil command returns a zero-valued Record.
func AsRecordPersisted(cmd *PersistedCommand) record.Record {
	//art-dupl:accept dep-isolated twin of query.AsRecord; lockstep Record population is by design
	if cmd == nil {
		return record.Record{}
	}

	md := cmd.Metadata()
	tracing := md.Tracing
	streamType := string(cmd.StreamType())

	var cause record.Cause
	if !tracing.CausationID.IsZero() {
		cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	}

	return record.Record{
		ID:         cmd.ID().String(),
		Type:       string(cmd.Type()),
		Payload:    cmd.Payload(),
		Encoding:   record.EncodingUnknown,
		StreamID:   record.NewStreamRefOrZero(streamType, cmd.StreamID().String()),
		StreamType: streamType,
		MetaData: record.CommonMetadata{
			CorrelationID:   metadata.BrandedString(tracing.CorrelationID),
			CausationID:     metadata.BrandedString(tracing.CausationID),
			Cause:           cause,
			ActorID:         metadata.ActorString(tracing),
			Actor:           metadata.RecordActor(tracing),
			ClientCreatedAt: cmd.ReceivedAt(),
			Received:        record.NewStamp(cmd.ReceivedAt()),
		},
	}
}
