package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestListingExampleCompiles(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("..", "..")
	cmd := exec.Command("go", "build", "./example/listing/...")
	cmd.Dir = dir

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("example/listing failed to compile: %s\n%s", err, output)
	}
}

func TestListingMainExists(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		t.Error("main.go does not exist in example/listing")
	}
}
