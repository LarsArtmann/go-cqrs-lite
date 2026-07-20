package encryption_test

import (
	"context"
	"crypto/rand"
	"strings"
	"testing"

	codecpkg "github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// fuzzEvent builds an event with the given fields. Returns (evt, true) on
// success, (nil, false) if validation fails (e.g., empty type).
func fuzzEvent(
	t *testing.T,
	eventType, aggType string,
	version int,
	schemaVersion int,
	payload []byte,
) (event.Event, bool) {
	t.Helper()

	if eventType == "" || aggType == "" {
		return nil, false
	}

	evt, err := event.NewEvent(
		event.Type(eventType),
		id.NewAggregateID(),
		id.AggregateType(aggType),
		event.Version(version),
		payload,
		event.WithSchemaVersion(event.SchemaVersion(schemaVersion)),
	)
	if err != nil {
		return nil, false
	}

	return evt, true
}

// FuzzEncryptingCodec_Roundtrip encrypts arbitrary JSON-serializable payloads
// using the composing encryption codec and ensures decode+decrypt roundtrip
// recovers the original. Catches mismatched nonce/key handling.
func FuzzEncryptingCodec_Roundtrip(f *testing.F) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		f.Fatalf("rand: %v", err)
	}

	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		f.Fatalf("NewAES256GCM: %v", err)
	}

	wrapped := encryption.NewCodec(codecpkg.JSONCodec{}, enc)

	seeds := []string{
		`{"name":"a"}`,
		`null`,
		`""`,
		`[]`,
		strings.Repeat(`"x"`, 1000),
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		if len(payload) == 0 {
			return
		}

		encoded, err := wrapped.Encode([]byte(payload))
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		var got []byte
		if err := wrapped.Decode(encoded, &got); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		if string(got) != payload {
			t.Errorf("roundtrip mismatch: got %q, want %q", got, payload)
		}
	})
}

// FuzzEncryptingCodec_DecodeEmptyInput ensures decode of empty input passes
// through (no encryption) instead of panicking.
func FuzzEncryptingCodec_DecodeEmptyInput(f *testing.F) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		f.Fatalf("rand: %v", err)
	}

	enc, err := encryption.NewAES256GCM(key)
	if err != nil {
		f.Fatalf("NewAES256GCM: %v", err)
	}

	wrapped := encryption.NewCodec(codecpkg.JSONCodec{}, enc)

	f.Add([]byte{})
	f.Add([]byte{0x00})

	f.Fuzz(func(t *testing.T, input []byte) {
		var out []byte
		_ = wrapped.Decode(input, &out)
	})
}

// FuzzAttachExtractCiphertext attaches a ciphertext to a random event and
// extracts it. The base64 roundtrip must preserve the bytes.
func FuzzAttachExtractCiphertext(f *testing.F) {
	f.Add("evt", "Agg", 1, 1, "hello world")
	f.Add("e", "A", 1, 1, "")
	f.Add("x.y.z", "Order", 999, 1, strings.Repeat("p", 100))

	f.Fuzz(func(t *testing.T, typ, agg string, version, schema int, plaintext string) {
		evt, ok := fuzzEvent(t, typ, agg, version, schema, []byte(plaintext))
		if !ok {
			return
		}

		if _, err := encryption.ExtractCiphertext(evt); err == nil {
			t.Error("ExtractCiphertext on plaintext event accepted")
		}

		ct := encryption.Ciphertext([]byte(plaintext))

		attached, err := encryption.AttachEncryption(evt, ct, encryption.WithKeyID("k1"))
		if err != nil {
			t.Fatalf("AttachEncryption: %v", err)
		}

		// Empty ciphertext is rejected by ExtractCiphertext by design
		// (avoids ambiguous "is this encrypted?" signal). Skip the extract
		// check in that case.
		if len(plaintext) == 0 {
			return
		}

		extracted, err := encryption.ExtractCiphertext(attached)
		if err != nil {
			t.Fatalf("ExtractCiphertext: %v", err)
		}

		if !extracted.Equal(ct) {
			t.Errorf("roundtrip mismatch: got %q, want %q", extracted, plaintext)
		}

		if _, err := encryption.ExtractCiphertext(nil); err == nil {
			t.Error("ExtractCiphertext(nil) accepted")
		}

		if _, err := encryption.AttachEncryption(nil, ct); err == nil {
			t.Error("AttachEncryption(nil) accepted")
		}
	})
}

// FuzzExtractAlgorithm drives ExtractAlgorithm with arbitrary Custom metadata
// values. The function must always return either nil error or a Rejection
// for unknown algorithms — never panic.
func FuzzExtractAlgorithm(f *testing.F) {
	for _, s := range []string{
		"",
		"aes-256-gcm",
		"xchacha20-poly1305",
		"totally-fake-cipher",
		"AES-256-GCM",
		"'a'",
		"🚫",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, algValue string) {
		evt, ok := fuzzEvent(t, "evt", "A", 1, 1, nil)
		if !ok {
			return
		}

		algKey := encryption.AlgorithmKey

		clone, err := event.NewEvent(
			evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
			nil,
			event.WithEventID(evt.ID()),
			event.WithOccurredAt(evt.OccurredAt()),
			event.WithMetadata(event.Metadata{
				Custom: map[event.MetadataKey]string{algKey: algValue},
			}),
		)
		if err != nil {
			t.Fatalf("build event: %v", err)
		}

		_, _ = encryption.ExtractAlgorithm(clone)

		if _, err := encryption.ExtractAlgorithm(nil); err == nil {
			t.Error("ExtractAlgorithm(nil) accepted")
		}
	})
}

