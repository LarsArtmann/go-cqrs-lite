package adoption

// Import-path-scoped call detection: resolves a qualifier through type info
// when available, so aliased imports match by path instead of local name.

import (
	"go/ast"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

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
