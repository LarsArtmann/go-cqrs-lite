package event

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// New creates a new event with a typed payload.
// If payload is []byte, it is used directly. Structs and maps are marshaled via json.Marshal.
// Returns an error if payload is nil.
func New(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version Version,
	payload any,
	opts ...Option,
) (*Core, error) {
	data, err := marshalPayload(payload, eventType)
	if err != nil {
		return nil, err
	}

	return NewEvent(eventType, aggregateID, aggregateType, version, data, opts...)
}

func marshalPayload(payload any, eventType Type) ([]byte, error) {
	if payload == nil {
		return nil, fmt.Errorf("%w: event type %q", ErrNilPayload, eventType)
	}

	switch v := payload.(type) {
	case []byte:
		return v, nil
	case json.RawMessage:
		return v, nil
	default:
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload for event type %q: %w", eventType, err)
		}

		return data, nil
	}
}
