package metadata

import (
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// RecordActor resolves a Tracing's producer into the structural
// record.Actor: the kind-discriminated ActorID when set, falling back to the
// legacy UserID as an ActorUser (a user ID is by definition a human user).
//
// It is the structural counterpart to [ActorString]: the "kind:raw" wire
// form survives only at the serialization edge, so Record consumers get the
// kind without paying the parse tax (review P3).
func RecordActor(tracing Tracing) record.Actor {
	if !tracing.ActorID.IsZero() {
		return record.Actor{
			Kind: recordActorKind(tracing.ActorID.Kind()),
			Raw:  tracing.ActorID.Raw(),
		}
	}

	if !tracing.UserID.IsZero() {
		return record.Actor{Kind: record.ActorUser, Raw: tracing.UserID.String()}
	}

	return record.Actor{}
}

// recordActorKind maps the id-layer actor kind onto its record-layer mirror.
// The two enums are declared independently (record/ is zero-dep); this map
// keeps them aligned at the single conversion point.
func recordActorKind(kind id.ActorKind) record.ActorKind {
	switch kind {
	case id.ActorUser:
		return record.ActorUser
	case id.ActorBot:
		return record.ActorBot
	case id.ActorSystem:
		return record.ActorSystem
	case id.ActorService:
		return record.ActorService
	default:
		return record.ActorUnknown
	}
}
