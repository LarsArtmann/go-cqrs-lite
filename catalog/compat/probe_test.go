package compat

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// TestProbeSpikeFixtures is a temporary probe; it always passes and prints the
// differ's class for every fixture the schema model can express.
func TestProbeSpikeFixtures(t *testing.T) {
	type c struct {
		name string
		old  *schema.Schema
		new  *schema.Schema
	}
	cases := []c{
		{"A1 add optional field", rootObj(map[string]schema.Property{"id": strProp()}, "id"), rootObj(map[string]schema.Property{"id": strProp(), "note": strProp()}, "id")},
		{"A2 add optional nested", rootObj(map[string]schema.Property{"meta": objProp(map[string]schema.Property{"a": strProp()})}), rootObj(map[string]schema.Property{"meta": objProp(map[string]schema.Property{"a": strProp(), "b": strProp()})})},
		{"A3 add field in items", rootObj(map[string]schema.Property{"xs": arrProp(objProp(map[string]schema.Property{"a": strProp()}))}), rootObj(map[string]schema.Property{"xs": arrProp(objProp(map[string]schema.Property{"a": strProp(), "b": strProp()}))})},
		{"A4 widen int to number", rootObj(map[string]schema.Property{"n": intProp()}), rootObj(map[string]schema.Property{"n": numProp()})},
		{"A5 add enum value", rootObj(map[string]schema.Property{"s": enumProp("a")}), rootObj(map[string]schema.Property{"s": enumProp("a", "b")})},
		{"A6 required becomes optional", rootObj(map[string]schema.Property{"a": strProp()}, "a"), rootObj(map[string]schema.Property{"a": strProp()})},
		{"B1 remove field", rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}), rootObj(map[string]schema.Property{"a": strProp()})},
		{"B2 rename field", rootObj(map[string]schema.Property{"a": strProp()}), rootObj(map[string]schema.Property{"b": strProp()})},
		{"B3 change field type", rootObj(map[string]schema.Property{"a": strProp()}), rootObj(map[string]schema.Property{"a": numProp()})},
		{"B4 add required field", rootObj(map[string]schema.Property{"a": strProp()}, "a"), rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}, "a", "b")},
		{"B5 narrow enum", rootObj(map[string]schema.Property{"s": enumProp("a", "b")}), rootObj(map[string]schema.Property{"s": enumProp("a")})},
		{"B6 become non nullable", rootObj(map[string]schema.Property{"a": {Type: schema.TypeString, Nullable: true}}), rootObj(map[string]schema.Property{"a": strProp()})},
		{"H3 flatten restructure", rootObj(map[string]schema.Property{"a": objProp(map[string]schema.Property{"b": strProp()})}), rootObj(map[string]schema.Property{"a_b": strProp()})},
		{"H4 add pattern", rootObj(map[string]schema.Property{"a": strProp()}), rootObj(map[string]schema.Property{"a": {Type: schema.TypeString, Pattern: "^[a-z]+$"}})},
		{"H5 relax pattern", rootObj(map[string]schema.Property{"a": {Type: schema.TypeString, Pattern: "^[a-z]+$"}}), rootObj(map[string]schema.Property{"a": strProp()})},
		{"H7 change format", rootObj(map[string]schema.Property{"a": {Type: schema.TypeString, Format: "date-time"}}), rootObj(map[string]schema.Property{"a": {Type: schema.TypeString, Format: "uuid"}})},
		{"H8 become nullable", rootObj(map[string]schema.Property{"a": strProp()}), rootObj(map[string]schema.Property{"a": {Type: schema.TypeString, Nullable: true}})},
		{"H9 add array items constraint", rootObj(map[string]schema.Property{"xs": {Type: schema.TypeArray}}), rootObj(map[string]schema.Property{"xs": arrProp(strProp())})},
		{"H10 add required next to existing", rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}, "a"), rootObj(map[string]schema.Property{"a": strProp(), "b": strProp()}, "a", "b")},
		{"H11 nested remove field", rootObj(map[string]schema.Property{"m": objProp(map[string]schema.Property{"a": strProp(), "b": strProp()})}), rootObj(map[string]schema.Property{"m": objProp(map[string]schema.Property{"a": strProp()})})},
		{"H12 int32 to int64 widen", rootObj(map[string]schema.Property{"n": {Type: schema.TypeInteger, Format: "int32"}}), rootObj(map[string]schema.Property{"n": {Type: schema.TypeInteger, Format: "int64"}})},
	}

	for _, tc := range cases {
		r := Compat(tc.old, tc.new)
		fmt.Printf("%-32s %-22s %v\n", tc.name, r.Class, uniqueKinds(r))
	}
}
