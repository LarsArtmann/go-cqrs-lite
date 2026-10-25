package bboltengine_test

import (
	"context"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/bboltengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// Stored-bytes pin for the value encoder (keycodec.EncodeJSONErr): the bytes
// in the cqrs_meta bucket must be the value's JSON rendering, never its %v
// string. The bucket name and key layout (keycodec.MapKey over
// keycodec.EncodeKeyStr) are on-disk format contracts — a database written by
// an older engine version must stay readable, so pinning the literals is the
// point. Expectations are constants on purpose: an expectation derived from
// the encoder cannot catch an encoder that agrees with itself.
func TestStoredBytes_MapSet(t *testing.T) {
	t.Parallel()

	db, err := bolt.Open(filepath.Join(t.TempDir(), "pin.db"), 0o600, nil)
	if err != nil {
		t.Skipf("bbolt not available: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := bboltengine.NewBboltEngineFromDB(db)
	if err != nil {
		t.Fatalf("NewBboltEngineFromDB: %v", err)
	}

	ctx := context.Background()

	if err := eng.(metaengine.MapBackend).MapSet(
		ctx, "pin_map", "k1", map[string]any{"title": "pin", "n": 7},
	); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	k := keycodec.MapKey("pin_map", keycodec.EncodeKeyStr("k1"))

	var stored []byte

	err = db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket([]byte("cqrs_meta")).Get(k)
		stored = append([]byte(nil), v...)

		return nil
	})
	if err != nil {
		t.Fatalf("read bucket: %v", err)
	}

	if want := `{"title":"pin","n":7}`; string(stored) != want {
		t.Fatalf("stored bytes = %q, want %q", stored, want)
	}
}
