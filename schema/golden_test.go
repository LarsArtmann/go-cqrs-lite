package schema_test

import (
	"encoding/json"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/schema/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func parseCorrID(s string) id.CorrelationID {
	v, err := id.ParseCorrelationID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func parseEventID(s string) id.EventID {
	v, err := id.ParseEventID(s)
	if err != nil {
		panic(err)
	}
	return v
}

var update = flag.Bool("update", false, "update golden files")

func TestGolden_UpcasterOutput(t *testing.T) {
	upcaster := schema.NewUpcaster(
		"UserCreated",
		1,
		func(evt event.Event) (*event.ImmutableEvent, error) {
			newPayload, _ := json.Marshal(map[string]any{
				"name":     "unknown",
				"email":    string(evt.Payload()),
				"upgraded": true,
			})

			return event.NewEvent(
				evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
				newPayload,
				event.WithEventID(evt.ID()),
				event.WithOccurredAt(evt.OccurredAt()),
				event.WithSchemaVersion(2),
				event.WithCorrelationID(parseCorrID("01HK1540X0841Y0A6BSX1VKR97")),
			)
		},
	)

	store := memory.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evtID := parseEventID("01HK1540X0841Y0A6BSX1VKR96")
	ref := event.NewAggregateRef("User", aggID)

	evt, err := event.NewEvent(
		"UserCreated", aggID, "User", 1,
		[]byte(`"alice@example.com"`),
		event.WithEventID(evtID),
		event.WithOccurredAt(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)),
		event.WithSchemaVersion(1),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := store.Save(t.Context(), ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("save: %v", err)
	}

	vs, err := schema.NewVersionedStore(store, upcaster)
	if err != nil {
		t.Fatalf("create versioned store: %v", err)
	}

	upcasted, err := vs.Load(t.Context(), ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	type snapshotEvent struct {
		Type          string `json:"type"`
		AggregateID   string `json:"aggregateId"`
		AggregateType string `json:"aggregateType"`
		Version       int    `json:"version"`
		SchemaVersion int    `json:"schemaVersion"`
		Payload       string `json:"payload"`
	}

	snapshots := make([]snapshotEvent, len(upcasted))
	for i, e := range upcasted {
		snapshots[i] = snapshotEvent{
			Type:          string(e.Type()),
			AggregateID:   e.AggregateID().String(),
			AggregateType: string(e.AggregateType()),
			Version:       e.Version().Int(),
			SchemaVersion: e.SchemaVersion().Int(),
			Payload:       string(e.Payload()),
		}
	}

	got, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "upcaster-output.json"),
		got,
		*update,
	)
}
