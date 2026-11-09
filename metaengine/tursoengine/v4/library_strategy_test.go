package tursoengine_test

import (
	"os"
	"testing"

	turso_libs "github.com/tursodatabase/turso-go-platform-libs"
	tursogo "turso.tech/database/tursogo"
)

// TestMain can run the whole suite against an externally built engine
// library: with TURSO_TEST_SYSTEM_LIB=1 the tursogo native library is
// pre-loaded via the SYSTEM strategy (LD_LIBRARY_PATH search for
// libturso_sync_sdk_kit.so) before any test opens a connection.
// tursogo.InitLibrary is a sync.Once and the first call wins — the
// driver's later embedded-strategy calls in Open/Setup become no-ops — so
// every engine in the process, including the library warmup ping, runs on
// the LD_LIBRARY_PATH library instead of the go.mod-pinned one.
//
// Purpose: verifying upstream engine fixes BEFORE they are tagged into
// github.com/tursodatabase/turso-go-platform-libs — build the core repo at
// a chosen ref (cargo build --profile lib-release --package
// turso_sync_sdk_kit, per that module's libs/build_libs.sh) and point
// LD_LIBRARY_PATH at the built libturso_sync_sdk_kit.so. InitLibrary
// panics when the library cannot be loaded, so a set-but-unreadable
// TURSO_TEST_SYSTEM_LIB fails loudly instead of silently testing the
// pinned engine. Unset, the suite behaves exactly as before.
func TestMain(m *testing.M) {
	if os.Getenv("TURSO_TEST_SYSTEM_LIB") == "1" {
		tursogo.InitLibrary(turso_libs.LoadTursoLibraryConfig{
			LoadStrategy: turso_libs.SystemLibraryLoadStrategy,
		})
	}

	os.Exit(m.Run())
}
