package adoption

import (
	"go/ast"
	"go/token"
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

// --- File-slice-scoped scan helpers ---
//
// These mirror the ctx-based helpers (importsPath, projectHasCallAny, etc.)
// but scan an explicit file slice instead of the whole workspace. The ctx
// helpers delegate to these so all callers share one code path.

func importsPathIn(files []*analyzer.GoFile, suffix string) bool {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}
		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}
			if strings.Contains(strings.Trim(imp.Path.Value, `"`), suffix) {
				return true
			}
		}
	}

	return false
}

func hasCallIn(files []*analyzer.GoFile, pkgName string, funcNames ...string) bool {
	nameSet := make(map[string]bool, len(funcNames))
	for _, n := range funcNames {
		nameSet[n] = true
	}

	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != pkgName {
				return true
			}

			if nameSet[sel.Sel.Name] {
				found = true
				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

func hasSelectorIn(files []*analyzer.GoFile, pkgName, selName string) bool {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			if pkg.Name == pkgName && sel.Sel.Name == selName {
				found = true
				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

func firstCallPosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
	pkgName, funcName string,
) (token.Position, bool) {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		var hit *ast.CallExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != pkgName {
				return true
			}

			if sel.Sel.Name == funcName {
				hit = call
				return false
			}

			return true
		})

		if hit != nil {
			return fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

func firstFilePosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
) (token.Position, bool) {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		if gf.AST.Package != token.NoPos {
			return fset.Position(gf.AST.Package), true
		}
	}

	return token.Position{}, false
}

func firstFuncDeclPosIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
	prefix string,
) (token.Position, bool) {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if strings.HasPrefix(fn.Name.Name, prefix) {
				return fset.Position(fn.Pos()), true
			}
		}
	}

	return token.Position{}, false
}

func firstCallByNameIn(
	fset *token.FileSet,
	files []*analyzer.GoFile,
	funcName string,
) (token.Position, bool) {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		var hit *ast.CallExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if sel.Sel.Name == funcName {
				hit = call
				return false
			}

			return true
		})

		if hit != nil {
			return fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

func countCallsIn(files []*analyzer.GoFile, pkgName, funcName string) int {
	count := 0

	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		astInspectCalls(gf.AST, func(call *ast.CallExpr) bool {
			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != pkgName {
				return true
			}

			if sel.Sel.Name == funcName {
				count++
			}

			return true
		})
	}

	return count
}
