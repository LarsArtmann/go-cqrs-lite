package event

import (
	"fmt"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"
)

// TombstoneStatus represents the soft-delete state of a stream.
//
// Deprecated: Tombstones violate event-stream immutability (ADR-0114).
// Express deletion as a domain event (e.g. "user.deleted") and handle it in
// your fold function instead. See docs/migration/tombstone-to-domain-events.md.
type TombstoneStatus int

const (
	// TombstoneActive means the stream is live and not soft-deleted.
	TombstoneActive TombstoneStatus = iota
	// TombstoneTombstoned means the stream has been soft-deleted.
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

// IsActive reports whether the stream is active (not tombstoned).
func (s TombstoneStatus) IsActive() bool { return s == TombstoneActive }

// IsTombstoned reports whether the stream is soft-deleted.
func (s TombstoneStatus) IsTombstoned() bool { return s == TombstoneTombstoned }

// IsKnown reports whether the status is determinable (not Undetermined).
func (s TombstoneStatus) IsKnown() bool { return s != TombstoneUndetermined }

// MetadataKeyTombstone marks an event as a tombstone action.
// When present with value "true" on an event, that event's stream
// is considered tombstoned. The tombstone status is determined by the
// LAST event in the stream.
//
// Deprecated: Use domain events for deletion semantics (ADR-0114).
const MetadataKeyTombstone MetadataKey = "tombstone"

// MetadataKeyRebirth marks an event as undoing a tombstone.
//
// Deprecated: Use domain events for restore semantics (ADR-0114).
const MetadataKeyRebirth MetadataKey = "rebirth"

// TombstoneMark is the typed representation of a tombstone or rebirth mark
// on an individual event (ADR-0031). It replaces the stringly-typed
// Custom[MetadataKeyTombstone] = "true" pattern while keeping the Custom
// map entries for v2 backward compatibility.
//
// Deprecated: Use domain events for deletion/restore (ADR-0114).
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
//
// Deprecated: Tombstones violate event-stream immutability (ADR-0114).
// Inspect event types directly instead — e.g. check whether the last event
// is a deletion event like "user.deleted". See
// docs/migration/tombstone-to-domain-events.md for migration patterns.
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
//
// Deprecated: Tombstones violate event-stream immutability (ADR-0114).
// Emit a dedicated deletion event (e.g. "user.deleted") instead of
// mutating metadata. See docs/migration/tombstone-to-domain-events.md.
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
//
// Deprecated: Tombstones violate event-stream immutability (ADR-0114).
// Emit a dedicated restore event (e.g. "user.restored") instead of
// mutating metadata. See docs/migration/tombstone-to-domain-events.md.
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
		return nil, errorfamily.NewRejection("event.nil_event", label+": event is required")
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
		streamID:      evt.StreamID(),
		streamType:    evt.StreamType(),
		version:       evt.Version(),
		schemaVersion: evt.SchemaVersion(),
		encoding:      encodingForCopy(evt),
		payload:       safePayload,
		metadata:      md,
		occurredAt:    evt.OccurredAt(),
		opts:          opts,
	}, nil
}
