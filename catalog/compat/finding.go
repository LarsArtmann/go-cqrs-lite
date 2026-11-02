package compat

import (
	"fmt"
	"sort"
	"strings"
)

// Kind is a stable identifier for one class of schema difference. Gates and
// waivers match on these strings, so they must not change without a
// deprecation path.
type Kind string

const (
	// KindSchemaAdded reports that a message gained a payload schema where
	// it previously documented only identity.
	KindSchemaAdded Kind = "schema-added"

	// KindSchemaRemoved reports that a message's payload schema is gone.
	KindSchemaRemoved Kind = "schema-removed"

	// KindFieldAdded reports a new optional property.
	KindFieldAdded Kind = "field-added"

	// KindFieldRemoved reports a property that no longer exists.
	KindFieldRemoved Kind = "field-removed"

	// KindFieldRenamed reports a removal and an addition whose property
	// shapes are identical, which the policy maps to RenameField.
	KindFieldRenamed Kind = "field-renamed"

	// KindFieldRequired reports that an existing property became required.
	KindFieldRequired Kind = "field-required"

	// KindFieldOptional reports that a required property became optional.
	KindFieldOptional Kind = "field-optional"

	// KindTypeChanged reports a property or payload type change.
	KindTypeChanged Kind = "type-changed"

	// KindEnumWidened reports enum values being added.
	KindEnumWidened Kind = "enum-widened"

	// KindEnumNarrowed reports enum values being removed, or an enum
	// appearing where there was none.
	KindEnumNarrowed Kind = "enum-narrowed"

	// KindNullableWidened reports a field becoming nullable.
	KindNullableWidened Kind = "nullable-widened"

	// KindNullableNarrowed reports a nullable field becoming non-nullable.
	KindNullableNarrowed Kind = "nullable-narrowed"

	// KindConstraintAdded reports a tightened constraint (format, pattern).
	KindConstraintAdded Kind = "constraint-added"

	// KindConstraintRemoved reports a dropped constraint.
	KindConstraintRemoved Kind = "constraint-removed"

	// KindDefaultChanged reports a changed documented default. The wire
	// shape is unaffected, so this is advisory.
	KindDefaultChanged Kind = "default-changed"

	// KindDeprecated reports a message or property being marked deprecated.
	KindDeprecated Kind = "deprecated"

	// KindMessageAdded reports a message that exists only in the new
	// catalog.
	KindMessageAdded Kind = "message-added"

	// KindMessageRemoved reports a message that existed only in the old
	// catalog. Removal is a domain decision, so it never blocks by itself.
	KindMessageRemoved Kind = "message-removed"

	// KindVersionNotBumped reports a breaking shape change that did not
	// move the message version. This is the only kind that blocks a gate.
	KindVersionNotBumped Kind = "version-not-bumped"
)

// The migration ops the policy prescribes. A finding names the op a reader
// must declare, so the report is actionable without consulting the policy
// document.
const (
	// OpAddField is [schema.AddField].
	OpAddField = "AddField"

	// OpRemoveField is [schema.RemoveField].
	OpRemoveField = "RemoveField"

	// OpRenameField is [schema.RenameField].
	OpRenameField = "RenameField"

	// OpTransform is [schema.Transform].
	OpTransform = "Transform"

	// OpRenameType is [schema.RenameType].
	OpRenameType = "RenameType"
)

// Finding is one classified difference.
type Finding struct {
	// Path locates the change using the catalog's dot-and-bracket
	// convention, for example "order_placed.shipping.city" or
	// "order_placed.lines[].sku".
	Path string `json:"path"`

	// Kind is the stable difference identifier.
	Kind Kind `json:"kind"`

	// Class is the compatibility verdict for this difference alone.
	Class Class `json:"class"`

	// Message explains the difference and its consequence.
	Message string `json:"message"`

	// Op names the migration op the policy requires, when one applies.
	Op string `json:"op,omitempty"`

	// Blocking marks the differences that must fail a compatibility gate.
	// Only a breaking change with an unchanged message version does.
	Blocking bool `json:"blocking,omitempty"`
}

// Report is the classified outcome of one comparison.
type Report struct {
	// Class is the most severe finding's class.
	Class Class `json:"class"`

	// Findings are sorted by path, then kind, so output is deterministic.
	Findings []Finding `json:"findings,omitempty"`
}

// Blocking returns the findings that must fail a compatibility gate.
func (r Report) Blocking() []Finding {
	var out []Finding

	for _, f := range r.Findings {
		if f.Blocking {
			out = append(out, f)
		}
	}

	return out
}

// OK reports whether the comparison requires no action: nothing blocks, and
// no finding is breaking.
func (r Report) OK() bool {
	return !r.Class.IsBreaking() && len(r.Blocking()) == 0
}

// String renders the report as a headline plus one line per finding.
func (r Report) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s (%d finding(s))", r.Class, len(r.Findings))

	for _, f := range r.Findings {
		fmt.Fprintf(&b, "\n  %s: %s [%s, %s]", f.Path, f.Message, f.Kind, f.Class)

		if f.Op != "" {
			fmt.Fprintf(&b, " op=%s", f.Op)
		}

		if f.Blocking {
			b.WriteString(" BLOCKING")
		}
	}

	return b.String()
}

// reportBuilder accumulates findings and tracks the worst class.
type reportBuilder struct {
	class    Class
	findings []Finding
}

func (b *reportBuilder) add(f Finding) {
	b.class = b.class.Worst(f.Class)
	b.findings = append(b.findings, f)
}

func (b *reportBuilder) report() Report {
	sort.SliceStable(b.findings, func(i, j int) bool {
		if b.findings[i].Path != b.findings[j].Path {
			return b.findings[i].Path < b.findings[j].Path
		}

		return b.findings[i].Kind < b.findings[j].Kind
	})

	return Report{Class: b.class, Findings: b.findings}
}
