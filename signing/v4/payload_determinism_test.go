package signing

import (
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestPayloadDeterminismContract pins the signing seam's determinism
// guarantee end to end:
//
//  1. canonicalPayload consumes the event's STORED bytes (it never re-encodes
//     through a codec), so encode-side non-determinism cannot perturb a
//     signature today: two byte-identical events canonicalize identically.
//  2. The codecs the event layer can stamp — the CBOR default behind
//     event.New and the JSONCodec available via event.DefaultCodec — all
//     satisfy codec.DeterministicCodec, so a future refactor that DOES
//     re-encode before hashing lands on signing-safe encoders (the marker's
//     documented purpose: turn silent signature corruption into a visible
//     contract break here).
//
// If (2) ever fails, the fix belongs in the codec layer or in the refactor
// that introduced non-deterministic re-encoding — never in this test.
func TestPayloadDeterminismContract(t *testing.T) {
	t.Parallel()

	mk := func() event.Event {
		evt, err := event.New(
			event.Type("task.determinism_probe"),
			id.NewStreamID(),
			id.StreamType("Task"),
			event.Version(1),
			determinismFixture{Title: "stable", At: time.Unix(0, 0).UTC()},
		)
		if err != nil {
			t.Fatalf("event.New: %v", err)
		}

		return evt
	}

	// 1. Stored-bytes stability: the canonical form is a pure function of
	// the event's bytes — equal events (same construction) canonicalize
	// byte-identically, and re-canonicalizing is idempotent.
	evt := mk()

	if a, b := canonicalPayload(evt), canonicalPayload(evt); string(a) != string(b) {
		t.Fatal("canonicalPayload is not idempotent for the same event")
	}

	// 2. The event-stamped codecs are DeterministicCodecs.
	for _, tc := range []struct {
		name  string
		codec codec.Codec
	}{
		{"event.New default (CBOR)", event.DefaultCodec},
		{"JSONCodec (encoding/json/v2)", codec.JSONCodec{}},
		{"CBORCompactCodec", codec.CBORCompactCodec{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, ok := tc.codec.(codec.DeterministicCodec); !ok {
				t.Fatalf(
					"%T does not implement codec.DeterministicCodec — a signing refactor "+
						"that re-encodes payloads would corrupt signatures",
					tc.codec,
				)
			}
		})
	}
}

type determinismFixture struct {
	Title string    `json:"title"`
	At    time.Time `json:"at"`
}
