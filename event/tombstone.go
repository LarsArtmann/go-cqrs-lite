package event

import (
	"fmt"
	"slices"
)

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

// TombstoneMark is the typed representation of a tombstone or rebirth mark
// on an individual event (ADR-0031). It replaces the stringly-typed
// Custom[MetadataKeyTombstone] = "true" pattern while keeping the Custom
// map entries for v2 backward compatibility.
type TombstoneMark struct {
	// Status is TombstoneTombstoned for a tombstone event, or
	// TombstoneActive for a rebirth event.
	Status TombstoneStatus
	// Reason is an optional human-readable note for audit trails.
	Reason string
}

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

	// Typed field takes precedence (ADR-0031).
	if md.Tombstone != nil {
		return md.Tombstone.Status
	}

	// Fall back to string-based Custom map for backward compatibility.
	if md.Custom != nil {
		if md.Custom[MetadataKeyRebirth] == "true" {
			return TombstoneActive
		}

		if md.Custom[MetadataKeyTombstone] == "true" {
			return TombstoneTombstoned
		}
	}

	return TombstoneUndetermined
}

// MarkTombstone copies an event and sets the tombstone metadata key.
// Returns a new event; the original is unmodified.
func MarkTombstone(evt Event) (*ImmutableEvent, error) {
	return copyWithTombstoneMark(
		evt,
		TombstoneMark{Status: TombstoneTombstoned},
		MetadataKeyTombstone,
		"mark tombstone",
	)
}

// MarkRebirth copies an event and sets the rebirth metadata key.
// Returns a new event; the original is unmodified.
func MarkRebirth(evt Event) (*ImmutableEvent, error) {
	return copyWithTombstoneMark(
		evt,
		TombstoneMark{Status: TombstoneActive},
		MetadataKeyRebirth,
		"mark rebirth",
	)
}

func copyWithTombstoneMark(
	evt Event,
	mark TombstoneMark,
	key MetadataKey,
	label string,
) (*ImmutableEvent, error) {
	if evt == nil {
		return nil, NewRejection("event.nil_event", label+": event is required")
	}

	rawPayload := payloadForDecode(evt)

	safePayload := slices.Clone(rawPayload)

	md := evt.Metadata()
	if md.Custom == nil {
		md.Custom = make(map[MetadataKey]string)
	}

	md.Custom[key] = "true"
	md.Tombstone = &mark

	deadline, hasDeadline := evt.Deadline()

	var opts *eventOptions
	if hasDeadline {
		opts = &eventOptions{deadline: deadline}
	}

	return &ImmutableEvent{
		id:            evt.ID(),
		eventType:     evt.Type(),
		aggregateID:   evt.AggregateID(),
		aggregateType: evt.AggregateType(),
		version:       evt.Version(),
		schemaVersion: evt.SchemaVersion(),
		encoding:      encodingForCopy(evt),
		payload:       safePayload,
		metadata:      md,
		occurredAt:    evt.OccurredAt(),
		opts:          opts,
	}, nil
}
