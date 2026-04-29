package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	builder := event.NewBuilder("TestEvent", aggID, "TestAggregate", event.Version(1))

	evt, err := builder.Build()
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

	builder := event.NewBuilder("TestEvent", aggID, "TestAggregate", event.Version(1)).
		WithPayload(payload)

	evt, err := builder.Build()
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

	builder := event.NewBuilder("TestEvent", aggID, "TestAggregate", event.Version(1)).
		WithCorrelationID(correlationID).
		WithCausationID(causationID).
		WithUserID(userID)

	evt, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := evt.Metadata()
	if meta == nil {
		t.Fatal("expected metadata, got nil")
	}

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
	builder := event.NewBuilder("", aggID, "TestAggregate", event.Version(1))

	_, err := builder.Build()
	if err == nil {
		t.Error("expected error for empty event type")
	}
}

func TestBuilder_MustBuild(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	builder := event.NewBuilder("TestEvent", aggID, "TestAggregate", event.Version(1))

	evt := builder.MustBuild()

	if evt.Type() != "TestEvent" {
		t.Errorf("expected type TestEvent, got %s", evt.Type())
	}
}

func TestBuilder_MustBuild_Panics(t *testing.T) {
	t.Parallel()

	builder := event.NewBuilder("", id.AggregateID{}, "", event.Version(0))

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid event")
		}
	}()

	builder.MustBuild()
}

func TestBuilder_Build_WithOptions(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	correlationID := id.NewCorrelationID()

	builder := event.NewBuilder("TestEvent", aggID, "TestAggregate", event.Version(1)).
		WithOptions(event.WithCorrelationID(correlationID))

	evt, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta := evt.Metadata()
	if meta == nil {
		t.Fatal("expected metadata, got nil")
	}

	if meta.CorrelationID != correlationID {
		t.Errorf("expected correlation ID %s, got %s", correlationID, meta.CorrelationID)
	}
}
