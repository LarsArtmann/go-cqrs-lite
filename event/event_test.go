package event_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

func TestNewEvent_Valid(t *testing.T) {
	evt, err := event.NewEvent(
		"UserCreated",
		"user-123",
		"User",
		1,
		[]byte(`{"name":"John"}`),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.ID() == "" {
		t.Error("event ID should not be empty")
	}
	if evt.Type() != "UserCreated" {
		t.Errorf("expected type UserCreated, got %s", evt.Type())
	}
	if evt.AggregateID() != "user-123" {
		t.Errorf("expected aggregate ID user-123, got %s", evt.AggregateID())
	}
	if evt.AggregateType() != "User" {
		t.Errorf("expected aggregate type User, got %s", evt.AggregateType())
	}
	if evt.Version() != 1 {
		t.Errorf("expected version 1, got %d", evt.Version())
	}
}

func TestNewEvent_MissingAggregateID(t *testing.T) {
	_, err := event.NewEvent("UserCreated", "", "User", 1, nil)
	if err == nil {
		t.Error("expected error for missing aggregate ID")
	}
}

func TestNewEvent_MissingAggregateType(t *testing.T) {
	_, err := event.NewEvent("UserCreated", "user-123", "", 1, nil)
	if err == nil {
		t.Error("expected error for missing aggregate type")
	}
}

func TestNewEvent_NegativeVersion(t *testing.T) {
	_, err := event.NewEvent("UserCreated", "user-123", "User", -1, nil)
	if err == nil {
		t.Error("expected error for negative version")
	}
}

func TestNewEvent_ErrorMessagesContainContext(t *testing.T) {
	tests := []struct {
		name          string
		eventType     event.EventType
		aggregateID   id.AggregateID
		aggregateType event.AggregateType
		version       int
		wantContains  []string
	}{
		{
			name:          "missing aggregate ID includes event type",
			eventType:     "TestEvent",
			aggregateID:   "",
			aggregateType: "User",
			version:       1,
			wantContains:  []string{"aggregate ID is required", "TestEvent"},
		},
		{
			name:          "missing aggregate type includes aggregate ID and event type",
			eventType:     "OrderCreated",
			aggregateID:   id.AggregateID("order-456"),
			aggregateType: "",
			version:       1,
			wantContains:  []string{"aggregate type is required", "order-456", "OrderCreated"},
		},
		{
			name:          "negative version includes version, aggregate ID and event type",
			eventType:     "PaymentProcessed",
			aggregateID:   id.AggregateID("payment-789"),
			aggregateType: "Payment",
			version:       -5,
			wantContains:  []string{"version must be non-negative", "-5", "payment-789", "PaymentProcessed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := event.NewEvent(tt.eventType, tt.aggregateID, tt.aggregateType, tt.version, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q does not contain %q", errMsg, want)
				}
			}
		})
	}
}

func TestEventOptions(t *testing.T) {
	evt, err := event.NewEvent(
		"TestEvent",
		id.AggregateID("agg-123"),
		"TestAggregate",
		1,
		nil,
		event.WithCorrelationID(id.CorrelationID("corr-123")),
		event.WithCausationID(id.CausationID("cause-456")),
		event.WithUserID(id.UserID("user-789")),
		event.WithRequestID(id.RequestID("req-001")),
		event.WithSource("test-service"),
		event.WithIPAddress("127.0.0.1"),
		event.WithUserAgent("test-agent"),
		event.WithCustom("key1", "value1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := evt.Metadata()
	if m.CorrelationID != id.CorrelationID("corr-123") {
		t.Errorf("expected correlation ID corr-123, got %s", m.CorrelationID)
	}
	if m.CausationID != id.CausationID("cause-456") {
		t.Errorf("expected causation ID cause-456, got %s", m.CausationID)
	}
	if m.UserID != id.UserID("user-789") {
		t.Errorf("expected user ID user-789, got %s", m.UserID)
	}
	if m.RequestID != id.RequestID("req-001") {
		t.Errorf("expected request ID req-001, got %s", m.RequestID)
	}
	if m.Source != "test-service" {
		t.Errorf("expected source test-service, got %s", m.Source)
	}
	if m.IPAddress != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", m.IPAddress)
	}
	if m.UserAgent != "test-agent" {
		t.Errorf("expected user agent test-agent, got %s", m.UserAgent)
	}
	if m.Custom["key1"] != "value1" {
		t.Errorf("expected custom key1=value1, got %s", m.Custom["key1"])
	}
}
