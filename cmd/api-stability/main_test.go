package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestEveryGoModDirIsInModulesList asserts that every directory containing a
// go.mod (except examples, integration, the root workspace, and this tool's own
// module) appears in the modules slice. This catches the class of omission
// where a published module ships without its API surface being tracked.
func TestEveryGoModDirIsInModulesList(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")

	// Build a set of tracked module paths for O(1) lookup.
	tracked := make(map[string]struct{}, len(modules))
	for _, m := range modules {
		tracked[m] = struct{}{}
	}

	// Directories that are intentionally excluded from the api-stability gate.
	excluded := map[string]string{
		".":                         "root workspace go.mod",
		"cmd/api-stability":         "the api-stability tool itself (circular)",
		"integration":               "workspace-only cross-module tests (published graph not self-contained)",
		"example/getting-started":   "example application",
		"example/readme-quickstart": "example application",
		"example/taskmanager":       "example application",
	}

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		// Skip hidden dirs and vendor.
		name := info.Name()
		if name == ".git" || name == "vendor" ||
			(len(name) > 0 && name[0] == '.' && path != projectRoot) {
			return filepath.SkipDir
		}
		// Check for go.mod in this directory.
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err != nil {
			return nil // no go.mod here, keep walking
		}
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		if reason, ok := excluded[rel]; ok {
			t.Logf("excluding %s (%s)", rel, reason)
			return nil
		}
		if _, ok := tracked[rel]; !ok {
			t.Errorf("directory %q has a go.mod but is NOT in the modules list in main.go — "+
				"add it to the modules slice so its API surface is tracked", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}

func TestAPISurfaceCheck(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skip("golden file does not exist; run with -update first")
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("API surface check failed:\n%s", out)
	}

	t.Logf("%s", out)
}

func TestAPISurfaceUpdateIdempotent(t *testing.T) {
	// Serial: writes the golden file. Must not overlap with TestAPISurfaceCheck
	// which reads the golden file concurrently.

	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skip("golden file does not exist; run with -update first")
	}

	original, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "run", ".", "-update")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update run failed: %s\n%s", err, out)
	}

	updated, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read updated golden: %v", err)
	}

	if string(original) != string(updated) {
		t.Errorf("golden file changed after update — API surface is not stable")
		if err := os.WriteFile(goldenPath, original, 0o600); err != nil {
			t.Logf("failed to restore golden: %v", err)
		}
	}
}
