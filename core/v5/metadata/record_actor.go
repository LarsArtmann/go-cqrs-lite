package metadata

import (
	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

// RecordActor resolves a Tracing's producer into the structural
// record.Actor. THE precedence rule (review P3/R7):
//
//  1. The structural Actor FIELD wins — it was set structurally, no parse.
//  2. Otherwise the stringly ActorID twin is parsed at this edge (its kind
//     was already discriminated at the id layer).
//  3. Otherwise the legacy UserID is an ActorUser (a user ID is by
//     definition a human user).
//
// The "kind:raw" wire form survives only at the serialization edge; Record
// consumers get the kind without paying the parse tax. [ActorString] is the
// deprecated wire-form view of the same chain (twin, then UserID).
func RecordActor(tracing Tracing) record.Actor {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	if !tracing.Actor.IsZero() {
		return tracing.Actor
	}

	if !tracing.ActorID.IsZero() {
		return record.Actor{
			Kind: recordActorKind(tracing.ActorID.Kind()),
			Raw:  tracing.ActorID.Raw(),
		}
	}

	if !tracing.UserID.IsZero() {
		return record.Actor{Kind: record.ActorUser, Raw: tracing.UserID.String()}
	}

	return record.Actor{Kind: record.ActorUnknown, Raw: ""}
}

// recordActorKind maps the id-layer actor kind onto its record-layer mirror.
// The two enums are declared independently (record/ is zero-dep); this map
// keeps them aligned at the single conversion point.
func recordActorKind(kind id.ActorKind) record.ActorKind {
	switch kind {
	case id.ActorUnknown:
		return record.ActorUnknown
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
