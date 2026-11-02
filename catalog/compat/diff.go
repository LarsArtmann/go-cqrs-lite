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

func diffSchema(path string, old, new *schema.Schema, b *reportBuilder) {
	switch {
	case old == nil && new == nil:
		return
	case new == nil:
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
		diffShape(shapeOf(path, old), shapeOf(path, new), b)
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

func diffShape(old, new objectShape, b *reportBuilder) {
	if kind, class, changed := typeVerdict(old.typ, new.typ); changed {
		b.add(Finding{
			Path:    old.path,
			Kind:    kind,
			Class:   class,
			Op:      OpTransform,
			Message: typeMessage(old.typ, new.typ, class),
		})

		return
	}

	diffProperties(old, new, b)
	diffRequired(old, new, b)
	diffItems(old, new, b)
}

// typeVerdict maps a type change to a policy class. integer -> number is the
// one genuine widening: every old integer value is still a valid number.
func typeVerdict(old, new schema.Type) (Kind, Class, bool) {
	switch {
	case old == "" || new == "":
		return KindTypeChanged, ClassIncomparable, true
	case old == new:
		return "", ClassUnchanged, false
	case old == schema.TypeInteger && new == schema.TypeNumber:
		return KindTypeChanged, ClassAdditive, true
	default:
		return KindTypeChanged, ClassNeedsVersionBump, true
	}
}

func typeMessage(old, new schema.Type, class Class) string {
	switch class {
	case ClassIncomparable:
		return fmt.Sprintf(
			"type is undeclared or opaque (%q -> %q): cannot relate the shapes",
			old,
			new,
		)
	case ClassAdditive:
		return fmt.Sprintf(
			"type widened %q -> %q: every old value is still representable",
			old,
			new,
		)
	default:
		return fmt.Sprintf(
			"type changed %q -> %q: stored values do not decode into the new shape",
			old,
			new,
		)
	}
}

// diffProperty compares one property node: its object-shaped parts plus the
// value constraints that cannot nest.
func diffProperty(path string, old, new schema.Property, b *reportBuilder) {
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
			typ:      new.Type,
			props:    new.Properties,
			required: new.Required,
			items:    new.Items,
		},
		b,
	)

	diffEnum(path, old.Enum, new.Enum, b)
	diffNullable(path, old.Nullable, new.Nullable, b)
	diffConstraints(path, old, new, b)
	diffAdvisory(path, old, new, b)
}
