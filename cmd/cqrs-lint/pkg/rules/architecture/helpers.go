package architecture

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// importsPathSuffix reports whether any non-test file imports a path containing
// suffix. Works with AST-level import declarations so it is testable via
// analyzer.BuildContextFromSource.
func importsPathSuffix(ctx *analyzer.AnalysisContext, suffix string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, imp := range gf.AST.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}

			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, suffix) {
				return true
			}
		}
	}

	return false
}

// projectCalls reports whether any non-test file calls pkgName.funcName.
func projectCalls(ctx *analyzer.AnalysisContext, pkgName, funcName string) bool {
	return projectCallsAny(ctx, pkgName, funcName)
}

// projectCallsAny reports whether any non-test file calls any of funcNames
// on pkgName.
func projectCallsAny(ctx *analyzer.AnalysisContext, pkgName string, funcNames ...string) bool {
	nameSet := make(map[string]bool, len(funcNames))
	for _, n := range funcNames {
		nameSet[n] = true
	}

	for _, gf := range ctx.GoFiles {
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

// findKeyBoolLit reports whether any non-test file contains a composite literal
// key-value pair where the key is keyName and the value is a boolean literal
// matching wantBool. Used to detect config flags like Enabled: false or
// BlockPublishUntilSubscriberAck: false.
func findKeyBoolLit(ctx *analyzer.AnalysisContext, keyName string, wantBool bool) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				ident, ok := kv.Key.(*ast.Ident)
				if !ok || ident.Name != keyName {
					continue
				}

				bid, ok := kv.Value.(*ast.Ident)
				if !ok {
					continue
				}

				if (wantBool && bid.Name == "true") || (!wantBool && bid.Name == "false") {
					found = true
					return false
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// countTypesWithSuffix counts non-test type declarations whose name ends with
// suffix. Used to detect adapter layers (E011) and dual-write buses (E012).
func countTypesWithSuffix(ctx *analyzer.AnalysisContext, suffix string) int {
	count := 0

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if strings.HasSuffix(ts.Name.Name, suffix) {
					count++
				}
			}
		}
	}

	return count
}

// typeExists reports whether any non-test file declares a type with the given
// name. Used to detect DualWrite types (E012).
func typeExists(ctx *analyzer.AnalysisContext, nameSubstring string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if strings.Contains(ts.Name.Name, nameSubstring) {
					return true
				}
			}
		}
	}

	return false
}

// firstCallPos returns the position of the first call to pkgName.funcName
// in any non-test file.
func firstCallPos(
	ctx *analyzer.AnalysisContext,
	pkgName, funcName string,
) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
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
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

// firstFilePos returns the package declaration position of the first non-test
// file. Used as an anchor for project-level findings without a specific call site.
func firstFilePos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if gf.AST.Package != token.NoPos {
			return ctx.Fset.Position(gf.AST.Package), true
		}
	}

	return token.Position{}, false
}

// singleFinding builds and returns a single finding with the common architecture
// defaults. Category is CategoryStructure, matching all existing E-series rules.
func singleFinding(
	ctx *analyzer.AnalysisContext,
	ruleID, message, suggestion string,
	pos token.Position,
	severity finding.Severity,
	confidence finding.Confidence,
) []finding.Finding {
	f, err := finding.NewBuilder(
		finding.RuleName(ruleID), toolName,
		message,
		severity,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(confidence).
		WithSuggestion(suggestion).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return nil
	}

	return []finding.Finding{f}
}
