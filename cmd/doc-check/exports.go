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

// buildExportIndex creates a map: package alias → set of exported symbols.
// It maps import paths to their package directories and parses the .go files
// to collect exported declarations. Every anomaly (unreadable dir, zero
// exports, unparseable file) is returned as a warning so the caller can gate
// on a zero-warning policy.
func buildExportIndex(imports []string, repoRoot string) (map[string]map[string]bool, []string) {
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

	var warnings []string

	for _, imp := range unique {
		dir := strings.TrimPrefix(imp, repoImportPrefix)

		dir = strings.TrimSuffix(dir, "/v4")
		if dir == "v3" {
			dir = "."
		}

		fullDir := filepath.Join(repoRoot, dir)
		if _, err := os.Stat(fullDir); os.IsNotExist(err) {
			stripped := strings.Replace(dir, "/v4/", "/", 1)
			if stripped != dir {
				fullDir = filepath.Join(repoRoot, stripped)
			}
		}

		pkgName := filepath.Base(dir)

		exports, pkgWarnings := parsePackageExports(fullDir)
		warnings = append(warnings, pkgWarnings...)

		if len(exports) == 0 {
			warnings = append(warnings, "no exports found in "+dir)

			continue
		}

		if _, ok := index[pkgName]; !ok {
			index[pkgName] = make(map[string]bool)
		}

		for sym := range exports {
			index[pkgName][sym] = true
		}
	}

	return index, warnings
}

// parsePackageExports parses all non-test .go files in dir and returns
// a set of exported symbol names (types, functions, vars, consts) plus any
// warnings (unreadable dir, unparseable file).
func parsePackageExports(dir string) (map[string]bool, []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, []string{fmt.Sprintf("cannot read %s: %v", dir, err)}
	}

	exports := make(map[string]bool)

	var warnings []string

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !shouldParseFile(entry.Name()) {
			continue
		}

		if warn := collectExports(fset, filepath.Join(dir, entry.Name()), exports); warn != "" {
			warnings = append(warnings, warn)
		}
	}

	return exports, warnings
}

func shouldParseFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// collectExports parses one file into exports. It returns a warning string
// when the file cannot be parsed (empty string on success).
func collectExports(fset *token.FileSet, path string, exports map[string]bool) string {
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return fmt.Sprintf("cannot parse %s: %v", path, err)
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

	return ""
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
