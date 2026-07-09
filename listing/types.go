package listing

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// AggregateListing is a summary of an aggregate stream.
// No derived state. Status is computed separately by the reader.
type AggregateListing struct {
	ID          id.AggregateID      `json:"id"`
	Type        event.AggregateType `json:"type"`
	Version     event.Version       `json:"version"`
	EventCount  uint                `json:"event_count"`   //nolint:tagliatelle // on-disk/external format uses snake_case
	LastEventAt time.Time           `json:"last_event_at"` //nolint:tagliatelle // on-disk/external format uses snake_case
}

// AggregateStatus pairs an aggregate with its computed tombstone state.
type AggregateStatus struct {
	Ref    AggregateListing
	Status event.TombstoneStatus
}

// Page is a cursor-based page of results.
// No TotalCount — append-only logs make counts stale and expensive.
type Page[T any] struct {
	Items   []T  `json:"items"`
	HasMore bool `json:"hasMore"`
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

func (p TombstonePolicy) String() string {
	switch p {
	case TombstoneExclude:
		return "exclude"
	case TombstoneInclude:
		return "include"
	case TombstoneOnly:
		return "only"
	default:
		return fmt.Sprintf("TombstonePolicy(%d)", p)
	}
}

func (s AggregateStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct { //nolint:wrapcheck // JSON serialization
		AggregateListing

		Status string `json:"status"`
	}{
		AggregateListing: s.Ref,
		Status:           s.Status.String(),
	})
}

// ListOptions controls aggregate listing queries.
type ListOptions struct {
	// Type is the aggregate type to list. Required for cursor pagination.
	Type event.AggregateType

	// After is the cursor for the next page.
	// Pass the last AggregateListing.ID from the previous Page.
	After id.AggregateID

	// Limit is the maximum number of items per page.
	// Zero defaults to the reader's default page size.
	Limit uint

	// Tombstone controls visibility of soft-deleted aggregates.
	// Default is TombstoneExclude.
	Tombstone TombstonePolicy
}
