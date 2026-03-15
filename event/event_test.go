package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
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

func TestEventOptions(t *testing.T) {
	evt, err := event.NewEvent(
		"TestEvent",
		"agg-123",
		"TestAggregate",
		1,
		nil,
		event.WithCorrelationID("corr-123"),
		event.WithCausationID("cause-456"),
		event.WithUserID("user-789"),
		event.WithRequestID("req-001"),
		event.WithSource("test-service"),
		event.WithIPAddress("127.0.0.1"),
		event.WithUserAgent("test-agent"),
		event.WithCustom("key1", "value1"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := evt.Metadata()
	if m.CorrelationID != "corr-123" {
		t.Errorf("expected correlation ID corr-123, got %s", m.CorrelationID)
	}
	if m.CausationID != "cause-456" {
		t.Errorf("expected causation ID cause-456, got %s", m.CausationID)
	}
	if m.UserID != "user-789" {
		t.Errorf("expected user ID user-789, got %s", m.UserID)
	}
	if m.RequestID != "req-001" {
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
