package event_test

import (
	"bytes"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestEventImmutability checks that Clone produces an independent copy.
func TestEventImmutability(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := event.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		streamID := id.NewStreamID()
		version := event.Version(rapid.IntRange(1, 1000).Draw(t, "version"))

		evt, err := event.NewEvent(typ, streamID, "Test", version, nil)
		if err != nil {
			t.Fatalf("create event: %v", err)
		}

		clone := evt.Clone()

		if evt.Type() != clone.Type() {
			t.Fatal("type mismatch after clone")
		}
		if evt.StreamID() != clone.StreamID() {
			t.Fatal("streamID mismatch after clone")
		}
		if evt.Version() != clone.Version() {
			t.Fatal("version mismatch after clone")
		}
		if evt.StreamType() != clone.StreamType() {
			t.Fatal("streamType mismatch after clone")
		}
		if evt.OccurredAt() != clone.OccurredAt() {
			t.Fatal("occurredAt mismatch after clone")
		}

		// The clone should be a different pointer
		if evt == clone {
			t.Fatal("clone returned same pointer")
		}
	})
}

// TestEventIDempotency checks that creating an event twice with the same
// parameters produces events with the same fields.
func TestEventIDempotency(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		typ := event.Type(rapid.StringMatching(`^[A-Za-z][A-Za-z0-9._-]+$`).Draw(t, "type"))
		streamID := id.NewStreamID()
		version := event.Version(rapid.IntRange(1, 1000).Draw(t, "version"))

		evt1, err1 := event.NewEvent(typ, streamID, "Test", version, nil)
		evt2, err2 := event.NewEvent(typ, streamID, "Test", version, nil)

		if err1 != nil || err2 != nil {
			t.Skip("creation error")
		}

		if evt1.Type() != evt2.Type() ||
			evt1.StreamID() != evt2.StreamID() ||
			evt1.Version() != evt2.Version() ||
			evt1.StreamType() != evt2.StreamType() {
			t.Fatal("identical parameters produced different events")
		}
	})
}

// TestBatchVersionMonotonicity checks that NewEvents creates events with
// strictly increasing versions.
func TestBatchVersionMonotonicity(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(2, 50).Draw(t, "count")
		startVersion := event.Version(rapid.IntRange(1, 100).Draw(t, "startVersion"))
		streamID := id.NewStreamID()

		types := make([]event.Type, count)
		payloads := make([]any, count)
		for i := range types {
			types[i] = "TestEvent"
			payloads[i] = struct{ N int }{N: i}
		}

		events, err := event.NewEvents(streamID, "Test", startVersion, types, payloads)
		if err != nil {
			t.Fatalf("create events: %v", err)
		}

		if len(events) != count {
			t.Fatalf("expected %d events, got %d", count, len(events))
		}

		for i := 1; i < len(events); i++ {
			if events[i].Version() != events[i-1].Version()+1 {
				t.Fatalf(
					"version not sequential: %d -> %d",
					events[i-1].Version(),
					events[i].Version(),
				)
			}
		}
	})
}

// TestPayloadIsolation_Property checks that Payload() returns independent copies
// under random payload sizes and content.
func TestPayloadIsolation_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(0, 4096).Draw(t, "size")
		payload := rapid.SliceOfN(rapid.Byte(), size, size).Draw(t, "payload")

		evt, err := event.NewEvent("Test", id.NewStreamID(), "Test", 1, payload)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		got := evt.Payload()

		if !bytes.Equal(got, payload) {
			t.Fatal("Payload() content mismatch")
		}

		if len(got) > 0 {
			got[0] ^= 0xff
			after := evt.Payload()
			if after[0] == got[0] {
				t.Fatal("mutating Payload() result affected internal state")
			}
		}
	})
}

// TestMetadataIsolation_Property checks that Metadata() returns independent copies.
func TestMetadataIsolation_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 10).Draw(t, "num_keys")

		seen := make(map[event.MetadataKey]string, n)
		opts := make([]event.Option, 0, n)

		for range n {
			k := event.MetadataKey(rapid.StringN(1, 20, 30).Draw(t, "key"))
			v := rapid.StringN(1, 20, 30).Draw(t, "val")
			seen[k] = v
			opts = append(opts, event.WithCustom(k, v))
		}

		evt, err := event.NewEvent("Test", id.NewStreamID(), "Test", 1, []byte(`{}`), opts...)
		if err != nil {
			t.Fatal(err)
		}

		md := evt.Metadata()
		for k := range md.Custom {
			md.Custom[k] = "MUTATED"
		}

		after := evt.Metadata()
		for k, want := range seen {
			if after.Custom[k] == "MUTATED" {
				t.Fatalf("mutating Metadata() result leaked for key %q", k)
			}

			if after.Custom[k] != want {
				t.Fatalf(
					"Metadata value changed for key %q: got %q, want %q",
					k,
					after.Custom[k],
					want,
				)
			}
		}
	})
}
