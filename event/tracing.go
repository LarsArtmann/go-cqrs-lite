package event

import "github.com/larsartmann/go-cqrs-lite/id/v3"

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
