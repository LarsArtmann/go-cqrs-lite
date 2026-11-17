package adoption

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRootForF030 walks up from the test's package directory to the repo root
// (the directory holding go.work).
func repoRootForF030(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.work")); statErr == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root (go.work) not found above %s", dir)
		}

		dir = parent
	}
}

// TestDeletedModuleRegistry_EntriesAreActuallyDeleted is the reverse drift
// gate: an entry whose module directory came back (a re-extraction, a
// revert of the ADR-0128 sweep) would make F030 lie about the repo. It also
// pins that every cited ADR exists, so the migration pointer never rots.
func TestDeletedModuleRegistry_EntriesAreActuallyDeleted(t *testing.T) {
	t.Parallel()

	root := repoRootForF030(t)

	for _, dep := range deletedModuleImports {
		// A module is deleted when its go.mod is gone. The DIRECTORY may
		// survive: go-cqrs-lite/idempotency/ still exists because its
		// sqlstore submodule lives under it.
		goMod := filepath.Join(
			root,
			strings.TrimPrefix(dep.fragment, "go-cqrs-lite/"),
			"go.mod",
		)

		if _, err := os.Stat(goMod); err == nil {
			t.Errorf("%s is listed as deleted but %s exists at HEAD",
				dep.fragment, goMod)
		}

		adrPath := filepath.Join(root, dep.adr)
		if _, err := os.Stat(adrPath); err != nil {
			t.Errorf("%s cites missing ADR %s", dep.fragment, dep.adr)
		}
	}
}

// TestPresentModuleFragments_StillExist keeps the suppression list honest: a
// present-module guard that outlives its module would silently mask a deleted
// ancestor's finding.
func TestPresentModuleFragments_StillExist(t *testing.T) {
	t.Parallel()

	root := repoRootForF030(t)

	for _, fragment := range presentModuleFragments {
		goMod := filepath.Join(
			root,
			strings.TrimPrefix(fragment, "go-cqrs-lite/"),
			"go.mod",
		)

		if _, err := os.Stat(goMod); err != nil {
			t.Errorf("present-module guard %s has no go.mod at HEAD (%s)",
				fragment, goMod)
		}
	}
}

// TestFragmentMatchesImport pins the segment-boundary contract behind F030's
// longest-fragment matching. Plain strings.Contains would make the deleted
// parent idempotency fragment match "go-cqrs-lite/idempotencyx" or a foreign
// "example.com/.../idempotency" path.
func TestFragmentMatchesImport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path, fragment string
		want           bool
	}{
		{"github.com/larsartmann/go-cqrs-lite/idempotency/v4", "go-cqrs-lite/idempotency", true},
		{"github.com/larsartmann/go-cqrs-lite/idempotency", "go-cqrs-lite/idempotency", true},
		{
			"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4",
			"go-cqrs-lite/idempotency",
			true,
		},
		{"github.com/other/idempotency", "go-cqrs-lite/idempotency", false},
		{"github.com/larsartmann/go-cqrs-lite/idempotencyx", "go-cqrs-lite/idempotency", false},
		{
			// Same path suffix under a different host still matches, exactly
			// like analyzer.pathBoundaryMatch: the fragment identifies a
			// module path, and a fork/vendored copy carries the same one.
			"example.com/larsartmann/go-cqrs-lite/idempotency",
			"go-cqrs-lite/idempotency",
			true,
		},
	}

	for _, tt := range tests {
		if got := fragmentMatchesImport(tt.path, tt.fragment); got != tt.want {
			t.Errorf("fragmentMatchesImport(%q, %q) = %v, want %v",
				tt.path, tt.fragment, got, tt.want)
		}
	}
}

// TestLongestDeletedModuleMatch pins specific-first resolution: a path that
// matches both a deleted ancestor and a more specific deleted module resolves
// to the specific one, and a still-present submodule suppresses the ancestor.
func TestLongestDeletedModuleMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		wantFrag string
		wantOK   bool
	}{
		{
			"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4",
			"go-cqrs-lite/idempotency/kvstore",
			true,
		},
		{"github.com/larsartmann/go-cqrs-lite/idempotency/v4", "go-cqrs-lite/idempotency", true},
		{"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4", "", false},
		{"github.com/larsartmann/go-idempotency", "", false},
	}

	for _, tt := range tests {
		got, ok := longestDeletedModuleMatch(tt.path)
		if ok != tt.wantOK || got.fragment != tt.wantFrag {
			t.Errorf("longestDeletedModuleMatch(%q) = (%q, %v), want (%q, %v)",
				tt.path, got.fragment, ok, tt.wantFrag, tt.wantOK)
		}
	}
}
