package schema

import (
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// EventSchema declares one event type's current wire contract: the event
// type, its CURRENT schema version (what new writes stamp), and the ops that
// upcast older stored versions toward it. Build values with [Event] and pass
// them to [Declare] — or hand the slice to a composition root that calls
// Declare (system.DomainConfig.Schema does exactly that).
//
// A Declaration is DATA, not a registry: no global state, no registration
// side effects. Validation happens in [Declare].
type EventSchema struct {
	eventType      event.Type
	currentVersion event.SchemaVersion
	ops            []Op
}

// Event declares the schema of one event type at its current version, with
// the upcast ops that migrate older stored versions toward it.
//
// Every op must target this event type (a [RenameType] op declares its FROM
// side here and may point anywhere; its target then continues in that
// type's own declaration). Ops with a source version >= the current version
// are rejected — an op may only migrate FROM a superseded version.
func Event(
	eventType event.Type,
	currentVersion event.SchemaVersion,
	ops ...Op,
) EventSchema {
	return EventSchema{
		eventType:      eventType,
		currentVersion: currentVersion,
		ops:            ops,
	}
}

// Type returns the declared event type.
func (s EventSchema) Type() event.Type { return s.eventType }

// CurrentVersion returns the schema version new writes of this event type
// stamp (the version the upcast chain converges toward).
func (s EventSchema) CurrentVersion() event.SchemaVersion { return s.currentVersion }

// Declare validates a full event-schema declaration set and compiles every
// op into ONE chain — matching, versioning, and policy rules are the
// [Compile] rules, applied across all declarations at once (two
// declarations claiming the same (type, version) match is a duplicate
// error).
//
// Additional declaration-level checks: event types must be non-empty and
// unique across the set, current versions positive, and every op must
// target its own declaration's event type with a source version BELOW the
// current version.
func Declare(schemas ...EventSchema) (*Chain, error) {
	seen := make(map[event.Type]struct{}, len(schemas))
	ops := make([]Op, 0, len(schemas))

	for _, declared := range schemas {
		if err := validateDeclaration(declared, seen); err != nil {
			return nil, err
		}

		seen[declared.eventType] = struct{}{}

		ops = append(ops, declared.ops...)
	}

	return Compile(ops...)
}

func validateDeclaration(declared EventSchema, seen map[event.Type]struct{}) error {
	if declared.eventType == "" {
		return errorfamily.NewRejection(
			"schema.invalid_declaration",
			"event schema declaration requires a non-empty event type",
		)
	}

	if _, duplicate := seen[declared.eventType]; duplicate {
		return errorfamily.NewRejection(
			"schema.duplicate_declaration",
			"event type "+string(declared.eventType)+" declared twice")
	}

	if !declared.currentVersion.IsPositive() {
		return errorfamily.NewRejection(
			"schema.invalid_declaration",
			"event type "+string(declared.eventType)+" requires a positive current version")
	}

	for _, op := range declared.ops {
		if err := validateOpTarget(declared, op); err != nil {
			return err
		}
	}

	return nil
}

func validateOpTarget(declared EventSchema, op Op) error {
	sourceType, sourceVersion := opSource(op)

	if sourceType != "" && sourceType != declared.eventType {
		return errorfamily.NewRejection(
			"schema.op_target_mismatch",
			"op for "+string(sourceType)+" declared inside "+string(declared.eventType)+
				"'s schema — ops belong in their own event's declaration")
	}

	if sourceType != "" && sourceVersion >= declared.currentVersion {
		return errorfamily.NewRejection(
			"schema.op_supersedes_current",
			"op source version "+strconv.Itoa(int(sourceVersion))+
				" is not below current version "+declared.currentVersion.String()+
				" for "+string(declared.eventType))
	}

	return nil
}

func opSource(op Op) (event.Type, event.SchemaVersion) {
	switch typed := op.(type) {
	case *fieldOp:
		return typed.sourceType, typed.sourceVersion
	case *transformOp:
		return typed.sourceType, typed.sourceVersion
	case *splitOp:
		return typed.sourceType, typed.sourceVersion
	case *renameTypeOp:
		return typed.from, 0
	case *dropOp:
		return typed.sourceType, 0
	default:
		return "", 0
	}
}
