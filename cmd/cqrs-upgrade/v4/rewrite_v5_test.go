package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestV5TableMatchesCoreTree pins the compiled-in rewrite table to the real
// core/v5 tree: every non-root, non-testdata Go package under core/v5 must
// be the target of exactly one table entry, and every table target must
// exist. Extending the fold without updating the table fails here first.
func TestV5TableMatchesCoreTree(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "core", "v5"))
	if err != nil {
		t.Fatalf("core root: %v", err)
	}

	tree := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != root && (name == "testdata" || name == ".git") {
			return filepath.SkipDir
		}
		if path == root {
			return nil // the module root package has no slice-1 v4 twin
		}
		goFiles, err := filepath.Glob(filepath.Join(path, "*.go"))
		if err != nil {
			return err
		}
		if len(goFiles) == 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		tree["github.com/larsartmann/go-cqrs-lite/core/v5/"+filepath.ToSlash(rel)] = true

		return nil
	})
	if err != nil {
		t.Fatalf("walk core/v5: %v", err)
	}

	tableTargets := make([]string, 0, len(v5ImportTable))
	for _, to := range v5ImportTable {
		tableTargets = append(tableTargets, to)
		if !tree[to] {
			t.Errorf("table targets %s but no such package exists under core/v5 (stale entry?)", to)
		}
	}
	sort.Strings(tableTargets)

	treeTargets := make([]string, 0, len(tree))
	for p := range tree {
		treeTargets = append(treeTargets, p)
		if _, ok := v5ImportTable[v4TwinOf(p)]; !ok {
			t.Errorf("core/v5 package %s has no table entry (extend the fold table)", p)
		}
	}
	sort.Strings(treeTargets)

	if strings.Join(tableTargets, "\n") != strings.Join(treeTargets, "\n") {
		t.Errorf(
			"table targets and core/v5 tree disagree:\ntable: %v\ntree:  %v",
			tableTargets,
			treeTargets,
		)
	}
}

// v4TwinOf derives the v4 import path a core/v5 package folds from:
// core/v5/<train>[/<rest>] -> github.com/larsartmann/go-cqrs-lite/<train>/v4[/<rest>].
func v4TwinOf(v5Path string) string {
	rel := strings.TrimPrefix(v5Path, "github.com/larsartmann/go-cqrs-lite/core/v5/")
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) == 1 {
		return v4CqrsPrefix + parts[0] + "/v4"
	}

	return v4CqrsPrefix + parts[0] + "/v4/" + parts[1]
}

const rewriteFixture = `package demo

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	cid "github.com/larsartmann/go-cqrs-lite/id/v4"
	_ "github.com/larsartmann/go-cqrs-lite/kv/v4/viewstoretest"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// github.com/larsartmann/go-cqrs-lite/event/v4 in a comment stays put.
const docString = "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = fmt.Sprint
`

func writeFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "demo.go")
	if err := os.WriteFile(path, []byte(rewriteFixture), 0o644); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	return path
}

