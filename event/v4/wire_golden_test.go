package event_test

// Wire-format goldens (disk-truth plan A20, PC-12): Causation and
// TombstoneMark ride inside every persisted event's metadata JSON. The tags
// on those structs pin the historical field names; these byte pins fail
// BEFORE any silent reshape of stored metadata can ship.

import (
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestGolden_CausationWireBytes(t *testing.T) {
	t.Parallel()

	commandID, err := id.ParseCommandID("01HKWREY000000000000000001")
	if err != nil {
		t.Fatalf("parse CommandID: %v", err)
	}

	causation := event.Causation{
		CommandType: "create_task",
		CommandID:   commandID,
	}

	got, err := json.Marshal(causation)
	if err != nil {
		t.Fatalf("marshal Causation: %v", err)
	}

	// CommandID marshals via MarshalText: the BARE value (the wire/storage
	// form), never the prefixed display form.
	want := `{"CommandType":"create_task","CommandID":"01HKWREY000000000000000001"}`

	if string(got) != want {
		t.Fatalf("Causation wire bytes drifted:\n got: %s\nwant: %s", got, want)
	}
}

func TestGolden_TombstoneMarkWireBytes(t *testing.T) {
	t.Parallel()

	mark := event.TombstoneMark{
		Status: event.TombstoneTombstoned,
		Reason: "gdpr-erasure",
	}

	got, err := json.Marshal(mark)
	if err != nil {
		t.Fatalf("marshal TombstoneMark: %v", err)
	}

	// TombstoneStatus has no MarshalJSON: the wire form is the INTEGER
	// (TombstoneTombstoned = 1). Pinned as-is — reshaping stored metadata
	// (e.g. switching to strings) is a wire break, not a cleanup.
	want := `{"Status":1,"Reason":"gdpr-erasure"}`

	if string(got) != want {
		t.Fatalf("TombstoneMark wire bytes drifted:\n got: %s\nwant: %s", got, want)
	}
}
