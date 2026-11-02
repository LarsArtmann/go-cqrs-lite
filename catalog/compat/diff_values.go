package compat

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

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
		if oldSet[value] {
			continue
		}

		b.add(Finding{
			Path:    path,
			Kind:    KindEnumWidened,
			Class:   ClassAdditive,
			Message: fmt.Sprintf("enum value %q added: the value set widened", value),
		})
	}

	for _, value := range old {
		if newSet[value] {
			continue
		}

		b.add(Finding{
			Path:  path,
			Kind:  KindEnumNarrowed,
			Class: ClassNeedsVersionBump,
			Op:    OpTransform,
			Message: fmt.Sprintf(
				"enum value %q removed: stored values may no longer validate",
				value,
			),
		})
	}
}

func diffNullable(path string, old, new bool, b *reportBuilder) {
	if old == new {
		return
	}

	if new {
		b.add(Finding{
			Path:    path,
			Kind:    KindNullableWidened,
			Class:   ClassAdditive,
			Message: "field became nullable: the accepted value set widened",
		})

		return
	}

	b.add(Finding{
		Path:    path,
		Kind:    KindNullableNarrowed,
		Class:   ClassNeedsVersionBump,
		Op:      OpTransform,
		Message: "field became non-nullable: stored nulls no longer validate",
	})
}

func diffConstraints(path string, old, new schema.Property, b *reportBuilder) {
	diffFormat(path, old, new, b)
	diffConstraint(path, "pattern", old.Pattern, new.Pattern, b)
}

// integerWidths maps the integer formats this repository renders to their bit
// widths. Only formats in this table can be ordered, so anything else (a
// timestamp-shaped integer format, say) falls through to the generic
// constraint rule rather than guessing.
var integerWidths = map[string]int{
	"int8":  8,
	"int16": 16,
	"int32": 32,
	"int64": 64,
}

// diffFormat treats a widened integer format as additive, because every stored
// value stays representable. This is the policy's own row ("Widening an int
// field (int32 -> int64)"); the generic constraint rule would otherwise report
// a needed op for a change that requires none.
func diffFormat(path string, old, new schema.Property, b *reportBuilder) {
	if old.Format == new.Format {
		return
	}

	if oldWidth, oldOK := integerWidths[old.Format]; oldOK {
		if newWidth, newOK := integerWidths[new.Format]; newOK && newWidth > oldWidth {
			b.add(Finding{
				Path:  path,
				Kind:  KindTypeChanged,
				Class: ClassAdditive,
				Message: fmt.Sprintf(
					"integer format widened %q -> %q: every stored value stays representable",
					old.Format,
					new.Format,
				),
			})

			return
		}
	}

	diffConstraint(path, "format", old.Format, new.Format, b)
}

func diffConstraint(path, name, old, new string, b *reportBuilder) {
	if old == new {
		return
	}

	if new == "" {
		b.add(Finding{
			Path:    path,
			Kind:    KindConstraintRemoved,
			Class:   ClassAdditive,
			Message: name + " constraint removed: the accepted value set widened",
		})

		return
	}

	b.add(Finding{
		Path:  path,
		Kind:  KindConstraintAdded,
		Class: ClassNeedsVersionBump,
		Op:    OpTransform,
		Message: fmt.Sprintf(
			"%s constraint changed %q -> %q: stored values may no longer validate",
			name,
			old,
			new,
		),
	})
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
			Path:  path,
			Kind:  KindDefaultChanged,
			Class: ClassAdditive,
			Message: fmt.Sprintf(
				"documented default changed %q -> %q (wire shape unaffected)",
				old.Default,
				new.Default,
			),
		})
	}
}
