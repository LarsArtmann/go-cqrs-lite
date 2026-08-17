package event_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

var updateGolden = flag.Bool("update", false, "update golden files")

// TestGolden_EventMetadataWithActor pins the full event metadata JSON shape
// with every tracing field set, including ActorID. This is the exact shape
// every SQL store persists in its metadata column and what consumers parse —
// a tag change on Tracing.ActorID (or a marshaler bypassing the struct tags)
// fails here before it can silently reshape stored events.
func TestGolden_EventMetadataWithActor(t *testing.T) {
	actor := id.NewServiceActor("order-api")

	meta := event.Metadata{
		Tracing: metadata.Tracing{
			CorrelationID: idtest.ParseCorrelationID(t, "01HK1540X0841Y0A6BSX1VKR97"),
			CausationID:   idtest.ParseCausationID(t, "01HK1540X0841Y0A6BSX1VKR98"),
			UserID:        idtest.ParseUserID(t, "01HK1540X0841Y0A6BSX1VKR99"),
			RequestID:     idtest.ParseRequestID(t, "01HK1540X0841Y0A6BSX1VKRA1"),
			ActorID:       actor,
		},
		Source:    "test-service",
		IPAddress: "10.0.0.1",
		UserAgent: "test-agent/1.0",
		Custom:    map[event.MetadataKey]string{"custom.trace": "abc123", "tenant": "acme"},
	}

	evt, err := event.NewEvent(
		"order.created",
		idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"Order", 1,
		[]byte(`{"item":"widget"}`),
		event.WithMetadata(meta),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	got, err := json.Marshal(
		evt.Metadata(),
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
		json.Deterministic(true),
	)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "event-metadata-actor.json"),
		got,
		*updateGolden,
	)

	// The persisted shape must reconstruct the same metadata — the load path
	// every SQL store takes via UnmarshalMetadataJSON.
	opts, err := event.UnmarshalMetadataJSON(got, "test.golden", "order.created")
	if err != nil {
		t.Fatalf("unmarshal metadata JSON: %v", err)
	}

	reconstructed, err := event.NewEvent(
		"order.created",
		idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		"Order", 1,
		[]byte(`{"item":"widget"}`),
		opts...,
	)
	if err != nil {
		t.Fatalf("reconstruct event: %v", err)
	}

	loaded := reconstructed.Metadata()
	if !loaded.ActorID.Equal(actor) {
		t.Errorf("actor lost through metadata JSON round-trip: got %q, want %q",
			loaded.ActorID.PrefixedString(), actor.PrefixedString())
	}

	if loaded.CorrelationID != meta.CorrelationID || loaded.UserID != meta.UserID {
		t.Errorf("tracing IDs lost through metadata JSON round-trip: got %+v", loaded.Tracing)
	}

	if loaded.Custom["tenant"] != "acme" || loaded.Custom["custom.trace"] != "abc123" {
		t.Errorf("custom metadata lost through round-trip: got %v", loaded.Custom)
	}
}
