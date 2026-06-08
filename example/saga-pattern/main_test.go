package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSagaPatternCompiles(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("..", "..")
	cmd := exec.CommandContext(context.Background(), "go", "build", "./example/saga-pattern/...")
	cmd.Dir = dir

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("example/saga-pattern failed to compile: %s\n%s", err, output)
	}
}

func TestSagaPatternMainExists(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		t.Error("main.go does not exist in example/saga-pattern")
	}
}
