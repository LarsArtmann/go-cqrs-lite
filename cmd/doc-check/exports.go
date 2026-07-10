package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// buildExportIndex creates a map: package alias → set of exported symbols.
// It maps import paths to their package directories and parses the .go files
// to collect exported declarations.
func buildExportIndex(imports []string, repoRoot string) map[string]map[string]bool {
	seen := make(map[string]bool)

	var unique []string

	for _, imp := range imports {
		if !seen[imp] {
			seen[imp] = true
			unique = append(unique, imp)
		}
	}

	sort.Strings(unique)

	index := make(map[string]map[string]bool)

	for _, imp := range unique {
		dir := strings.TrimPrefix(imp, repoImportPrefix)

		dir = strings.TrimSuffix(dir, "/v3")
		if dir == "v3" {
			dir = "."
		}

		fullDir := filepath.Join(repoRoot, dir)
		if _, err := os.Stat(fullDir); os.IsNotExist(err) {
			stripped := strings.Replace(dir, "/v3/", "/", 1)
			if stripped != dir {
				fullDir = filepath.Join(repoRoot, stripped)
			}
		}

		pkgName := filepath.Base(dir)

		exports := parsePackageExports(fullDir)
		if len(exports) == 0 {
			log.Printf("warning: no exports found in %s", dir) //nolint:gosec,lll // G706: CLI tool

			continue
		}

		if _, ok := index[pkgName]; !ok {
			index[pkgName] = make(map[string]bool)
		}

		for sym := range exports {
			index[pkgName][sym] = true
		}
	}

	return index
}

// parsePackageExports parses all non-test .go files in dir and returns
// a set of exported symbol names (types, functions, vars, consts).
func parsePackageExports(dir string) map[string]bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("warning: cannot read %s: %v", dir, err) //nolint:gosec,lll // G706: CLI tool

		return nil
	}

	exports := make(map[string]bool)

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !shouldParseFile(entry.Name()) {
			continue
		}

		collectExports(fset, filepath.Join(dir, entry.Name()), exports)
	}

	return exports
}

func shouldParseFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func collectExports(fset *token.FileSet, path string, exports map[string]bool) {
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				exports[d.Name.Name] = true
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				collectGenDeclExports(spec, exports)
			}
		}
	}
}

func collectGenDeclExports(spec ast.Spec, exports map[string]bool) {
	if vs, ok := spec.(*ast.ValueSpec); ok {
		for _, name := range vs.Names {
			if name.IsExported() {
				exports[name.Name] = true
			}
		}
	}

	if ts, ok := spec.(*ast.TypeSpec); ok {
		if ts.Name.IsExported() {
			exports[ts.Name.Name] = true
		}
	}
}
