package schema

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// TypedEventSchema is an [EventSchema] bound to its Go payload type T. The
// binding lets a composition root derive typed projection decoders from the
// SAME declaration that drives upcasting — no parallel registration list to
// keep in sync. Build values with [EventOf].
//
// T exists for type inference only: no sample is decoded here and no value
// of T is stored beyond the zero-value sample the constructor captures.
type TypedEventSchema[T any] struct {
	declaration EventSchema
	sample      T
}

// EventOf declares one event type's schema bound to its Go payload type:
// wire name, CURRENT schema version, and the upcast ops migrating older
// stored versions toward it. It is the typed variant of [Event].
//
//	userCreated := schema.EventOf[UserCreated]("user.created", 2,
//		schema.RenameField("user.created", 1, "name", "displayName"))
func EventOf[T any](
	eventType event.Type,
	currentVersion event.SchemaVersion,
	ops ...Op,
) TypedEventSchema[T] {
	return TypedEventSchema[T]{
		declaration: Event(eventType, currentVersion, ops...),
		sample:      *new(T),
	}
}

// Declaration returns the untyped [EventSchema] — pass these to [Declare] or
// a composition root (system.DomainConfig.Schema).
func (s TypedEventSchema[T]) Declaration() EventSchema { return s.declaration }

// Sample returns the zero-value payload sample captured at declaration
// time. Decoder builders use it for payload-type inference.
func (s TypedEventSchema[T]) Sample() T { return s.sample }

// Type returns the declared event type.
func (s TypedEventSchema[T]) Type() event.Type { return s.declaration.Type() }

// CurrentVersion returns the schema version new writes of this event type
// stamp.
func (s TypedEventSchema[T]) CurrentVersion() event.SchemaVersion {
	return s.declaration.CurrentVersion()
}
