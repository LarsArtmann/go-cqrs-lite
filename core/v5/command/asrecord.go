package command

import (
	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

// AsRecord converts a *BasicCommand into a [record.Record], mapping all
// structural fields. This is the command-side counterpart to
// [event.AsRecord], bridging command-driven pipelines into the metaengine's
// Record-aware fold system (ADR-0111, ADR-0112).
//
// Commands differ from events in several ways that affect the mapping:
//   - Commands have no stream identity in this thin form → Stream zero
//     (a BasicCommand carries a stream ID but no stream TYPE; under
//     ADR-0166 an empty-type ref is illegal, so the Record states "not a
//     stream member" honestly instead of fabricating a ref).
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
//   - Payload       ← zero (absent — commands are typed structs, not
//     blobs)
//   - Stream        ← zero — no fabricated identity: the BasicCommand's
//     stream ID has no stream TYPE to pair with, and ADR-0166 makes an
//     empty-type ref illegal; use [AsRecordPersisted] for full stream
//     identity
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
// A nil command — or a zero STRUCT (no ID, no type) — returns a zero-valued
// Record (review R10).
//
// Deprecated: planned for removal at v5. This bridges the thin in-memory
// form only — the Record it returns has no payload and an empty stream type.
// For persisted commands use [AsRecordPersisted], which carries the payload,
// the full stream identity, and the receive stamps. Behavior is unchanged
// until the v5 cut (see the AsRecord asymmetry section of
// docs/reviews/2026-09-13_event-command-duplication-hypothesis-review.md).
func AsRecord(cmd *BasicCommand) record.Record {
	//art-dupl:accept dep-isolated twin of query.AsRecord; lockstep Record population is by design
	// A zero STRUCT is treated exactly like nil (review R10).
	if cmd == nil || cmd.ID().IsZero() || cmd.Type() == "" {
		return record.Record{} //nolint:exhaustruct_v5 // the zero Record IS the verdict
	}

	md := cmd.Metadata()
	tracing := md.Tracing

	var cause record.Cause
	if !tracing.CausationID.IsZero() {
		cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	}

	return record.Record{
		ID:     record.InstanceID(cmd.ID().String()),
		Type:   cmd.Type(),
		Stream: record.StreamRef{},
		MetaData: record.CommonMetadata{
			CorrelationID: record.CorrelationID(metadata.BrandedString(tracing.CorrelationID)),
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
//   - Stream         ← cmd.StreamRef() at full fidelity — both parts of
//     the identity, validated at PersistedCommand construction; where the
//     BasicCommand bridge leaves Stream zero
//   - Received       ← cmd.ReceivedAt() — the honest home for the
//     server-receive clock. Created stays zero: PersistedCommand carries no
//     client clock.
//
// Field mapping:
//
//   - ID              ← cmd.ID() — the command's instance identity
//   - Type            ← cmd.Type()
//   - Payload         ← record.PayloadFrom(cmd.Payload(),
//     EncodingUnknown) — bytes and stamp fused; the stamp stays Unknown
//     because the bridge cannot know whether the bytes are an ADR-0044
//     envelope (producers that do know stamp EncodingEnvelope)
//   - Stream           ← cmd.StreamRef() (validated at PersistedCommand
//     construction — full fidelity, never malformed)
//   - Version          ← 0 (commands have no version)
//   - CorrelationID   ← cmd.Metadata().Tracing.CorrelationID
//   - CausationID     ← cmd.Metadata().Tracing.CausationID (Deprecated: removed in v5)
//   - Cause           ← {CauseUnknown, Tracing.CausationID} when set — the
//     tracing chain does not discriminate the causer's kind, so the Cause
//     states that honestly instead of guessing
//   - ActorID         ← Tracing.ActorID ("kind:raw") when set, else Tracing.UserID
//     (Deprecated: removed in v5)
//   - Actor           ← metadata.RecordActor(tracing): same precedence,
//     resolved structurally (kind explicit, no parse tax)
//   - ClientCreatedAt ← zero (Deprecated field, in lockstep with Created
//     until the cut; PersistedCommand carries NO client clock — the old
//     relabeling of ReceivedAt as a client clock was the R8 lie)
//   - Created         ← zero (no client clock on the command surface)
//   - Received        ← NewStamp(cmd.ReceivedAt()) — the honest home for
//     the server-receive clock
//   - SchemaVersion   ← 0 (commands have no schema version)
//
// A nil command — or a zero STRUCT (no ID, no type) — returns a zero-valued
// Record (review R10).
func AsRecordPersisted(cmd *PersistedCommand) record.Record {
	//art-dupl:accept dep-isolated twin of query.AsRecord; lockstep Record population is by design
	// A zero STRUCT is treated exactly like nil (review R10).
	if cmd == nil || cmd.ID().IsZero() || cmd.Type() == "" {
		return record.Record{} //nolint:exhaustruct_v5 // the zero Record IS the verdict
	}

	md := cmd.Metadata()
	tracing := md.Tracing

	var cause record.Cause
	if !tracing.CausationID.IsZero() {
		cause = record.Cause{Kind: record.CauseUnknown, ID: tracing.CausationID.String()}
	}

	return record.Record{
		ID:      record.InstanceID(cmd.ID().String()),
		Type:    cmd.Type(),
		Payload: record.PayloadFrom(cmd.Payload(), record.EncodingUnknown),
		Stream:  cmd.StreamRef(),
		MetaData: record.CommonMetadata{
			CorrelationID: record.CorrelationID(metadata.BrandedString(tracing.CorrelationID)),
			CausationID:   metadata.BrandedString(tracing.CausationID),
			Cause:         cause,
			ActorID:       metadata.ActorString(tracing),
			Actor:         metadata.RecordActor(tracing),
			Received:      record.NewStamp(cmd.ReceivedAt()),
		},
	}
}
