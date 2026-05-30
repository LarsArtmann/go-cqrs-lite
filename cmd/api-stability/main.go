package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var modules = []string{
	"command",
	"event",
	"query",
	"decider",
	"id",
	"dispatcher",
	"memory",
	"catalog",
	"middleware",
	"signing",
	"projection",
	"listing",
	"otel",
	"storage",
	"event/eventtest",
	"watermill",
}

func main() {
	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	var exports []string

	for _, mod := range modules {
		modPath := filepath.Join(projectRoot, mod)
		if _, err := os.Stat(modPath); os.IsNotExist(err) {
			continue
		}

		exps, err := collectExports(modPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", mod, err)
			continue
		}

		for _, e := range exps {
			exports = append(exports, mod+"/"+e)
		}
	}

	sort.Strings(exports)

	if len(os.Args) > 1 && os.Args[1] == "-update" {
		err := os.MkdirAll(filepath.Dir(goldenPath), 0o755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
			os.Exit(1)
		}

		err = os.WriteFile(goldenPath, []byte(strings.Join(exports, "\n")+"\n"), 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated %s (%d exports)\n", goldenPath, len(exports))
		os.Exit(0)
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(
				os.Stderr,
				"golden file %s does not exist; run with -update to create\n",
				goldenPath,
			)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}

	expected := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(exports) != len(expected) {
		missing, added := diff(expected, exports)
		fmt.Fprintf(
			os.Stderr,
			"API surface mismatch: %d expected, %d actual\n",
			len(expected),
			len(exports),
		)

		if len(missing) > 0 {
			fmt.Fprintf(os.Stderr, "REMOVED exports:\n  %s\n", strings.Join(missing, "\n  "))
		}

		if len(added) > 0 {
			fmt.Fprintf(os.Stderr, "NEW exports:\n  %s\n", strings.Join(added, "\n  "))
		}

		os.Exit(1)
	}

	for i, exp := range expected {
		if exports[i] != exp {
			fmt.Fprintf(os.Stderr, "export %d: expected %q, got %q\n", i, exp, exports[i])
			os.Exit(1)
		}
	}

	fmt.Printf("API surface OK: %d exports verified\n", len(exports))
}

func collectExports(dir string) ([]string, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nonTestFilter, 0)
	if err != nil {
		return nil, fmt.Errorf("parse dir %s: %w", dir, err)
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

				if genDecl.Tok.String() == "var" || genDecl.Tok.String() == "const" {
					prefix := strings.ToLower(genDecl.Tok.String())
					for _, spec := range genDecl.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, name := range vs.Names {
							if name.IsExported() {
								exports = append(exports, prefix+" "+name.Name)
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
