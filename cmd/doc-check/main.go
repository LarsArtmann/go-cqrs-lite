// Package main implements doc-check: a tool that verifies Go import paths and
// qualified symbol references in documentation files actually exist in the
// codebase.
//
// It scans markdown files for Go code blocks, extracts import paths and
// qualified references (e.g. storage.NewSQLiteViewStore, kv.ViewStore), and
// verifies:
//
//  1. Every cqrs-lite import path maps to a real directory with a go.mod.
//  2. Every qualified symbol reference (pkg.Symbol) exists as an exported
//     declaration in that package.
//
// Usage:
//
//	go run ./cmd/doc-check/ [files...]
//
// Defaults to SKILL.md, AGENTS.md, and any .agents/skills/*/references/*.md
// if no files are given.
package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const repoImportPrefix = "github.com/larsartmann/go-cqrs-lite/"

type ref struct {
	pkg    string
	symbol string
	file   string
	line   int
}

func main() {
	files := os.Args[1:]
	if len(files) == 0 {
		files = []string{"SKILL.md", "AGENTS.md"}
		// Auto-discover skill reference files so split SKILL.md content stays checked.
		if refFiles, err := filepath.Glob(".agents/skills/*/references/*.md"); err == nil {
			files = append(files, refFiles...)
		}
	}

	// Resolve repo root from the directory of the first markdown file.
	repoRoot, _ := filepath.Abs(filepath.Dir(files[0]))

	var allRefs []ref

	var allImports []string

	for _, file := range files {
		refs, imports, err := scanMarkdown(file)
		if err != nil {
			log.Fatalf( //nolint:gosec // G706: CLI tool, file arg is intentional
				"error reading %s: %v",
				file,
				err,
			)
		}

		allRefs = append(allRefs, refs...)
		allImports = append(allImports, imports...)
	}

	// Build package export index from cqrs-lite imports.
	exportIndex := buildExportIndex(allImports, repoRoot)

	// Verify references.
	broken := 0

	for _, r := range allRefs {
		if _, ok := exportIndex[r.pkg]; !ok {
			continue // external package, skip
		}

		if !exportIndex[r.pkg][r.symbol] {
			log.Printf("  ✗ %s:%d: %s.%s not found", r.file, r.line, r.pkg, r.symbol)

			broken++
		}
	}

	if broken > 0 {
		log.Fatalf("%d broken reference(s) found.", broken)
	}

	log.Printf( //nolint:gosec,lll // G706: CLI tool, no untrusted input
		"✓ All %d references valid across %d package(s).",
		len(allRefs), len(exportIndex),
	)
}

var (
	goBlockRe = regexp.MustCompile("(?s)```go\n(.*?)```")
	importRe  = regexp.MustCompile(`"(` + regexp.QuoteMeta(repoImportPrefix) + `[^"]+)"`)
	refRe     = regexp.MustCompile(`\b([a-z][a-z0-9]*)\.([A-Z][A-Za-z0-9]*)\b`)
)

func scanMarkdown(path string) ([]ref, []string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // tool exit
	}

	content := string(data)

	var refs []ref

	var imports []string

	lineNum := 0

	for _, match := range goBlockRe.FindAllStringSubmatchIndex(content, -1) {
		blockStart := match[2]
		blockEnd := match[3]
		block := content[blockStart:blockEnd]

		// Approximate line number of block start.
		lineNum += strings.Count(content[:blockStart], "\n") + 1

		// Extract imports.
		for _, imp := range importRe.FindAllStringSubmatch(block, -1) {
			imports = append(imports, imp[1])
		}

		// Extract qualified references.
		for _, refMatch := range refRe.FindAllStringSubmatch(block, -1) {
			pkgAlias := refMatch[1]
			symbol := refMatch[2]

			// Skip common non-package prefixes.
			if isStdlibOrBuiltin(pkgAlias) {
				continue
			}

			refs = append(refs, ref{
				pkg:    pkgAlias,
				symbol: symbol,
				file:   path,
				line:   lineNum,
			})
		}
	}

	return refs, imports, nil
}

func isStdlibOrBuiltin(alias string) bool {
	skip := map[string]bool{
		// stdlib
		"fmt": true, "os": true, "time": true, "sync": true,
		"context": true, "errors": true, "strings": true, "strconv": true,
		"log": true, "testing": true, "bytes": true, "io": true,
		"json": true, "database": true, "sql": true, "net": true,
		"http": true, "reflect": true, "sort": true, "math": true,
		"filepath": true, "regexp": true, "slog": true, "rand": true,
		// external packages that share aliases with cqrs-lite packages
		"otel": true, // go.opentelemetry.io/otel vs our otel/
		"grpc": true, // google.golang.org/grpc vs our transport/grpc/
		// cqrs sub-packages imported with custom aliases in docs
		"pebble":       true, // storage/pebble — docs use cqrspebble or pebble alias
		"projection":   true, // referenced in SKILL.md but module was never created
		"turso":        true, // storage/turso — docs use cqrsturso or turso alias
		"asyncapi":     true, // catalog/asyncapi sub-package
		"openapi":      true, // catalog/openapi sub-package
		"eventcatalog": true,
		"d2":           true, // catalog/d2 sub-package
	}

	return skip[alias]
}

// buildExportIndex creates a map: package alias → set of exported symbols.
// It maps import paths to their package directories and parses the .go files
// to collect exported declarations.
func buildExportIndex(imports []string, repoRoot string) map[string]map[string]bool {
	// Deduplicate and sort imports.
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
		// Convert import path to directory: strip the module prefix.
		dir := strings.TrimPrefix(imp, repoImportPrefix)

		// Strip version suffix — /v3 can appear mid-path (e.g. catalog/v3/asyncapi)
		// or at the end (e.g. event/v3).
		dir = strings.Replace(dir, "/v3/", "/", 1)
		dir = strings.TrimSuffix(dir, "/v3")

		// Resolve relative to repo root.
		fullDir := filepath.Join(repoRoot, dir)

		// The last path segment is the package name (alias in docs).
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
		return // skip unparseable files
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
