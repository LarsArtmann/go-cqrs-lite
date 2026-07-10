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

	errorfamily "github.com/larsartmann/go-error-family"
)

func main() {
	modules := []string{
		// Layer 0: leaf modules
		"id",
		"id/idtest",
		"dispatcher",
		"codec",
		"kv",
		"dedup",
		// Layer 1
		"event",
		"event/eventtest",
		"command",
		"query",
		"query/querytest",
		"idempotency",
		// Layer 2
		"schema",
		"snapshot",
		"projection",
		"deriver",
		// Layer 3
		"decider",
		"graph",
		"scenario",
		"projectionhost",
		"scheduling",
		"metadata",
		// Layer 4
		"memory",
		"signing",
		"encryption",
		"otel",
		// Layer 5
		"middleware",
		"storage",
		"storage/sql",
		"storage/memory",
		"storage/pebble",
		"storage/turso",
		"listing",
		"watermill",
		"pebble",
		"turso",
		"prometheus",
		"transport/http",
		"transport/grpc",
		// Composition (Bundle layer)
		"stack",
		"stack/memory",
		"stack/sqlite",
		"stack/pebble",
		"stack/postgres",
		"stack/turso",
		// Tooling + catalog
		"testutil",
		"catalog",
	}

	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	exports := collectAllModuleExports(modules, projectRoot)

	sort.Strings(exports)

	if len(os.Args) > 1 && os.Args[1] == "-update" {
		writeGoldenFile(goldenPath, exports)

		return
	}

	verifyGoldenFile(goldenPath, exports)
}

func collectAllModuleExports(modules []string, projectRoot string) []string {
	var exports []string

	for _, mod := range modules {
		modPath := filepath.Join(projectRoot, mod)

		_, err := os.Stat(modPath)
		if os.IsNotExist(err) {
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

	return exports
}

const (
	goldenDirPerms  = 0o750
	goldenFilePerms = 0o600
)

func writeGoldenFile(goldenPath string, exports []string) {
	cleanPath := filepath.Clean(goldenPath)

	err := os.MkdirAll(filepath.Dir(cleanPath), goldenDirPerms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(cleanPath, []byte(strings.Join(exports, "\n")+"\n"), goldenFilePerms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Updated %s (%d exports)\n", cleanPath, len(exports))

	os.Exit(0)
}

func verifyGoldenFile(goldenPath string, exports []string) {
	cleanPath := filepath.Clean(goldenPath)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		handleReadError(cleanPath, err)
	}

	expected := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(exports) != len(expected) {
		reportMismatch(expected, exports)
	}

	for i, exp := range expected {
		if exports[i] != exp {
			fmt.Fprintf(os.Stderr, "export %d: expected %q, got %q\n", i, exp, exports[i])
			os.Exit(1)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "API surface OK: %d exports verified\n", len(exports))
}

func handleReadError(goldenPath string, err error) {
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

func reportMismatch(expected, exports []string) {
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

func collectExports(dir string) ([]string, error) {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"api_stability.read_dir",
			"read dir %s",
			dir,
		)
	}

	pkgFiles := make(map[string][]*ast.File)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return nil, errorfamily.Wrapf(
				err,
				errorfamily.Infrastructure,
				"api_stability.parse_file",
				"parse file %s",
				entry.Name(),
			)
		}

		pkgName := file.Name.Name
		pkgFiles[pkgName] = append(pkgFiles[pkgName], file)
	}

	var exports []string

	for _, files := range pkgFiles {
		for _, file := range files {
			exports = append(exports, collectFileExports(file)...)
		}
	}

	sort.Strings(exports)

	return exports, nil
}

func collectFileExports(file *ast.File) []string {
	var exports []string

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			exports = append(exports, collectGenDeclExports(d)...)
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				if d.Recv == nil {
					exports = append(exports, "func "+d.Name.Name)
				} else {
					exports = append(exports, "method "+d.Name.Name)
				}
			}
		}
	}

	return exports
}

func collectGenDeclExports(genDecl *ast.GenDecl) []string {
	var exports []string

	for _, spec := range genDecl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !s.Name.IsExported() {
				continue
			}

			exports = append(exports, typeExportName(s))
		case *ast.ValueSpec:
			prefix := strings.ToLower(genDecl.Tok.String())

			for _, name := range s.Names {
				if name.IsExported() {
					exports = append(exports, prefix+" "+name.Name)
				}
			}
		}
	}

	return exports
}

func typeExportName(ts *ast.TypeSpec) string {
	switch ts.Type.(type) {
	case *ast.InterfaceType:
		return "interface " + ts.Name.Name
	case *ast.StructType:
		return "struct " + ts.Name.Name
	default:
		return "type " + ts.Name.Name
	}
}

func diff(expected, actual []string) ([]string, []string) {
	expSet := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		expSet[e] = struct{}{}
	}

	actSet := make(map[string]struct{}, len(actual))
	for _, a := range actual {
		actSet[a] = struct{}{}
	}

	var missing []string

	for _, e := range expected {
		if _, ok := actSet[e]; !ok {
			missing = append(missing, e)
		}
	}

	var added []string

	for _, a := range actual {
		if _, ok := expSet[a]; !ok {
			added = append(added, a)
		}
	}

	return missing, added
}
