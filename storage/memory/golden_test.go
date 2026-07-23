package memory_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_EventStoreRoundTrip(t *testing.T) {
	store := memory.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	ref := id.NewStreamRef("Order", aggID)

	types := []struct {
		typ     string
		version int
		payload string
	}{
		{"OrderCreated", 1, `{"customer":"Alice"}`},
		{"ItemAdded", 2, `{"sku":"W-001","qty":3}`},
		{"ItemAdded", 3, `{"sku":"G-002","qty":1}`},
		{"OrderShipped", 4, `{"tracking":"TRK-123"}`},
	}

	events := make([]event.Event, 0, len(types))

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for i, tc := range types {
		evtID := idtest.ParseEventID(t, "01HK1540X0841Y0A6BSX1VKR9"+string(rune('A'+i)))

		evt, err := event.NewEvent(
			event.Type(tc.typ), aggID, "Order", event.Version(tc.version),
			[]byte(tc.payload),
			event.WithEventID(evtID),
			event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Minute)),
		)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(t.Context(), ref, events, event.Version(0)); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(t.Context(), ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	type snapEvent struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Version  int    `json:"version"`
		Payload  string `json:"payload"`
		Occurred string `json:"occurredAt"`
	}

	snaps := make([]snapEvent, len(loaded))
	for i := range loaded {
		evt := loaded[i]
		snaps[i] = snapEvent{
			ID:       evt.ID().String(),
			Type:     string(evt.Type()),
			Version:  evt.Version().Int(),
			Payload:  string(evt.Payload()),
			Occurred: evt.OccurredAt().Format(time.RFC3339Nano),
		}
	}

	got, err := json.Marshal(snaps, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "event-store-roundtrip.json"),
		got,
		*update,
	)
}

func TestGolden_SnapshotStoreRoundTrip(t *testing.T) {
	store := memory.NewMemorySnapshotStore()
	t.Cleanup(func() { _ = store.Close() })

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	ref := id.NewStreamRef("User", aggID)

	state, err := json.Marshal(struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}{Name: "Bob", Role: "admin"})
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}

	snap := snapshot.Snapshot{
		StreamID:   aggID,
		StreamType: "User",
		Version:    event.Version(10),
		State:      state,
		CreatedAt:  time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC),
	}

	if err := store.Save(t.Context(), snap); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(t.Context(), ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	got, err := json.Marshal(struct {
		StreamID   string `json:"aggregateId"`
		StreamType string `json:"aggregateType"`
		Version    int    `json:"version"`
		State      string `json:"state"`
		CreatedAt  string `json:"createdAt"`
	}{
		StreamID:   loaded.StreamID.String(),
		StreamType: string(loaded.StreamType),
		Version:    loaded.Version.Int(),
		State:      string(loaded.State),
		CreatedAt:  loaded.CreatedAt.Format(time.RFC3339Nano),
	}, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "snapshot-store-roundtrip.json"),
		got,
		*update,
	)
}
