package catalog

import (
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

// Version identity (proposal §8, ADR-0155): the wire keeps the integer
// schema version; the declaration owns semantics; the catalog semver string
// is DERIVED display metadata. SemverFromWire maps wire int N to "N.0.0" —
// chains compose on +1 steps and compat classes are enforced by declared
// ops, so MAJOR/MINOR/PATCH distinctions would be decorative.

// SemverFromWire derives the catalog display version from a wire schema
// version: 3 → "3.0.0".
func SemverFromWire(wireVersion int) string {
	return strconv.Itoa(wireVersion) + ".0.0"
}

// FromTypedSchema renders a typed event-schema declaration
// ([schema.EventOf]) as a catalog event message, so the SAME declaration
// that drives upcasting and projection decoding also drives the governance
// export — no parallel catalog declarations to keep in sync.
//
// The wire name is the message ID, the payload Go type T is the schema, and
// the version string is derived from the declared current wire version via
// [SemverFromWire]. Direction must be explicit (Sends or Receives).
// [WithVersion] overrides the derived string; [WithName] overrides the
// payload-type-derived display name; all other message options apply.
func FromTypedSchema[T any](
	decl schema.TypedEventSchema[T],
	direction Direction,
	opts ...MessageOption,
) MessageConfig {
	derived := append([]MessageOption{
		WithVersion(SemverFromWire(int(decl.CurrentVersion()))),
	}, opts...)

	return Event[T](MessageID(decl.Type()), direction, derived...)
}
