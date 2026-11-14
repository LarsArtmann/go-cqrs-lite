package main

import (
	"os"
	"path/filepath"
)

// repoRoot walks up from the current working directory (the module dir when
// run via `go test`) to the nearest .git marker. The module moved from
// cmd/cqrs-upgrade to cmd/cqrs-upgrade/v4 (2026-11-10 dir move), so
// relative "../.." joins resolved one level short and broke three tests that
// reference repo-sibling trees (core/v5, event, example). Walking to the
// repo root survives any future module-dir nesting.
func repoRoot(t interface {
	Fatalf(format string, args ...any)
},
) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no .git marker found above %s", dir)
		}
		dir = parent
	}
}
