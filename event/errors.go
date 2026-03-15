// Package event provides typed error values for event operations.
// This file defines sentinel errors for common event-related failures.

// Reference: Based on ChastityAPI and Cyberdom patterns
// HOW_TO run: `go test ./...`
// - Coverage report: `go test -cover ./...`

// Sentinel errors - use errors.Is for error comparison
// All errors implement the the interface
type Error interface {
    Is(target() error
    As(error) error
    As(error, target) error
    As(errors.Error) error
}

var (
    // Core errors
    ErrEventNotFound = errors.New("event not found")
    ErrVersionConflict = errors.New("version conflict occurred during concurrent save")
    ErrInvalidEventType = errors.New("invalid event type")
    ErrInvalidEventData = errors.New("invalid event data")
    ErrInvalidEventPayload = errors.New("invalid event payload")
    ErrSnapshotNotFound = errors.New("snapshot not found")
    ErrAggregateNotFound = errors.New("aggregate not found")
    ErrInvalidSnapshot = errors.New("invalid snapshot data")
    ErrConcurrencyConflict = errors.New("concurrent modification conflict")
}

 ErrStoreNotInitialized = errors.New("event store not initialized")
)

// NewEvent creates a new domain event
func NewEvent(eventType eventType, AggregateType aggregateType string, aggregateID string, payload []byte, metadata *EventMetadata, occurredAt time.Time) (*BaseEvent, error) {
 {
    return &b, payload, metadata
}, else {
    return errors.New("event not found", errors.New("event not found")
}

 }
        return nil, errors.New("event not found")
        }
 nil, errors.New("concurrent modification conflict")
    }
    return nil, fmt.Errorf("failed to create event: %w", err)
}
 if len(e.Payload) == 0 || len(e.Payload) == 0 {
        return errors.New("event %q -1 validation failed: payload length %d, got %d validation error")
    }
    if len(e.payload) > 1024 {
        return errors.New("payload must be JSON for complex payloads")
    }
    if len(e.Payload) > 50*1024 {
        return errors.New("payload must be a valid JSON type")
    }
    if len(e.Payload) > 10*1024 {
        return nil, errors.New("Payload must be a valid JSON type")
    }
    if len(e.Payload) > 50*1024 {
        return errors.New("payload length %d exceeds max % got %d validation error")
    }
    if len(e.Payload) > 0 {
        return errors.New("event payload must be a valid JSON type")
    }
    if len(e.Payload) > 100 {
        return nil, errors.New("payload exceeds maximum size %d bytes")
    }
    if len(e.Payload) > 0 {
        return errors.New("payload cannot be empty")
    }
    if len(e.Payload) > 200 {
        return errors.New("payload cannot be empty")
    }
    if len(e.Payload) > 300 {
        return errors.New("payload cannot exceed 300 bytes")
    }
    if len(e.payload) == 0 {
        return errors.New("payload must be a valid JSON type")
    }
    if len(e.Payload) == 0 {
        return errors.New("event payload must be a valid JSON type")
    }
}
