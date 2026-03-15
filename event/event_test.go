package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func TestNewEvent(t *testing.T) {
	tests := []struct {
		name          string
		eventType     event.EventType
		aggregateID   string
		aggregateType event.AggregateType
		version       int
		wantErr       bool
	}{
		{
			name:          "valid event",
			eventType:     "UserCreated",
			aggregateID:   "user-123",
			aggregateType: "User",
			version:       1,
			wantErr:       false,
		},
		{
			name:          "missing aggregate ID",
			eventType:     "UserCreated",
			aggregateID:   "",
			aggregateType: "User",
			version:       1,
			wantErr:       true,
		},
		{
			name:          "missing aggregate type",
			eventType:     "UserCreated",
			aggregateID:   "user-123",
			aggregateType: "",
			version:       1,
			wantErr:       true,
		},
		{
			name:          "negative version",
			eventType:     "UserCreated",
			aggregateID:   "user-123",
			aggregateType: "User",
			version:       -1,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt, err := event.NewEvent(
				tt.eventType,
				tt.aggregateID,
				tt.aggregateType,
				tt.version,
				[]byte(`{"test":"data"}`),
			)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if evt.ID() == "" {
				t.Error("event ID should not be empty")
			}
			if evt.Type() != tt.eventType {
				t.Errorf("expected type %s, got %s", tt.eventType, evt.Type())
			}
			if evt.AggregateID() != tt.aggregateID {
				t.Errorf("expected aggregate ID %s, got %s", tt.aggregateID, evt.AggregateID())
			}
		})
	}
}

func TestEventOptions(t *testing.T) {
	evt, err := event.NewEvent(
		"TestEvent",
		"agg-123",
		"TestAggregate",
		1,
		nil,
		event.WithCorrelationID("corr-123"),
		event.WithUserID("user-456"),
		event.WithCustom("key", "value"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Metadata().CorrelationID != "corr-123" {
		t.Errorf("expected correlation ID corr-123, got %s", evt.Metadata().CorrelationID)
	}
	if evt.Metadata().UserID != "user-456" {
		t.Errorf("expected user ID user-456, got %s", evt.Metadata().UserID)
	}
	if evt.Metadata().Custom["key"] != "value" {
		t.Errorf("expected custom key=value, got %s", evt.Metadata().Custom["key"])
	}
}
