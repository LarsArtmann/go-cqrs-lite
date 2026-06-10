package event

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	b := newBuilder("TestEvent", aggID, "TestAggregate", Version(1))

	evt, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Type() != "TestEvent" {
		t.Errorf("expected type TestEvent, got %s", evt.Type())
	}

	if evt.AggregateID() != aggID {
		t.Errorf("expected aggregate ID %s, got %s", aggID, evt.AggregateID())
	}

	if evt.AggregateType() != "TestAggregate" {
		t.Errorf("expected aggregate type TestAggregate, got %s", evt.AggregateType())
	}

	if evt.Version() != 1 {
		t.Errorf("expected version 1, got %d", evt.Version())
	}
}

func TestBuilder_Build_WithPayload(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	payload := []byte(`{"key":"value"}`)

	b := newBuilder("TestEvent", aggID, "TestAggregate", Version(1)).
		WithPayload(payload)

	evt, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(evt.Payload()) != string(payload) {
		t.Errorf("expected payload %s, got %s", payload, evt.Payload())
	}
}

func TestBuilder_Build_WithMetadata(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	correlationID := id.NewCorrelationID()
	causationID := id.NewCausationID()
	userID := id.NewUserID()

	b := newBuilder("TestEvent", aggID, "TestAggregate", Version(1)).
		WithCorrelationID(correlationID).
		WithCausationID(causationID).
		WithUserID(userID)

	evt, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := evt.Metadata()
	if meta.CorrelationID != correlationID {
		t.Errorf("expected correlation ID %s, got %s", correlationID, meta.CorrelationID)
	}

	if meta.CausationID != causationID {
		t.Errorf("expected causation ID %s, got %s", causationID, meta.CausationID)
	}

	if meta.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, meta.UserID)
	}
}

func TestBuilder_Build_InvalidEventType(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	b := newBuilder("", aggID, "TestAggregate", Version(1))

	_, err := b.Build()
	if err == nil {
		t.Error("expected error for empty event type")
	}
}

func TestBuilder_Build_ReturnsEvent(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	b := newBuilder("TestEvent", aggID, "TestAggregate", Version(1))

	evt, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if evt.Type() != "TestEvent" {
		t.Errorf("expected type TestEvent, got %s", evt.Type())
	}
}

func TestBuilder_Build_WithOptions(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	correlationID := id.NewCorrelationID()

	b := newBuilder("TestEvent", aggID, "TestAggregate", Version(1)).
		WithOptions(WithCorrelationID(correlationID))

	evt, err := b.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := evt.Metadata()
	if meta.CorrelationID != correlationID {
		t.Errorf("expected correlation ID %s, got %s", correlationID, meta.CorrelationID)
	}
}
