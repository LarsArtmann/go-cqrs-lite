package analyzer

import (
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// writeTestTree writes a module fixture: go.mod at root, plus the given
// relative files (dirs created as needed).
func writeTestTree(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
}

func TestCollectTestImportPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestTree(t, root, map[string]string{
		"go.mod":                  "module test.example\n\ngo 1.27\n",
		"main.go":                 "package main\nimport \"fmt\"\nfunc main() { fmt.Println(1) }\n",
		"main_test.go":            "package main\nimport (\n\t\"testing\"\n\t\"github.com/larsartmann/go-cqrs-lite/systemscenario/v4\"\n)\nfunc TestA(t *testing.T) {}\n",
		"internal/boot/x.go":      "package boot\nfunc F() {}\n",
		"internal/boot/x_test.go": "package boot\nimport \"github.com/larsartmann/go-cqrs-lite/scenario/v4\"\nfunc TestB(t *testing.T) {}\n",
		// Same import twice across files: result must be deduplicated.
		"dupe_test.go": "package main\nimport \"github.com/larsartmann/go-cqrs-lite/systemscenario/v4\"\nfunc TestC(t *testing.T) {}\n",
		// Skipped dirs: vendor, testdata.
		"vendor/x_test.go":   "package v\nimport \"github.com/larsartmann/go-cqrs-lite/otel/v4\"\n",
		"testdata/x_test.go": "package td\nimport \"github.com/larsartmann/go-cqrs-lite/kv/v4\"\n",
		"broken/x_test.go":   "this is not go source",
		"notatest.go":        "package main\nimport \"github.com/larsartmann/go-cqrs-lite/event/v4\"\n",
	})

	got, err := CollectTestImportPaths(root)
	if err != nil {
		t.Fatalf("CollectTestImportPaths: %v", err)
	}

	want := []string{
		"github.com/larsartmann/go-cqrs-lite/scenario/v4",
		"github.com/larsartmann/go-cqrs-lite/systemscenario/v4",
		"testing",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d import paths (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("path %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCollectTestImportPaths_NoModules(t *testing.T) {
	t.Parallel()

	// No go.mod anywhere: the root itself is scanned (mirrors BuildContext).
	root := t.TempDir()
	writeTestTree(t, root, map[string]string{
		"a_test.go": "package a\nimport \"github.com/larsartmann/go-cqrs-lite/systemscenario/v4\"\n",
	})

	got, err := CollectTestImportPaths(root)
	if err != nil {
		t.Fatalf("CollectTestImportPaths: %v", err)
	}
	if len(got) != 1 || got[0] != "github.com/larsartmann/go-cqrs-lite/systemscenario/v4" {
		t.Fatalf("got %v, want [systemscenario path]", got)
	}
}

// TestDetectUsedModules_TestOnlyImports pins the issue-#69 contract: a module
// imported ONLY from _test.go files counts as UsageImported. The typed load
// (Tests: false) supplies zero packages and zero GoFiles for such imports —
// the testImportPaths pass is their only route into the usage map.
func TestDetectUsedModules_TestOnlyImports(t *testing.T) {
	t.Parallel()

	usage := DetectUsedModules(nil, nil, []string{
		"github.com/larsartmann/go-cqrs-lite/systemscenario/v4",
		"github.com/larsartmann/go-cqrs-lite/scenario/v4",
	}, DefaultCatalog)

	u := usage["systemscenario"]
	if u.Status != UsageImported {
		t.Errorf("systemscenario: got %s, want used (test-only import)", u.Status)
	}
	if !strings.Contains(u.Evidence, "systemscenario") {
		t.Errorf("systemscenario evidence: got %q, want the import path", u.Evidence)
	}

	checkStatus(t, usage, "scenario", UsageImported)
	checkStatus(t, usage, "kv", UsageAbsent)
	checkStatus(t, usage, "otel", UsageAbsent)
}

// TestScorecard_TestOnlyImport_Integration composes the exact runScorecard
// pipeline (CollectTestImportPaths → DetectUsedModules → ComputeScorecard)
// over a disk fixture whose ONLY cqrs import lives in a _test.go file: the
// System Scenario Testing row must land in Used, not Missing.
func TestScorecard_TestOnlyImport_Integration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestTree(t, root, map[string]string{
		"go.mod":       "module test.example\n\ngo 1.27\n",
		"main.go":      "package main\nfunc main() {}\n",
		"boot_test.go": "package main\nimport \"github.com/larsartmann/go-cqrs-lite/systemscenario/v4\"\nfunc TestBoot(t *testing.T) { _ = systemscenario.System }\n",
	})

	testImports, err := CollectTestImportPaths(root)
	if err != nil {
		t.Fatalf("CollectTestImportPaths: %v", err)
	}

	usage := DetectUsedModules(nil, nil, testImports, DefaultCatalog)

	if u := usage["systemscenario"]; u.Status != UsageImported {
		t.Fatalf("systemscenario usage: got %s, want used", u.Status)
	}
}

// TestDetectUsedModules_TestOnlyProvenance pins the provenance split: a module
// matched ONLY through _test.go imports carries TestOnly=true, while a module
// matched by production imports (Pass 1b AST) stays TestOnly=false even when
// test files import it too.
func TestDetectUsedModules_TestOnlyProvenance(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	gofiles := []*GoFile{
		parseTestFile(
			t,
			fset,
			"main.go",
			"package main\n\nimport \"github.com/larsartmann/go-cqrs-lite/kv/v4\"\n\nfunc main() {}\n",
		),
	}

	usage := DetectUsedModules(
		nil,
		gofiles,
		[]string{
			"github.com/larsartmann/go-cqrs-lite/systemscenario/v4",
			"github.com/larsartmann/go-cqrs-lite/kv/v4",
		},
		DefaultCatalog,
	)

	if u := usage["systemscenario"]; !u.TestOnly {
		t.Errorf("systemscenario: TestOnly = false, want true (matched only via _test.go imports)")
	}
	if u := usage["kv"]; u.TestOnly {
		t.Errorf("kv: TestOnly = true, want false (production import is the adoption evidence)")
	}
}

// TestCollectTestImportPaths_MonorepoNestedModules pins the monorepo contract
// (issue #69 tail): a repo with NESTED go.mod files must have every module's
// _test.go imports collected — findGoModDirs discovers both module roots and
// the walk covers the inner module's test files.
func TestCollectTestImportPaths_MonorepoNestedModules(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestTree(t, root, map[string]string{
		"go.mod":                "module test.example/outer\n\ngo 1.27\n",
		"main.go":               "package main\nfunc main() {}\n",
		"inner/go.mod":          "module test.example/inner\n\ngo 1.27\n",
		"inner/svc.go":          "package svc\nfunc F() {}\n",
		"inner/svc_scenario.go": "package svc\n", // no cqrs production import
		"inner/svc_test.go":     "package svc\nimport \"github.com/larsartmann/go-cqrs-lite/systemscenario/v4\"\nfunc TestX(t *testing.T) {}\n",
	})

	testImports, err := CollectTestImportPaths(root)
	if err != nil {
		t.Fatalf("CollectTestImportPaths: %v", err)
	}

	found := slices.Contains(testImports, "github.com/larsartmann/go-cqrs-lite/systemscenario/v4")
	if !found {
		t.Fatalf("inner module test import not collected: %v", testImports)
	}

	usage := DetectUsedModules(nil, nil, testImports, DefaultCatalog)
	if u := usage["systemscenario"]; u.Status != UsageImported || !u.TestOnly {
		t.Fatalf(
			"inner-module test import: status=%s testOnly=%v, want used+testOnly",
			u.Status, u.TestOnly,
		)
	}
}

// TestCollectTestImportPaths_SymlinkedDirsNotFollowed pins the WalkDir
// semantics the walk relies on: symlinked directories are never descended, so
// _test.go files reachable ONLY through a symlink are invisible by contract.
func TestCollectTestImportPaths_SymlinkedDirsNotFollowed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestTree(t, root, map[string]string{
		"go.mod":       "module test.example\n\ngo 1.27\n",
		"main_test.go": "package main\nimport \"github.com/larsartmann/go-cqrs-lite/event/v4\"\n",
	})

	// The otel import is reachable ONLY via the symlink: WalkDir must skip it.
	outside := t.TempDir()
	writeTestTree(t, outside, map[string]string{
		"linked/x_test.go": "package linked\nimport \"github.com/larsartmann/go-cqrs-lite/otel/v4\"\n",
	})
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, err := CollectTestImportPaths(root)
	if err != nil {
		t.Fatalf("CollectTestImportPaths: %v", err)
	}

	for _, p := range got {
		if p == "github.com/larsartmann/go-cqrs-lite/otel/v4" {
			t.Fatalf("import behind symlink must not be collected, got %v", got)
		}
	}
	if len(got) != 1 || got[0] != "github.com/larsartmann/go-cqrs-lite/event/v4" {
		t.Fatalf("got %v, want only the real file's event import", got)
	}
}
