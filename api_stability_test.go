// api_stability_test.go verifies that exported API symbols are not accidentally
// removed between versions. Run with: go test -run TestAPIStability ./...
//
// This test uses go/ast to parse source files and snapshot the exported
// symbols. A golden file (docs/api_surface.txt) stores the expected surface.
// If the surface changes intentionally, regenerate with:
//
//	go test -run TestAPIStability -update ./...
package api_stability

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
}

var modules = []string{
	"core/command",
	"core/event",
	"core/query",
	"core/decider",
	"core/pkg/id",
	"core/pkg/dispatcher",
	"memory",
	"catalog",
	"middleware",
	"signing",
	"projection",
	"saga",
	"stream",
	"otel",
	"storage",
	"testhelpers",
	"watermill",
}

func TestAPIStability(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	var exports []string

	for _, mod := range modules {
		modPath := filepath.Join(projectRoot, mod)
		if _, err := os.Stat(modPath); os.IsNotExist(err) {
			continue
		}

		exps, err := collectExports(modPath)
		if err != nil {
			t.Logf("skip %s: %v", mod, err)

			continue
		}

		for _, e := range exps {
			exports = append(exports, mod+"/"+e)
		}
	}

	sort.Strings(exports)

	if os.Getenv("UPDATE") == "1" || os.Getenv("update") == "1" {
		err := os.MkdirAll(filepath.Dir(goldenPath), 0o755)
		if err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		err = os.WriteFile(goldenPath, []byte(strings.Join(exports, "\n")+"\n"), 0o644)
		if err != nil {
			t.Fatalf("write golden: %v", err)
		}

		t.Logf("Updated %s (%d exports)", goldenPath, len(exports))

		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("golden file %s does not exist; run with -update to create", goldenPath)
		}

		t.Fatalf("read golden: %v", err)
	}

	expected := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(exports) != len(expected) {
		missing, added := diff(expected, exports)

		t.Errorf("API surface mismatch: %d expected, %d actual", len(expected), len(exports))

		if len(missing) > 0 {
			t.Errorf("Removed exports:\n  %s", strings.Join(missing, "\n  "))
		}

		if len(added) > 0 {
			t.Logf("New exports:\n  %s", strings.Join(added, "\n  "))
		}

		return
	}

	for i, exp := range expected {
		if exports[i] != exp {
			t.Errorf("export %d: expected %q, got %q", i, exp, exports[i])
		}
	}
}

func collectExports(dir string) ([]string, error) {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, dir, nonTestFilter, 0)
	if err != nil {
		return nil, err
	}

	var exports []string

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				for _, spec := range genDecl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}

					switch ts.Type.(type) {
					case *ast.InterfaceType:
						exports = append(exports, "interface "+ts.Name.Name)
					case *ast.StructType:
						exports = append(exports, "struct "+ts.Name.Name)
					default:
						exports = append(exports, "type "+ts.Name.Name)
					}
				}

				if genDecl.Tok.String() == "var" {
					for _, spec := range genDecl.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}

						for _, name := range vs.Names {
							if name.IsExported() {
								exports = append(exports, "var "+name.Name)
							}
						}
					}
				}

				if genDecl.Tok.String() == "const" {
					for _, spec := range genDecl.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}

						for _, name := range vs.Names {
							if name.IsExported() {
								exports = append(exports, "const "+name.Name)
							}
						}
					}
				}
			}

			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !fn.Name.IsExported() {
					continue
				}

				if fn.Recv == nil {
					exports = append(exports, "func "+fn.Name.Name)
				} else {
					exports = append(exports, "method "+fn.Name.Name)
				}
			}
		}
	}

	sort.Strings(exports)

	return exports, nil
}

func nonTestFilter(fi os.FileInfo) bool {
	return !strings.HasSuffix(fi.Name(), "_test.go")
}

func diff(expected, actual []string) (missing, added []string) {
	expSet := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		expSet[e] = struct{}{}
	}

	actSet := make(map[string]struct{}, len(actual))
	for _, a := range actual {
		actSet[a] = struct{}{}
	}

	for _, e := range expected {
		if _, ok := actSet[e]; !ok {
			missing = append(missing, e)
		}
	}

	for _, a := range actual {
		if _, ok := expSet[a]; !ok {
			added = append(added, a)
		}
	}

	return missing, added
}
