package tursoengine_test

import (
	"testing"

	tursoengine "github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"
)

// TestNew_ColdCacheLibraryWarmup pins the cold-start path through the
// engine's driver registration: with a fresh TURSO_GO_CACHE_DIR the first
// Open must warm the library (serialized by the warmup lockfile) and return
// a usable engine instead of the pre-fix "cached library file hash sum
// mismatch" panic from tursogo.InitLibrary. The file-existence and
// cross-process legs live in storage/turso's child-mode test — this process
// may have warmed earlier via another test, so only the clean Open is
// assertable here.
func TestNew_ColdCacheLibraryWarmup(t *testing.T) {
	t.Setenv("TURSO_GO_CACHE_DIR", t.TempDir())

	engine, err := tursoengine.New("")
	if err != nil {
		t.Fatalf("New with cold cache dir: %v", err)
	}

	if err := engine.Close(); err != nil {
		t.Fatalf("close engine: %v", err)
	}
}