// TestRewriteV5ImportsApply covers the apply path: aliased, blank, and plain
// folded imports rewrite; the string, the comment, and non-Go content stay
// byte-identical; the not-yet-folded system import is reported as remaining.
func TestRewriteV5ImportsApply(t *testing.T) {
	path := writeFixture(t)

	changes, remaining, err := rewriteV5ImportsInFile(path, false)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if len(changes) != 3 {
		t.Fatalf(
			"want 3 changes (command, aliased id, blank viewstoretest), got %d: %+v",
			len(changes),
			changes,
		)
	}
	want := map[string]string{
		"github.com/larsartmann/go-cqrs-lite/command/v4":          "github.com/larsartmann/go-cqrs-lite/core/v5/command",
		"github.com/larsartmann/go-cqrs-lite/id/v4":               "github.com/larsartmann/go-cqrs-lite/core/v5/id",
		"github.com/larsartmann/go-cqrs-lite/kv/v4/viewstoretest": "github.com/larsartmann/go-cqrs-lite/core/v5/kv/viewstoretest",
	}
	for _, c := range changes {
		if want[c.From] != c.To {
			t.Errorf("change %s -> %s wrong (want %s)", c.From, c.To, want[c.From])
		}
		delete(want, c.From)
	}
	if len(want) != 0 {
		t.Errorf("missing changes: %v", want)
	}
	if len(remaining) != 1 || remaining[0] != "github.com/larsartmann/go-cqrs-lite/system/v4" {
		t.Errorf("remaining = %v, want [system/v4]", remaining)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reread: %v", err)
	}
	src := string(out)
	for _, gone := range []string{
		`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
		`cid "github.com/larsartmann/go-cqrs-lite/id/v4"`,
	} {
		if strings.Contains(src, gone) {
			t.Errorf("import literal still present: %s", gone)
		}
	}
	if !strings.Contains(src, `cid "github.com/larsartmann/go-cqrs-lite/core/v5/id"`) {
		t.Error("alias not preserved on rewrite")
	}
	if !strings.Contains(src, `"github.com/larsartmann/go-cqrs-lite/event/v4"`) {
		t.Error("the const string must NOT be rewritten")
	}
	if !strings.Contains(
		src,
		"// github.com/larsartmann/go-cqrs-lite/event/v4 in a comment stays put.",
	) {
		t.Error("the comment must NOT be rewritten")
	}
	if !strings.Contains(src, `"github.com/larsartmann/go-cqrs-lite/system/v4"`) {
		t.Error("the not-yet-folded import must stay")
	}

	// Idempotency: a second pass finds nothing left to fold.
	changes2, remaining2, err := rewriteV5ImportsInFile(path, false)
	if err != nil {
		t.Fatalf("second rewrite: %v", err)
	}
	if len(changes2) != 0 {
		t.Errorf("second pass rewrote again: %+v", changes2)
	}
	if len(remaining2) != 1 {
		t.Errorf("second pass remaining = %v, want system/v4 only", remaining2)
	}
}

// TestRewriteV5ImportsDryRun proves --dry-run plans without writing.
func TestRewriteV5ImportsDryRun(t *testing.T) {
	path := writeFixture(t)

	changes, _, err := rewriteV5ImportsInFile(path, true)
	if err != nil {
		t.Fatalf("dry-run rewrite: %v", err)
	}
	if len(changes) != 3 {
		t.Fatalf("dry run planned %d changes, want 3", len(changes))
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reread: %v", err)
	}
	if string(out) != rewriteFixture {
		t.Error("dry run must not modify the file")
	}
}

// TestIsCqrsV4Import pins v4-import classification vectors, including the
// v5 twin (which must NOT classify as v4).
func TestIsCqrsV4Import(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"github.com/larsartmann/go-cqrs-lite/event/v4", true},
		{"github.com/larsartmann/go-cqrs-lite/v4", true},
		{"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4", true},
		{"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest", true},
		{"github.com/larsartmann/go-cqrs-lite/core/v5/event", false},
		{"github.com/larsartmann/go-error-family", false},
		{"fmt", false},
	} {
		if got := isCqrsV4Import(tc.path); got != tc.want {
			t.Errorf("isCqrsV4Import(%s) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestStrictV5GateError covers the --strict --to-v5 verdict ladder.
func TestStrictV5GateError(t *testing.T) {
	cases := []struct {
		name    string
		strict  bool
		reports []v5ModuleReport
		wantNil bool
		wantErr string
	}{
		{
			"not strict ignores everything", false,
			[]v5ModuleReport{{Error: "boom", RemainingV4: []string{"x/v4"}}},
			true, "",
		},
		{
			"errored module fails first", true,
			[]v5ModuleReport{{Error: "boom"}},
			false, "module error",
		},
		{
			"remaining v4 fails",
			true,
			[]v5ModuleReport{
				{RemainingV4: []string{"github.com/larsartmann/go-cqrs-lite/system/v4"}},
			},
			false,
			"not-yet-folded",
		},
		{
			"clean passes", true,
			[]v5ModuleReport{{FilesScanned: 3}},
			true, "",
		},
	}
	for _, tc := range cases {
		err := strictV5GateError(tc.strict, tc.reports)
		if tc.wantNil {
			if err != nil {
				t.Errorf("%s: want nil, got %v", tc.name, err)
			}

			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: want error containing %q, got %v", tc.name, tc.wantErr, err)
		}
	}
}

// TestParseFlagsToV5 pins the flag wiring.
func TestParseFlagsToV5(t *testing.T) {
	cfg, err := parseFlags([]string{"--to-v5", "--dry-run"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !cfg.toV5 || !cfg.dryRun {
		t.Errorf("toV5=%v dryRun=%v, want both true", cfg.toV5, cfg.dryRun)
	}
}
