package architecture

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

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

// findKeyBoolLitInTypedComposite reports whether any non-test file contains a
// composite literal key-value pair where the key is keyName, the value is a
// boolean literal matching wantBool, AND the composite literal's type name
// contains one of typeSubstrings. This prevents false positives where an
// unrelated struct happens to have the same key name (e.g., Enabled: false
// in a feature-flag config vs a signing/encryption config).
func findKeyBoolLitInTypedComposite(
	ctx *analyzer.AnalysisContext,
	keyName string,
	wantBool bool,
	typeSubstrings ...string,
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

			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			typeName := compositeTypeName(cl.Type)
			if typeName == "" {
				return true
			}

			lowerType := strings.ToLower(typeName)
			matchesType := false
			for _, sub := range typeSubstrings {
				if strings.Contains(lowerType, strings.ToLower(sub)) {
					matchesType = true
					break
				}
			}

			if !matchesType {
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

// compositeTypeName extracts the type name from a composite literal type
// expression (e.g., "signing.Config" from *ast.CompositeLit.Type).
func compositeTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.StarExpr:
		return compositeTypeName(t.X)
	case *ast.ArrayType:
		return compositeTypeName(t.Elt)
	}

	return ""
}

// firstKeyBoolPosInTypedComposite returns the position of the first composite
// literal key-value pair matching keyName/wantBool within a typed composite
// literal whose type contains one of typeSubstrings.
func firstKeyBoolPosInTypedComposite(
	ctx *analyzer.AnalysisContext,
	keyName string,
	wantBool bool,
	typeSubstrings ...string,
) (token.Position, bool) {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit ast.Node

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}

			typeName := compositeTypeName(cl.Type)
			if typeName == "" {
				return true
			}

			lowerType := strings.ToLower(typeName)
			matchesType := false
			for _, sub := range typeSubstrings {
				if strings.Contains(lowerType, strings.ToLower(sub)) {
					matchesType = true
					break
				}
			}

			if !matchesType {
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
					hit = kv
					return false
				}
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}
