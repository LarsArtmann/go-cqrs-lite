package schema

import (
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// benchUpcastChain is the declaration under measurement: a rename + a
// nested-object transform — the two op families a real fleet consumer
// (DiscordSync-class) declares.
func benchUpcastChain(tb testing.TB) *Chain {
	tb.Helper()

	chain, err := Compile(
		Transform("user.profiled", 1, func(fields map[string]any) (map[string]any, error) {
			if profile, ok := fields["profile"].(map[string]any); ok {
				profile["displayName"] = profile["name"]
			}

			return fields, nil
		}),
		RenameType("user.renamed_legacy", "user.profiled"),
	)
	if err != nil {
		tb.Fatalf("Compile: %v", err)
	}

	return chain
}

func benchPayloadEvent(tb testing.TB, codecFor codec.Codec) event.Event {
	tb.Helper()

	payload := map[string]any{
		"id":    "bench-1",
		"rank":  float64(3),
		"email": "bench@example.com",
		"flags": []any{float64(1), float64(2), float64(3)},
		"profile": map[string]any{
			"name":   "Bench",
			"avatar": "https://example.com/a.png",
			"bio":    "representative nested object",
		},
	}

	return newPayloadEvent(tb, "user.profiled", 1, payload, event.WithCodec(codecFor))
}

// benchHandrolledUpcast is the pre-chain idiom the named ops replace: decode
// the field map by hand, mutate, re-encode, rebuild the event — same work
// per event, minus the chain's op matching.
func benchHandrolledUpcast(
	tb testing.TB,
	codecFor codec.Codec,
	evt event.Event,
) (event.Event, error) {
	tb.Helper()

	var fields map[string]any
	if err := codecFor.Decode(evt.Payload(), &fields); err != nil {
		return nil, err
	}

	if profile, ok := fields["profile"].(map[string]any); ok {
		profile["displayName"] = profile["name"]
	}

	encoded, err := codecFor.Encode(fields)
	if err != nil {
		return nil, err
	}

	return event.New(
		evt.Type(),
		evt.StreamID(),
		evt.StreamType(),
		evt.Version(),
		encoded,
		event.WithEventID(evt.ID()),
		event.WithEncoding(evt.Encoding()),
	)
}

func BenchmarkChainSourceTransformJSON(b *testing.B) {
	chain := benchUpcastChain(b)
	source := benchPayloadEvent(b, codec.JSONCodec{})
	events := []event.Event{source}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := chain.SourceTransform()(events); err != nil {
			b.Fatalf("transform: %v", err)
		}
	}
}

func BenchmarkChainSourceTransformCBOR(b *testing.B) {
	chain := benchUpcastChain(b)
	source := benchPayloadEvent(b, codec.CBORCodec{})
	events := []event.Event{source}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := chain.SourceTransform()(events); err != nil {
			b.Fatalf("transform: %v", err)
		}
	}
}

func BenchmarkHandrolledUpcastJSON(b *testing.B) {
	source := benchPayloadEvent(b, codec.JSONCodec{})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := benchHandrolledUpcast(b, codec.JSONCodec{}, source); err != nil {
			b.Fatalf("upcast: %v", err)
		}
	}
}

func BenchmarkHandrolledUpcastCBOR(b *testing.B) {
	source := benchPayloadEvent(b, codec.CBORCodec{})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := benchHandrolledUpcast(b, codec.CBORCodec{}, source); err != nil {
			b.Fatalf("upcast: %v", err)
		}
	}
}
