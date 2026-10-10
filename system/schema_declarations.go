package system

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
)

// SchemaDeclarations is the built result of the [Schemas] builder: the
// upcasting declarations (for [DomainConfig.Schema]) and the derived typed
// decoder (for [DomainConfig.ProjectionTypeDecoder]) produced from ONE
// declaration list. Pass both to the DomainConfig — the declaration, not the
// wiring, stays the single source of truth.
type SchemaDeclarations struct {
	declarations []schema.EventSchema
	decoder      *projectionadapter.TypeDecoder
}

// Declarations returns the event schemas for [DomainConfig.Schema].
func (d SchemaDeclarations) Declarations() []schema.EventSchema { return d.declarations }

// TypeDecoder returns the projection decoder derived from the same list, for
// [DomainConfig.ProjectionTypeDecoder]. It is nil when no event was declared.
func (d SchemaDeclarations) TypeDecoder() *projectionadapter.TypeDecoder { return d.decoder }

// Schemas starts a schema-declaration builder. Each [SchemaSet.Event] binds
// one event type's wire name, CURRENT schema version, upcast ops, AND Go
// payload type; [SchemaSet.Build] yields both DomainConfig inputs from that
// one list:
//
//	decls := system.Schemas().
//		Event[UserCreated]("user.created", 2,
//			schema.RenameField("user.created", 1, "name", "displayName")).
//		Event[UserRenamed]("user.renamed", 1).
//		Build()
//
//	sys, err := system.New(ctx, system.DomainConfig{
//		Schema:                decls.Declarations(),
//		ProjectionTypeDecoder: decls.TypeDecoder(),
//	}, deployment)
//
// Duplicate event types are rejected at Build with the same rule [Declare]
// enforces — fail fast, before composition.
func Schemas() *SchemaSet {
	return &SchemaSet{}
}

// SchemaSet accumulates typed schema declarations. It is not safe for
// concurrent use; build once during composition.
type SchemaSet struct {
	schemaDeclarations []schema.EventSchema
	registrations      []projectionadapter.EventRegistration
	seen               map[event.Type]struct{}
}

// Event declares one event type: wire name, CURRENT schema version, upcast
// ops migrating older stored versions toward it, and the Go payload type T
// (inferred from the type parameter — no sample argument). It returns the
// builder for chaining.
func (s *SchemaSet) Event[T any](
	eventType event.Type,
	currentVersion event.SchemaVersion,
	ops ...schema.Op,
) *SchemaSet {
	if s.seen == nil {
		s.seen = make(map[event.Type]struct{})
	}

	s.seen[eventType] = struct{}{}
	s.schemaDeclarations = append(
		s.schemaDeclarations, schema.Event(eventType, currentVersion, ops...),
	)
	s.registrations = append(
		s.registrations, projectionadapter.Register(eventType, *new(T)),
	)

	return s
}

// Build validates the accumulated set (unique, non-empty event types) and
// returns the declarations plus the derived decoder.
func (s *SchemaSet) Build() (SchemaDeclarations, error) {
	if len(s.schemaDeclarations) == 0 {
		return SchemaDeclarations{}, nil
	}

	if len(s.seen) != len(s.schemaDeclarations) {
		counts := make(map[event.Type]int, len(s.schemaDeclarations))
		for _, declared := range s.schemaDeclarations {
			counts[declared.Type()]++
		}

		for declared, count := range counts {
			if count > 1 {
				return SchemaDeclarations{}, errorfamily.NewRejection(
					"system.duplicate_schema_declaration",
					"event type "+string(declared)+" declared twice in the schema builder",
				)
			}
		}
	}

	return SchemaDeclarations{
		declarations: s.schemaDeclarations,
		decoder:      projectionadapter.NewTypeDecoder(s.registrations...),
	}, nil
}
