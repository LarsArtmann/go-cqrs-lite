package pebble

import (
	"testing"

	"github.com/cockroachdb/pebble"

	cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
)

// applyOptions mirrors New's option application without opening a database.
func applyOptions(opts ...Option) *pebble.Options {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg.pebbleOpts
}

// TestKnobs_DefaultsByteIdentical pins the F38 contract: building a preset
// with NO knob options leaves every tuned field exactly as
// cqrspebble.DefaultOptions() set it. Knobs are pure opt-in.
func TestKnobs_DefaultsByteIdentical(t *testing.T) {
	t.Parallel()

	built := applyOptions()
	want := cqrspebble.DefaultOptions()

	if built.MemTableSize != want.MemTableSize {
		t.Errorf("MemTableSize = %d, want default %d", built.MemTableSize, want.MemTableSize)
	}

	if built.WALBytesPerSync != want.WALBytesPerSync {
		t.Errorf("WALBytesPerSync = %d, want default %d", built.WALBytesPerSync, want.WALBytesPerSync)
	}

	if built.Cache != want.Cache {
		t.Errorf("Cache = %v, want default %v", built.Cache, want.Cache)
	}

	for i := range want.Levels {
		if built.Levels[i].Compression != want.Levels[i].Compression {
			t.Errorf("Levels[%d].Compression = %v, want default %v",
				i, built.Levels[i].Compression, want.Levels[i].Compression)
		}
	}
}

// TestKnobs_ApplyOnlyTheirField proves each knob touches exactly its field
// and leaves every other default untouched. The block cache is exercised in
// TestKnobs_BlockCacheLifecycle below.
func TestKnobs_ApplyOnlyTheirField(t *testing.T) {
	t.Parallel()

	built := applyOptions(
		WithMemTableSize(16<<20),
		WithWALBytesPerSync(64<<10),
		WithPebbleCompression(pebble.NoCompression),
	)
	want := cqrspebble.DefaultOptions()

	if built.MemTableSize != 16<<20 {
		t.Errorf("MemTableSize = %d, want %d", built.MemTableSize, 16<<20)
	}

	if built.WALBytesPerSync != 64<<10 {
		t.Errorf("WALBytesPerSync = %d, want %d", built.WALBytesPerSync, 64<<10)
	}

	for i := range built.Levels {
		if built.Levels[i].Compression != pebble.NoCompression {
			t.Errorf("Levels[%d].Compression = %v, want NoCompression", i, built.Levels[i].Compression)
		}
	}

	if built.Cache != want.Cache {
		t.Errorf("Cache = %v, untouched default expected", built.Cache)
	}
}

// TestKnobs_BlockCacheLifecycle proves WithBlockCacheSize installs a cache
// on the options and a repeated call releases the previous one (no ref leak).
func TestKnobs_BlockCacheLifecycle(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	WithBlockCacheSize(1 << 20)(&cfg)
	first := cfg.blockCache
	if cfg.pebbleOpts.Cache != first {
		t.Fatal("WithBlockCacheSize did not install the cache on pebbleOpts")
	}

	WithBlockCacheSize(2 << 20)(&cfg)
	if cfg.blockCache == first {
		t.Fatal("second WithBlockCacheSize should replace the cache")
	}

	if cfg.pebbleOpts.Cache != cfg.blockCache {
		t.Fatal("replacement cache not installed on pebbleOpts")
	}

	cfg.blockCache.Unref()
}
