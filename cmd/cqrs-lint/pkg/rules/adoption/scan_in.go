package adoption

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// File-slice-scoped scan helpers.
//
// These are the base scanners over an explicit file slice. The ctx-based
// helpers (projectHasCallAny, firstCallPos, firstFuncDeclPos, ...) delegate
// to them so all callers share one code path.

// importsPathIn reports whether any non-test file imports a path matching one
// of the fragments. Matching tolerates Go major-version segments, so a
// version-less fragment ("…/metaengine") also matches
// "…/metaengine/v4/graphadapter" (see analyzer.PathContainsFragment).
func importsPathIn(files []*analyzer.GoFile, suffixes ...string) bool {
	for _, gf := range files {
		if gf.IsTest {
			continue
		}
		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}
			path := strings.Trim(imp.Path.Value, `"`)
			for _, suffix := range suffixes {
				if analyzer.PathContainsFragment(path, suffix) {
					return true
				}
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
// importLocalNames returns the local identifiers a project's imports of the
// given path fragments are bound to: the explicit alias when one is present,
// otherwise the path's last non-major-version segment.
//
// Detectors that look for `pkg.Func(...)` call shapes must use this instead of
// hardcoding the package name: an aliased import (`metaeng "…/metaengine/v4"`)
// otherwise evades the rule silently, and a rule that cannot see the code it
// coaches is worse than no rule.
func importLocalNames(files []*analyzer.GoFile, fragments ...string) map[string]bool {
	names := make(map[string]bool)

	for _, gf := range files {
		if gf.IsTest {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			path := strings.Trim(imp.Path.Value, `"`)

			matched := false
			for _, fragment := range fragments {
				if analyzer.PathContainsFragment(path, fragment) {
					matched = true

					break
				}
			}

			if !matched {
				continue
			}

			if imp.Name != nil {
				names[imp.Name.Name] = true

				continue
			}

			if name := packageNameFromPath(path); name != "" {
				names[name] = true
			}
		}
	}

	return names
}

// packageNameFromPath derives the default import identifier of a package path,
// skipping trailing major-version segments ("…/metaengine/v4" → "metaengine").
func packageNameFromPath(path string) string {
	segments := strings.Split(path, "/")

	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i] == "" || isMajorVersionSegment(segments[i]) {
			continue
		}

		return segments[i]
	}

	return ""
}

func isMajorVersionSegment(segment string) bool {
	if len(segment) < 2 || segment[0] != 'v' {
		return false
	}

	for i := 1; i < len(segment); i++ {
		if segment[i] < '0' || segment[i] > '9' {
			return false
		}
	}

	return true
}

// hasCallToLocal reports whether a non-test file calls one of the given local
// package identifiers with the function name.
func hasCallToLocal(files []*analyzer.GoFile, localNames map[string]bool, funcName string) bool {
	if len(localNames) == 0 {
		return false
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
			if !ok || sel.Sel.Name != funcName {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok || !localNames[ident.Name] {
				return true
			}

			found = true

			return false
		})

		if found {
			return true
		}
	}

	return false
}

// hasCallToNameAnywhere reports whether ANY non-test file calls `.Name(...)`,
// whatever the receiver is. Use it for APIs reachable both as `pkg.Name(...)`
// and as a method on a value (`reader.Name(...)`): the method form has no
// package identifier to match on.
func hasCallToNameAnywhere(files []*analyzer.GoFile, name string) bool {
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
			if !ok || sel.Sel.Name != name {
				return true
			}

			found = true

			return false
		})

		if found {
			return true
		}
	}

	return false
}
