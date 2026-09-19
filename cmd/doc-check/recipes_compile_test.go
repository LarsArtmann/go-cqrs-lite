package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
	return []map[string]recipeSpec{recipeCatalogA, recipeCatalogB, recipeCatalogB2}
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

// extractBodyImports pulls `import (...)` groups and single `import "x"`
// lines out of a snippet body, merging their paths into the scaffold's
// import set (scaffold entries win on path conflicts). Returns the body
// without import statements plus the merged import lines.
func extractBodyImports(code string, imports []string) (string, []string) {
	have := make(map[string]bool, len(imports))
	for _, imp := range imports {
		if path := importPath(imp); path != "" {
			have[path] = true
		}
	}
	var body, extra []string
	inGroup := false
	for _, ln := range strings.Split(code, "\n") {
		tl := strings.TrimSpace(ln)
		switch {
		case !inGroup && strings.HasPrefix(tl, "import ("):
			inGroup = true
		case inGroup && tl == ")":
			inGroup = false
		case inGroup:
			extra = append(extra, tl)
		case !inGroup && strings.HasPrefix(tl, "import "):
			extra = append(extra, strings.TrimSpace(strings.TrimPrefix(tl, "import ")))
		default:
			body = append(body, ln)
		}
	}
	for _, imp := range extra {
		if path := importPath(imp); path != "" && !have[path] {
			have[path] = true
			imports = append(imports, imp)
		}
	}

	return strings.Join(body, "\n"), imports
}

// importPath extracts the quoted path from one import line (with or without
// an alias); "" if the line carries none.
func importPath(line string) string {
	i := strings.IndexByte(line, '"')
	if i < 0 {
		return ""
	}
	if j := strings.LastIndexByte(line, '"'); j > i {
		return line[i+1 : j]
	}

	return ""
}

// generateRecipeSource renders one block into a compilable package file.
func generateRecipeSource(b RecipeBlock, spec recipeSpec) []byte {
	if spec.wholeProgram || isWholeProgram(b.Code) {
		return []byte(b.Code)
	}
	body, imports := extractBodyImports(b.Code, spec.imports)
	decls, stmts := splitTypeDecls(spec.preamble + "\n" + body)
	var sb strings.Builder
	sb.WriteString("package main\n\nimport (\n")
	for _, imp := range imports {
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
	sb.WriteString(strings.Join(stmts, "\n"))
	if spec.trailers != "" {
		sb.WriteString("\n" + spec.trailers)
	}
	sb.WriteString("\n")
}

// workspaceFor renders a workspace file for the snippet module: the repo's
// go.work with every use path absolutized, plus the snippet dir itself (the
// go command requires the working directory to be inside the workspace).
func workspaceFor(t *testing.T, root, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		t.Fatalf("read go.work: %v", err)
	}
	var sb strings.Builder
	inUse := false
	for _, ln := range strings.Split(string(raw), "\n") {
		tl := strings.TrimSpace(ln)
		switch {
		case tl == "use (":
			inUse = true
			sb.WriteString(ln + "\n")
		case inUse && tl == ")":
			inUse = false
			sb.WriteString("\t" + dir + "\n")
			sb.WriteString(ln + "\n")
		case inUse && tl != "" && !strings.HasPrefix(tl, "//"):
			rel := strings.TrimSpace(tl)
			abs := rel
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(root, rel)
			}
			sb.WriteString("\t" + abs + "\n")
		default:
			sb.WriteString(ln + "\n")
		}
	}
	if inUse { // defensive: unterminated use block in the repo file
		t.Fatal("repo go.work has an unterminated use block")
	}

	return sb.String()
}

// TestRecipesCompile compiles every non-skipped recipes.md block as its own
// package against the workspace. This is the gate that turns
// "reference-verified" snippets into compile-verified ones: when an API
// drifts, the doc snippet stops building and this test names it.
func TestRecipesCompile(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Fatalf("go toolchain not on PATH: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	dir := t.TempDir()
	if keep := os.Getenv("RECIPES_COMPILE_KEEP"); keep != "" {
		dir = keep
		_ = os.MkdirAll(dir, 0o755)
		t.Setenv("RECIPES_COMPILE_DIR", dir)
		t.Logf("keeping snippet module at %s", dir)
	}
	gomod := "module recipescompile\n\ngo 1.27.1\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "go.work"),
		[]byte(workspaceFor(t, root, dir)),
		0o644,
	); err != nil {
		t.Fatalf("write go.work: %v", err)
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
		"GOWORK="+filepath.Join(dir, "go.work"),
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
