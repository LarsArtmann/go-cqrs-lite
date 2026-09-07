package tursoengine_test

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestTursoEncryption_RoundTrip(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		cipher tursoengine.Cipher
		keyLen int
	}{
		{name: "aes256gcm", cipher: tursoengine.CipherAES256GCM, keyLen: 32},
		{name: "aegis256", cipher: tursoengine.CipherAEGIS256, keyLen: 32},
		{name: "aegis128l", cipher: tursoengine.CipherAEGIS128L, keyLen: 16},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := make([]byte, tt.keyLen)
			if _, err := rand.Read(key); err != nil {
				t.Fatal(err)
			}

			dsn := filepath.Join(t.TempDir(), "enc.db")

			eng, err := tursoengine.New(dsn, tursoengine.WithEncryption(tt.cipher, hex.EncodeToString(key)))
			if err != nil {
				t.Fatalf("open encrypted engine: %v", err)
			}

			mb, ok := eng.(metaengine.MapBackend)
			if !ok {
				_ = eng.Close()
				t.Fatal("engine does not implement MapBackend")
			}

			if err := mb.MapSet(t.Context(), "secrets", "k1", map[string]any{"v": "s3cret"}); err != nil {
				_ = eng.Close()
				t.Fatalf("MapSet: %v", err)
			}

			if err := eng.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			eng2, err := tursoengine.New(dsn, tursoengine.WithEncryption(tt.cipher, hex.EncodeToString(key)))
			if err != nil {
				t.Fatalf("reopen with correct key: %v", err)
			}

			if err := eng2.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			wrongKey := hex.EncodeToString(make([]byte, tt.keyLen))
			eng3, err := tursoengine.New(dsn, tursoengine.WithEncryption(tt.cipher, wrongKey))
			if err == nil {
				_ = eng3.Close()
				t.Fatal("reopen with all-zero key unexpectedly succeeded")
			}
		})
	}
}

func TestTursoEncryption_WithMaterializedViews(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	dsn := filepath.Join(t.TempDir(), "enc-mv.db")

	specs := []metaengine.MaterializedViewSpec{
		{Collection: "orders", Fn: metaengine.MatViewCount},
	}

	eng, err := tursoengine.New(dsn,
		tursoengine.WithEncryption(tursoengine.CipherAEGIS256, hex.EncodeToString(key)),
		tursoengine.WithMaterializedViews(specs),
	)
	if err != nil {
		t.Fatalf("encrypted engine with materialized views: %v", err)
	}

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		_ = eng.Close()
		t.Fatal("engine does not implement MapBackend")
	}

	if err := mb.MapSet(t.Context(), "orders", "o1", map[string]any{"v": 1}); err != nil {
		_ = eng.Close()
		t.Fatalf("MapSet: %v", err)
	}

	if err := eng.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
