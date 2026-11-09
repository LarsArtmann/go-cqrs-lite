package adoption

// Metaengine-native capability detection for the coaching rules.

import (
	"go/ast"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

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
