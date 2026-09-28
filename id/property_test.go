package id_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestParseRoundTrip checks that parsing a stringified ID returns the same ID.
func TestParseRoundTrip(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		original := id.NewStreamID()
		parsed, err := id.ParseStreamID(original.String())
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if original != parsed {
			t.Fatalf("parse round-trip failed: %s != %s", original.String(), parsed.String())
		}
	})
}

// TestNewIDUniqueness checks that two newly generated IDs are different.
func TestNewIDUniqueness(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		id1 := id.NewStreamID()
		id2 := id.NewStreamID()
		if id1 == id2 {
			t.Fatal("two new IDs are equal (should be unique)")
		}
	})
}

// TestIDStringLength checks that all ID types produce strings of the same length.
func TestIDStringLength(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		aggID := id.NewStreamID()
		evtID := id.NewEventID()
		cmdID := id.NewCommandID()

		// ULID strings are always 26 characters
		if len(aggID.String()) != 26 {
			t.Fatalf("aggregate ID length %d != 26", len(aggID.String()))
		}
		if len(evtID.String()) != 26 {
			t.Fatalf("event ID length %d != 26", len(evtID.String()))
		}
		if len(cmdID.String()) != 26 {
			t.Fatalf("command ID length %d != 26", len(cmdID.String()))
		}
	})
}

// TestParseInvalidString checks that parsing invalid strings fails.
// StreamID accepts any non-empty string, so empty string is the only
// universally invalid input.
func TestParseInvalidString(t *testing.T) {
	t.Parallel()

	_, err := id.ParseStreamID("")
	if err == nil {
		t.Fatal("expected parse to fail for empty string")
	}
}
