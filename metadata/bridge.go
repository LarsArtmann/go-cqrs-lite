package metadata

import (
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ToCommonMetadata converts Tracing fields into a record.CommonMetadata.
// The branded string types (id.CorrelationID, etc.) are unwrapped to plain
// strings via their String() method. Fields not present in Tracing
// (timestamps, SchemaVersion) are left zero-valued — callers that need them
// should set them after conversion.
func (t Tracing) ToCommonMetadata() record.CommonMetadata {
	return record.CommonMetadata{
		CorrelationID: t.CorrelationID.String(),
		CausationID:   t.CausationID.String(),
		ActorID:       t.UserID.String(),
		RequestID:     t.RequestID.String(),
	}
}

// FromCommonMetadata returns a Tracing populated from the common metadata fields.
// The plain string fields are wrapped into branded types via Parse.
func FromCommonMetadata(cm record.CommonMetadata) Tracing {
	correlation, _ := id.ParseCorrelationID(cm.CorrelationID)
	causation, _ := id.ParseCausationID(cm.CausationID)
	user, _ := id.ParseUserID(cm.ActorID)
	request, _ := id.ParseRequestID(cm.RequestID)

	return Tracing{
		CorrelationID: correlation,
		CausationID:   causation,
		UserID:        user,
		RequestID:     request,
	}
}
