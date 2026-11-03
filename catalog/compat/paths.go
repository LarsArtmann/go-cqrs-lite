package compat

import (
	"reflect"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func sortedKeys[V any](m map[string]V) []string {
	// art-dupl:accept the collect-and-sort map-keys idiom is irreducible Go; twins in diagnostics/plan_diff are the same 6-line idiom, not shared logic
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func stringSet(values []string) map[string]bool {
	// art-dupl:accept the build-a-set idiom is irreducible Go; the plan_diff twin is the same idiom, not shared logic
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
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

// propertiesEqual is the rename heuristic's shape test. A wrong answer only
// costs a less precise op hint (a removal plus an addition instead of a
// rename); it never changes the verdict class.
func propertiesEqual(a, b schema.Property) bool {
	return reflect.DeepEqual(a, b)
}
