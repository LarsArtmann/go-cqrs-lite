package event

import (
	"slices"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// Clone returns a deep copy of the event. The returned event is fully independent —
// mutations to its payload or metadata will not affect the original.
func (e *ImmutableEvent) Clone() *ImmutableEvent {
	var clonedOpts *eventOptions
	if e.opts != nil {
		cloned := *e.opts
		clonedOpts = &cloned
	}

	return &ImmutableEvent{
		id:            e.id,
		eventType:     e.eventType,
		aggregateID:   e.aggregateID,
		aggregateType: e.aggregateType,
		version:       e.version,
		schemaVersion: e.schemaVersion,
		encoding:      e.encoding,
		payload:       slices.Clone(e.payload),
		metadata:      e.Metadata(),
		occurredAt:    e.occurredAt,
		opts:          clonedOpts,
	}
}

// NewEvent creates a new event with validation.
func NewEvent(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload []byte,
	opts ...Option,
) (*ImmutableEvent, error) {
	err := validateEventParams(
		eventType,
		aggregateID,
		aggregateType,
		version,
		payload,
	)
	if err != nil {
		return nil, err
	}

	safePayload := slices.Clone(payload)

	return buildEvent(eventType, aggregateID, aggregateType, version, safePayload, opts), nil
}

// buildEvent constructs an ImmutableEvent from already-validated, already-copied fields.
func buildEvent(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload []byte,
	opts []Option,
) *ImmutableEvent {
	schemaV, _ := ParseSchemaVersion(1)

	evt := &ImmutableEvent{
		id:            id.NewEventID(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
		schemaVersion: schemaV,
		payload:       payload,
		metadata:      NewMetadata(),
	}

	for _, opt := range opts {
		opt(evt)
	}

	if evt.occurredAt.IsZero() {
		clk := defaultClock
		if evt.opts != nil && evt.opts.clock != nil {
			clk = evt.opts.clock
		}
		evt.occurredAt = clk()
	}

	return evt
}
