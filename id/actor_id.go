package id

import (
	"fmt"
	"strings"
)

// ActorKind discriminates between human and machine actors that produce
// records (events, commands). The zero value is ActorUnknown.
type ActorKind uint8

// Actor kind string representations used in the wire format "kind:raw".
const (
	kindUserStr    = "user"
	kindBotStr     = "bot"
	kindSystemStr  = "system"
	kindServiceStr = "service"
	kindUnknownStr = "unknown"
)

const (
	// ActorUnknown is the zero value, meaning the kind was not set.
	// An ActorID with this kind and an empty raw value is considered zero.
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

// String returns the lowercase name of the kind, used as the prefix in
// ActorID.PrefixedString (e.g. "user", "bot", "system", "service").
func (k ActorKind) String() string {
	switch k {
	case ActorUser:
		return kindUserStr
	case ActorBot:
		return kindBotStr
	case ActorSystem:
		return kindSystemStr
	case ActorService:
		return kindServiceStr
	default:
		return kindUnknownStr
	}
}

// ActorID is a kind-discriminated identifier for any actor that produces a
// record. It unifies users, bots, system processes, and services under one
// type, enabling audit trails and trust-level decisions without separate
// fields or string conventions.
//
// The wire format is "kind:raw" (e.g. "user:01JXYZ...", "system:scheduler",
// "service:api-gateway"). This is self-describing: the kind can be recovered
// from the serialized form via ParseActorID.
//
// The zero value (ActorUnknown, "") represents "no actor" and IsZero returns
// true. Construct via NewUserActor, NewBotActor, NewSystemActor,
// NewServiceActor, or ParseActorID.
type ActorID struct {
	kind ActorKind
	raw  string
}

// NewUserActor creates an ActorID for a human user from their UserID.
// The raw value is the UserID's string form.
func NewUserActor(userID UserID) ActorID {
	return ActorID{kind: ActorUser, raw: userID.String()}
}

// NewBotActor creates an ActorID for an automated bot or API token.
// The raw value is the bot's identifier (typically a ULID or token name).
func NewBotActor(raw string) ActorID {
	return ActorID{kind: ActorBot, raw: raw}
}

// NewSystemActor creates an ActorID for an internal system process.
// name examples: "scheduler", "indexer", "gc", "migration".
func NewSystemActor(name string) ActorID {
	return ActorID{kind: ActorSystem, raw: name}
}

// NewServiceActor creates an ActorID for a named service.
// serviceID examples: "api-gateway", "order-service", "notification-worker".
func NewServiceActor(serviceID string) ActorID {
	return ActorID{kind: ActorService, raw: serviceID}
}

// ParseActorID reconstructs an ActorID from its prefixed string form
// ("kind:raw"). Returns an error if the format is invalid or the kind
// prefix is unrecognized.
func ParseActorID(s string) (ActorID, error) {
	if s == "" {
		return ActorID{}, nil
	}

	kindStr, raw, found := strings.Cut(s, ":")
	if !found {
		return ActorID{}, fmt.Errorf("actor id %q: missing kind prefix (expected \"kind:raw\")", s)
	}

	kind, ok := parseActorKind(kindStr)
	if !ok {
		return ActorID{}, fmt.Errorf("actor id %q: unknown kind %q", s, kindStr)
	}

	return ActorID{kind: kind, raw: raw}, nil
}

func parseActorKind(s string) (ActorKind, bool) {
	switch s {
	case kindUserStr:
		return ActorUser, true
	case kindBotStr:
		return ActorBot, true
	case kindSystemStr:
		return ActorSystem, true
	case kindServiceStr:
		return ActorService, true
	default:
		return ActorUnknown, false
	}
}

// Kind returns the actor kind (User, Bot, System, Service).
func (a ActorID) Kind() ActorKind { return a.kind }

// Raw returns the raw identifier string (the actor's ID or name without
// the kind prefix).
func (a ActorID) Raw() string { return a.raw }

// String returns the raw identifier without the kind prefix.
// Use PrefixedString for the self-describing form.
func (a ActorID) String() string { return a.raw }

// PrefixedString returns the self-describing "kind:raw" form.
// This is the format used for JSON serialization and ParseActorID.
func (a ActorID) PrefixedString() string {
	if a.IsZero() {
		return ""
	}

	return a.kind.String() + ":" + a.raw
}

// IsZero reports whether the ActorID is the zero value (no actor set).
func (a ActorID) IsZero() bool {
	return a.raw == ""
}

// Equal reports whether two ActorIDs have the same kind and raw value.
func (a ActorID) Equal(other ActorID) bool {
	return a.kind == other.kind && a.raw == other.raw
}

// GoString returns a debug-friendly representation.
func (a ActorID) GoString() string {
	if a.IsZero() {
		return "ActorID{}"
	}

	return fmt.Sprintf("ActorID{%s:%s}", a.kind, a.raw)
}

// Format implements fmt.Formatter for consistent verb behavior.
func (a ActorID) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		fmt.Fprint(f, a.PrefixedString())
	case 'q':
		fmt.Fprintf(f, "%q", a.PrefixedString())
	default:
		fmt.Fprint(f, a.PrefixedString())
	}
}
