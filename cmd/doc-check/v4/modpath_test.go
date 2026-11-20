package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestModuleExistsAtHEAD pins the presence ladder against the real repo
// layout: a module exists when any path-segment prefix carries a go.mod,
// directly or under a major-version directory. Deleted modules match no
// prefix.
func TestModuleExistsAtHEAD(t *testing.T) {
	t.Parallel()

	root := findRepoRoot()

	cases := []struct {
		rel  string
		want bool
	}{
		{"event", true},                // via event/v4/go.mod
		{"event/v4", true},             // direct
		{"event/v4/eventtest", true},   // sub-package tail
		{"storage/memory/v4", true},    // nested module
		{"stack/sqlite", true},         // via stack/sqlite/v4/go.mod
		{"transport/http", false},      // deleted 2026-10-17 (ADR-0127)
		{"transport/http/v4", false},   // deleted, version suffix
		{"transport/grpc", false},      // deleted 2026-10-17 (ADR-0127)
		{"idempotency", false},         // shim deleted 2026-08-15 (ADR-0128)
		{"idempotency/sqlstore", true}, // still present, deprecated
		{"idempotency/kvstore", false}, // deleted 2026-10-17
		{"does/not/exist", false},
	}

	for _, tc := range cases {
		if got := moduleExistsAtHEAD(root, tc.rel); got != tc.want {
			t.Errorf("moduleExistsAtHEAD(%q) = %v, want %v", tc.rel, got, tc.want)
		}
	}
}

// TestCheckModulePaths covers the three citation postures over a synthetic
// tree: live modules and marked historical citations stay silent; an
// unmarked citation of a module with no go.mod fails with file:line.
func TestCheckModulePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "live", "v4"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "live", "v4", "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	doc := `# Doc

Live citation: github.com/larsartmann/go-cqrs-lite/live/v4 stays silent.
Marked history: github.com/larsartmann/go-cqrs-lite/dead/v4 was removed in 2026.
Plain dead citation: import github.com/larsartmann/go-cqrs-lite/dead/v4 here.
`
	path := filepath.Join(root, "doc.md")
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	issues := checkModulePaths([]string{path}, root)

	if len(issues) != 1 {
		t.Fatalf("got %d issue(s), want 1: %+v", len(issues), issues)
	}

	wantLine := 5
	if issues[0].Line != wantLine {
		t.Errorf("issue line = %d, want %d", issues[0].Line, wantLine)
	}

	if !strings.Contains(issues[0].Msg, "go-cqrs-lite/dead/v4") {
		t.Errorf("issue message should name the dead path: %q", issues[0].Msg)
	}
}

// TestCheckModulePaths_FaultInjection is the mutation leg: deleting the
// go.mod of a previously live module must flip its citation to a failure —
// the gate must notice a deletion that happens AFTER docs were written,
// which is the entire drift class C27 exists to catch.
func TestCheckModulePaths_FaultInjection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "mod", "v4"), 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := filepath.Join(root, "mod", "v4", "go.mod")
	if err := os.WriteFile(goMod, []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "doc.md")
	writeDoc := func() {
		t.Helper()
		doc := "Use github.com/larsartmann/go-cqrs-lite/mod/v4 for everything.\n"
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeDoc()
	if issues := checkModulePaths([]string{path}, root); len(issues) != 0 {
		t.Fatalf("live module citation flagged: %+v", issues)
	}

	if err := os.Remove(goMod); err != nil {
		t.Fatal(err)
	}

	issues := checkModulePaths([]string{path}, root)
	if len(issues) != 1 {
		t.Fatalf("after deletion: got %d issue(s), want 1: %+v", len(issues), issues)
	}
}
