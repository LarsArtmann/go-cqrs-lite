package catalog

import (
	"cmp"
	"maps"
	"slices"
)

func sortedCopy[K cmp.Ordered, V any, S any](m map[K]V, copyFn func(V) S) []S {
	keys := slices.Sorted(maps.Keys(m))
	result := make([]S, 0, len(m))
	for _, key := range keys {
		result = append(result, copyFn(m[key]))
	}
	return result
}

func copyPtr[T any](fn func(*T) T, v T) *T {
	cp := fn(&v)
	return &cp
}
