package compat

import (
	"sort"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func strProp() schema.Property {
	return schema.Property{Type: schema.TypeString}
}

func intProp() schema.Property {
	return schema.Property{Type: schema.TypeInteger}
}

func numProp() schema.Property {
	return schema.Property{Type: schema.TypeNumber}
}

func objProp(props map[string]schema.Property, required ...string) schema.Property {
	return schema.Property{Type: schema.TypeObject, Properties: props, Required: required}
}

func arrProp(items schema.Property) schema.Property {
	return schema.Property{Type: schema.TypeArray, Items: &items}
}

func rootObj(props map[string]schema.Property, required ...string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeObject, Properties: props, Required: required}
}

func rootArr(items schema.Property) *schema.Schema {
	return &schema.Schema{Type: schema.TypeArray, Items: &items}
}

func enumProp(values ...string) schema.Property {
	return schema.Property{Type: schema.TypeString, Enum: values}
}

func hasKind(r Report, kind Kind) bool {
	for _, f := range r.Findings {
		if f.Kind == kind {
			return true
		}
	}

	return false
}

func findKind(r Report, kind Kind) (Finding, bool) {
	for _, f := range r.Findings {
		if f.Kind == kind {
			return f, true
		}
	}

	return Finding{}, false
}

func uniqueKinds(r Report) []string {
	seen := make(map[string]bool, len(r.Findings))
	kinds := make([]string, 0, len(r.Findings))

	for _, f := range r.Findings {
		if seen[string(f.Kind)] {
			continue
		}

		seen[string(f.Kind)] = true
		kinds = append(kinds, string(f.Kind))
	}

	sort.Strings(kinds)

	return kinds
}
