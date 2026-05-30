package listing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
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

type aggregateStatusJSON struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Version    int    `json:"version"`
	EventCount uint   `json:"event_count"`   //nolint:tagliatelle
	LastEvent  string `json:"last_event_at"` //nolint:tagliatelle
	Status     string `json:"status"`
}

func (s AggregateStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal( //nolint:wrapcheck // JSON serialization, not domain error
		aggregateStatusJSON{
			ID:         s.Ref.ID.String(),
			Type:       string(s.Ref.Type),
			Version:    s.Ref.Version.Int(),
			EventCount: s.Ref.EventCount,
			LastEvent:  s.Ref.LastEventAt.Format(time.RFC3339),
			Status:     s.Status.String(),
		},
	)
}

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
