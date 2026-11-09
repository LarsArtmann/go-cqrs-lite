package compat

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// The fixtures below are the CUE-subsume spike's own pairs, restricted to the
// rows this repository's schema model can express (it has no closedness or
// maxLength field, so those three spike rows cannot appear in a catalog at
// all). docs/evidence/2026-11-02_cue-subsume-spike.md records the CUE scores;
// this table pins the alternative that shipped. ADR-0163 is the decision.
//
// The point is not that this differ is clever. It is that it is SOUND: across
// every row it is asked about it never reports "compatible" for a change the
// policy calls breaking. CUE's best configuration missed two of these rows in
// exactly that direction.
func TestCompatAgreesWithPolicyOnCueSpikeFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		old  *schema.Schema
		new  *schema.Schema
		want Class
	}{
		{
			name: "A1 add optional field",
			old:  rootObj(map[string]schema.Property{"id": strProp()}, "id"),
			new:  rootObj(map[string]schema.Property{"id": strProp(), "note": strProp()}, "id"),
			want: ClassAdditive,
		},
		{
			name: "A2 add optional nested field",
			old: rootObj(
				map[string]schema.Property{
					"meta": objProp(map[string]schema.Property{"a": strProp()}),
				},
			),
			new: rootObj(
				map[string]schema.Property{
					"meta": objProp(map[string]schema.Property{"a": strProp(), "b": strProp()}),
				},
			),
			want: ClassAdditive,
		},
		{
			name: "A3 add field inside array items",
			old: rootObj(
				map[string]schema.Property{
					"xs": arrProp(objProp(map[string]schema.Property{"a": strProp()})),
				},
			),
			new: rootObj(
				map[string]schema.Property{
					"xs": arrProp(
						objProp(map[string]schema.Property{"a": strProp(), "b": strProp()}),
					),
				},
			),
			want: ClassAdditive,
		},
		{
			name: "A4 widen integer to number",
			old:  rootObj(map[string]schema.Property{"n": intProp()}),
			new:  rootObj(map[string]schema.Property{"n": numProp()}),
			want: ClassAdditive,
		},
		{
			name: "A5 add enum value",
			old:  rootObj(map[string]schema.Property{"s": enumProp("a")}),
			new:  rootObj(map[string]schema.Property{"s": enumProp("a", "b")}),
			want: ClassAdditive,
		},
		{
			name: "A6 required becomes optional",
			old:  rootObj(map[string]schema.Property{"a": strProp()}, "a"),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassAdditive,
		},
		{
			name: "B1 remove field",
			old:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "B2 rename field",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"b": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "B3 change field type",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": numProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "B4 add required field",
			old:  rootObj(map[string]schema.Property{"a": strProp()}, "a"),
			new:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}, "a", "b"),
			want: ClassNeedsVersionBump,
		},
		{
			name: "B5 narrow enum",
			old:  rootObj(map[string]schema.Property{"s": enumProp("a", "b")}),
			new:  rootObj(map[string]schema.Property{"s": enumProp("a")}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "B6 become non-nullable",
			old: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Nullable: true}},
			),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "H3 flatten restructure",
			old: rootObj(
				map[string]schema.Property{
					"a": objProp(map[string]schema.Property{"b": strProp()}),
				},
			),
			new:  rootObj(map[string]schema.Property{"a_b": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "H4 add pattern constraint",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Pattern: "^[a-z]+$"}},
			),
			want: ClassNeedsVersionBump,
		},
		{
			name: "H5 relax pattern constraint",
			old: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Pattern: "^[a-z]+$"}},
			),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassAdditive,
		},
		{
			name: "H7 change format",
			old: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Format: "date-time"}},
			),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Format: "uuid"}},
			),
			want: ClassNeedsVersionBump,
		},
		{
			name: "H8 become nullable",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Nullable: true}},
			),
			want: ClassAdditive,
		},
		{
			name: "H9 add array items constraint",
			old:  rootObj(map[string]schema.Property{"xs": {Type: schema.TypeArray}}),
			new:  rootObj(map[string]schema.Property{"xs": arrProp(strProp())}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "H10 add required next to existing",
			old:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}, "a"),
			new:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}, "a", "b"),
			want: ClassNeedsVersionBump,
		},
		{
			name: "H11 nested remove field",
			old: rootObj(
				map[string]schema.Property{
					"m": objProp(map[string]schema.Property{"a": strProp(), "b": strProp()}),
				},
			),
			new: rootObj(
				map[string]schema.Property{
					"m": objProp(map[string]schema.Property{"a": strProp()}),
				},
			),
			want: ClassNeedsVersionBump,
		},
		{
			name: "H12 int32 to int64 widen",
			old: rootObj(
				map[string]schema.Property{"n": {Type: schema.TypeInteger, Format: "int32"}},
			),
			new: rootObj(
				map[string]schema.Property{"n": {Type: schema.TypeInteger, Format: "int64"}},
			),
			want: ClassAdditive,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := Compat(tc.old, tc.new).Class; got != tc.want {
				t.Fatalf("Compat(%s).Class = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestCompatNeverApprovesABreakingChange is the soundness pin. Its two cases
// are precisely the rows CUE could not see: format is not a CUE constraint, and
// an open CUE list subsumes a constrained one. Both were reported "compatible"
// by every CUE configuration measured in the spike; a gate that approves them
// would license a replay-breaking release. See ADR-0163.
func TestCompatNeverApprovesABreakingChange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		old  *schema.Schema
		new  *schema.Schema
	}{
		{
			name: "format swap",
			old: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Format: "date-time"}},
			),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Format: "uuid"}},
			),
		},
		{
			name: "newly constrained array items",
			old:  rootObj(map[string]schema.Property{"xs": {Type: schema.TypeArray}}),
			new:  rootObj(map[string]schema.Property{"xs": arrProp(strProp())}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report := Compat(tc.old, tc.new)
			if !report.Class.IsBreaking() {
				t.Fatalf("Compat(%s).Class = %v, want a breaking class", tc.name, report.Class)
			}
			if report.OK() {
				t.Fatalf(
					"Compat(%s).OK() = true, want false (a breaking change must not read as OK)",
					tc.name,
				)
			}
		})
	}
}

