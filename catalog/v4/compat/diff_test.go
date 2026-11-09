package compat

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// TestCompatPolicyTable pins every row of docs/SCHEMA-COMPATIBILITY.md as an
// executable case. The old/new pairs are the same fixtures the CUE fidelity
// spike used, so the evidence file and this table cannot drift apart.
func TestCompatPolicyTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		old  *schema.Schema
		new  *schema.Schema
		want Class
	}{
		{
			name: "add optional field",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}),
			want: ClassAdditive,
		},
		{
			name: "add optional nested field",
			old: rootObj(
				map[string]schema.Property{
					"a": objProp(map[string]schema.Property{"x": strProp()}),
				},
			),
			new: rootObj(map[string]schema.Property{
				"a": objProp(map[string]schema.Property{"x": strProp(), "y": strProp()}),
			}),
			want: ClassAdditive,
		},
		{
			name: "add field inside array items",
			old: rootObj(
				map[string]schema.Property{
					"xs": arrProp(objProp(map[string]schema.Property{"x": strProp()})),
				},
			),
			new: rootObj(map[string]schema.Property{
				"xs": arrProp(objProp(map[string]schema.Property{"x": strProp(), "y": strProp()})),
			}),
			want: ClassAdditive,
		},
		{
			name: "widen integer to number",
			old:  rootObj(map[string]schema.Property{"n": intProp()}),
			new:  rootObj(map[string]schema.Property{"n": numProp()}),
			want: ClassAdditive,
		},
		{
			name: "required becomes optional",
			old:  rootObj(map[string]schema.Property{"a": strProp()}, "a"),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassAdditive,
		},
		{
			name: "add enum value",
			old:  rootObj(map[string]schema.Property{"s": enumProp("a", "b")}),
			new:  rootObj(map[string]schema.Property{"s": enumProp("a", "b", "c")}),
			want: ClassAdditive,
		},
		{
			name: "become nullable",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Nullable: true}},
			),
			want: ClassAdditive,
		},
		{
			name: "mark deprecated",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Deprecated: true}},
			),
			want: ClassAdditive,
		},
		{
			name: "schema appears where none existed",
			old:  nil,
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassAdditive,
		},

		{
			name: "remove field",
			old:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "rename field",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"b": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "change field type",
			old:  rootObj(map[string]schema.Property{"v": strProp()}),
			new:  rootObj(map[string]schema.Property{"v": intProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "restructure flatten",
			old: rootObj(
				map[string]schema.Property{
					"a": objProp(map[string]schema.Property{"x": strProp()}),
				},
			),
			new:  rootObj(map[string]schema.Property{"a_x": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "add required field",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}, "b"),
			want: ClassNeedsVersionBump,
		},
		{
			name: "narrow enum",
			old:  rootObj(map[string]schema.Property{"s": enumProp("a", "b")}),
			new:  rootObj(map[string]schema.Property{"s": enumProp("a")}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "add enum where none existed",
			old:  rootObj(map[string]schema.Property{"s": strProp()}),
			new:  rootObj(map[string]schema.Property{"s": enumProp("a")}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "become non-nullable",
			old: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Nullable: true}},
			),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassNeedsVersionBump,
		},
		{
			name: "add pattern constraint",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Pattern: "^a"}},
			),
			want: ClassNeedsVersionBump,
		},
		{
			name: "add array item constraint",
			old:  rootArr(strProp()),
			new:  rootArr(intProp()),
			want: ClassNeedsVersionBump,
		},
		{
			name: "schema disappears",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  nil,
			want: ClassNeedsVersionBump,
		},
		{
			name: "change format",
			old: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Format: "date"}},
			),
			new: rootObj(
				map[string]schema.Property{"a": {Type: schema.TypeString, Format: "email"}},
			),
			want: ClassNeedsVersionBump,
		},
		{
			name: "change payload type",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootArr(strProp()),
			want: ClassNeedsVersionBump,
		},

		{
			name: "identical shapes",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassUnchanged,
		},
		{
			name: "both schemas absent",
			old:  nil,
			new:  nil,
			want: ClassUnchanged,
		},

		{
			name: "opaque payload type",
			old:  &schema.Schema{},
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			want: ClassIncomparable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := Compat(tc.old, tc.new)

			if got.Class != tc.want {
				t.Fatalf("Compat class = %s, want %s (%s)", got.Class, tc.want, got)
			}

			if len(got.Blocking()) != 0 {
				t.Fatalf("single-schema Compat must never block, got %v", got.Blocking())
			}
		})
	}
}

