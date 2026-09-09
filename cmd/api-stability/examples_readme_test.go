package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEveryExampleHasREADME fails when an example module ships without a
// README — the examples are the copy-paste surface for new consumers, and a
// missing README is a missing landing page (metaengine-quickstart shipped
// without one for weeks before this tripwire existed).
func TestEveryExampleHasREADME(t *testing.T) {
	t.Parallel()

	examplesDir := filepath.Join(".", "..", "..", "example")

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("read example dir: %v", err)
	}

	var missing []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(examplesDir, entry.Name())

		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
			continue // not a module (e.g. shared testdata)
		}

		if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
			missing = append(missing, entry.Name())
		}
	}

	if len(missing) > 0 {
		t.Errorf("example modules without README.md (the copy-paste surface): %v", missing)
	}
}
