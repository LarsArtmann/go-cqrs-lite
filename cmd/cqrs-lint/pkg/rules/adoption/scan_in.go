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
				if strings.Contains(path, suffix) {
					return true
				}
			}
		}
	}

	return false
}

// usesMetaengineGraph reports whether the files import metaengine and
// reference the native Graph ADT. Edge and EdgeRemoval folds are the supported
// metaengine way to project relations, so a consumer using them already has a
// first-party graph facility and F010 must stay silent.
func usesMetaengineGraph(files []*analyzer.GoFile) bool {
	if !importsPathIn(files, "go-cqrs-lite/metaengine") {
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

			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if sel.Sel.Name == "Edge" || sel.Sel.Name == "EdgeRemoval" {
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

// hasCallOnImportedPackage reports whether any non-test file calls one of
// funcNames on a qualifier whose import path contains pathFragment. Package
// aliases are resolved through type info when available (IsQualifierFor).
func hasCallOnImportedPackage(
	files []*analyzer.GoFile,
	pathFragment string,
	funcNames ...string,
) bool {
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

			if !analyzer.IsQualifierFor(gf, sel, pathFragment) {
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
