package api

import (
	"go/ast"
	"go/token"
	"path/filepath"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// dispatcherModulePaths are the go-cqrs-lite modules owning a Dispatcher with
// middleware. `.Use` is a method there (command/query/dispatcher
// Dispatcher.Use), but also on chi/gorilla routers, loggers and many other
// types — the name alone says nothing.
//
//nolint:gochecknoglobals // read-only lookup table
var dispatcherModulePaths = []string{"go-cqrs-lite/dispatcher", "go-cqrs-lite/command", "go-cqrs-lite/query"}

// isDispatcherWiring reports whether call is `pkg.NewDispatcher(...)` or a
// `.Use(...)` on a go-cqrs-lite dispatcher: the qualifier is a package of a
// dispatcher module (alias-aware through the import table), or the receiver's
// type — from go/types when loaded — lives in one. On a receiver variable
// without type info (parse-only loads, unit tests) the file must assign a
// variable from a go-cqrs-lite NewDispatcher for `.Use` to count.
func isDispatcherWiring(gf *analyzer.GoFile, call *ast.CallExpr) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	ident, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return false
	}

	switch sel.Sel.Name {
	case "NewDispatcher":
		return qualifierTargetsDispatcher(gf, ident)
	case "Use":
		return useTargetsDispatcher(gf, sel)
	default:
		return false
	}
}

func useTargetsDispatcher(gf *analyzer.GoFile, sel *ast.SelectorExpr) bool {
	if gf.Pkg != nil && gf.Pkg.TypesInfo != nil {
		if selection, ok := gf.Pkg.TypesInfo.Selections[sel]; ok {
			return typeInDispatcherModule(selection.Recv().String())
		}

		// A package qualifier (never a method receiver): `pkg.Use` is not a
		// go-cqrs-lite call.
		if _, isVar := gf.Pkg.TypesInfo.Uses[sel.X.(*ast.Ident)]; !isVar {
			return false
		}
	} else if qualifierTargetsDispatcher(gf, sel.X.(*ast.Ident)) {
		return false
	}

	return fileAssignsLiteDispatcher(gf)
}

func qualifierTargetsDispatcher(gf *analyzer.GoFile, ident *ast.Ident) bool {
	for _, module := range dispatcherModulePaths {
		if lintutil.QualifierTargetsModule(gf, ident, module) {
			return true
		}
	}

	return false
}

// typeInDispatcherModule reports whether a go/types type string names a type
// from a dispatcher module (paths carry the /vN suffix in type strings).
func typeInDispatcherModule(typeStr string) bool {
	for _, module := range dispatcherModulePaths {
		if analyzer.PathContainsFragment(typeStr, module) {
			return true
		}
	}

	return false
}

// fileAssignsLiteDispatcher reports whether the file assigns a variable from
// a NewDispatcher call whose qualifier targets a dispatcher module.
func fileAssignsLiteDispatcher(gf *analyzer.GoFile) bool {
	found := false

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || found {
			return !found
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok || sel.Sel.Name != "NewDispatcher" {
			return true
		}

		if ident, ok := sel.X.(*ast.Ident); ok && qualifierTargetsDispatcher(gf, ident) {
			found = true
		}

		return true
	})

	return found
}

// dispatcherSite is one go-cqrs-lite dispatcher wiring position. Findings are
// emitted per module (first site wins), mirroring the E009/E016 grouping.
type dispatcherSite struct {
	module string
	pos    token.Position
}

type dispatcherSites struct {
	seen  map[string]bool
	sites []dispatcherSite
}

func newDispatcherSites() *dispatcherSites {
	return &dispatcherSites{seen: map[string]bool{}}
}

func (d *dispatcherSites) add(ctx *analyzer.AnalysisContext, gf *analyzer.GoFile, call *ast.CallExpr) {
	pos := ctx.Fset.Position(call.Pos())

	// ModuleDir is empty on hand-built (unit-test) contexts; the file's
	// directory is the equivalent key there.
	module := gf.ModuleDir
	if module == "" {
		module = filepath.Dir(pos.Filename)
	}

	if d.seen[module] {
		return
	}

	d.seen[module] = true
	d.sites = append(d.sites, dispatcherSite{module: module, pos: pos})
}

func a016Finding(ctx *analyzer.AnalysisContext, site dispatcherSite) (finding.Finding, error) {
	return findingTemplate.Builder(
		"A016",
		"Command dispatcher lacks idempotency middleware — duplicate commands may execute twice",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(site.pos.Filename), site.pos.Line, site.pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(finding.ConfidenceLow).
		WithSuggestion("Add middleware.CommandIdempotency(store, ttl, nil) to your dispatcher").
		WithSnippet(ctx.SourceLine(site.pos.Filename, site.pos.Line)).
		Build()
}
