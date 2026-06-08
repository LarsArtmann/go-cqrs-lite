// Package snaptest provides minimal snapshot testing without external dependencies.
// Set UPDATE_SNAPS=true to regenerate golden files.
package snaptest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Match compares data against a golden file. If UPDATE_SNAPS is set,
// it writes the data to the golden file instead.
func Match(tb testing.TB, name string, data []byte) {
	tb.Helper()

	golden := filepath.Join("testdata", "snapshots", name+".snap")
	dir := filepath.Dir(golden)

	if os.Getenv("UPDATE_SNAPS") == "true" {
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			tb.Fatalf("create snapshot dir: %v", err)
		}

		err = os.WriteFile(golden, data, 0o600)
		if err != nil {
			tb.Fatalf("write snapshot: %v", err)
		}

		tb.Logf("updated snapshot: %s", golden)

		return
	}

	expected, err := os.ReadFile(golden)
	if err != nil {
		if os.IsNotExist(err) {
			tb.Fatalf("snapshot not found: %s (run with UPDATE_SNAPS=true to create)", golden)
		}

		tb.Fatalf("read snapshot: %v", err)
	}

	if !bytes.Equal(data, expected) {
		diff := fmt.Sprintf("snapshot mismatch: %s\n--- expected ---\n%s\n--- actual ---\n%s",
			golden, string(expected), string(data))

		tb.Fatal(diff)
	}
}