// TestCompatFindingsNameThePolicyOp pins that every actionable finding carries
// the migration op docs/SCHEMA-COMPATIBILITY.md prescribes.
func TestCompatFindingsNameThePolicyOp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		old  *schema.Schema
		new  *schema.Schema
		kind Kind
		op   string
	}{
		{
			name: "removal",
			old:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": strProp()}),
			kind: KindFieldRemoved,
			op:   OpRemoveField,
		},
		{
			name: "rename",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"b": strProp()}),
			kind: KindFieldRenamed,
			op:   OpRenameField,
		},
		{
			name: "type change",
			old:  rootObj(map[string]schema.Property{"v": strProp()}),
			new:  rootObj(map[string]schema.Property{"v": intProp()}),
			kind: KindTypeChanged,
			op:   OpTransform,
		},
		{
			name: "addition",
			old:  rootObj(map[string]schema.Property{"a": strProp()}),
			new:  rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}),
			kind: KindFieldAdded,
			op:   OpAddField,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			finding, ok := findKind(Compat(tc.old, tc.new), tc.kind)
			if !ok {
				t.Fatalf("missing finding kind %s", tc.kind)
			}

			if finding.Op != tc.op {
				t.Fatalf("finding %s op = %q, want %q", tc.kind, finding.Op, tc.op)
			}
		})
	}
}

func TestCompatPathConventions(t *testing.T) {
	t.Parallel()

	old := rootObj(map[string]schema.Property{
		"shipping": objProp(map[string]schema.Property{"city": strProp()}),
		"lines":    arrProp(objProp(map[string]schema.Property{"sku": strProp()})),
	})
	new := rootObj(map[string]schema.Property{
		"shipping": objProp(map[string]schema.Property{"postcode": intProp()}),
		"lines":    arrProp(objProp(map[string]schema.Property{"code": intProp()})),
	})

	report := Compat(old, new)

	for _, want := range []string{
		"shipping.city",
		"shipping.postcode",
		"lines[].sku",
		"lines[].code",
	} {
		if _, ok := findKindAt(report, want); !ok {
			t.Fatalf("missing finding at path %q; got %v", want, pathsOf(report))
		}
	}
}

func TestReportStringAndOK(t *testing.T) {
	t.Parallel()

	clean := Compat(
		rootObj(map[string]schema.Property{"a": strProp()}),
		rootObj(map[string]schema.Property{"a": strProp()}),
	)
	if !clean.OK() {
		t.Fatalf("identical shapes should be OK, got %s", clean)
	}

	if got := clean.String(); got == "" {
		t.Fatal("Report.String() must not be empty")
	}

	breaking := Compat(rootObj(map[string]schema.Property{"a": strProp()}), rootObj(nil))
	if breaking.OK() {
		t.Fatal("a removal must not be OK")
	}

	if line := breaking.String(); len(line) <= len(breaking.Class.String()) {
		t.Fatalf("Report.String() should list findings, got %q", line)
	}
}

func findKindAt(r Report, path string) (Finding, bool) {
	for _, f := range r.Findings {
		if f.Path == path {
			return f, true
		}
	}

	return Finding{}, false
}

func pathsOf(r Report) []string {
	paths := make([]string, 0, len(r.Findings))
	for _, f := range r.Findings {
		paths = append(paths, f.Path)
	}

	return paths
}
