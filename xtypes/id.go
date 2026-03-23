// Package xtypes provides type-safe extensions for go-cqrs-lite.
//
// This package offers strongly-typed wrappers around the core types,
// using branded identifiers to prevent mixing up IDs at compile time.
//
// Example:
//
//	userID := id.New[id.UserID]()
//	evt, _ := xtypes.NewEventBuilder("UserCreated", userID, "User", 1).Build()
package xtypes

import (
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

// Re-export ID types from pkg/id for convenience

type (
	// AggregateID is a strongly-typed identifier for aggregate roots.
	AggregateID = id.AggregateID
	// EventID is a strongly-typed identifier for domain events.
	EventID = id.EventID
	// UserID is a strongly-typed identifier for users.
	UserID = id.UserID
	// CorrelationID is a strongly-typed identifier for distributed tracing.
	CorrelationID = id.CorrelationID
	// CausationID is a strongly-typed identifier for causation tracking.
	CausationID = id.CausationID
	// RequestID is a strongly-typed identifier for HTTP requests.
	RequestID = id.RequestID
)

// commandMarker is a phantom type for branding CommandIDs.
type commandMarker struct{}

// CommandID is a strongly-typed identifier for commands (for idempotency).
type CommandID = id.Of[commandMarker]

// NewCommandID generates a new random CommandID.
func NewCommandID() CommandID {
	return id.New[commandMarker]()
}

// ParseCommandID converts a string to a CommandID.
func ParseCommandID(s string) (CommandID, error) {
	return id.Parse[commandMarker](s)
}
