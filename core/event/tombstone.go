package event

import "fmt"

// TombstoneStatus represents the soft-delete state of an aggregate.
type TombstoneStatus int

const (
	// TombstoneActive means the aggregate is live and not soft-deleted.
	TombstoneActive TombstoneStatus = iota
	// TombstoneTombstoned means the aggregate has been soft-deleted.
	TombstoneTombstoned
	// TombstoneUndetermined means the status cannot be determined
	// (e.g., no tombstone/rebirth metadata found, or no detector configured).
	TombstoneUndetermined
)

// String returns the human-readable name of the tombstone status.
func (s TombstoneStatus) String() string {
	switch s {
	case TombstoneActive:
		return "active"
	case TombstoneTombstoned:
		return "tombstoned"
	case TombstoneUndetermined:
		return "undetermined"
	default:
		return fmt.Sprintf("TombstoneStatus(%d)", s)
	}
}

// IsActive reports whether the aggregate is active (not tombstoned).
func (s TombstoneStatus) IsActive() bool { return s == TombstoneActive }

// IsTombstoned reports whether the aggregate is soft-deleted.
func (s TombstoneStatus) IsTombstoned() bool { return s == TombstoneTombstoned }

// IsKnown reports whether the status is determinable (not Undetermined).
func (s TombstoneStatus) IsKnown() bool { return s != TombstoneUndetermined }

// MetadataKeyTombstone marks an event as a tombstone action.
// When present with value "true" on an event, that event's aggregate
// is considered tombstoned. The tombstone status is determined by the
// LAST event in the stream.
const MetadataKeyTombstone MetadataKey = "tombstone"

// MetadataKeyRebirth marks an event as undoing a tombstone.
const MetadataKeyRebirth MetadataKey = "rebirth"

// DetectTombstone inspects an event stream and returns the tombstone status.
// Returns Undetermined if the stream is empty or no tombstone/rebirth metadata is found.
//
// Rebirth takes precedence (newest event wins).
func DetectTombstone(events []Event) TombstoneStatus {
	if len(events) == 0 {
		return TombstoneUndetermined
	}

	last := events[len(events)-1]
	md := last.Metadata()
	if md.Custom == nil {
		return TombstoneUndetermined
	}

	// Rebirth takes precedence (newest event wins)
	if md.Custom[MetadataKeyRebirth] == "true" {
		return TombstoneActive
	}

	if md.Custom[MetadataKeyTombstone] == "true" {
		return TombstoneTombstoned
	}

	return TombstoneUndetermined
}

// MarkTombstone copies an event and sets the tombstone metadata key.
// Returns a new event; the original is unmodified.
func MarkTombstone(evt Event) (*ImmutableEvent, error) {
	if evt == nil {
		return nil, NewRejection("event.nil_event", "mark tombstone: event is required")
	}

	return NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		evt.Payload(),
		WithEventID(evt.ID()),
		WithOccurredAt(evt.OccurredAt()),
		WithSchemaVersion(evt.SchemaVersion()),
		WithMetadata(evt.Metadata()),
		WithCustom(MetadataKeyTombstone, "true"),
	)
}

// MarkRebirth copies an event and sets the rebirth metadata key.
// Returns a new event; the original is unmodified.
func MarkRebirth(evt Event) (*ImmutableEvent, error) {
	if evt == nil {
		return nil, NewRejection("event.nil_event", "mark rebirth: event is required")
	}

	return NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		evt.Payload(),
		WithEventID(evt.ID()),
		WithOccurredAt(evt.OccurredAt()),
		WithSchemaVersion(evt.SchemaVersion()),
		WithMetadata(evt.Metadata()),
		WithCustom(MetadataKeyRebirth, "true"),
	)
}
