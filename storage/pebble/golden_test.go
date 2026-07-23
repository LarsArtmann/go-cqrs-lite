package pebble_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	pb "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_EventStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()

	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	store, err := pb.NewStore(db, nil)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	aggID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	ref := id.NewStreamRef("Order", aggID)

	types := []struct {
		typ     event.Type
		version int
		payload string
	}{
		{"OrderCreated", 1, `{"customer":"Bob"}`},
		{"ItemAdded", 2, `{"sku":"W-001","qty":5}`},
		{"OrderShipped", 3, `{"tracking":"TRK-456"}`},
	}

	baseTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	evts := make([]event.Event, len(types))

	for i, tc := range types {
		evtID := idtest.ParseEventID(t, "01HK1540X0841Y0A6BSX1VKR9"+string(rune('A'+i)))

		evt, err := event.NewEvent(
			tc.typ, aggID, "Order", event.Version(tc.version),
			[]byte(tc.payload),
			event.WithEventID(evtID),
			event.WithOccurredAt(baseTime.Add(time.Duration(i)*time.Hour)),
		)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		evts[i] = evt
	}

	if err := store.Save(t.Context(), ref, evts, event.Version(0)); err != nil {
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
		e := loaded[i]
		snaps[i] = snapEvent{
			ID:       e.ID().String(),
			Type:     string(e.Type()),
			Version:  e.Version().Int(),
			Payload:  string(e.Payload()),
			Occurred: e.OccurredAt().Format(time.RFC3339Nano),
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
