package signing_test

import (
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
)

func TestCloneEvent_PreservesPayloadEncoding(t *testing.T) {
	t.Parallel()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")

	newEvent := func(c codec.Codec) event.Event {
		t.Helper()

		evt, err := event.New(
			"test.created",
			streamID,
			"Test",
			1,
			map[string]any{"key": "value"},
			event.WithCodec(c),
		)
		if err != nil {
			t.Fatalf("create %s event: %v", c.Encoding(), err)
		}

		return evt
	}

	t.Run("JSON codec event clones as encoding=JSON", func(t *testing.T) {
		t.Parallel()

		evt := newEvent(codec.JSONCodec{})

		clone, err := signing.CloneEvent(evt, signing.MetadataKey, "sig-value")
		if err != nil {
			t.Fatalf("clone: %v", err)
		}

		if clone.Encoding() != evt.Encoding() {
			t.Fatalf("clone encoding = %s, want original %s", clone.Encoding(), evt.Encoding())
		}

		if clone.Encoding() != codec.EncodingJSON {
			t.Fatalf("clone encoding = %s, want %s", clone.Encoding(), codec.EncodingJSON)
		}

		jsonCodec := codec.JSONCodec{}

		var decoded map[string]any
		if err := jsonCodec.Decode(clone.Payload(), &decoded); err != nil {
			t.Fatalf("clone payload no longer decodes as stamped JSON encoding: %v", err)
		}

		if decoded["key"] != "value" {
			t.Fatalf("decoded payload = %v, want key=value", decoded)
		}
	})

	t.Run("CBOR codec event clones as encoding=CBOR", func(t *testing.T) {
		t.Parallel()

		evt := newEvent(codec.CBORCodec{})

		clone, err := signing.CloneEvent(evt, signing.MetadataKey, "sig-value")
		if err != nil {
			t.Fatalf("clone: %v", err)
		}

		if clone.Encoding() != evt.Encoding() {
			t.Fatalf("clone encoding = %s, want original %s", clone.Encoding(), evt.Encoding())
		}

		if clone.Encoding() != codec.EncodingCBOR {
			t.Fatalf("clone encoding = %s, want %s", clone.Encoding(), codec.EncodingCBOR)
		}
	})

	t.Run("AttachSignature preserves encoding", func(t *testing.T) {
		t.Parallel()

		evt := newEvent(codec.JSONCodec{})

		signer, err := signing.NewHMAC([]byte("my-secret-key-thirty-two-bytes!!"))
		if err != nil {
			t.Fatalf("new hmac signer: %v", err)
		}

		sig, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		signed, err := signing.AttachSignature(evt, sig)
		if err != nil {
			t.Fatalf("attach: %v", err)
		}

		if signed.Encoding() != codec.EncodingJSON {
			t.Fatalf("signed encoding = %s, want %s", signed.Encoding(), codec.EncodingJSON)
		}

		if err := signer.Verify(signed, sig); err != nil {
			t.Fatalf("verify signature on encoding-preserving clone: %v", err)
		}
	})
}