// TestCompatIntegerWidthOrdering pins the narrowing direction, which is the
// half of the int-width rule that must NOT be forgiven: int64 -> int32 makes
// stored values unrepresentable.
func TestCompatIntegerWidthOrdering(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		oldFormat  string
		newFormat  string
		wantClass  Class
		wantReason Kind
	}{
		{"int8 to int64", "int8", "int64", ClassAdditive, KindTypeChanged},
		{"int32 to int64", "int32", "int64", ClassAdditive, KindTypeChanged},
		{"int64 to int32", "int64", "int32", ClassNeedsVersionBump, KindConstraintAdded},
		{"unbounded to int32", "", "int32", ClassNeedsVersionBump, KindConstraintAdded},
		{"int32 to unbounded", "int32", "", ClassAdditive, KindConstraintRemoved},
		{"int32 to timestamp", "int32", "date-time", ClassNeedsVersionBump, KindConstraintAdded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report := Compat(
				rootObj(
					map[string]schema.Property{
						"n": {Type: schema.TypeInteger, Format: tc.oldFormat},
					},
				),
				rootObj(
					map[string]schema.Property{
						"n": {Type: schema.TypeInteger, Format: tc.newFormat},
					},
				),
			)

			if report.Class != tc.wantClass {
				t.Fatalf("Class = %v, want %v", report.Class, tc.wantClass)
			}
			if !hasKind(report, tc.wantReason) {
				t.Fatalf("kinds = %v, want %v", uniqueKinds(report), tc.wantReason)
			}
		})
	}
}
