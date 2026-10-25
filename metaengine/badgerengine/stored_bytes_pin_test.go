package badgerengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/badger/v4"

	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// Stored-bytes pin for the value encoder (keycodec.EncodeJSONErr): the bytes
// Badger persists must be the value's JSON rendering, never its %v string.
// The key layout (keycodec.MapKey over keycodec.EncodeKeyStr) is an on-disk
// format contract, and the expectation is a literal constant on purpose — an
// expectation derived from the encoder cannot catch an encoder that agrees
// with itself.
func TestStoredBytes_MapSet(t *testing.T) {
	t.Parallel()

	db, err := badger.Open(badger.DefaultOptions(filepath.Join(t.TempDir(), "pin")))
	if err != nil {
		t.Skipf("badger not available: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eng, err := badgerengine.NewBadgerEngineFromDB(db)
	if err != nil {
		t.Fatalf("NewBadgerEngineFromDB: %v", err)
	}

	ctx := context.Background()

	if err := eng.(metaengine.MapBackend).MapSet(
		ctx, "pin_map", "k1", map[string]any{"title": "pin", "n": 7},
	); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	var stored []byte

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(keycodec.MapKey("pin_map", keycodec.EncodeKeyStr("k1")))
		if err != nil {
			return err
		}

		stored, err = item.ValueCopy(nil)

		return err
	})
	if err != nil {
		t.Fatalf("read key: %v", err)
	}

	if want := `{"title":"pin","n":7}`; string(stored) != want {
		t.Fatalf("stored bytes = %q, want %q", stored, want)
	}
}
