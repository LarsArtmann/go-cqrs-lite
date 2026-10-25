package pebbleengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Stored-bytes pins for the value encoder (keycodec.EncodeJSONErr): the bytes
// Pebble persists must be the value's JSON rendering, never its %v string.
// Expectations are literal constants on purpose — an expectation derived from
// the encoder cannot catch an encoder that agrees with itself. Values use
// structs, not maps: encoding/json/v2 does not sort map keys, so map bytes are
// iteration-order dependent and un-pinnable.
const (
	pinValueJSON  = `{"title":"pin","n":7}`
	pinStringJSON = `"raw-string"`
	pinIntJSON    = "42"
)

type pinStruct struct {
	Title string `json:"title"`
	N     int    `json:"n"`
}

func TestStoredBytes_MapSet(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rr := eng.(metaengine.RawValueReader)

	writes := []struct {
		key  string
		val  any
		want string
	}{
		{"k1", pinStruct{Title: "pin", N: 7}, pinValueJSON},
		{"k2", "raw-string", pinStringJSON},
		{"k3", 42, pinIntJSON},
	}

	for _, w := range writes {
		if err := mb.MapSet(ctx, "pin_map", w.key, w.val); err != nil {
			t.Fatalf("MapSet(%s): %v", w.key, err)
		}

		raw, found, err := rr.GetRawValue(ctx, "pin_map", w.key)
		if err != nil || !found {
			t.Fatalf("GetRawValue(%s): found=%v err=%v", w.key, found, err)
		}

		if string(raw) != w.want {
			t.Fatalf("stored bytes for %s = %q, want %q", w.key, raw, w.want)
		}
	}
}

// TestStoredBytes_StreamAppend_DecodesTyped pins the stream path at decode
// level: a %v stringification of the map would round-trip as a flat STRING,
// not a map. Raw stream bytes live at engine-internal keys, so the pin asserts
// the decoded shape StreamRead produces from whatever was stored.
func TestStoredBytes_StreamAppend_DecodesTyped(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
	ctx := context.Background()
	sl := eng.(metaengine.StreamLogBackend)

	err := sl.StreamAppend(ctx, "pin_streams", "s-1", []any{
		map[string]any{"title": "pin", "n": 7}, // map: typed-decode assert, no byte literal (v2 map order is random)
		"raw-string",
		42,
	})
	if err != nil {
		t.Fatalf("StreamAppend: %v", err)
	}

	got, err := sl.StreamRead(ctx, "pin_streams", "s-1")
	if err != nil {
		t.Fatalf("StreamRead: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("stream holds %d values, want 3", len(got))
	}

	obj, ok := got[0].(map[string]any)
	if !ok {
		t.Fatalf("stream value 0 = %T (%v), want map[string]any — the %v fallback stringifies", got[0], got[0], got[0])
	}

	if obj["title"] != "pin" || obj["n"] != float64(7) {
		t.Fatalf("stream value 0 = %#v, want {title:pin n:7}", obj)
	}

	if got[1] != "raw-string" {
		t.Fatalf("stream value 1 = %#v, want raw-string", got[1])
	}

	if got[2] != float64(42) {
		t.Fatalf("stream value 2 = %#v, want 42", got[2])
	}
}

// TestStoredBytes_MapUpdate_ReEncodes pins the update path: the RMW result is
// re-encoded through the same rejection encoder, so an update-path encoder
// change cannot hide behind MapSet's pin.
func TestStoredBytes_MapUpdate_ReEncodes(t *testing.T) {
	t.Parallel()

	eng := mustNewPebbleEngine(t)
	ctx := context.Background()
	mb := eng.(metaengine.MapBackend)
	rr := eng.(metaengine.RawValueReader)
	mu := eng.(metaengine.MapUpdater)

	if err := mb.MapSet(ctx, "pin_update", "k1", pinStruct{Title: "pin", N: 7}); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := mu.MapUpdate(ctx, "pin_update", "k1", func(prev any) any {
		return pinStruct{Title: "updated", N: 9}
	}); err != nil {
		t.Fatalf("MapUpdate: %v", err)
	}

	raw, found, err := rr.GetRawValue(ctx, "pin_update", "k1")
	if err != nil || !found {
		t.Fatalf("GetRawValue: found=%v err=%v", found, err)
	}

	if want := `{"title":"updated","n":9}`; string(raw) != want {
		t.Fatalf("stored bytes after update = %q, want %q", raw, want)
	}
}
