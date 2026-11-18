package analyzer

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// analysisSkipDir reports whether a directory name is excluded from analysis
// walks (shared by findGoModDirs and hasGoSourceFiles).
func analysisSkipDir(base string) bool {
	return base == "vendor" || base == ".git" || base == "node_modules" ||
		base == "testdata" || base == "dist" || base == "build"
}

// findGoModDirs walks the directory tree under root and returns all directories
// containing a go.mod file. Skips vendor, .git, node_modules, and similar dirs.
func findGoModDirs(root string) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip inaccessible paths, continue walking
		}

		if d.IsDir() {
			if analysisSkipDir(d.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if d.Name() == "go.mod" {
			dir := filepath.Dir(path)
			dirs = append(dirs, dir)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	return dirs, nil
}

// hasGoSourceFiles reports whether any .go file exists under the given module
// directories (stopping at the first hit). Used by the silent-empty guard to
// tell "repo has no Go code" (legitimate clean verdict) from "Go code exists
// but go/packages returned nothing" (swallowed driver failure — loud error).
func hasGoSourceFiles(dirs []string) bool {
	for _, dir := range dirs {
		found := false

		err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
			if err != nil || found {
				return nil //nolint:nilerr // skip inaccessible paths, continue walking
			}

			if d.IsDir() {
				if analysisSkipDir(d.Name()) {
					return filepath.SkipDir
				}

				return nil
			}

			if strings.HasSuffix(d.Name(), ".go") {
				found = true

				return filepath.SkipAll
			}

			return nil
		})
		if err != nil || found {
			return found
		}
	}

	return false
}

// ModuleDirs returns every directory under root containing a go.mod, skipping
// vendor, .git, node_modules, testdata, dist, and build. Exported for toolsdk
// hosts that replicate the CLI's per-module walk semantics (see
// BuildContextWithEnv).
func ModuleDirs(root string) ([]string, error) {
	return findGoModDirs(root)
}
