package compat

import (
	"fmt"
	"reflect"
	"sort"

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
	if class == ClassIncomparable {
		return fmt.Sprintf("type is undeclared or opaque (%q -> %q): cannot relate the shapes", old, new)
	}

	if class == ClassAdditive {
		return fmt.Sprintf("type widened %q -> %q: every old value is still representable", old, new)
	}

	return fmt.Sprintf(
		"type changed %q -> %q: stored values do not decode into the new shape",
		old,
		new,
	)
}

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

func diffProperties(old, new objectShape, b *reportBuilder) {
	removed, added := splitProperties(old.props, new.props)

	for from, to := range pairRenames(removed, added) {
		b.add(Finding{
			Path:  joinPath(old.path, from),
			Kind:  KindFieldRenamed,
			Class: ClassNeedsVersionBump,
			Op:    OpRenameField,
			Message: fmt.Sprintf(
				"field renamed to %q with an unchanged shape: declare RenameField",
				to,
			),
		})

		delete(removed, from)
		delete(added, to)
	}

	diffFieldSets(old.path, removed, added, stringSet(new.required), b)

	for name, oldProp := range old.props {
		if newProp, ok := new.props[name]; ok {
			diffProperty(joinPath(old.path, name), oldProp, newProp, b)
		}
	}
}

func diffFieldSets(path string, removed, added map[string]schema.Property, required map[string]bool, b *reportBuilder) {
	for name := range removed {
		b.add(Finding{
			Path:    joinPath(path, name),
			Kind:    KindFieldRemoved,
			Class:   ClassNeedsVersionBump,
			Op:      OpRemoveField,
			Message: "field removed: old events carry a value the new shape rejects",
		})
	}

	for name := range added {
		b.add(fieldAdded(path, name, required[name]))
	}
}

func fieldAdded(path, name string, required bool) Finding {
	if required {
		return Finding{
			Path:    joinPath(path, name),
			Kind:    KindFieldRequired,
			Class:   ClassNeedsVersionBump,
			Op:      OpTransform,
			Message: "required field added: old events that omit it no longer validate",
		}
	}

	return Finding{
		Path:    joinPath(path, name),
		Kind:    KindFieldAdded,
		Class:   ClassAdditive,
		Op:      OpAddField,
		Message: "optional field added: old events decode with the zero value",
	}
}

func diffRequired(old, new objectShape, b *reportBuilder) {
	oldReq, newReq := stringSet(old.required), stringSet(new.required)

	for name := range oldReq {
		if newReq[name] {
			continue
		}

		if _, declared := new.props[name]; !declared {
			continue // the field itself was removed, already reported
		}

		b.add(Finding{
			Path:    joinPath(old.path, name),
			Kind:    KindFieldOptional,
			Class:   ClassAdditive,
			Message: "field became optional: old events keep decoding",
		})
	}

	for name := range newReq {
		if oldReq[name] {
			continue
		}

		if _, existed := old.props[name]; !existed {
			continue // a newly added required field, reported by diffFieldSets
		}

		b.add(Finding{
			Path:    joinPath(old.path, name),
			Kind:    KindFieldRequired,
			Class:   ClassNeedsVersionBump,
			Op:      OpTransform,
			Message: "field became required: old events that omit it no longer validate",
		})
	}
}

func diffItems(old, new objectShape, b *reportBuilder) {
	switch {
	case old.items == nil && new.items == nil:
		return
	case new.items == nil:
		b.add(Finding{
			Path:    old.path + "[]",
			Kind:    KindConstraintRemoved,
			Class:   ClassAdditive,
			Message: "array item schema removed: the element constraint widened",
		})
	case old.items == nil:
		b.add(Finding{
			Path:    old.path + "[]",
			Kind:    KindConstraintAdded,
			Class:   ClassNeedsVersionBump,
			Op:      OpTransform,
			Message: "array item schema added: existing elements are now constrained",
		})
	default:
		diffProperty(old.path+"[]", *old.items, *new.items, b)
	}
}

func diffEnum(path string, old, new []string, b *reportBuilder) {
	switch {
	case len(old) == 0 && len(new) == 0:
		return
	case len(new) == 0:
		b.add(Finding{
			Path:    path,
			Kind:    KindEnumWidened,
			Class:   ClassAdditive,
			Message: "enum constraint removed: the value set widened",
		})
	case len(old) == 0:
		b.add(Finding{
			Path:    path,
			Kind:    KindEnumNarrowed,
			Class:   ClassNeedsVersionBump,
			Op:      OpTransform,
			Message: "enum constraint added: previously accepted values are now rejected",
		})
	default:
		diffEnumValues(path, old, new, b)
	}
}

