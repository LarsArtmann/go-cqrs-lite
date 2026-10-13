package analyzer

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// CollectTestImportPaths walks every module under root and returns the
// sorted, deduplicated import paths declared in _test.go files.
//
// The typed package load runs with Tests: false (loader.go), so test files
// appear in neither pkg.Imports nor ctx.GoFiles — test-only modules
// (systemscenario, scenario, the stack/memory test preset) are structurally
// invisible to DetectUsedModules without this walk, and their scorecard rows
// can never leave "missing" no matter how faithfully the consumer follows
// the ADR-0153 testing pattern. Imports need no type information, so plain
// go/parser with parser.ImportsOnly keeps the walk cheap.
//
// Best effort by design: a _test.go file that does not parse is skipped (the
// heavy loader owns module-graph error reporting; a broken test file must
// not kill the scorecard), while filesystem-level walk errors fail loudly.
func CollectTestImportPaths(root string) ([]string, error) {
	modDirs, err := findGoModDirs(root)
	if err != nil {
		return nil, err
	}

	if len(modDirs) == 0 {
		modDirs = []string{root}
	}

	fset := token.NewFileSet()
	seen := make(map[string]struct{})

	for _, dir := range modDirs {
		if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // skip inaccessible paths, continue walking
			}

			if d.IsDir() {
				if analysisSkipDir(d.Name()) {
					return filepath.SkipDir
				}

				return nil
			}

			if !strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}

			// A nil src reads the file from disk; ImportsOnly stops after the
			// import block, so even large test files parse in microseconds.
			file, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if perr != nil {
				return nil //nolint:nilerr // broken test file: skip, loader owns error reporting
			}

			for _, imp := range file.Imports {
				if imp.Path == nil {
					continue
				}

				seen[strings.Trim(imp.Path.Value, `"`)] = struct{}{}
			}

			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk %s: %w", dir, err)
		}
	}

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	slices.Sort(paths)

	return paths, nil
}
