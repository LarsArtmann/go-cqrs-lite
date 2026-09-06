package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// parsePackageExports parses all non-test .go files in dir and returns the
// exported symbol names, the package clause, and any warnings (unreadable
// dir, unparseable file). The clause is the authoritative Go package name,
// which may differ from the directory base name.
func parsePackageExports(dir string) (map[string]bool, string, []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", []string{fmt.Sprintf("cannot read %s: %v", dir, err)}
	}

	exports := make(map[string]bool)

	var warnings []string

	clause := ""

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() || !shouldParseFile(entry.Name()) {
			continue
		}

		fileClause, warn := collectExports(fset, filepath.Join(dir, entry.Name()), exports)
		if clause == "" {
			clause = fileClause
		}

		if warn != "" {
			warnings = append(warnings, warn)
		}
	}

	return exports, clause, warnings
}

func shouldParseFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// collectExports parses one file into exports and returns its package clause
// (empty when the file cannot be parsed, together with a warning string).
func collectExports(fset *token.FileSet, path string, exports map[string]bool) (string, string) {
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return "", fmt.Sprintf("cannot parse %s: %v", path, err)
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

	return file.Name.Name, ""
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
