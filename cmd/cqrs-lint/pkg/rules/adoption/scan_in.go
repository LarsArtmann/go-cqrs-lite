package adoption

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// File-slice-scoped scan helpers.
//
// These mirror the ctx-based helpers (importsPath, projectHasCallAny, etc.)
// but scan an explicit file slice instead of the whole workspace. The ctx
// helpers delegate to these so all callers share one code path.

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
