package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// recipeSpec decides how the compile gate treats one recipes.md block.
//
// Every block in recipes.md MUST have exactly one catalog entry (in
// recipeCatalogA or recipeCatalogB): either a compiled scaffold (imports,
// preamble for free variables, trailers blanking unused locals) or a skip
// with a reason. A new or reworded section therefore fails the coverage test
// until it is classified — snippets can no longer rot silently.
type recipeSpec struct {
	skip         string   // non-empty: documented reason the block is not compiled
	imports      []string // raw import lines (aliases/blank imports allowed)
	preamble     string   // declarations placed above func main (free vars, types)
	trailers     string   // statements appended inside main to blank unused locals
	wholeProgram bool     // block already starts with "package " — use verbatim
	errFunc      bool     // wrap statements in `func run() error` (block returns errors)
}

const recipesRelPath = ".agents/skills/go-cqrs-lite/references/recipes.md"

func loadRecipeBlocks(t *testing.T) []RecipeBlock {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	md, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(recipesRelPath)))
	if err != nil {
		t.Fatalf("read recipes.md: %v", err)
	}
	return extractGoBlocks(string(md))
}

func allRecipeSpecs() []map[string]recipeSpec {
	return []map[string]recipeSpec{recipeCatalogA, recipeCatalogB}
}

func lookupSpec(key string) (recipeSpec, bool) {
	for _, cat := range allRecipeSpecs() {
		if spec, ok := cat[key]; ok {
			return spec, true
		}
	}
	return recipeSpec{}, false
}

// TestRecipesCatalogCoversFile enforces full classification of every block.
// It is fast (no toolchain) and is the ratchet: doc edits must keep the
// catalog honest.
func TestRecipesCatalogCoversFile(t *testing.T) {
	blocks := loadRecipeBlocks(t)
	if len(blocks) < 50 {
		t.Fatalf("extracted only %d blocks — extraction is broken", len(blocks))
	}
	seen := make(map[string]bool, len(blocks))
	for _, b := range blocks {
		key := recipeKey(b)
		spec, ok := lookupSpec(key)
		if !ok {
			t.Errorf("unclassified block %s (line %d):\n  heading: %q\n  first line: %q",
				key, b.Line, b.Heading, firstLine(b.Code))
			continue
		}
		if spec.skip == "" && !spec.wholeProgram && len(spec.imports) == 0 {
			t.Errorf("block %s: compiled entries need imports (or wholeProgram/skip)", key)
		}
		seen[key] = true
	}
	for _, cat := range allRecipeSpecs() {
		for key := range cat {
			if !seen[key] {
				t.Errorf("stale catalog entry (no matching block — heading drifted?): %q", key)
			}
		}
	}
}

func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}

// generateRecipeSource renders one block into a compilable package file.
func generateRecipeSource(b RecipeBlock, spec recipeSpec) []byte {
	if spec.wholeProgram || isWholeProgram(b.Code) {
		return []byte(b.Code)
	}
	decls, stmts := splitTypeDecls(b.Code)
	var sb strings.Builder
	sb.WriteString("package main\n\nimport (\n")
	for _, imp := range spec.imports {
		sb.WriteString("\t" + imp + "\n")
	}
	sb.WriteString(")\n\n")
	if d := strings.TrimSpace(strings.Join(decls, "\n")); d != "" {
		sb.WriteString(d + "\n\n")
	}
	if spec.errFunc {
		sb.WriteString("func run() error {\n")
		writeBody(&sb, spec, stmts)
		sb.WriteString("\treturn nil\n}\n\nfunc main() {\n\t_ = run()\n}\n")
	} else {
		sb.WriteString("func main() {\n")
		writeBody(&sb, spec, stmts)
		sb.WriteString("}\n")
	}
	return []byte(sb.String())
}

func writeBody(sb *strings.Builder, spec recipeSpec, stmts []string) {
	if spec.preamble != "" {
		sb.WriteString(spec.preamble)
		if !strings.HasSuffix(spec.preamble, "\n") {
			sb.WriteString("\n")
		}
	}
	sb.WriteString(strings.Join(stmts, "\n"))
	if spec.trailers != "" {
		sb.WriteString("\n" + spec.trailers)
	}
	sb.WriteString("\n")
}

// TestRecipesCompile compiles every non-skipped recipes.md block as its own
// package against the workspace (GOWORK=repo/go.work). This is the gate that
// turns "reference-verified" snippets into compile-verified ones: when an
// API drifts, the doc snippet stops building and this test names it.
func TestRecipesCompile(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Fatalf("go toolchain not on PATH: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	dir := t.TempDir()
	gomod := "module recipescompile\n\ngo 1.26.7\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	compiled := 0
	for _, b := range loadRecipeBlocks(t) {
		spec, ok := lookupSpec(recipeKey(b))
		if !ok || spec.skip != "" {
			continue
		}
		pkgName := fmt.Sprintf("recipe_l%d", b.Line)
		pkgDir := filepath.Join(dir, pkgName)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", pkgDir, err)
		}
		src := generateRecipeSource(b, spec)
		if err := os.WriteFile(filepath.Join(pkgDir, "snippet.go"), src, 0o644); err != nil {
			t.Fatalf("write snippet %s: %v", pkgName, err)
		}
		compiled++
	}
	if compiled == 0 {
		t.Fatal("no blocks selected for compilation — catalog misconfigured")
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOWORK="+filepath.Join(root, "go.work"),
		"GOEXPERIMENT=jsonv2",
		"CGO_ENABLED=0",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build of %d recipe snippet packages failed:\n%s", compiled, out)
	}
	t.Logf("compiled %d recipe snippet packages", compiled)
}

// TestRecipesExtractor unit-checks the fence scanner on a small fixture.
func TestRecipesExtractor(t *testing.T) {
	md := strings.Join([]string{
		"# Title",
		"",
		"## A",
		"```go",
		"x := 1",
		"```",
		"```sql",
		"SELECT 1",
		"```",
		"```go",
		"y := 2",
		"```",
		"## B",
		"```go",
		"z := 3",
		"```",
	}, "\n")
	blocks := extractGoBlocks(md)
	if len(blocks) != 3 {
		t.Fatalf("want 3 go blocks, got %d", len(blocks))
	}
	wantKeys := []string{"## A #1", "## A #2", "## B #1"}
	for i, w := range wantKeys {
		if got := recipeKey(blocks[i]); got != w {
			t.Errorf("block %d key = %q, want %q", i, got, w)
		}
	}
	if blocks[0].Code != "x := 1" || blocks[2].Code != "z := 3" {
		t.Errorf("code extraction wrong: %q / %q", blocks[0].Code, blocks[2].Code)
	}
	if !isWholeProgram("package main\n\nfunc main() {}\n") {
		t.Error("whole-program detection failed")
	}
	decls, stmts := splitTypeDecls(
		"type A struct{ X int }\nx := A{}\ntype B struct {\n\tY int\n}\n_ = B{}",
	)
	if len(decls) != 4 || len(stmts) != 2 || stmts[0] != "x := A{}" {
		t.Errorf("type hoist wrong: decls=%q stmts=%q", decls, stmts)
	}
}
