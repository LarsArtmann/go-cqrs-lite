package compat

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func diffEnum(path string, old, next []string, b *reportBuilder) {
	switch {
	case len(old) == 0 && len(next) == 0:
		return
	case len(next) == 0:
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
		diffEnumValues(path, old, next, b)
	}
}

func diffEnumValues(path string, old, next []string, b *reportBuilder) {
	oldSet, newSet := stringSet(old), stringSet(next)

	for _, value := range next {
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

func diffNullable(path string, old, next bool, b *reportBuilder) {
	if old == next {
		return
	}

	if next {
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

func diffConstraints(path string, old, next schema.Property, b *reportBuilder) {
	diffFormat(path, old, next, b)
	diffConstraint(path, "pattern", old.Pattern, next.Pattern, b)
}

// integerWidth reports the bit width of the integer formats this repository
// renders. Only these formats can be ordered, so anything else (a
// timestamp-shaped integer format, say) reports false and falls through to the
// generic constraint rule rather than guessing.
const (
	widthInt8  = 8
	widthInt16 = 16
	widthInt32 = 32
	widthInt64 = 64
)

func integerWidth(format string) (int, bool) {
	switch format {
	case "int8":
		return widthInt8, true
	case "int16":
		return widthInt16, true
	case "int32":
		return widthInt32, true
	case "int64":
		return widthInt64, true
	default:
		return 0, false
	}
}

// diffFormat treats a widened integer format as additive, because every stored
// value stays representable. This is the policy's own row ("Widening an int
// field (int32 -> int64)"); the generic constraint rule would otherwise report
// a needed op for a change that requires none.
func diffFormat(path string, old, next schema.Property, b *reportBuilder) {
	if old.Format == next.Format {
		return
	}

	if oldWidth, ok := integerWidth(old.Format); ok {
		if nextWidth, nextOK := integerWidth(next.Format); nextOK && nextWidth > oldWidth {
			b.add(Finding{
				Path:  path,
				Kind:  KindTypeChanged,
				Class: ClassAdditive,
				Message: fmt.Sprintf(
					"integer format widened %q -> %q: every stored value stays representable",
					old.Format,
					next.Format,
				),
			})

			return
		}
	}

	diffConstraint(path, "format", old.Format, next.Format, b)
}

func diffConstraint(path, name, old, next string, b *reportBuilder) {
	if old == next {
		return
	}

	if next == "" {
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
			next,
		),
	})
}

func diffAdvisory(path string, old, next schema.Property, b *reportBuilder) {
	if !old.Deprecated && next.Deprecated {
		b.add(Finding{
			Path:    path,
			Kind:    KindDeprecated,
			Class:   ClassAdditive,
			Message: "field marked deprecated: existing readers keep working",
		})
	}

	if old.Default != next.Default {
		b.add(Finding{
			Path:  path,
			Kind:  KindDefaultChanged,
			Class: ClassAdditive,
			Message: fmt.Sprintf(
				"documented default changed %q -> %q (wire shape unaffected)",
				old.Default,
				next.Default,
			),
		})
	}
}
