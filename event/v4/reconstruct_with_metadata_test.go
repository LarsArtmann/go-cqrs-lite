package event_test

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// TestReconstructEventWithMetadata_Equivalence verifies that reconstructing an
// event from a decoded Metadata value produces the same event as the
// marshal-to-JSON/unmarshal path used by SQL stores. Engines that decode the
// whole envelope in one step rely on this equivalence to skip the JSON
// round-trip.
func TestReconstructEventWithMetadata_Equivalence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata event.Metadata
	}{
		{
			name:     "empty metadata",
			metadata: event.Metadata{},
		},
		{
			name: "tracing metadata",
			metadata: event.Metadata{
				Tracing: metadata.Tracing{
					CorrelationID: id.NewCorrelationID(),
					CausationID:   id.NewCausationID(),
					RequestID:     id.NewRequestID(),
					ActorID:       id.NewUserActor(id.NewUserID()),
				},
				Source:    "test-suite",
				IPAddress: "127.0.0.1",
				UserAgent: "equivalence-test/1.0",
			},
		},
		{
			name: "custom metadata",
			metadata: event.Metadata{
				Tracing: metadata.Tracing{CorrelationID: id.NewCorrelationID()},
				Custom:  map[event.MetadataKey]string{"tenant": "acme", "region": "eu-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eventID := id.NewEventID()
			streamID := id.NewStreamID()
			occurredAt := time.Date(2026, 8, 16, 1, 2, 3, 4, time.UTC)

			metadataJSON, err := event.MarshalMetadataJSON(tt.metadata, "test")
			if err != nil {
				t.Fatalf("MarshalMetadataJSON: %v", err)
			}

			viaJSON, err := event.ReconstructEventFromFields(
				eventID, "UserCreated", "User", streamID, 7, 2,
				[]byte(`{"name":"Alice"}`), metadataJSON, occurredAt, "cbor", "test",
			)
			if err != nil {
				t.Fatalf("ReconstructEventFromFields: %v", err)
			}

			direct, err := event.ReconstructEventWithMetadata(
				eventID, "UserCreated", "User", streamID, 7, 2,
				[]byte(`{"name":"Alice"}`), tt.metadata, occurredAt, "cbor", "test",
			)
			if err != nil {
				t.Fatalf("ReconstructEventWithMetadata: %v", err)
			}

			if direct.ID() != viaJSON.ID() ||
				direct.Type() != viaJSON.Type() ||
				direct.StreamID() != viaJSON.StreamID() ||
				direct.StreamType() != viaJSON.StreamType() ||
				direct.Version() != viaJSON.Version() ||
				direct.SchemaVersion() != viaJSON.SchemaVersion() ||
				direct.Encoding() != viaJSON.Encoding() ||
				!direct.OccurredAt().Equal(viaJSON.OccurredAt()) {
				t.Fatalf("field mismatch:\ndirect: %+v\nviaJSON: %+v", direct, viaJSON)
			}

			directMeta := direct.Metadata()
			viaJSONMeta := viaJSON.Metadata()
			if directMeta.CorrelationID != viaJSONMeta.CorrelationID ||
				directMeta.CausationID != viaJSONMeta.CausationID ||
				directMeta.RequestID != viaJSONMeta.RequestID ||
				directMeta.ActorID != viaJSONMeta.ActorID ||
				directMeta.Source != viaJSONMeta.Source ||
				directMeta.IPAddress != viaJSONMeta.IPAddress ||
				directMeta.UserAgent != viaJSONMeta.UserAgent {
				t.Fatalf("metadata mismatch:\ndirect: %+v\nviaJSON: %+v", directMeta, viaJSONMeta)
			}

			if len(directMeta.Custom) != len(viaJSONMeta.Custom) {
				t.Fatalf("custom map length mismatch: direct=%d viaJSON=%d",
					len(directMeta.Custom), len(viaJSONMeta.Custom))
			}

			for k, v := range directMeta.Custom {
				if viaJSONMeta.Custom[k] != v {
					t.Fatalf("custom[%q] = %q, want %q", k, v, viaJSONMeta.Custom[k])
				}
			}
		})
	}
}

// TestReconstructEventWithMetadata_NilCustomMapPreserved checks that the
// direct path preserves a nil Custom map exactly (the JSON round-trip also
// yields nil, so equivalence must hold for the zero Custom map as well).
func TestReconstructEventWithMetadata_NilCustomMapPreserved(t *testing.T) {
	t.Parallel()

	meta := event.Metadata{Source: "nil-custom-test"}
	if meta.Custom != nil {
		t.Fatal("test setup: Custom should start nil")
	}

	evt, err := event.ReconstructEventWithMetadata(
		id.NewEventID(), "UserCreated", "User", id.NewStreamID(), 1, 0,
		[]byte(`{}`), meta, time.Now().UTC(), "", "test",
	)
	if err != nil {
		t.Fatalf("ReconstructEventWithMetadata: %v", err)
	}

	got := evt.Metadata()
	if got.Custom != nil {
		t.Fatalf("Custom = %v, want nil", got.Custom)
	}

	if got.Source != "nil-custom-test" {
		t.Fatalf("Source = %q, want %q", got.Source, "nil-custom-test")
	}
}
