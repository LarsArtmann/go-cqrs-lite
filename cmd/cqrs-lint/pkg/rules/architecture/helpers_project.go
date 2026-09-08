package architecture

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

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

// projectCallsMethodOnType scans for calls to any of methodNames on any
// receiver whose type path contains any of typePathSubstrs. Uses type info
// when available (real analysis); falls back to variable-name matching when
// type info is absent (unit tests with AST-only context).
//
// Example: projectCallsMethodOnType(ctx, []string{"Save", "Append"},
// []string{"go-cqrs-lite/event", "go-cqrs-lite/storage"}) matches
// `myStore.Save(...)` when myStore's type path contains "go-cqrs-lite/event"
// or "go-cqrs-lite/storage".
func projectCallsMethodOnType(
	ctx *analyzer.AnalysisContext,
	methodNames []string,
	typePathSubstrs []string,
) (token.Position, bool) {
	nameSet := make(map[string]bool, len(methodNames))
	for _, n := range methodNames {
		nameSet[n] = true
	}

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

			if !nameSet[sel.Sel.Name] {
				return true
			}

			if receiverTypeMatches(gf, sel.X, typePathSubstrs) {
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

// receiverTypeMatches checks whether the expression's type path contains any
// of the substrings. Uses go/types when available, falls back to variable-name
// heuristic otherwise.
func receiverTypeMatches(
	gf *analyzer.GoFile,
	expr ast.Expr,
	typePathSubstrs []string,
) bool {
	// Try type info first (real analysis path).
	if gf.Pkg != nil && gf.Pkg.TypesInfo != nil {
		if tv, ok := gf.Pkg.TypesInfo.Types[expr]; ok {
			typeStr := tv.Type.String()
			for _, sub := range typePathSubstrs {
				if strings.Contains(typeStr, sub) {
					return true
				}
			}
			// Type info available but type doesn't match — trust the type info.
			return false
		}
	}

	// Fallback: no type info (unit tests). Use variable-name heuristic.
	if ident, ok := expr.(*ast.Ident); ok {
		name := strings.ToLower(ident.Name)
		for _, sub := range typePathSubstrs {
			switch {
			case strings.Contains(sub, "event") || strings.Contains(sub, "storage"):
				if slices.Contains([]string{"store", "eventstore", "repo", "es"}, name) {
					return true
				}
			case strings.Contains(sub, "projectionhost"):
				if slices.Contains([]string{"host", "projectionhost", "proj"}, name) {
					return true
				}
			case strings.Contains(sub, "signing") || strings.Contains(sub, "encryption"):
				if slices.Contains([]string{"signer", "encryptor", "enc", "sign"}, name) {
					return true
				}
			}
		}
	}

	return false
}

// projectHasMethodCallContaining reports whether any non-test file has a call
// to a method whose name contains substr. This is a broad check used for
// suppression (e.g., any .Execute() call suppresses E010).
func projectHasMethodCallContaining(
	ctx *analyzer.AnalysisContext,
	substr string,
) bool {
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

			if strings.Contains(sel.Sel.Name, substr) {
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
