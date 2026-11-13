package analyzer

import (
	"os"
	"path/filepath"
	"strings"
)

// findEnclosingGoModDir walks UP from root's parent to the filesystem root
// looking for the nearest go.mod — the module that owns a root sitting
// inside a module (no go.mod at or under it). This mirrors how the go tool
// resolves module context for commands run from a subdirectory.
func findEnclosingGoModDir(root string) (string, bool) {
	for dir := filepath.Dir(root); ; dir = filepath.Dir(dir) {
		if stat, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !stat.IsDir() {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
	}
}

// isUnderDir reports whether path equals dir or lives underneath it.
func isUnderDir(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+string(os.PathSeparator))
}

// primaryModuleProfile selects the profile of the project's primary module from
// a per-module map. The primary module is the go.mod at projectRoot when one
// exists; otherwise the shallowest module dir. Callers only invoke this when
// the map is non-empty.
func primaryModuleProfile(
	profiles map[string]FeatureProfile,
	projectRoot string,
	modDirs []string,
) FeatureProfile {
	if p, ok := profiles[projectRoot]; ok {
		return p
	}

	// Shallowest dir wins (closest to the filesystem root). Ties break on
	// path name so the selection is deterministic (map iteration is not).
	var best string
	for dir := range profiles {
		if best == "" || pathDepth(dir) < pathDepth(best) ||
			(pathDepth(dir) == pathDepth(best) && dir < best) {
			best = dir
		}
	}
	if best != "" {
		return profiles[best]
	}

	// Last resort: first module dir.
	if len(modDirs) > 0 {
		return profiles[modDirs[0]]
	}

	return FeatureProfile{}
}
