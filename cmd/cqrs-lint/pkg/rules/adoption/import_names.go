package adoption

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

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

	for i := 1; i < len(segment); i++ { //art-dupl:accept digit scan, not a rule-ID validator
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
