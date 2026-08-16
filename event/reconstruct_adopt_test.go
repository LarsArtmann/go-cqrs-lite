package event

import (
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func adoptTestInputs(t *testing.T) (id.EventID, Type, id.StreamType, id.StreamID, []byte, Metadata, time.Time) {
	t.Helper()

	return id.NewEventID(), "UserCreated", "User", id.NewStreamID(),
		[]byte(`{"name":"Alice","email":"alice@example.com","age":42}`),
		NewMetadata(), time.Unix(1700000000, 0).UTC()
}

// TestReconstructEventWithAdoptedPayload_Equivalence pins that the zero-copy
// variant produces an event identical (every accessor) to the cloning variant.
func TestReconstructEventWithAdoptedPayload_Equivalence(t *testing.T) {
	t.Parallel()

	eventID, eventType, streamType, streamID, payload, meta, occurredAt := adoptTestInputs(t)

	cloned, err := ReconstructEventWithMetadata(
		eventID, eventType, streamType, streamID, 7, 2,
		payload, meta, occurredAt, "cbor", "test",
	)
	if err != nil {
		t.Fatalf("cloning variant: %v", err)
	}

	adopted, err := ReconstructEventWithAdoptedPayload(
		eventID, eventType, streamType, streamID, 7, 2,
		payload, meta, occurredAt, "cbor", "test",
	)
	if err != nil {
		t.Fatalf("adopting variant: %v", err)
	}

	if adopted.ID() != cloned.ID() ||
		adopted.Type() != cloned.Type() ||
		adopted.StreamID() != cloned.StreamID() ||
		adopted.StreamType() != cloned.StreamType() ||
		adopted.Version() != cloned.Version() ||
		adopted.SchemaVersion() != cloned.SchemaVersion() ||
		adopted.Encoding() != cloned.Encoding() ||
		!adopted.OccurredAt().Equal(cloned.OccurredAt()) {
		t.Fatal("adopted event differs from cloned event in structural fields")
	}

	if string(adopted.Payload()) != string(cloned.Payload()) {
		t.Fatalf("payload mismatch: %q vs %q", adopted.Payload(), cloned.Payload())
	}
}

// TestReconstructEventWithAdoptedPayload_AdoptsBuffer proves the payload is
// stored by reference (no clone): the internal payload must alias the input
// slice, while the cloning variant must NOT alias it.
func TestReconstructEventWithAdoptedPayload_AdoptsBuffer(t *testing.T) {
	t.Parallel()

	eventID, eventType, streamType, streamID, payload, meta, occurredAt := adoptTestInputs(t)

	adopted, err := ReconstructEventWithAdoptedPayload(
		eventID, eventType, streamType, streamID, 1, 1,
		payload, meta, occurredAt, "cbor", "test",
	)
	if err != nil {
		t.Fatalf("adopting variant: %v", err)
	}

	if &adopted.payload[0] != &payload[0] {
		t.Fatal("adopt variant copied the payload — expected the input buffer to be adopted")
	}

	// Defensive accessor stays defensive: Payload() returns a clone.
	if &adopted.Payload()[0] == &payload[0] {
		t.Fatal("Payload() aliased the internal buffer — defensive clone contract broken")
	}
}

func TestReconstructEventWithAdoptedPayload_ConcurrentReads(t *testing.T) {
	t.Parallel()

	eventID, eventType, streamType, streamID, payload, meta, occurredAt := adoptTestInputs(t)

	adopted, err := ReconstructEventWithAdoptedPayload(
		eventID, eventType, streamType, streamID, 1, 1,
		payload, meta, occurredAt, "cbor", "test",
	)
	if err != nil {
		t.Fatalf("adopting variant: %v", err)
	}

	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 100 {
				if string(adopted.Payload()) == "" {
					t.Error("empty payload read")
				}

				_ = adopted.Metadata()
				_ = adopted.ID()
			}
		}()
	}

	wg.Wait()
}
