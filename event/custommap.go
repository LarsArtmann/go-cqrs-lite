package event

import "maps"

// MergeCustomMaps returns a new map containing every entry from base overlaid
// with every entry from other. When other is empty the original base map is
// returned unchanged (no allocation).
//
// This is the shared merge logic for the Custom maps carried by event,
// command, and query metadata (ADR-0031). It is generic over the key type so
// each module can use its own strongly-typed MetadataKey without duplicating
// the copy/overlay algorithm.
func MergeCustomMaps[K ~string](base, other map[K]string) map[K]string {
	if len(other) == 0 {
		return base
	}

	merged := make(map[K]string, len(base)+len(other))
	maps.Copy(merged, base)
	maps.Copy(merged, other)

	return merged
}
