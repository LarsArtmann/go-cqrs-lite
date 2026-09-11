package tursoengine_test

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newEncryptedEngine builds an encrypted engine over a fresh file DB and
// returns it with its MapBackend view.
func newEncryptedEngine(t *testing.T, dsn string) (metaengine.Engine, metaengine.MapBackend) {
	t.Helper()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	eng, err := tursoengine.New(dsn,
		tursoengine.WithEncryption(tursoengine.CipherAEGIS256, hex.EncodeToString(key)))
	if err != nil {
		t.Fatalf("open encrypted engine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatal("engine does not implement MapBackend")
	}

	return eng, mb
}

// The `file:` DSN scheme must round-trip under encryption exactly like a
// plain path (the README documents both forms as embedded use).
func TestTursoEncryption_FileDSNRoundTrip(t *testing.T) {
	t.Parallel()

	dsn := "file:" + t.TempDir() + "/file-scheme.db"

	_, mb := newEncryptedEngine(t, dsn)

	if err := mb.MapSet(t.Context(), "secrets", "k1", "v1"); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	val, ok, err := mb.MapGet(t.Context(), "secrets", "k1")
	if err != nil || !ok {
		t.Fatalf("MapGet after write on file: DSN (ok=%v err=%v)", ok, err)
	}

	if val != "v1" {
		t.Fatalf("round-trip must preserve the value, got %v", val)
	}
}

// Encryption covers EVERY ADT surface, not just the map table: counters (a
// second table shape) round-trip and survive close/reopen with the key.
func TestTursoEncryption_SecondADTRoundTrip(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	dsn := t.TempDir() + "/counter-enc.db"
	hexKey := hex.EncodeToString(key)

	eng, err := tursoengine.New(dsn,
		tursoengine.WithEncryption(tursoengine.CipherAEGIS256, hexKey))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	cb := eng.(metaengine.CounterBackend)
	if err := cb.CounterIncrement(t.Context(), "counts", metaengine.Delta{"created": 3, "done": 2}); err != nil {
		_ = eng.Close()
		t.Fatalf("CounterIncrement: %v", err)
	}

	sl := eng.(metaengine.StreamLogBackend)
	if err := sl.StreamAppend(t.Context(), "journal", "s1", []any{"e1", "e2"}); err != nil {
		_ = eng.Close()
		t.Fatalf("StreamAppend: %v", err)
	}

	if err := eng.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	eng2, err := tursoengine.New(dsn,
		tursoengine.WithEncryption(tursoengine.CipherAEGIS256, hexKey))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	defer eng2.Close() //nolint:errcheck // test cleanup

	counters, err := eng2.(metaengine.CounterBackend).CounterGet(t.Context(), "counts")
	if err != nil {
		t.Fatalf("CounterGet after reopen: %v", err)
	}

	if counters["created"] != 3 || counters["done"] != 2 {
		t.Fatalf("counters must survive close/reopen under encryption, got %v", counters)
	}

	stream, err := eng2.(metaengine.StreamLogBackend).StreamRead(t.Context(), "journal", "s1")
	if err != nil || len(stream) != 2 {
		t.Fatalf("journal must survive close/reopen under encryption (len=%d err=%v)", len(stream), err)
	}
}

// A materialized view on an ENCRYPTED engine must actually SERVE the
// aggregate (the coexistence test constructs and writes but never queries):
// the count aggregate reads back through the encrypted view machinery.
func TestTursoEncryption_MatViewServesAggregate(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	dsn := t.TempDir() + "/enc-mv-serve.db"

	eng, err := tursoengine.New(dsn,
		tursoengine.WithEncryption(tursoengine.CipherAEGIS256, hex.EncodeToString(key)),
		tursoengine.WithMaterializedViews([]metaengine.MaterializedViewSpec{
			{Collection: "orders", Fn: metaengine.MatViewCount},
		}),
	)
	if err != nil {
		t.Fatalf("open encrypted engine with matview: %v", err)
	}

	defer eng.Close() //nolint:errcheck // test cleanup

	mb := eng.(metaengine.MapBackend)
	for i := range 3 {
		if err := mb.MapSet(t.Context(), "orders", string(rune('a'+i)), map[string]any{"amount": i + 1}); err != nil {
			t.Fatalf("MapSet %d: %v", i, err)
		}
	}

	agg := eng.(metaengine.AggregateReader)
	count, err := agg.Aggregate(t.Context(), "orders", metaengine.MatViewCount, "", nil)
	if err != nil {
		t.Fatalf("Aggregate count on encrypted engine: %v", err)
	}

	if count != 3 {
		t.Fatalf("matview-served count on encrypted engine must be 3, got %v", count)
	}
}

// The key-size error carries the exact actionable hint (cipher name + the
// openssl one-liner); pinning the text keeps the operator contract stable.
func TestTursoEncryption_KeySizeHintExactText(t *testing.T) {
	t.Parallel()

	// 16 raw bytes offered to a 32-byte cipher.
	_, err := tursoengine.New(t.TempDir()+"/hint.db",
		tursoengine.WithEncryption(tursoengine.CipherAEGIS256, hex.EncodeToString(make([]byte, 16))))
	if err == nil {
		t.Fatal("undersized key must be rejected")
	}

	want := "encryption key is 16 bytes, want 32 for aegis256 (openssl rand -hex 32)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("size hint must carry the exact remedy text\nwant substring: %s\ngot: %v", want, err)
	}
}
