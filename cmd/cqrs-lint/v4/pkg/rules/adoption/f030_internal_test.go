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

// moduleGoMods returns every go.mod belonging to the module at rel: the root
// module (rel/go.mod) plus any major-version module (rel/v4/go.mod). Both
// forms exist in this workspace, so "does the module exist" must ask for both.
func moduleGoMods(root, rel string) []string {
	var out []string

	for _, pattern := range []string{
		filepath.Join(root, rel, "go.mod"),
		filepath.Join(root, rel, "v[0-9]*", "go.mod"),
	} {
		matches, _ := filepath.Glob(pattern)
		out = append(out, matches...)
	}

	return out
}

// TestModulePresenceRegistry_StateMatchesSource is the bidirectional drift gate
// over the single registry:
//
//   - FORWARD: an entry claiming moduleDeleted must have no go.mod at HEAD. If
//     a deleted module comes back (a re-extraction, a revert of the ADR-0128
//     sweep), F030 would lie about the repo, and consumers would be coached off
//     a module that is alive.
//   - REVERSE: an entry claiming modulePresent must have a go.mod at HEAD. A
//     present-module guard that outlives its module silently masks the deleted
//     ancestor's finding.
//
// It also pins that every cited ADR and every replacement string exists, so the
// migration pointer never rots into a dead reference.
func TestModulePresenceRegistry_StateMatchesSource(t *testing.T) {
	t.Parallel()

	root := repoRootForF030(t)

	for _, entry := range modulePresenceRegistry {
		// A module's directory may survive deletion: go-cqrs-lite/idempotency/
		// still exists because its sqlstore submodule lives under it. Presence
		// is a go.mod question, never a directory question.
		rel := strings.TrimPrefix(entry.fragment, "go-cqrs-lite/")
		found := moduleGoMods(root, rel)

		switch entry.state {
		case moduleDeleted:
			if len(found) > 0 {
				t.Errorf("%s is registered as deleted but %v exists at HEAD",
					entry.fragment, found)
			}
		case modulePresent:
			if len(found) == 0 {
				t.Errorf("%s is registered as present but has no go.mod at HEAD",
					entry.fragment)
			}
		default:
			t.Errorf("%s has unknown presence state %d", entry.fragment, entry.state)
		}

		if entry.removal == "" || entry.replacement == "" {
			t.Errorf("%s needs both removal and replacement text", entry.fragment)
		}

		if _, err := os.Stat(filepath.Join(root, entry.adr)); err != nil {
			t.Errorf("%s cites missing ADR %s", entry.fragment, entry.adr)
		}
	}
}

// TestModulePresenceRegistry_NoDuplicateFragments fails on a repeated
// fragment: longest-match can only rank DIFFERENT lengths, so a duplicate row
// with a different state would resolve arbitrarily depending on table order.
// (A shorter prefix row is legitimate and stays: it is the unique winner for
// the bare ancestor import.)
func TestModulePresenceRegistry_NoDuplicateFragments(t *testing.T) {
	t.Parallel()

	seen := make(map[string]modulePresence, len(modulePresenceRegistry))

	for _, entry := range modulePresenceRegistry {
		if prev, dup := seen[entry.fragment]; dup {
			t.Errorf("%s is registered twice (states %v and %v)",
				entry.fragment, prev, entry.state)

			continue
		}

		seen[entry.fragment] = entry.state
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

// TestResolveModulePresence pins specific-first resolution across BOTH states:
// a path matching a deleted ancestor and a more specific row resolves to the
// specific one, and a still-present submodule resolves to itself rather than to
// the deleted ancestor above it.
func TestResolveModulePresence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		wantFrag string
		wantSt   modulePresence
		wantOK   bool
	}{
		{
			"github.com/larsartmann/go-cqrs-lite/idempotency/kvstore/v4",
			"go-cqrs-lite/idempotency/kvstore",
			moduleDeleted,
			true,
		},
		{
			"github.com/larsartmann/go-cqrs-lite/idempotency/v4",
			"go-cqrs-lite/idempotency",
			moduleDeleted,
			true,
		},
		{
			"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4",
			"go-cqrs-lite/idempotency/sqlstore",
			modulePresent,
			true,
		},
		{"github.com/larsartmann/go-idempotency", "", modulePresent, false},
		{"github.com/larsartmann/go-cqrs-lite/event/v4", "", modulePresent, false},
	}

	for _, tt := range tests {
		got, ok := resolveModulePresence(tt.path)
		if ok != tt.wantOK || got.fragment != tt.wantFrag || got.state != tt.wantSt {
			t.Errorf("resolveModulePresence(%q) = (%q, %v, %v), want (%q, %v, %v)",
				tt.path, got.fragment, got.state, ok, tt.wantFrag, tt.wantSt, tt.wantOK)
		}
	}
}
