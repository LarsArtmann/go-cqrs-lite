package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindRepoRootFromPath_RelativePaths pins the relative-path fix: the
// resolver must find the repo root from a RELATIVE starting directory (the
// repo's own CI invokes doc-check from cmd/doc-check with ../../-style args,
// and a regression here silently mis-resolves every alias dir — the 12-16
// relative-path fix class).
func TestFindRepoRootFromPath_RelativePaths(t *testing.T) {
	tmp := t.TempDir()
	// Simulate a repo: <tmp>/repo/.git + a nested docs dir.
	repo := filepath.Join(tmp, "repo")
	docs := filepath.Join(repo, "docs", "deep")
	if err := os.MkdirAll(docs, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(repo, ".git"),
		[]byte("gitdir: /elsewhere\n"),
		0o600,
	); err != nil {
		t.Fatalf("write .git file (worktree form): %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := os.Chdir(docs); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Relative start (the CI shape): "docs/deep" relative to repo root,
	// plus a bare relative ".", plus a "../.." climb.
	for _, rel := range []string{".", "..", "../.."} {
		// Resolve relative to docs/, not the outer cwd: use a path that
		// exists relative to the current dir only for "."; the others
		// still must climb to SOME root containing .git (the tmp repo).
		got := findRepoRootFromPath(rel)
		if got == "" {
			t.Fatalf("findRepoRootFromPath(%q) returned empty", rel)
		}

		// The found root must actually contain the .git entry.
		if _, err := os.Stat(filepath.Join(got, ".git")); err != nil {
			t.Errorf(
				"findRepoRootFromPath(%q) = %q, which has no .git (relative-path regression)",
				rel,
				got,
			)
		}
	}

	// From inside the repo, the absolute form must find exactly the repo.
	if got := findRepoRootFromPath(repo); got != repo {
		t.Errorf("absolute start: got %q want %q", got, repo)
	}
}