func diffEnumValues(path string, old, new []string, b *reportBuilder) {
	oldSet, newSet := stringSet(old), stringSet(new)

	for _, value := range new {
		if !oldSet[value] {
			b.add(Finding{
				Path:    path,
				Kind:    KindEnumWidened,
				Class:   ClassAdditive,
				Message: fmt.Sprintf("enum value %q added: the value set widened", value),
			})
		}
	}

	for _, value := range old {
		if !newSet[value] {
			b.add(Finding{
				Path:    path,
				Kind:    KindEnumNarrowed,
				Class:   ClassNeedsVersionBump,
				Op:      OpTransform,
				Message: fmt.Sprintf(
					"enum value %q removed: stored values may no longer validate",
					value,
				),
			})
		}
	}
}

func diffNullable(path string, old, new bool, b *reportBuilder) {
	switch {
	case old == new:
		return
	case new:
		b.add(Finding{
			Path:    path,
			Kind:    KindNullableWidened,
			Class:   ClassAdditive,
			Message: "field became nullable: the accepted value set widened",
		})
	default:
		b.add(Finding{
			Path:    path,
			Kind:    KindNullableNarrowed,
			Class:   ClassNeedsVersionBump,
			Op:      OpTransform,
			Message: "field became non-nullable: stored nulls no longer validate",
		})
	}
}

func diffConstraints(path string, old, new schema.Property, b *reportBuilder) {
	diffConstraint(path, "format", old.Format, new.Format, b)
	diffConstraint(path, "pattern", old.Pattern, new.Pattern, b)
}

func diffConstraint(path, name, old, new string, b *reportBuilder) {
	switch {
	case old == new:
		return
	case new == "":
		b.add(Finding{
			Path:    path,
			Kind:    KindConstraintRemoved,
			Class:   ClassAdditive,
			Message: name + " constraint removed: the accepted value set widened",
		})
	default:
		b.add(Finding{
			Path:    path,
			Kind:    KindConstraintAdded,
			Class:   ClassNeedsVersionBump,
			Op:      OpTransform,
			Message: fmt.Sprintf("%s constraint changed %q -> %q: stored values may no longer validate", name, old, new),
		})
	}
}

func diffAdvisory(path string, old, new schema.Property, b *reportBuilder) {
	if !old.Deprecated && new.Deprecated {
		b.add(Finding{
			Path:    path,
			Kind:    KindDeprecated,
			Class:   ClassAdditive,
			Message: "field marked deprecated: existing readers keep working",
		})
	}

	if old.Default != new.Default {
		b.add(Finding{
			Path:    path,
			Kind:    KindDefaultChanged,
			Class:   ClassAdditive,
			Message: fmt.Sprintf(
				"documented default changed %q -> %q (wire shape unaffected)",
				old.Default,
				new.Default,
			),
		})
	}
}

func splitProperties(old, new map[string]schema.Property) (map[string]schema.Property, map[string]schema.Property) {
	removed := make(map[string]schema.Property)
	added := make(map[string]schema.Property)

	for name, prop := range old {
		if _, ok := new[name]; !ok {
			removed[name] = prop
		}
	}

	for name, prop := range new {
		if _, ok := old[name]; !ok {
			added[name] = prop
		}
	}

	return removed, added
}

// pairRenames matches removals against additions whose shape is identical, so
// the report names RenameField instead of a bare removal plus addition. The
// pairing is advisory: a wrong pair still yields the same class, only a less
// precise op hint. Names are sorted so the pairing is deterministic.
func pairRenames(removed, added map[string]schema.Property) map[string]string {
	pairs := make(map[string]string)
	if len(removed) == 0 || len(added) == 0 {
		return pairs
	}

	used := make(map[string]bool, len(added))

	for _, from := range sortedKeys(removed) {
		for _, to := range sortedKeys(added) {
			if used[to] || !reflect.DeepEqual(removed[from], added[to]) {
				continue
			}

			pairs[from] = to
			used[to] = true

			break
		}
	}

	return pairs
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}

	return set
}

func joinPath(base, name string) string {
	switch {
	case base == "":
		return name
	case name == "":
		return base
	default:
		return base + "." + name
	}
}
