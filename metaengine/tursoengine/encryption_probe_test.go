package tursoengine_test

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbe_EncryptionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "enc.db")

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	hexKey := hex.EncodeToString(key)
	dsn := dbPath + "?experimental=encryption&encryption_cipher=aes256gcm&encryption_hexkey=" + hexKey

	eng, err := tursoengine.New(dsn)
	if err != nil {
		t.Fatalf("open encrypted failed: %v", err)
	}

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		_ = eng.Close()
		t.Fatal("no MapBackend")
	}

	if err := mb.MapSet(t.Context(), "t", "k", map[string]any{"v": "secret"}); err != nil {
		_ = eng.Close()
		t.Fatalf("MapSet failed: %v", err)
	}

	if err := eng.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	eng2, err := tursoengine.New(dsn)
	if err != nil {
		t.Fatalf("reopen WITH key failed: %v", err)
	}
	_ = eng2.Close()

	badDSN := dbPath + "?experimental=encryption&encryption_cipher=aes256gcm&encryption_hexkey=" + hex.EncodeToString(make([]byte, 32))
	eng3, err := tursoengine.New(badDSN)
	if err != nil {
		t.Logf("reopen with WRONG key failed as expected: %v", err)
		return
	}
	_ = eng3.Close()
	t.Fatal("reopen with WRONG key unexpectedly succeeded")
}
