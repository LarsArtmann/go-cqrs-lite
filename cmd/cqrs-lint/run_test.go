package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr redirects os.Stderr to a pipe, runs fn, and returns whatever
// was written. NOT safe for parallel tests — callers must not use t.Parallel().
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}

// TestRun_BrokenProject_ExitsNonZero verifies that run() returns
// errFindingsWithErrors when the project has unresolvable dependencies and no
// packages could be loaded. This is the regression test for the original
// silent-failure bug where cqrs-lint reported "Nothing to lint" on a broken build.
func TestRun_BrokenProject_ExitsNonZero(t *testing.T) {
	dir := t.TempDir()

	goMod := `module testproject

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v9.9.9
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	goFile := `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func main() {
	_ = event.New
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")

	err := run(context.Background(), &AppConfig{Path: dir})
	if !errors.Is(err, errFindingsWithErrors) {
		t.Errorf("run() on broken project: got %v, want errFindingsWithErrors", err)
	}
}

// TestRun_StrictMode verifies that --strict makes any load error fatal, even
// when the error message is different from the no-files path.
func TestRun_StrictMode(t *testing.T) {
	dir := t.TempDir()

	goMod := `module testproject

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v9.9.9
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	goFile := `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func main() {
	_ = event.New
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")

	cfg := &AppConfig{
		Path:       dir,
		StrictLoad: true,
	}

	err := run(context.Background(), cfg)
	if !errors.Is(err, errFindingsWithErrors) {
		t.Errorf("run() with --strict on broken project: got %v, want errFindingsWithErrors", err)
	}
}

// TestRun_NoGoFiles_Message verifies the "No Go files found" message when a
// project has a go.mod but no Go source files.
func TestRun_NoGoFiles_Message(t *testing.T) {
	dir := t.TempDir()

	goMod := `module emptyproject

go 1.26.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")

	cfg := &AppConfig{Path: dir}

	stderr := captureStderr(t, func() {
		err := run(context.Background(), cfg)
		if err != nil {
			t.Errorf("run() on empty project: got %v, want nil", err)
		}
	})

	if !strings.Contains(stderr, "No Go files found") {
		t.Errorf("expected 'No Go files found' in stderr, got: %s", stderr)
	}
}

// TestRun_NoCQRSImports_Message verifies the "Found Go files but none import
// go-cqrs-lite" message when a project compiles but has no go-cqrs-lite imports.
func TestRun_NoCQRSImports_Message(t *testing.T) {
	dir := t.TempDir()

	goMod := `module cleanproject

go 1.26.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	goFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")

	cfg := &AppConfig{Path: dir}

	stderr := captureStderr(t, func() {
		err := run(context.Background(), cfg)
		if err != nil {
			t.Errorf("run() on non-cqrs project: got %v, want nil", err)
		}
	})

	if !strings.Contains(stderr, "Found Go files but none import go-cqrs-lite") {
		t.Errorf(
			"expected 'Found Go files but none import go-cqrs-lite' in stderr, got: %s",
			stderr,
		)
	}
}
