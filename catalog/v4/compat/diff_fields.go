package compat

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func diffProperties(old, next objectShape, b *reportBuilder) {
	removed, added := splitProperties(old.props, next.props)

	for previousName, nextName := range pairRenames(removed, added) {
		b.add(Finding{
			Path:  joinPath(old.path, previousName),
			Kind:  KindFieldRenamed,
			Class: ClassNeedsVersionBump,
			Op:    OpRenameField,
			Message: fmt.Sprintf(
				"field renamed to %q with an unchanged shape: declare RenameField",
				nextName,
			),
		})

		delete(removed, previousName)
		delete(added, nextName)
	}

	diffFieldSets(old.path, removed, added, stringSet(next.required), b)

	for name, oldProp := range old.props {
		if newProp, ok := next.props[name]; ok {
			diffProperty(joinPath(old.path, name), oldProp, newProp, b)
		}
	}
}

func diffFieldSets(
	path string,
	removed, added map[string]schema.Property,
	required map[string]bool,
	b *reportBuilder,
) {
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

func diffRequired(old, next objectShape, b *reportBuilder) {
	oldReq, newReq := stringSet(old.required), stringSet(next.required)

	diffBecameOptional(old, newReq, oldReq, b)
	diffBecameRequired(old, oldReq, newReq, b)
}

func diffBecameOptional(old objectShape, newReq, oldReq map[string]bool, b *reportBuilder) {
	for name := range oldReq {
		if newReq[name] {
			continue
		}

		if _, declared := old.props[name]; !declared {
			continue
		}

		b.add(Finding{
			Path:    joinPath(old.path, name),
			Kind:    KindFieldOptional,
			Class:   ClassAdditive,
			Message: "field became optional: the accepted value set widened",
		})
	}
}

func diffBecameRequired(old objectShape, oldReq, newReq map[string]bool, b *reportBuilder) {
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

func diffItems(old, next objectShape, b *reportBuilder) {
	switch {
	case old.items == nil && next.items == nil:
		return
	case next.items == nil:
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
		diffProperty(old.path+"[]", *old.items, *next.items, b)
	}
}

func splitProperties(
	old, next map[string]schema.Property,
) (map[string]schema.Property, map[string]schema.Property) {
	removed := make(map[string]schema.Property)
	added := make(map[string]schema.Property)

	for name, prop := range old {
		if _, ok := next[name]; !ok {
			removed[name] = prop
		}
	}

	for name, prop := range next {
		if _, ok := old[name]; !ok {
			added[name] = prop
		}
	}

	return removed, added
}

// pairRenames matches removals against additions whose shape is identical, so
// the report names RenameField instead of a bare removal plus addition. The
// pairing is advisory: a wrong pair still yields the same class, only a less
// precise op hint. Names are sorted so the pairing stays deterministic.
func pairRenames(removed, added map[string]schema.Property) map[string]string {
	pairs := make(map[string]string)
	if len(removed) == 0 || len(added) == 0 {
		return pairs
	}

	used := make(map[string]bool, len(added))

	for _, previousName := range sortedKeys(removed) {
		for _, nextName := range sortedKeys(added) {
			if used[nextName] || !propertiesEqual(removed[previousName], added[nextName]) {
				continue
			}

			pairs[previousName] = nextName
			used[nextName] = true

			break
		}
	}

	return pairs
}
