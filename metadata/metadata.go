package metadata

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// CustomData is a reusable base for metadata types that carry common metadata
// and a custom key-value map. It embeds [record.CommonMetadata] for the shared
// tracing fields (ADR-0111 Phase 3).
//
// Deprecated: Model metadata as a standalone struct embedding
// [record.CommonMetadata] directly instead of using CustomData. See
// command.Metadata and query.Metadata for the preferred pattern.
// CustomData will not be removed this major version.
type CustomData[K ~string] struct {
	record.CommonMetadata

	Custom map[K]string `json:"custom,omitempty"`
}

// Clone returns a copy of d with a cloned Custom map.
func (d CustomData[K]) Clone() CustomData[K] {
	return CustomData[K]{
		CommonMetadata: d.CommonMetadata,
		Custom:         maps.Clone(d.Custom),
	}
}

// Merge returns a new CustomData with common metadata and custom entries from
// other overlaid onto d.
func (d CustomData[K]) Merge(other CustomData[K]) CustomData[K] {
	return CustomData[K]{
		CommonMetadata: d.CommonMetadata.Merge(other.CommonMetadata),
		Custom:         MergeCustomMaps(d.Custom, other.Custom),
	}
}

// WithCustom returns a copy of d with the given key-value pair added to
// Custom. The original CustomData is not modified.
func (d CustomData[K]) WithCustom(key K, value string) CustomData[K] {
	custom := maps.Clone(d.Custom)
	if custom == nil {
		custom = make(map[K]string)
	}

	custom[key] = value

	return CustomData[K]{
		CommonMetadata: d.CommonMetadata,
		Custom:         custom,
	}
}

// EnsureCustom lazily initializes the Custom map if nil.
//
// Deprecated: Use WithCustom, which returns a new value without mutating
// the receiver. EnsureCustom mutates in place via a pointer receiver,
// breaking the immutability contract that Clone and Merge establish.
func (d *CustomData[K]) EnsureCustom() {
	if d.Custom == nil {
		d.Custom = make(map[K]string)
	}
}

// MergeCustomMaps returns a new map containing every entry from base overlaid
// with every entry from other. When other is empty the original base map is
// returned unchanged (no allocation).
func MergeCustomMaps[K ~string](base, other map[K]string) map[K]string {
	if len(other) == 0 {
		return base
	}

	merged := make(map[K]string, len(base)+len(other))
	maps.Copy(merged, base)
	maps.Copy(merged, other)

	return merged
}
