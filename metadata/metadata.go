package metadata

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// Tracing holds the cross-cutting tracing identifiers shared by event,
// command, and query metadata. Each module embeds Tracing rather than
// aliasing event.Metadata, keeping module boundaries clean (ADR-0031).
//
// When embedded anonymously in a struct, encoding/json promotes these
// fields to the parent level, preserving the existing JSON shape:
// {"correlationId": "...", "causationId": "...", ...}.
type Tracing struct {
	CorrelationID id.CorrelationID `json:"correlationId"`
	CausationID   id.CausationID   `json:"causationId"`
	UserID        id.UserID        `json:"userId"`
	RequestID     id.RequestID     `json:"requestId"`
}

// IsZero returns true when no tracing field has been set.
func (t Tracing) IsZero() bool {
	return t.CorrelationID.IsZero() &&
		t.CausationID.IsZero() &&
		t.UserID.IsZero() &&
		t.RequestID.IsZero()
}

// Merge returns a Tracing with non-zero fields from other overlaid onto t.
func (t Tracing) Merge(other Tracing) Tracing {
	result := t

	if !other.CorrelationID.IsZero() {
		result.CorrelationID = other.CorrelationID
	}

	if !other.CausationID.IsZero() {
		result.CausationID = other.CausationID
	}

	if !other.UserID.IsZero() {
		result.UserID = other.UserID
	}

	if !other.RequestID.IsZero() {
		result.RequestID = other.RequestID
	}

	return result
}

// CustomData is the shared base for command.Metadata and query.Metadata
// (ADR-0031). It carries tracing identifiers and a custom key-value map,
// providing the Clone, Merge, and EnsureCustom operations so each module does
// not duplicate the logic. Each module keeps its own Metadata type (which
// embeds CustomData with its own MetadataKey) — the types stay separate but
// the behaviour is shared.
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
