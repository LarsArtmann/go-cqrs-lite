package record

// ActorKind mirrors id.ActorKind: the discriminator for who or what produced
// a record. The zero value is ActorUnknown, mirroring id.ActorUnknown.
//
// The mirror exists because record/ is zero-dep by design (ADR-0111): it
// cannot import id/, so the union is restated structurally instead of being
// smuggled through "kind:raw" strings (review P3).
type ActorKind uint8

const (
	// ActorUnknown is the zero value: the kind was not set. An Actor with
	// this kind is considered zero.
	ActorUnknown ActorKind = iota

	// ActorUser is a human user (authenticated via WebAuthn, OAuth2, etc.).
	ActorUser

	// ActorBot is an automated bot or API token.
	ActorBot

	// ActorSystem is an internal system process (scheduler, indexer, GC).
	ActorSystem

	// ActorService is a named service producing records on behalf of users.
	ActorService
)

// String returns the lowercase kind name, used as the prefix in Actor.String
// and matching the id.ActorID wire format ("user", "bot", "system",
// "service").
func (k ActorKind) String() string {
	switch k {
	case ActorUser:
		return "user"
	case ActorBot:
		return "bot"
	case ActorSystem:
		return "system"
	case ActorService:
		return "service"
	default:
		return "unknown"
	}
}

// Actor is the structural mirror of id.ActorID — the kind-discriminated
// producer of a record, explicit at the type level. Every consumer of a
// Record gets the kind without paying the "kind:raw" parse tax the stringly
// ActorID field charged (review P3).
//
// The zero value means "no actor". Convert from tracing metadata with
// metadata.RecordActor; the "kind:raw" wire form survives only at the
// serialization edge (Actor.String).
type Actor struct {
	// Kind says what the producer was: user, bot, system, or service.
	Kind ActorKind

	// Raw is the producer's identifier (user ID, bot name, process name).
	// Empty when Kind is ActorUnknown.
	Raw string
}

// IsZero reports whether no actor is recorded.
func (a Actor) IsZero() bool { return a.Kind == ActorUnknown }

// String returns the self-describing "kind:raw" wire form (e.g.
// "user:01JXYZ...", "system:scheduler"), identical to
// id.ActorID.PrefixedString. The zero Actor returns "".
func (a Actor) String() string {
	if a.Kind == ActorUnknown {
		return ""
	}

	return a.Kind.String() + ":" + a.Raw
}
