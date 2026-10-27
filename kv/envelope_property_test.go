package kv_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// Property tests for the ADR-0044 envelope as the blind store sees it. The
// single-instance cases live in typed_store_test.go; these generalise them to
// arbitrary values, which is where a codec assumption actually breaks (a field
// JSON can round-trip but CBOR cannot, or vice versa).
//
// Testing suite T27: the envelope's whole promise is that the WRITER's codec
// does not constrain the READER. If that ever regresses, every deployment that
// switches its default codec silently reads zero values, and no table test
// would notice because each table entry uses one codec end to end.

// TestProperty_EnvelopeCodecIndependence — a value written through one codec
// is readable through a store configured with the other.
func TestProperty_EnvelopeCodecIndependence(t *testing.T) {
	t.Parallel()

	pairs := map[string]struct {
		writer codec.Codec
		reader codec.Codec
	}{
		"cbor-write/json-read": {writer: codec.CBORCodec{}, reader: codec.JSONCodec{}},
		"json-write/cbor-read": {writer: codec.JSONCodec{}, reader: codec.CBORCodec{}},
	}

	for name, pair := range pairs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				backend := kv.NewMemStore()
				defer func() { _ = backend.Close() }()

				ctx := context.Background()
				id := genID().Draw(rt, "id")
				user := genUser().Draw(rt, "user")

				writer := kv.NewTypedStore[testUser, testID](backend,
					kv.WithTypedCodec[testUser, testID](pair.writer))
				reader := kv.NewTypedStore[testUser, testID](backend,
					kv.WithTypedCodec[testUser, testID](pair.reader))

				if err := writer.Set(ctx, id, &user); err != nil {
					rt.Fatalf("Set failed: %v", err)
				}

				got, err := reader.Get(ctx, id)
				if err != nil {
					rt.Fatalf("Get with a different codec failed: %v", err)
				}
				if *got != user {
					rt.Fatalf("cross-codec round-trip mismatch: wrote %+v, read %+v", user, *got)
				}
			})
		})
	}
}

// TestProperty_LegacyRawCodecBytesStillDecode — pre-envelope rows (the value
// encoded with no envelope, i.e. bytes written before ADR-0044) stay readable
// regardless of which codec the reading store is configured with. This is the
// migration path: an operator flips the codec and old rows must not vanish.
func TestProperty_LegacyRawCodecBytesStillDecode(t *testing.T) {
	t.Parallel()

	encoders := map[string]codec.Codec{
		"json": codec.JSONCodec{},
		"cbor": codec.CBORCodec{},
	}
	readers := map[string]codec.Codec{
		"json": codec.JSONCodec{},
		"cbor": codec.CBORCodec{},
	}

	for encName, enc := range encoders {
		for readName, read := range readers {
			t.Run(encName+"-legacy/"+readName+"-reader", func(t *testing.T) {
				t.Parallel()

				rapid.Check(t, func(rt *rapid.T) {
					backend := kv.NewMemStore()
					defer func() { _ = backend.Close() }()

					ctx := context.Background()
					id := genID().Draw(rt, "id")
					user := genUser().Draw(rt, "user")

					raw, err := enc.Encode(user)
					if err != nil {
						rt.Fatalf("legacy encode failed: %v", err)
					}
					// The default key function is K.String(), so a raw byte
					// write is exactly what a pre-envelope version produced.
					if err := backend.Set(ctx, []byte(id.String()), raw); err != nil {
						rt.Fatalf("raw backend Set failed: %v", err)
					}

					reader := kv.NewTypedStore[testUser, testID](backend,
						kv.WithTypedCodec[testUser, testID](read))
					got, err := reader.Get(ctx, id)
					if err != nil {
						rt.Fatalf(
							"legacy %s bytes unreadable by a %s store: %v",
							encName,
							readName,
							err,
						)
					}
					if *got != user {
						rt.Fatalf("legacy decode mismatch: wrote %+v, read %+v", user, *got)
					}
				})
			})
		}
	}
}
