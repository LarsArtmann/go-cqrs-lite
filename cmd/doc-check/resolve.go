package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// resolver verifies qualified references against block-scoped imports first,
// then falls back to a repo-wide package-alias index for references whose
// import is not visible in the doc block. Parsing is memoized per import path.
type resolver struct {
	repoRoot    string
	exports     map[string]map[string]bool
	clauses     map[string]string
	aliasLoaded bool
	aliasDirs   map[string][]string
	aliasExps   map[string]map[string]bool
}

func newResolver(repoRoot string) *resolver {
	return &resolver{
		repoRoot:  repoRoot,
		exports:   make(map[string]map[string]bool),
		clauses:   make(map[string]string),
		aliasDirs: make(map[string][]string),
		aliasExps: make(map[string]map[string]bool),
	}
}

// warm parses one import path once and returns any parse warnings. The
// zero-warning gate stays meaningful: every doc-visible import is pre-warmed.
func (r *resolver) warm(imp string) []string {
	if _, ok := r.exports[imp]; ok {
		return nil
	}

	exports, clause, warns := parsePackageExports(r.fullDir(imp))
	r.exports[imp] = exports
	r.clauses[imp] = clause

	if len(exports) == 0 {
		warns = append(warns, "no exports found in "+importDir(imp))
	}

	return warns
}

// resolve reports whether one reference is valid. A reference whose alias is
// neither a block import nor a repo package alias is treated as external and
// skipped (documented limitation, matches the historical behavior).
func (r *resolver) resolve(b block, rr ref) bool {
	if paths := r.blockPaths(b, rr.pkg); len(paths) > 0 {
		for _, p := range paths {
			if r.exports[p][rr.symbol] {
				return true
			}
		}

		return false
	}

	if alias := r.aliasExports(rr.pkg); alias != nil {
		return alias[rr.symbol]
	}

	return true
}

// blockPaths returns the block's import paths whose package name matches the
// reference alias (package clause or directory base name).
func (r *resolver) blockPaths(b block, alias string) []string {
	var out []string

	for _, imp := range b.imports {
		if r.clauses[imp] == alias || filepath.Base(importDir(imp)) == alias {
			out = append(out, imp)
		}
	}

	return out
}

// aliasExports returns the union of exports across every repo package with
// the given package name, or nil when no repo package has that name.
func (r *resolver) aliasExports(alias string) map[string]bool {
	if !r.aliasLoaded {
		r.loadAliasDirs()

		r.aliasLoaded = true
	}

	if exp, ok := r.aliasExps[alias]; ok {
		return exp
	}

	if len(r.aliasDirs[alias]) == 0 {
		r.aliasExps[alias] = nil

		return nil
	}

	union := make(map[string]bool)

	for _, dir := range r.aliasDirs[alias] {
		exports, _, _ := parsePackageExports(filepath.Join(r.repoRoot, dir))
		for sym := range exports {
			union[sym] = true
		}
	}

	r.aliasExps[alias] = union

	return union
}

// loadAliasDirs walks the repo once and maps package names to directories.
func (r *resolver) loadAliasDirs() {
	_ = filepath.WalkDir(r.repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil //nolint:nilerr // unreadable entries are skipped silently
		}

		if name := d.Name(); name == ".git" || name == "vendor" ||
			name == "node_modules" || strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}

		r.indexPackageDir(path)

		return nil //nolint:nilerr // always continue the walk
	})
}

// indexPackageDir records one directory under its package-clause name when it
// contains at least one non-test .go file.
func (r *resolver) indexPackageDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() || !shouldParseFile(entry.Name()) {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.PackageClauseOnly)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(r.repoRoot, dir)
		if err != nil {
			return
		}

		r.aliasDirs[file.Name.Name] = append(r.aliasDirs[file.Name.Name], rel)

		return
	}
}

// verifyBlocks checks every reference block-scoped and returns the broken
// count, the total, and any parse warnings for the zero-warning gate.
func verifyBlocks(blocks []block, allImports []string, res *resolver) (int, int, []string) {
	var warnings []string

	for _, imp := range dedupe(allImports) {
		warnings = append(warnings, res.warm(imp)...)
	}

	broken, total := 0, 0

	for _, b := range blocks {
		for _, rr := range b.refs {
			total++

			if !res.resolve(b, rr) {
				log.Printf("  ✗ %s:%d: %s.%s not found", rr.file, rr.line, rr.pkg, rr.symbol)

				broken++
			}
		}
	}

	return broken, total, warnings
}

// dedupe preserves first-seen order.
func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))

	var out []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true

			out = append(out, item)
		}
	}

	return out
}

// importDir maps an import path to its repo-relative directory.
func importDir(imp string) string {
	dir := strings.TrimPrefix(imp, repoImportPrefix)
	dir = strings.TrimSuffix(dir, "/v4")

	if dir == "v3" {
		dir = "."
	}

	return dir
}

// fullDir resolves an import path to a filesystem directory, applying the
// historical /v4/ in-path fallback when the canonical directory is absent.
func (r *resolver) fullDir(imp string) string {
	dir := importDir(imp)
	full := filepath.Join(r.repoRoot, dir)

	if _, err := os.Stat(full); os.IsNotExist(err) {
		if stripped := strings.Replace(dir, "/v4/", "/", 1); stripped != dir {
			full = filepath.Join(r.repoRoot, stripped)
		}
	}

	return full
}
