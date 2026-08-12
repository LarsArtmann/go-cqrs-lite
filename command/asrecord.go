package command

import (
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
//   - Type          ← cmd.Type()
//   - Payload       ← nil (commands are typed structs, not blobs)
//   - StreamID      ← record.NewStreamRef("", cmd.StreamID().String())
//   - StreamType    ← "" (commands do not carry a stream type)
//   - Version       ← 0 (commands have no version)
//   - CorrelationID ← cmd.Metadata().Tracing.CorrelationID
//   - CausationID   ← cmd.Metadata().Tracing.CausationID
//   - ActorID       ← cmd.Metadata().Tracing.UserID (the "who")
//   - SchemaVersion ← 0 (commands have no schema version)
//
// A nil command returns a zero-valued Record.
func AsRecord(cmd *BasicCommand) record.Record {
	if cmd == nil {
		return record.Record{}
	}

	md := cmd.Metadata()
	tracing := md.Tracing

	return record.Record{
		Type:       string(cmd.Type()),
		StreamID:   record.NewStreamRef("", cmd.StreamID().String()),
		StreamType: "",
		MetaData: record.CommonMetadata{
			CorrelationID: brandedString(tracing.CorrelationID),
			CausationID:   brandedString(tracing.CausationID),
			ActorID:       brandedString(tracing.UserID),
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
