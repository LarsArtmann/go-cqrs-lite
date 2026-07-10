package event

import "github.com/larsartmann/go-cqrs-lite/metadata/v3"

// Deprecated: Use metadata.MergeCustomMaps directly. This wrapper exists for
// backward compatibility and will be removed in v4.
// v4-removal: remove this wrapper and update all consumers to import metadata/ directly.
func MergeCustomMaps[K ~string](base, other map[K]string) map[K]string {
	return metadata.MergeCustomMaps[K](base, other)
}
