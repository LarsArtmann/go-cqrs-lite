package listing

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// StreamListing is a summary of an event stream.
// No derived state. Status is computed separately by the reader.
type StreamListing struct {
	ID          id.StreamID   `json:"id"`
	Type        id.StreamType `json:"type"`
	Version     event.Version `json:"version"`
	EventCount  uint          `json:"event_count"`   //nolint:tagliatelle // on-disk/external format uses snake_case
	LastEventAt time.Time     `json:"last_event_at"` //nolint:tagliatelle // on-disk/external format uses snake_case
}

// Deprecated: use StreamListing.
type AggregateListing = StreamListing

// StreamStatus pairs a stream listing with its computed tombstone state.
type StreamStatus struct {
	Ref    StreamListing
	Status event.TombstoneStatus
}

// Deprecated: use StreamStatus.
type AggregateStatus = StreamStatus

// Page is a cursor-based page of results.
// No TotalCount — append-only logs make counts stale and expensive.
type Page[T any] struct {
	Items   []T  `json:"items"`
	HasMore bool `json:"hasMore"`
}

// TombstonePolicy controls visibility of soft-deleted streams.
type TombstonePolicy int

const (
	// TombstoneExclude hides tombstoned streams (default).
	TombstoneExclude TombstonePolicy = iota
	// TombstoneInclude shows all streams, with Status.
	TombstoneInclude
	// TombstoneOnly shows only tombstoned streams.
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

func (s StreamStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct { //nolint:wrapcheck // JSON serialization
		StreamListing

		Status string `json:"status"`
	}{
		StreamListing: s.Ref,
		Status:        s.Status.String(),
	})
}

// ListOptions controls stream listing queries.
type ListOptions struct {
	// Type is the stream type to list. Required for cursor pagination.
	Type id.StreamType

	// After is the cursor for the next page.
	// Pass the last StreamListing.ID from the previous Page.
	After id.StreamID

	// Limit is the maximum number of items per page.
	// Zero defaults to the reader's default page size.
	Limit uint

	// Tombstone controls visibility of soft-deleted streams.
	// Default is TombstoneExclude.
	Tombstone TombstonePolicy
}
