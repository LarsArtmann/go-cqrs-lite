package event

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/id"
)

// Clone returns a deep copy of the event. The returned event is fully independent —
// mutations to its payload or metadata will not affect the original.
func (e *ImmutableEvent) Clone() *ImmutableEvent {
	var payloadCopy []byte
	if e.payload != nil {
		payloadCopy = make([]byte, len(e.payload))
		copy(payloadCopy, e.payload)
	}

	return &ImmutableEvent{
		id:            e.id,
		eventType:     e.eventType,
		aggregateID:   e.aggregateID,
		aggregateType: e.aggregateType,
		version:       e.version,
		schemaVersion: e.schemaVersion,
		encoding:      e.encoding,
		payload:       payloadCopy,
		metadata:      e.Metadata(),
		occurredAt:    e.occurredAt,
		clock:         e.clock,
		deadline:      e.deadline,
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

	var safePayload []byte
	if payload != nil {
		safePayload = make([]byte, len(payload))
		copy(safePayload, payload)
	}

	schemaV, _ := ParseSchemaVersion(1)

	clk := defaultClock

	evt := &ImmutableEvent{
		id:            id.NewEventID(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
		schemaVersion: schemaV,
		payload:       safePayload,
		metadata:      NewMetadata(),
		occurredAt:    time.Time{},
		clock:         clk,
	}

	for _, opt := range opts {
		opt(evt)
	}

	if evt.occurredAt.IsZero() {
		evt.occurredAt = evt.clock()
	}

	return evt, nil
}
