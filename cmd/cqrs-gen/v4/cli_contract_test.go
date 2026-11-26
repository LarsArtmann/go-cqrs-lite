package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
)

const markedCommand = "package cmds\n\n//cqrs:command user.create\ntype CreateUserCmd struct {\n\tName string\n}\n"

func buildTestCLI(t *testing.T) *cmdguard.CLI[AppConfig] {
	t.Helper()

	cli, err := buildCLI()
	if err != nil {
		t.Fatalf("buildCLI: %v", err)
	}

	return cli
}

func runCLI(t *testing.T, cli *cmdguard.CLI[AppConfig], args ...string) error {
	t.Helper()

	return cli.ExecuteWithArgs(context.Background(), args)
}

func TestCLIPositionalPathsReachScan(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "commands.go", markedCommand)

	out := filepath.Join(dir, "gen_out.go")

	if err := runCLI(t, buildTestCLI(t), "--type", "command", "--output", out, dir); err != nil {
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

func TestCLIRejectsUnknownType(t *testing.T) {
	err := runCLI(t, buildTestCLI(t), "--type", "bogus", t.TempDir())
	if err == nil {
		t.Fatal("expected error for --type bogus, got nil")
	}

	if !strings.Contains(err.Error(), "invalid --type") {
		t.Errorf("expected invalid-type error, got: %v", err)
	}
}

func TestCLINoMarkersIsClean(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "plain.go", "package plain\n\nfunc Hi() {}\n")

	if err := runCLI(t, buildTestCLI(t), dir); err != nil {
		t.Errorf("no-marker scan should be clean, got: %v", err)
	}
}

// TestCLINoPathArgumentScansWorkingDirectory pins the documented default: with
// no path argument the generator scans ".". The hidden-directory prune used to
// match the root entry itself (WalkDir reports it under the name "."), so the
// default invocation reported "no cqrs markers found" for every project.
func TestCLINoPathArgumentScansWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "commands.go", markedCommand)

	t.Chdir(dir)

	if err := runCLI(t, buildTestCLI(t)); err != nil {
		t.Fatalf("default-path invocation: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, generatedFile))
	if err != nil {
		t.Fatalf("generated file missing: %v", err)
	}

	if !strings.Contains(string(data), "func RegisterCreateUserCmd[State any](") {
		t.Errorf("generated file should target the composition root by default, got:\n%s", data)
	}
}

func TestCLICheckMode(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "commands.go", markedCommand)

	if err := runCLI(t, buildTestCLI(t), dir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	if err := runCLI(t, buildTestCLI(t), "--check", dir); err != nil {
		t.Fatalf("fresh generated file must pass --check: %v", err)
	}

	path := filepath.Join(dir, generatedFile)

	if err := os.WriteFile(path, []byte("// stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := runCLI(t, buildTestCLI(t), "--check", dir)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale generated file must fail --check, got: %v", err)
	}

	if current, readErr := os.ReadFile(path); readErr != nil || string(current) != "// stale\n" {
		t.Errorf("--check must not write, file is now %q (%v)", current, readErr)
	}

	empty := t.TempDir()

	err = runCLI(t, buildTestCLI(t), "--check", empty)
	if err == nil || !strings.Contains(err.Error(), "no cqrs markers found") {
		t.Fatalf("--check over a path without markers must fail instead of passing vacuously, got: %v", err)
	}
}

func TestCLIOutputRefusesSeveralPackages(t *testing.T) {
	root := t.TempDir()

	writeFixture(t, root, "a/a.go", "package a\n\n//cqrs:command a.one\ntype OneCmd struct{}")
	writeFixture(t, root, "b/b.go", "package b\n\n//cqrs:command b.two\ntype TwoCmd struct{}")

	err := runCLI(t, buildTestCLI(t), "--output", filepath.Join(root, "out.go"), root)
	if err == nil || !strings.Contains(err.Error(), "2 packages declare markers") {
		t.Fatalf("one --output file cannot serve two packages, got: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(root, "out.go")); statErr == nil {
		t.Error("refused run must not write the output file")
	}
}

func TestCLIOneFilePerPackage(t *testing.T) {
	root := t.TempDir()

	writeFixture(t, root, "a/a.go", "package a\n\n//cqrs:command a.one\ntype OneCmd struct{}")
	writeFixture(t, root, "b/b.go", "package b\n\n//cqrs:command b.two\ntype TwoCmd struct{}")

	if err := runCLI(t, buildTestCLI(t), "--type", "command", root); err != nil {
		t.Fatalf("generate: %v", err)
	}

	for dir, want := range map[string]string{"a": "package a", "b": "package b"} {
		data, err := os.ReadFile(filepath.Join(root, dir, generatedFile))
		if err != nil {
			t.Fatalf("generated file missing in %s: %v", dir, err)
		}

		if !strings.Contains(string(data), want) {
			t.Errorf("%s should hold %q, got:\n%s", dir, want, data)
		}
	}

	if err := runCLI(t, buildTestCLI(t), "--type", "command", "--check", root); err != nil {
		t.Fatalf("per-package output must satisfy --check: %v", err)
	}
}

func TestCLIListWritesNothing(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "commands.go", markedCommand)

	cli := buildTestCLI(t)

	if err := runCLI(t, cli, "--list", "--target", "dispatcher", dir); err != nil {
		t.Fatalf("list: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, generatedFile)); err == nil {
		t.Error("--list must not write a generated file")
	}
}

func TestCLIRejectsPkgWithoutOutput(t *testing.T) {
	err := runCLI(t, buildTestCLI(t), "--pkg", "other", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "--pkg only applies") {
		t.Fatalf("--pkg without --output must fail before scanning, got: %v", err)
	}
}
