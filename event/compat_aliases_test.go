package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestV3CompatAliases verifies the deprecated v3-compat aliases re-exported
// from the event package still resolve and behave identically to the canonical
// id package types, guarding the ADR-0058 backward-compatibility surface.
func TestV3CompatAliases(t *testing.T) {
	t.Parallel()

	// Type aliases cross-assign to the canonical id package types.
	streamID := id.NewStreamID()
	var _ id.StreamID = event.AggregateID(streamID)
	var _ id.StreamType = event.AggregateType("User")
	var _ id.StreamRef = event.AggregateRef(id.NewStreamRef("User", streamID))

	// v3-compat constructor and parser re-exports.
	ref := event.NewAggregateRef(id.StreamType("Order"), streamID)
	if ref.ID != streamID || ref.Type != "Order" {
		t.Errorf("NewAggregateRef = %+v, want {Order %s}", ref, streamID)
	}

	got, err := event.ParseAggregateType("Invoice")
	if err != nil {
		t.Fatalf("ParseAggregateType: %v", err)
	}
	if got != id.StreamType("Invoice") {
		t.Errorf("ParseAggregateType = %q, want Invoice", got)
	}
}

// TestDeprecatedEventMethods verifies the deprecated AggregateID/AggregateType
// accessors on ImmutableEvent return the same values as the canonical methods.
func TestDeprecatedEventMethods(t *testing.T) {
	t.Parallel()

	streamID := id.NewStreamID()
	evt, err := event.NewEvent("UserCreated", streamID, "User", event.Version(1), []byte(`{}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if evt.AggregateID() != evt.StreamID() {
		t.Error("AggregateID() does not match StreamID()")
	}

	if evt.AggregateType() != evt.StreamType() {
		t.Error("AggregateType() does not match StreamType()")
	}
}

// TestDeprecatedErrorSentinels verifies the deprecated error aliases resolve
// to the same sentinel values as their canonical replacements.
func TestDeprecatedErrorSentinels(t *testing.T) {
	t.Parallel()

	if event.ErrNilAggregateID != event.ErrNilStreamID {
		t.Error("ErrNilAggregateID is not ErrNilStreamID")
	}

	if event.ErrEmptyAggregateType != event.ErrEmptyStreamType {
		t.Error("ErrEmptyAggregateType is not ErrEmptyStreamType")
	}

	if event.ErrAggregateNotFound != event.ErrStreamNotFound {
		t.Error("ErrAggregateNotFound is not ErrStreamNotFound")
	}
}
