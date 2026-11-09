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
// with itself. The value is a struct because encoding/json/v2 does not sort
// map keys (struct field order is the only deterministic shape).
type pinStruct struct {
	Title string `json:"title"`
	N     int    `json:"n"`
}

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
		ctx, "pin_map", "k1", pinStruct{Title: "pin", N: 7},
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

// TestStoredBytes_MapUpdate_ReEncodes pins the update path: the RMW result is
// re-encoded through the same rejection encoder, so an update-path encoder
// change cannot hide behind MapSet's pin.
func TestStoredBytes_MapUpdate_ReEncodes(t *testing.T) {
	t.Parallel()

	db, err := badger.Open(badger.DefaultOptions(filepath.Join(t.TempDir(), "pin-update")))
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
		ctx, "pin_update", "k1", pinStruct{Title: "pin", N: 7},
	); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := eng.(metaengine.MapUpdater).MapUpdate(ctx, "pin_update", "k1", func(prev any) any {
		return pinStruct{Title: "updated", N: 9}
	}); err != nil {
		t.Fatalf("MapUpdate: %v", err)
	}

	var stored []byte

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(keycodec.MapKey("pin_update", keycodec.EncodeKeyStr("k1")))
		if err != nil {
			return err
		}

		stored, err = item.ValueCopy(nil)

		return err
	})
	if err != nil {
		t.Fatalf("txn read: %v", err)
	}

	if want := `{"title":"updated","n":9}`; string(stored) != want {
		t.Fatalf("stored bytes after update = %q, want %q", stored, want)
	}
}
