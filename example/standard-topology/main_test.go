package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestStandardTopologyBootsAndDispatches proves the whole ADR-0165 story:
// an engines-only deployment JSON loads, system.Standard fills the
// three-instance topology, system.New boots sqlite, one command dispatches,
// and the Lookup projection folds it back.
func TestStandardTopologyBootsAndDispatches(t *testing.T) {
	dir := t.TempDir()

	config := `{
		"engines": {
			"primary": {"driver": "sqlite", "dsn": "file:` + filepath.ToSlash(filepath.Join(dir, "notes.db")) + `"}
		}
	}`

	path := filepath.Join(dir, "deployment.json")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// run prints the view and the plan; success IS the assertion — any break
	// in the load → Standard → New → dispatch → fold chain fails the run.
	if err := run(context.Background(), path); err != nil {
		t.Fatalf("standard-topology run: %v", err)
	}
}
