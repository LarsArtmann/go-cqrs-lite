package compat

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// Compat classifies the change from oldSchema to newSchema against the policy
// rows in docs/SCHEMA-COMPATIBILITY.md. A nil schema means "this message
// documents identity only": gaining one is additive, losing one is breaking.
func Compat(oldSchema, newSchema *schema.Schema) Report {
	var b reportBuilder

	diffSchema("", oldSchema, newSchema, &b)

	return b.report()
}

func diffSchema(path string, old, next *schema.Schema, b *reportBuilder) {
	switch {
	case old == nil && next == nil:
		return
	case next == nil:
		b.add(Finding{
			Path:    path,
			Kind:    KindSchemaRemoved,
			Class:   ClassNeedsVersionBump,
			Op:      OpTransform,
			Message: "payload schema removed: consumers lose the declared shape",
		})
	case old == nil:
		b.add(Finding{
			Path:    path,
			Kind:    KindSchemaAdded,
			Class:   ClassAdditive,
			Op:      OpAddField,
			Message: "payload schema added: nothing depended on the previous absence",
		})
	default:
		diffShape(shapeOf(path, old), shapeOf(path, next), b)
	}
}

// objectShape holds the object-shaped parts of a schema node so one walk
// serves both the payload root and every nested property.
type objectShape struct {
	path     string
	typ      schema.Type
	props    map[string]schema.Property
	required []string
	items    *schema.Property
}

func shapeOf(path string, s *schema.Schema) objectShape {
	return objectShape{
		path:     path,
		typ:      s.Type,
		props:    s.Properties,
		required: s.Required,
		items:    s.Items,
	}
}

func diffShape(old, next objectShape, b *reportBuilder) {
	if kind, class, changed := typeVerdict(old.typ, next.typ); changed {
		b.add(Finding{
			Path:    old.path,
			Kind:    kind,
			Class:   class,
			Op:      OpTransform,
			Message: typeMessage(old.typ, next.typ, class),
		})

		return
	}

	diffProperties(old, next, b)
	diffRequired(old, next, b)
	diffItems(old, next, b)
}

// typeVerdict maps a type change to a policy class. integer -> number is the
// one genuine widening: every old integer value is still a valid number.
func typeVerdict(old, next schema.Type) (Kind, Class, bool) {
	switch {
	case old == "" || next == "":
		return KindTypeChanged, ClassIncomparable, true
	case old == next:
		return "", ClassUnchanged, false
	case old == schema.TypeInteger && next == schema.TypeNumber:
		return KindTypeChanged, ClassAdditive, true
	default:
		return KindTypeChanged, ClassNeedsVersionBump, true
	}
}

func typeMessage(old, next schema.Type, class Class) string {
	switch class {
	case ClassIncomparable:
		return fmt.Sprintf(
			"type is undeclared or opaque (%q -> %q): cannot relate the shapes",
			old,
			next,
		)
	case ClassAdditive:
		return fmt.Sprintf(
			"type widened %q -> %q: every old value is still representable",
			old,
			next,
		)
	default:
		return fmt.Sprintf(
			"type changed %q -> %q: stored values do not decode into the new shape",
			old,
			next,
		)
	}
}

// diffProperty compares one property node: its object-shaped parts plus the
// value constraints that cannot nest.
func diffProperty(path string, old, next schema.Property, b *reportBuilder) {
	diffShape(
		objectShape{
			path:     path,
			typ:      old.Type,
			props:    old.Properties,
			required: old.Required,
			items:    old.Items,
		},
		objectShape{
			path:     path,
			typ:      next.Type,
			props:    next.Properties,
			required: next.Required,
			items:    next.Items,
		},
		b,
	)

	diffEnum(path, old.Enum, next.Enum, b)
	diffNullable(path, old.Nullable, next.Nullable, b)
	diffConstraints(path, old, next, b)
	diffAdvisory(path, old, next, b)
}