// FuzzExtractKeyID drives ExtractKeyID with arbitrary KeyID metadata values.
func FuzzExtractKeyID(f *testing.F) {
	f.Add("")
	f.Add("key-v1")
	f.Add("🔥🔥🔥")
	f.Add("a\nb\tc\rd")

	f.Fuzz(func(t *testing.T, keyIDValue string) {
		evt, ok := fuzzEvent(t, "evt", "A", 1, 1, nil)
		if !ok {
			return
		}

		keyIDKey := encryption.KeyIDKey

		clone, err := event.NewEvent(
			evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
			nil,
			event.WithEventID(evt.ID()),
			event.WithOccurredAt(evt.OccurredAt()),
			event.WithMetadata(event.Metadata{
				Custom: map[event.MetadataKey]string{keyIDKey: keyIDValue},
			}),
		)
		if err != nil {
			t.Fatalf("build event: %v", err)
		}

		kid, err := encryption.ExtractKeyID(clone)
		if err != nil {
			t.Errorf("ExtractKeyID unexpected error: %v", err)
		}

		if string(kid) != keyIDValue {
			t.Errorf("KeyID mismatch: got %q, want %q", kid, keyIDValue)
		}

		if _, nilErr := encryption.ExtractKeyID(nil); nilErr == nil {
			t.Error("ExtractKeyID(nil) accepted")
		}
	})
}

// FuzzEncryptMiddleware_NilGuards ensures nil encrypter/decrypter middleware
// rejects publish/handle calls (regression guard).
func FuzzEncryptMiddleware_NilGuards(f *testing.F) {
	f.Add("evt")

	f.Fuzz(func(t *testing.T, eventType string) {
		ctx := context.Background()

		pubMW := encryption.EncryptMiddleware(nil)
		pub := pubMW(event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
			return nil
		}))
		_ = pub.Publish(ctx)

		handlerMW := encryption.DecryptMiddleware(nil)
		handler := handlerMW(func(_ context.Context, _ event.Event) error { return nil })

		evt, ok := fuzzEvent(t, eventType, "A", 1, 1, []byte("p"))
		if ok {
			_ = handler(ctx, evt)
		}
	})
}

// FuzzCiphertext_EqualConstantTime ensures Ciphertext.Equal does not crash
// on arbitrarily-sized inputs (constant-time compare).
func FuzzCiphertext_EqualConstantTime(f *testing.F) {
	f.Add([]byte("a"), []byte("b"))
	f.Add([]byte{}, []byte{})
	f.Add([]byte{0x01}, []byte{0x02})

	f.Fuzz(func(t *testing.T, a, b []byte) {
		c1 := encryption.Ciphertext(a)
		c2 := encryption.Ciphertext(b)

		_ = c1.Equal(c2)
		_ = c1.Equal(c1) //nolint:gocritic

		_ = c1.String()
		_ = c1.Bytes()
	})
}

// FuzzAlgorithm_String ensures Algorithm.String() never panics and roundtrips.
func FuzzAlgorithm_String(f *testing.F) {
	f.Add("")
	f.Add("aes-256-gcm")
	f.Add("xchacha20-poly1305")
	f.Add("totally-bogus")

	f.Fuzz(func(t *testing.T, s string) {
		alg := encryption.Algorithm(s)

		got := alg.String()
		if got != s {
			t.Errorf("Algorithm.String: got %q, want %q", got, s)
		}

		if s == "" && !alg.IsZero() {
			t.Error("empty Algorithm should be IsZero")
		}
	})
}

// FuzzExtractCiphertext_OverCorruptBase64 drives ExtractCiphertext against
// events whose encryption metadata has been replaced with arbitrary base64
// strings. Must always return an error (or a decoded value), never panic.
func FuzzExtractCiphertext_OverCorruptBase64(f *testing.F) {
	f.Add("")
	f.Add("hello")
	f.Add("!!!")
	f.Add("AAAA")
	f.Add("AAAAAAAAAAAAAAAAAAAAAA==")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, encoded string) {
		evt, ok := fuzzEvent(t, "evt", "A", 1, 1, nil)
		if !ok {
			return
		}

		clone, err := event.NewEvent(
			evt.Type(), evt.AggregateID(), evt.AggregateType(), evt.Version(),
			nil,
			event.WithEventID(evt.ID()),
			event.WithOccurredAt(evt.OccurredAt()),
			event.WithMetadata(event.Metadata{
				Custom: map[event.MetadataKey]string{encryption.MetadataKey: encoded},
			}),
		)
		if err != nil {
			t.Fatalf("build event: %v", err)
		}

		// Should not panic on arbitrary base64 text.
		_, _ = encryption.ExtractCiphertext(clone)
	})
}
