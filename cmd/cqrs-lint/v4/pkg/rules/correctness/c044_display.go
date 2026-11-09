package correctness

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

type displaySpan struct{ start, end token.Pos }

// displayArgRanges records the spans of arguments passed to display-context
// calls; a .String() anywhere inside such a span is display by construction,
// at any nesting depth (strings.ToUpper(s.String()) inside slog.Info).
func displayArgRanges(gf *analyzer.GoFile) []displaySpan {
	var spans []displaySpan

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isDisplayContextCall(gf, call) {
			return true
		}

		for _, arg := range call.Args {
			spans = append(spans, displaySpan{start: arg.Pos(), end: arg.End()})
		}

		return true
	})

	return spans
}

// isDisplayContextCall classifies one call as display (logging, structured
// attributes, message formatting, or an ID re-parse round-trip). Qualifier
// resolution is TYPED first (aliased imports and versioned module paths —
// …/id/v4 binds as `id`, which the import-table name match cannot see),
// with the syntactic import table as fallback for syntax-only contexts.
func isDisplayContextCall(gf *analyzer.GoFile, call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return displayContextCalleeNames[fun.Name]
	case *ast.SelectorExpr:
		if isIDReparseCall(gf, fun) {
			return true // documented round-trip law: ParseStreamID strips the prefix
		}

		if !displayContextCalleeNames[fun.Sel.Name] {
			return false
		}

		if path, resolved := qualifierImportPath(gf, fun); resolved {
			if path == "" {
				return true // method on a logger/formatting receiver
			}

			return isDisplayImportPath(path)
		}

		if path := importPathOf(gf.AST, fun); path != "" {
			return isDisplayImportPath(path)
		}

		return true
	default:
		return false
	}
}

// isIDReparseCall reports whether sel is an id-module Parse* constructor
// call — re-parsing a display-form string per the documented round-trip law.
func isIDReparseCall(gf *analyzer.GoFile, sel *ast.SelectorExpr) bool {
	if !strings.HasPrefix(sel.Sel.Name, "Parse") {
		return false
	}

	if path, resolved := qualifierImportPath(gf, sel); resolved {
		return analyzer.PathContainsFragment(path, "go-cqrs-lite/id")
	}

	return strings.Contains(importPathOf(gf.AST, sel), "go-cqrs-lite/id")
}

// qualifierImportPath resolves a selector's package qualifier through the
// type checker; resolved=false means type info is unavailable and the caller
// must fall back to the syntactic import table. A resolved EMPTY path means
// the identifier is a value (logger receiver), not a package.
func qualifierImportPath(gf *analyzer.GoFile, sel *ast.SelectorExpr) (path string, resolved bool) {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	return analyzer.ResolveQualifierTyped(gf, ident)
}

// isDisplayImportPath matches logging and formatting packages: stdlib log,
// log/slog (and vendored slog variants), fmt, and log-attribute packages
// (go.opentelemetry.io/otel/attribute and peers).
func isDisplayImportPath(path string) bool {
	last := path[strings.LastIndex(path, "/")+1:]

	if last == "attribute" {
		return true
	}

	return path == "log" || path == "fmt" ||
		strings.HasPrefix(path, "log/") ||
		strings.Contains(path, "/slog") ||
		strings.Contains(path, "/log") ||
		strings.Contains(path, "/fmt")
}

func resolveNamed(t types.Type) (*types.Named, bool) {
	if alias, ok := t.(*types.Alias); ok {
		t = types.Unalias(alias)
	}

	named, ok := t.(*types.Named)

	return named, ok
}

// importPathOf resolves the import path behind a selector's X identifier
// when it is a package name.
func importPathOf(file *ast.File, sel *ast.SelectorExpr) string {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			if imp.Name.Name == ident.Name {
				return path
			}

			continue
		}

		if path[strings.LastIndex(path, "/")+1:] == ident.Name {
			return path
		}
	}

	return ""
}
