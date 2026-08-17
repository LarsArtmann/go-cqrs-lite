package event_test

import (
	"reflect"
	"testing"

	codecpkg "github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// TestMetadata_CBORRoundtrip_PreservesActor locks the guarantee that
// id.ActorID survives a CBOR round-trip of full event.Metadata — the binary
// form (ActorID.MarshalBinary, the "kind:raw" prefixed string) must come back
// intact, not as a zero value. Custom stores that CBOR-encode metadata rely
// on this; dropping the actor here is silent audit-trail data loss.
func TestMetadata_CBORRoundtrip_PreservesActor(t *testing.T) {
	t.Parallel()

	actor := id.NewServiceActor("order-api")

	cmdID, err := id.ParseCommandID("01HK1540X0841Y0A6BSX1VKRA0")
	if err != nil {
		t.Fatalf("parse command ID: %v", err)
	}

	reqID, err := id.ParseRequestID("01HK1540X0841Y0A6BSX1VKRA1")
	if err != nil {
		t.Fatalf("parse request ID: %v", err)
	}

	original := event.Metadata{
		Tracing: metadata.Tracing{
			CorrelationID: idtest.ParseCorrelationID(t, "01HK1540X0841Y0A6BSX1VKR97"),
			CausationID:   idtest.ParseCausationID(t, "01HK1540X0841Y0A6BSX1VKR98"),
			UserID:        idtest.ParseUserID(t, "01HK1540X0841Y0A6BSX1VKR99"),
			RequestID:     reqID,
			ActorID:       actor,
		},
		Source:    "test-service",
		IPAddress: "10.0.0.1",
		UserAgent: "test-agent/1.0",
		Causation: &event.Causation{
			CommandType: "CreateOrder",
			CommandID:   cmdID,
		},
		Custom: map[event.MetadataKey]string{"custom.trace": "abc123"},
	}

	codec := codecpkg.CBORCodec{}

	data, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("CBOR encode metadata: %v", err)
	}

	var decoded event.Metadata

	if err := codec.Decode(data, &decoded); err != nil {
		t.Fatalf("CBOR decode metadata: %v", err)
	}

	if !decoded.ActorID.Equal(actor) {
		t.Errorf("actor lost through CBOR round-trip: got %q, want %q",
			decoded.ActorID.PrefixedString(), actor.PrefixedString())
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Errorf(
			"full metadata mismatch after CBOR round-trip:\ngot  %+v\nwant %+v",
			decoded,
			original,
		)
	}
}

// TestMetadata_CBORRoundtrip_ZeroActorOmitted verifies the zero ActorID stays
// zero (and CBOR-encodes without error) when no actor is set.
func TestMetadata_CBORRoundtrip_ZeroActorOmitted(t *testing.T) {
	t.Parallel()

	original := event.Metadata{Source: "test-service"}

	codec := codecpkg.CBORCodec{}

	data, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("CBOR encode zero-actor metadata: %v", err)
	}

	var decoded event.Metadata

	if err := codec.Decode(data, &decoded); err != nil {
		t.Fatalf("CBOR decode zero-actor metadata: %v", err)
	}

	if !decoded.ActorID.IsZero() {
		t.Errorf("zero actor came back non-zero: %q", decoded.ActorID.PrefixedString())
	}
}
