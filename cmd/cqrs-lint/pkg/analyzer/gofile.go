package analyzer

import (
	"fmt"
	"go/ast"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// GoFile wraps a parsed Go file with its package context.
type GoFile struct {
	Path   string
	Pkg    *packages.Package
	AST    *ast.File
	IsTest bool
	// ModuleDir is the filesystem path of the go.mod directory this file belongs
	// to. Set in BuildContext so feature detection can be partitioned per module
	// (a multi-module workspace gets one FeatureProfile per module instead of a
	// single cross-module merge). Empty when the file's module is unknown.
	ModuleDir string
}

// ProfileForFile returns the FeatureProfile for the module that owns the given
// file path. This lets per-file detectors evaluate each finding against its own
// module's features rather than a workspace-wide merge (so a server call in an
// examples/ module does not enable server-only rules for the library module).
// Falls back to the primary FeatureProfile when no per-module profile matches
// (files outside any known module). When the per-module map is non-empty but
// NOTHING matches, a stderr warning fires once: that shape hid the
// relative-key bug fixed 2026-10-12 (a relative BuildContext root produced
// relative keys that never matched absolute finding positions, silently
// downgrading every per-file-gated rule to the primary profile) and must
// never be silent again.
func (ctx *AnalysisContext) ProfileForFile(path string) FeatureProfile {
	if len(ctx.FeatureProfiles) == 0 {
		return ctx.FeatureProfile
	}

	// A file belongs to the module whose dir is the longest matching prefix of
	// the file path (handles nested modules: /repo/examples/basic owns files
	// under /repo/examples/basic/...).
	var bestDir string
	for dir := range ctx.FeatureProfiles {
		if !strings.HasPrefix(path, dir+string(os.PathSeparator)) && path != dir {
			continue
		}
		if len(dir) > len(bestDir) {
			bestDir = dir
		}
	}

	if bestDir != "" {
		return ctx.FeatureProfiles[bestDir]
	}

	ctx.profileFallbackWarn.Do(func() {
		fmt.Fprintf(
			os.Stderr,
			"cqrs-lint: WARNING no per-module feature profile matches %s — "+
				"falling back to the primary module's profile; per-file-gated rules "+
				"evaluate against the WRONG module here. Profile dirs: %d; this usually "+
				"means profile keys and file positions use different path forms\n",
			path,
			len(ctx.FeatureProfiles),
		)
	})

	return ctx.FeatureProfile
}
