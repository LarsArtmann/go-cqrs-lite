package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The README quick-start is compile-verified exactly like recipes.md (T06,
// 2026-11-04). Its four Go fences form one NARRATIVE program - state and fold
// declared, read-model query declared, the composition root booted, a command
// dispatched and the view read - so the compile unit is the concatenation of
// all fences in document order, not each fence alone. When an API drifts, the
// quick-start stops building HERE instead of in a consumer's editor.

// readmeFences is the ratchet: every Go fence in README.md must be keyed here
// and no stale keys may remain, so adding or removing a fence forces a
// deliberate catalog edit (the recipes.md TestRecipesCatalogCoversFile
// pattern, applied to the repo's front door).
var readmeFences = map[string]struct{}{
	"### 1. Declare the domain — events, state, decide logic #1":       {},
	"### 2. Declare the read model — one fold per event type #1":       {},
	"### 3. Boot the system — and never touch infrastructure again #1": {},
	"### 3. Boot the system — and never touch infrastructure again #2": {},
}

// readmeQuickStart is the compile spec for the concatenated fences: the
// imports a consumer pasting the whole quick-start needs, the two free
// variables the README prose leaves implicit (a context and a stream ID),
// and trailers blanking the locals the prose never consumes.
var readmeQuickStart = recipeSpec{
	imports: []string{
		`"context"`,
		`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
		`"github.com/larsartmann/go-cqrs-lite/decider/v4"`,
		`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
		`"github.com/larsartmann/go-cqrs-lite/id/v4"`,
		`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
		`"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"`,
		`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
		`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
	},
	preamble: "var ctx = context.Background()\n\nvar counterID = id.NewStreamID()",
	trailers: "\t_ = err\n\t_ = view\n\t_ = found",
	errFunc:  true,
}

func loadReadmeBlocks(t *testing.T) []RecipeBlock {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	md, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	blocks := extractGoBlocks(string(md))
	if len(blocks) < 4 {
		t.Fatalf(
			"extracted only %d Go fences - extraction is broken or the quick-start was rewritten",
			len(blocks),
		)
	}

	return blocks
}

// TestReadmeFencesClassified is the fast ratchet (no toolchain): README Go
// fences and the readmeFences catalog must agree exactly in both directions.
func TestReadmeFencesClassified(t *testing.T) {
	blocks := loadReadmeBlocks(t)
	seen := make(map[string]bool, len(blocks))
	for _, b := range blocks {
		key := recipeKey(b)
		if _, ok := readmeFences[key]; !ok {
			t.Errorf(
				"unclassified README Go fence %s (line %d) - add it to readmeFences and extend readmeQuickStart if it joins the narrative",
				key,
				b.Line,
			)

			continue
		}
		seen[key] = true
	}
	for key := range readmeFences {
		if !seen[key] {
			t.Errorf("stale readmeFences entry (no matching fence - heading drifted?): %q", key)
		}
	}
}

// TestReadmeQuickStartCompiles builds the concatenated quick-start fences as
// one package against the workspace (the TestRecipesCompile machinery).
func TestReadmeQuickStartCompiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Fatalf("go toolchain not on PATH: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	blocks := loadReadmeBlocks(t)
	bodies := make([]string, 0, len(blocks))
	for _, b := range blocks {
		bodies = append(bodies, b.Code)
	}
	joined := RecipeBlock{Code: strings.Join(bodies, "\n\n")}

	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "go.mod"),
		[]byte("module readmecompile\n\ngo 1.27\n"),
		0o644,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "go.work"),
		[]byte(workspaceFor(t, root, dir)),
		0o644,
	); err != nil {
		t.Fatalf("write go.work: %v", err)
	}
	pkgDir := filepath.Join(dir, "quickstart")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", pkgDir, err)
	}
	src := generateRecipeSource(joined, readmeQuickStart)
	if err := os.WriteFile(filepath.Join(pkgDir, "snippet.go"), src, 0o644); err != nil {
		t.Fatalf("write snippet: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", os.DevNull, "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOWORK="+filepath.Join(dir, "go.work"),
		"CGO_ENABLED=0",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build of the README quick-start failed:\n%s", out)
	}
	t.Logf(
		"compiled README quick-start (%d fences, %d lines)",
		len(blocks),
		strings.Count(joined.Code, "\n")+1,
	)
}
