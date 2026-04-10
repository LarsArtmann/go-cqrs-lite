package event_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

func TestNewEvent_Valid(t *testing.T) {
	t.Parallel()

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

func TestNewEvent_InvalidInputErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		eventType     event.Type
		aggregateID   id.AggregateID
		aggregateType event.AggregateType
		version       int
	}{
		{
			name:          "missing aggregate ID",
			eventType:     "UserCreated",
			aggregateID:   "",
			aggregateType: "User",
			version:       1,
		},
		{
			name:          "missing aggregate type",
			eventType:     "UserCreated",
			aggregateID:   "user-123",
			aggregateType: "",
			version:       1,
		},
		{
			name:          "negative version",
			eventType:     "UserCreated",
			aggregateID:   "user-123",
			aggregateType: "User",
			version:       -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := event.NewEvent(
				tt.eventType,
				tt.aggregateID,
				tt.aggregateType,
				tt.version,
				nil,
			)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestNewEvent_ErrorMessagesContainContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		eventType     event.Type
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
			wantContains:  []string{"version -5 invalid", "payment-789", "PaymentProcessed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := event.NewEvent(
				tt.eventType,
				tt.aggregateID,
				tt.aggregateType,
				tt.version,
				nil,
			)
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
	t.Parallel()

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
		event.WithSource(event.Source("test-service")),
		event.WithIPAddress(event.IPAddress("127.0.0.1")),
		event.WithUserAgent(event.UserAgent("test-agent")),
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

	if m.Source.String() != "test-service" {
		t.Errorf("expected source test-service, got %s", m.Source)
	}

	if m.IPAddress.String() != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", m.IPAddress)
	}

	if m.UserAgent.String() != "test-agent" {
		t.Errorf("expected user agent test-agent, got %s", m.UserAgent)
	}

	if m.Custom["key1"] != "value1" {
		t.Errorf("expected custom key1=value1, got %s", m.Custom["key1"])
	}
}

func TestNewMetadata(t *testing.T) {
	t.Parallel()

	m := event.NewMetadata()
	if m == nil {
		t.Fatal("NewMetadata() should return non-nil")
	}

	if m.Custom == nil {
		t.Error("Custom map should be initialized")
	}

	if m.CorrelationID != "" {
		t.Errorf("CorrelationID should be empty, got %s", m.CorrelationID)
	}
}

func TestEventAccessors(t *testing.T) {
	t.Parallel()

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

	if evt.Payload() == nil || string(evt.Payload()) != `{"name":"John"}` {
		t.Errorf("expected payload, got %v", evt.Payload())
	}

	if evt.OccurredAt().IsZero() {
		t.Error("OccurredAt should not be zero")
	}

	if evt.Metadata() == nil {
		t.Error("Metadata should not be nil")
	}
}
