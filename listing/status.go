package listing

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Status is the lifecycle state of a stream, derived from domain event
// types (ADR-0114): deletion and restoration are domain events, and a
// stream's status follows from its LAST event's type.
//
// Numeric values intentionally match the legacy event.TombstoneStatus
// wire values (active=0, tombstoned=1, undetermined=2).
type Status int

const (
	// StatusActive means the stream is live and not deleted.
	StatusActive Status = iota
	// StatusTombstoned means the stream's last event is a deletion event.
	StatusTombstoned
	// StatusUndetermined means no classifier is configured, so status
	// cannot be derived from event types.
	StatusUndetermined
)

// String returns the human-readable name of the status.
func (s Status) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusTombstoned:
		return "tombstoned"
	case StatusUndetermined:
		return "undetermined"
	default:
		return fmt.Sprintf("Status(%d)", int(s))
	}
}

// IsActive reports whether the stream is active (not deleted).
func (s Status) IsActive() bool { return s == StatusActive }

// IsTombstoned reports whether the stream is deleted.
func (s Status) IsTombstoned() bool { return s == StatusTombstoned }

// IsKnown reports whether the status is determinable (a classifier is
// configured).
func (s Status) IsKnown() bool { return s != StatusUndetermined }

// StatusClassifier derives stream Status from event types (ADR-0114).
//
// A stream whose LAST event is a delete type is StatusTombstoned; a rebirth
// type makes it StatusActive again. Any other last event means StatusActive.
// With no types configured at all, classification returns
// StatusUndetermined — matching the legacy no-metadata behavior.
//
// This replaces metadata tombstone marks (event.MarkTombstone /
// listing.StatusMiddleware), which mutate immutable streams and are
// removed in v5.
type StatusClassifier struct {
	deletes  map[event.Type]struct{}
	rebirths map[event.Type]struct{}
}

// NewStatusClassifier builds a classifier from deletion and rebirth event
// types, e.g.
//
//	listing.NewStatusClassifier(
//	    []event.Type{"user.deleted"},
//	    []event.Type{"user.reactivated"},
//	)
func NewStatusClassifier(deleteTypes, rebirthTypes []event.Type) StatusClassifier {
	return StatusClassifier{
		deletes:  event.NewTypeSet(deleteTypes),
		rebirths: event.NewTypeSet(rebirthTypes),
	}
}

// configured reports whether any event types are classified.
func (c StatusClassifier) configured() bool {
	return len(c.deletes) > 0 || len(c.rebirths) > 0
}

// ClassifyLast returns the Status implied by a stream's last event.
func (c StatusClassifier) ClassifyLast(evt event.Event) Status {
	if !c.configured() {
		return StatusUndetermined
	}

	eventType := evt.Type()

	if _, ok := c.deletes[eventType]; ok {
		return StatusTombstoned
	}

	if _, ok := c.rebirths[eventType]; ok {
		return StatusActive
	}

	return StatusActive
}

// ReaderOption configures an InMemoryStreamReader.
type ReaderOption func(*readerConfig)

type readerConfig struct {
	classifier StatusClassifier
}

// WithStatusClassifier derives each stream's Status from its last event's
// type instead of legacy tombstone metadata.
func WithStatusClassifier(classifier StatusClassifier) ReaderOption {
	return func(cfg *readerConfig) {
		cfg.classifier = classifier
	}
}
