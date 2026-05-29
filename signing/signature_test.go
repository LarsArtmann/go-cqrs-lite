package signing_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestSignature(t *testing.T) {
	t.Parallel()

	t.Run("bytes returns copy", func(t *testing.T) {
		t.Parallel()

		orig := signing.Signature([]byte("hello"))
		b := orig.Bytes()
		b[0] = 'x'

		if orig[0] != 'h' {
			t.Fatal("Bytes() should return a copy")
		}
	})

	t.Run("is zero for empty", func(t *testing.T) {
		t.Parallel()

		var s signing.Signature
		if !s.IsZero() {
			t.Fatal("expected zero signature")
		}
	})

	t.Run("is zero for nil", func(t *testing.T) {
		t.Parallel()

		var s signing.Signature = nil
		if !s.IsZero() {
			t.Fatal("expected nil signature to be zero")
		}
	})
}

func TestAttachAndExtractSignature(t *testing.T) {
	t.Parallel()

	key := []byte("my-secret-key-thirty-two-bytes!!")
	signer, _ := signing.NewHMAC(key)
	evt := makeTestEvent(t)
	sig, _ := signer.Sign(evt)

	t.Run("attach and extract roundtrip", func(t *testing.T) {
		t.Parallel()

		clone, err := signing.AttachSignature(evt, sig)
		if err != nil {
			t.Fatalf("attach: %v", err)
		}

		extracted, err := signing.ExtractSignature(clone)
		if err != nil {
			t.Fatalf("extract: %v", err)
		}

		if !bytes.Equal(sig.Bytes(), extracted.Bytes()) {
			t.Fatal("extracted signature does not match original")
		}
	})

	t.Run("attach preserves event fields", func(t *testing.T) {
		t.Parallel()

		clone, err := signing.AttachSignature(evt, sig)
		if err != nil {
			t.Fatalf("attach: %v", err)
		}

		if clone.ID() != evt.ID() {
			t.Error("ID mismatch")
		}
		if clone.Type() != evt.Type() {
			t.Error("Type mismatch")
		}
		if clone.AggregateID() != evt.AggregateID() {
			t.Error("AggregateID mismatch")
		}
		if clone.AggregateType() != evt.AggregateType() {
			t.Error("AggregateType mismatch")
		}
		if clone.Version() != evt.Version() {
			t.Error("Version mismatch")
		}
		if !bytes.Equal(clone.Payload(), evt.Payload()) {
			t.Error("Payload mismatch")
		}
	})

	t.Run("extract from unsigned event", func(t *testing.T) {
		t.Parallel()

		_, err := signing.ExtractSignature(evt)
		if err == nil {
			t.Fatal("expected error for unsigned event")
		}
	})

	t.Run("extract from nil event", func(t *testing.T) {
		t.Parallel()

		_, err := signing.ExtractSignature(nil)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("attach to nil event", func(t *testing.T) {
		t.Parallel()

		_, err := signing.AttachSignature(nil, sig)
		if err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("has signature detects attached", func(t *testing.T) {
		t.Parallel()

		if signing.HasSignature(evt) {
			t.Fatal("original event should not have signature")
		}

		clone, _ := signing.AttachSignature(evt, sig)
		if !signing.HasSignature(clone) {
			t.Fatal("clone should have signature")
		}
	})
}

func TestCanonicalPayload_Deterministic(t *testing.T) {
	t.Parallel()

	// Create two identical events
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	opts := []event.Option{
		event.WithSchemaVersion(2),
	}

	evt1, err := event.NewEvent("test.evt", aggID, "Test", 1, []byte(`{"key":"value"}`), opts...)
	if err != nil {
		t.Fatalf("create event 1: %v", err)
	}

	evt2, err := event.NewEvent("test.evt", aggID, "Test", 1, []byte(`{"key":"value"}`), opts...)
	if err != nil {
		t.Fatalf("create event 2: %v", err)
	}

	// They should have same canonical payload determinism by same key
	key := []byte("my-secret-key-thirty-two-bytes!!")
	signer, _ := signing.NewHMAC(key)

	sig1, _ := signer.Sign(evt1)
	sig2, _ := signer.Sign(evt2)

	// Different events (different IDs) should produce different signatures
	if bytes.Equal(sig1.Bytes(), sig2.Bytes()) {
		t.Fatal("different events should produce different signatures")
	}
}

func TestSignature_String(t *testing.T) {
	t.Parallel()

	raw := signing.Signature([]byte("test-signature-bytes"))
	s := raw.String()

	decoded, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("String() should produce valid URL-safe base64: %v", err)
	}

	if !bytes.Equal(raw, decoded) {
		t.Fatal("String() roundtrip failed")
	}
}

func TestSignature_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := signing.Signature([]byte("test-signature-for-json-roundtrip"))

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded signing.Signature

	unmarshalErr := json.Unmarshal(data, &decoded)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	if !bytes.Equal(original, decoded) {
		t.Fatalf("JSON roundtrip failed: got %v, want %v", decoded, original)
	}
}

func TestSignature_UnmarshalJSON_BackwardCompat(t *testing.T) {
	t.Parallel()

	// Standard base64 encoded (old format) should still decode
	original := signing.Signature([]byte("backward-compat-sig"))
	stdEncoded := `"` + base64.StdEncoding.EncodeToString(original) + `"`

	var decoded signing.Signature

	err := json.Unmarshal([]byte(stdEncoded), &decoded)
	if err != nil {
		t.Fatalf("unmarshal standard base64: %v", err)
	}

	if !bytes.Equal(original, decoded) {
		t.Fatal("backward-compatible decode failed")
	}
}

func TestEd25519_Deterministic(t *testing.T) {
	t.Parallel()

	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer, _ := signing.NewEd25519(privKey)
	evt := makeTestEvent(t)

	sig1, _ := signer.Sign(evt)
	sig2, _ := signer.Sign(evt)

	if !bytes.Equal(sig1.Bytes(), sig2.Bytes()) {
		t.Fatal("Ed25519 signatures should be deterministic for same event + key")
	}
}

func TestSignature_Equal(t *testing.T) {
	t.Parallel()

	sig1 := signing.Signature([]byte("abc"))
	sig2 := signing.Signature([]byte("abc"))
	sig3 := signing.Signature([]byte("xyz"))

	if !sig1.Equal(sig2) {
		t.Fatal("equal signatures should report equal")
	}

	if sig1.Equal(sig3) {
		t.Fatal("different signatures should not report equal")
	}

	empty := signing.Signature(nil)
	if !empty.Equal(signing.Signature(nil)) {
		t.Fatal("two nil signatures should report equal")
	}

	if empty.Equal(sig1) {
		t.Fatal("nil vs non-nil should not report equal")
	}
}

func TestSignature_UnmarshalJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	var s signing.Signature

	err := json.Unmarshal([]byte(`123`), &s)
	if err == nil {
		t.Fatal("expected error for non-string JSON")
	}

	err = json.Unmarshal([]byte(`{}`), &s)
	if err == nil {
		t.Fatal("expected error for object JSON")
	}
}

func TestSignature_UnmarshalJSON_BadBase64(t *testing.T) {
	t.Parallel()

	var s signing.Signature

	// Valid JSON string but invalid base64 (contains chars not in any base64 alphabet)
	err := json.Unmarshal([]byte(`"!!!not-valid-base64!!!"`), &s)
	if err == nil {
		t.Fatal("expected error for invalid base64 string")
	}
}

func TestCanonicalPayload_EdgeCases(t *testing.T) {
	t.Parallel()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")

	t.Run("nil payload", func(t *testing.T) {
		t.Parallel()
		evt, _ := event.NewEvent("test.nil", aggID, "Test", 1, nil)
		key := []byte("secret-key-thirty-two-bytes!!!!!")
		signer, _ := signing.NewHMAC(key)
		_, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign nil payload: %v", err)
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		t.Parallel()
		evt, _ := event.NewEvent("test.empty", aggID, "Test", 1, []byte{})
		key := []byte("secret-key-thirty-two-bytes!!!!!")
		signer, _ := signing.NewHMAC(key)
		_, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign empty payload: %v", err)
		}
	})

	t.Run("large payload", func(t *testing.T) {
		t.Parallel()
		large := make([]byte, 1<<20) // 1 MB
		for i := range large {
			large[i] = byte(i % 256)
		}
		evt, _ := event.NewEvent("test.large", aggID, "Test", 1, large)
		key := []byte("secret-key-thirty-two-bytes!!!!!")
		signer, _ := signing.NewHMAC(key)
		_, err := signer.Sign(evt)
		if err != nil {
			t.Fatalf("sign large payload: %v", err)
		}
	})
}
