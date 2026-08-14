package metadata

import (
	"maps"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
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
	ActorID       id.ActorID       `json:"actorId,omitempty"`
}

// IsZero returns true when no tracing field has been set.
func (t Tracing) IsZero() bool {
	return t.CorrelationID.IsZero() &&
		t.CausationID.IsZero() &&
		t.UserID.IsZero() &&
		t.RequestID.IsZero() &&
		t.ActorID.IsZero()
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

	if !other.ActorID.IsZero() {
		result.ActorID = other.ActorID
	}

	return result
}

// Metadata is the canonical metadata shape shared by all record types:
// tracing identifiers plus a custom key-value map (ADR-0031, WAL unification).
//
// command.Metadata and query.Metadata are type aliases for this generic with
// their module-local key types. The key type parameter keeps the module
// boundaries clean: a command.MetadataKey never collides with a
// query.MetadataKey, and neither gains event-only fields (Tombstone,
// Causation).
//
// When embedded anonymously in a struct, encoding/json promotes these fields
// to the parent level, preserving the JSON shape:
// {"correlationId": "...", "causationId": "...", "custom": {...}}.
type Metadata[K ~string] struct {
	Tracing

	Custom map[K]string `json:"custom,omitempty"`
}

// Clone returns a copy of m with a cloned Custom map.
func (m Metadata[K]) Clone() Metadata[K] {
	return Metadata[K]{
		Tracing: m.Tracing,
		Custom:  maps.Clone(m.Custom),
	}
}

// Merge returns a new Metadata with tracing and custom entries from other
// overlaid onto m.
func (m Metadata[K]) Merge(other Metadata[K]) Metadata[K] {
	return Metadata[K]{
		Tracing: m.Tracing.Merge(other.Tracing),
		Custom:  MergeCustomMaps(m.Custom, other.Custom),
	}
}

// WithCustom returns a copy of m with the given key-value pair added to
// Custom. The original Metadata is not modified.
func (m Metadata[K]) WithCustom(key K, value string) Metadata[K] {
	custom := maps.Clone(m.Custom)
	if custom == nil {
		custom = make(map[K]string)
	}

	custom[key] = value

	return Metadata[K]{
		Tracing: m.Tracing,
		Custom:  custom,
	}
}

// EnsureCustom lazily initializes the Custom map if nil.
//
// Deprecated: Use WithCustom, which returns a new value without mutating
// the receiver. EnsureCustom mutates in place via a pointer receiver,
// breaking the immutability contract that Clone and Merge establish.
func (m *Metadata[K]) EnsureCustom() {
	if m.Custom == nil {
		m.Custom = make(map[K]string)
	}
}

// CustomData is the pre-unification name for [Metadata].
//
// Deprecated: Use Metadata[K] directly. CustomData is kept as a generic type
// alias so existing consumer code keeps compiling; it will not be removed
// this major version.
type CustomData[K ~string] = Metadata[K]

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
