package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

// skippedDirNames are directory names whose subtrees never contain modules
// worth upgrading: vendored copies, test fixtures, and VCS metadata.
var skippedDirNames = map[string]bool{
	"vendor":     true,
	"testdata":   true,
	".git":       true,
	"node_modules": true,
}

// findGoMods returns every directory under root (inclusive) that contains a
// go.mod, sorted for deterministic output. Used by --workspace mode to run
// the upgrade pipeline across a multi-module tree.
func findGoMods(root string) ([]string, error) {
	var mods []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skippedDirNames[d.Name()] {
				return fs.SkipDir
			}

			return nil
		}

		if d.Name() == "go.mod" {
			mods = append(mods, filepath.Dir(path))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	sort.Strings(mods)

	return mods, nil
}
