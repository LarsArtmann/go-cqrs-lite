package adoption

import (
	"os"
	"sort"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// moduleScope bundles the feature profile and file set for a single Go module
// so coaching rules can evaluate each module independently. In a multi-module
// workspace (e.g. a library + example apps), this prevents an example app's
// server/store signals from leaking into the library module's coaching.
type moduleScope struct {
	dir     string
	profile analyzer.FeatureProfile
	files   []*analyzer.GoFile
}

// coachingScopes returns one moduleScope per module to evaluate. For a single-
// module project (no FeatureProfiles) it returns one scope with all non-test
// files and the primary profile — preserving the pre-per-module behavior.
// For a multi-module workspace each module gets its own scope so coaching
// rules fire per-module (an example app gets server coaching; the library
// does not).
func coachingScopes(ctx *analyzer.AnalysisContext) []moduleScope {
	if len(ctx.FeatureProfiles) == 0 {
		return []moduleScope{{profile: ctx.FeatureProfile, files: nonTestFiles(ctx)}}
	}

	dirs := make([]string, 0, len(ctx.FeatureProfiles))
	for d := range ctx.FeatureProfiles {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	filesByDir := make(map[string][]*analyzer.GoFile, len(dirs))
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}
		dir := attributeModule(gf, dirs)
		filesByDir[dir] = append(filesByDir[dir], gf)
	}

	scopes := make([]moduleScope, 0, len(dirs))
	for _, dir := range dirs {
		files := filesByDir[dir]
		if len(files) == 0 {
			continue
		}
		scopes = append(scopes, moduleScope{
			dir:     dir,
			profile: ctx.FeatureProfiles[dir],
			files:   files,
		})
	}

	return scopes
}

// attributeModule returns the module dir that owns gf. Prefers the explicit
// ModuleDir set by the loader (real usage); falls back to longest-prefix path
// matching against the known module dirs (for test contexts where ModuleDir
// is unset). Mirrors the ProfileForFile resolution logic.
func attributeModule(gf *analyzer.GoFile, sortedDirs []string) string {
	if gf.ModuleDir != "" {
		for _, d := range sortedDirs {
			if d == gf.ModuleDir {
				return d
			}
		}
	}

	var best string
	for _, d := range sortedDirs {
		if gf.Path == d || strings.HasPrefix(gf.Path, d+string(os.PathSeparator)) {
			if len(d) > len(best) {
				best = d
			}
		}
	}

	return best
}

func nonTestFiles(ctx *analyzer.AnalysisContext) []*analyzer.GoFile {
	var out []*analyzer.GoFile
	for _, gf := range ctx.GoFiles {
		if !gf.IsTest {
			out = append(out, gf)
		}
	}

	return out
}
