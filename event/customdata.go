package event

import "maps"

// CustomData is the shared base for command.Metadata and query.Metadata
// (ADR-0031). It carries tracing identifiers and a custom key-value map,
// providing the Clone, Merge, and EnsureCustom operations so each module does
// not duplicate the logic. Each module keeps its own Metadata type (which
// embeds CustomData with its own MetadataKey) — the types stay separate but
// the behaviour is shared.
//
// event.Metadata does NOT use CustomData: it has additional fields (Tombstone,
// Causation, Source, etc.) that require its own Clone/Merge.
type CustomData[K ~string] struct {
	Tracing
	Custom map[K]string `json:"custom,omitempty"`
}

// Clone returns a copy of d with a cloned Custom map.
func (d CustomData[K]) Clone() CustomData[K] {
	return CustomData[K]{
		Tracing: d.Tracing,
		Custom:  maps.Clone(d.Custom),
	}
}

// Merge returns a new CustomData with tracing and custom entries from other
// overlaid onto d.
func (d CustomData[K]) Merge(other CustomData[K]) CustomData[K] {
	return CustomData[K]{
		Tracing: d.Tracing.Merge(other.Tracing),
		Custom:  MergeCustomMaps(d.Custom, other.Custom),
	}
}

// EnsureCustom lazily initializes the Custom map if nil.
func (d *CustomData[K]) EnsureCustom() {
	if d.Custom == nil {
		d.Custom = make(map[K]string)
	}
}
