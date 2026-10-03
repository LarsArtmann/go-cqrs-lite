package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIPositionalPathsReachScan pins the 2026-10-03 fix: positional
// arguments are scan paths. Cobra's default Args validator (active because
// help/completion register as subcommands) rejected them as "Unknown
// command" — the documented `cqrs-gen ./...` invocation was dead code.
func TestCLIPositionalPathsReachScan(t *testing.T) {
	dir := t.TempDir()
	src := "package cmds\n\n//cqrs:command CreateUser\ntype CreateUserCmd struct {\n\tName string\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "commands.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "gen_out.go")

	cli, err := buildCLI()
	if err != nil {
		t.Fatalf("buildCLI: %v", err)
	}

	if err := cli.ExecuteWithArgs(context.Background(), []string{
		"--type", "command", "--output", out, dir,
	}); err != nil {
		t.Fatalf("positional path invocation: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("generated file missing: %v", err)
	}
	if !strings.Contains(string(data), "CreateUser") {
		t.Errorf("generated file should register the marker type, got:\n%s", data)
	}
}

// TestCLIRejectsUnknownType pins fail-fast on an invalid --type: the error
// surfaces before any scanning happens.
func TestCLIRejectsUnknownType(t *testing.T) {
	cli, err := buildCLI()
	if err != nil {
		t.Fatalf("buildCLI: %v", err)
	}

	err = cli.ExecuteWithArgs(context.Background(), []string{"--type", "bogus", t.TempDir()})
	if err == nil {
		t.Fatal("expected error for --type bogus, got nil")
	}
	if !strings.Contains(err.Error(), "invalid type") {
		t.Errorf("expected invalid-type error, got: %v", err)
	}
}

// TestCLINoMarkersIsClean pins the zero-finding exit: scanning a directory
// without cqrs markers prints a notice and exits 0 (CI-safe no-op).
func TestCLINoMarkersIsClean(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "plain.go"),
		[]byte("package plain\n\nfunc Hi() {}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cli, err := buildCLI()
	if err != nil {
		t.Fatalf("buildCLI: %v", err)
	}

	if err := cli.ExecuteWithArgs(context.Background(), []string{dir}); err != nil {
		t.Errorf("no-marker scan should be clean, got: %v", err)
	}
}
