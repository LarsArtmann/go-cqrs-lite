package event

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestSingle(t *testing.T) {
	aggID := id.NewAggregateID()
	aggType := id.StreamType("User")

	events, err := Single(
		"user.created",
		aggID,
		aggType,
		Version(1),
		struct{ Name string }{Name: "Alice"},
	)
	if err != nil {
		t.Fatalf("Single() error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Single() returned %d events, want 1", len(events))
	}

	evt := events[0]
	if evt.Type() != "user.created" {
		t.Errorf("Type() = %q, want %q", evt.Type(), "user.created")
	}
	if evt.StreamType() != aggType {
		t.Errorf("StreamType() = %q, want %q", evt.StreamType(), aggType)
	}
	if evt.Version() != Version(1) {
		t.Errorf("Version() = %d, want 1", evt.Version())
	}
}

func TestSingle_WithNilPayload(t *testing.T) {
	aggID := id.NewAggregateID()

	_, err := Single(
		"user.created",
		aggID,
		"User",
		Version(1),
		nil,
	)
	if err == nil {
		t.Fatal("Single() with nil payload should return error")
	}
}

func TestSingle_WithCorrelationID(t *testing.T) {
	aggID := id.NewAggregateID()
	corrID := id.NewCorrelationID()

	events, err := Single(
		"user.created",
		aggID,
		"User",
		Version(1),
		struct{ Name string }{Name: "Bob"},
		WithCorrelationID(corrID),
	)
	if err != nil {
		t.Fatalf("Single() error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Single() returned %d events, want 1", len(events))
	}

	md := events[0].Metadata()
	if md.CorrelationID != corrID {
		t.Errorf("CorrelationID mismatch: got %v, want %v", md.CorrelationID, corrID)
	}
}
