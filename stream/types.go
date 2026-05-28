package stream

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// AggregateRef is a lightweight identity reference to an aggregate stream.
// No derived state. Status is computed separately by the reader.
type AggregateRef struct {
	ID          id.AggregateID
	Type        event.AggregateType
	Version     event.Version
	EventCount  uint
	LastEventAt time.Time
}

// AggregateStatus pairs an aggregate with its computed tombstone state.
type AggregateStatus struct {
	Ref    AggregateRef
	Status event.TombstoneStatus
}

// Page is a cursor-based page of results.
// No TotalCount — append-only logs make counts stale and expensive.
type Page[T any] struct {
	Items   []T
	HasMore bool
}

// TombstonePolicy controls visibility of soft-deleted aggregates.
type TombstonePolicy int

const (
	// TombstoneExclude hides tombstoned aggregates (default).
	TombstoneExclude TombstonePolicy = iota
	// TombstoneInclude shows all aggregates, with Status.
	TombstoneInclude
	// TombstoneOnly shows only tombstoned aggregates.
	TombstoneOnly
)

// ListOptions controls aggregate listing queries.
type ListOptions struct {
	// Type is the aggregate type to list. Required for cursor pagination.
	Type event.AggregateType

	// After is the cursor for the next page.
	// Pass the last AggregateRef.ID from the previous Page.
	After id.AggregateID

	// Limit is the maximum number of items per page.
	// Zero defaults to the reader's default page size.
	Limit uint

	// Tombstone controls visibility of soft-deleted aggregates.
	// Default is TombstoneExclude.
	Tombstone TombstonePolicy
}
