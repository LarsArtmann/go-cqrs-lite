package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAPISurfaceCheck(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skip("golden file does not exist; run with -update first")
	}

	cmd := exec.Command("go", "run", ".")
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

	cmd := exec.Command("go", "run", ".", "-update")
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
		if err := os.WriteFile(goldenPath, original, 0o644); err != nil {
			t.Logf("failed to restore golden: %v", err)
		}
	}
}
