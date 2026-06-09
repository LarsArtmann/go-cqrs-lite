package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSagaPatternCompiles(t *testing.T) {
	t.Parallel()

	rootDir := filepath.Join("..", "..")
	buildCmd := exec.Command("go", "build", "./example/saga-pattern/...")
	buildCmd.Dir = rootDir

	output, buildErr := buildCmd.CombinedOutput()
	if buildErr != nil {
		t.Errorf("example/saga-pattern failed to compile: %s\n%s", buildErr, output)
	}
}

func TestSagaPatternMainExists(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		t.Error("main.go does not exist in example/saga-pattern")
	}
}
