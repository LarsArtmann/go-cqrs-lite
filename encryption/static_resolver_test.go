package encryption

import (
	"testing"
)

func TestStaticKeyResolver_Resolve(t *testing.T) {
	t.Parallel()

	decV1, _ := NewAES256GCM(aes256Key())
	decV2, _ := NewAES256GCM(aes256Key())

	keys := map[KeyID]Decrypter{
		"key-v1": decV1,
		"key-v2": decV2,
	}

	resolver := NewStaticKeyResolver(keys)

	dec, err := resolver.Resolve("key-v1")
	if err != nil {
		t.Fatalf("resolve key-v1: %v", err)
	}

	if dec != decV1 {
		t.Error("expected decV1")
	}

	dec, err = resolver.Resolve("key-v2")
	if err != nil {
		t.Fatalf("resolve key-v2: %v", err)
	}

	if dec != decV2 {
		t.Error("expected decV2")
	}
}

func TestStaticKeyResolver_UnknownKey(t *testing.T) {
	t.Parallel()

	dec, _ := NewAES256GCM(aes256Key())
	resolver := NewStaticKeyResolver(map[KeyID]Decrypter{"key-v1": dec})

	_, err := resolver.Resolve("key-v99")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}

	want := `encryption: unknown key "key-v99" (available: key-v1)`
	if got := err.Error(); got != want {
		t.Errorf("error message:\n got: %s\nwant: %s", got, want)
	}
}

func TestStaticKeyResolver_AvailableKeys(t *testing.T) {
	t.Parallel()

	dec, _ := NewAES256GCM(aes256Key())
	resolver := NewStaticKeyResolver(map[KeyID]Decrypter{
		"key-beta":  dec,
		"key-alpha": dec,
	})

	_, err := resolver.Resolve("key-missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}

	want := `encryption: unknown key "key-missing" (available: key-alpha, key-beta)`
	if got := err.Error(); got != want {
		t.Errorf("error message:\n got: %s\nwant: %s", got, want)
	}
}

func TestStaticKeyResolver_DefensiveCopy(t *testing.T) {
	t.Parallel()

	dec, _ := NewAES256GCM(aes256Key())
	original := map[KeyID]Decrypter{"key-v1": dec}
	resolver := NewStaticKeyResolver(original)

	original["key-v2"] = dec

	_, err := resolver.Resolve("key-v2")
	if err == nil {
		t.Error("resolver should not see mutations to original map")
	}
}

func TestStaticKeyResolver_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ KeyResolver = NewStaticKeyResolver(nil)

	resolver := NewStaticKeyResolver(nil)
	_, err := resolver.Resolve("any")
	if err == nil {
		t.Fatal("empty resolver should reject all keys")
	}
}

func TestStaticKeyResolver_RoundTrip(t *testing.T) {
	t.Parallel()

	key := aes256Key()
	enc, _ := NewAES256GCM(key)
	dec, _ := NewAES256GCM(key)

	resolver := NewStaticKeyResolver(map[KeyID]Decrypter{"key-v1": dec})

	plaintext := []byte(`{"secret":"data"}`)
	ct, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	gotDec, err := resolver.Resolve("key-v1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	result, err := gotDec.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(result) != string(plaintext) {
		t.Errorf("round-trip mismatch:\n got: %s\nwant: %s", result, plaintext)
	}
}

func aes256Key() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}


