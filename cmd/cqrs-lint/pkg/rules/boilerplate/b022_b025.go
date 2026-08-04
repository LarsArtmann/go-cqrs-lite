package boilerplate

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B022: Manual correlation enricher instead of CommandCausalityEnricher.
// Detects custom enricher functions passed to decider.NewRepository that are
// not event.CommandCausalityEnricher. Custom enrichers miss the typed
// command causality metadata that CommandCausalityEnricher provides.
//
//nolint:ireturn // factory returns public interface
func NewB022Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B022-manual-correlation-enricher",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					// Find NewRepository calls from the decider package.
					if sel.Sel.Name != "NewRepository" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "decider" {
						return true
					}

					// Scan arguments for enricher calls that are not
					// CommandCausalityEnricher.
					for _, arg := range call.Args {
						argCall, ok := arg.(*ast.CallExpr)
						if !ok {
							continue
						}

						argSel, ok := analyzer.SelectorFromExpr(argCall.Fun)
						if !ok {
							continue
						}

						argName := argSel.Sel.Name
						if !containsEnricher(argName) {
							continue
						}

						// CommandCausalityEnricher is the recommended enricher.
						if argName == "CommandCausalityEnricher" {
							continue
						}

						// WithCommandCausality is also acceptable (the option
						// form that wraps the enricher).
						if argName == "WithCommandCausality" {
							continue
						}

						// WithEnricher(event.CommandCausalityEnricher) wraps the
						// canonical enricher — not a custom one. Inspect its
						// arguments to avoid flagging the recommended pattern.
						if argName == "WithEnricher" && wrapsCanonicalEnricher(argCall) {
							continue
						}

						pos := ctx.Fset.Position(argCall.Pos())

						f, err := finding.NewBuilder(
							"B022", toolName,
							"Custom enricher ("+argName+") passed to decider.NewRepository — "+
								"use event.CommandCausalityEnricher for typed command causality",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Replace the custom enricher with event.CommandCausalityEnricher — " +
								"it stamps metadata.command.type and metadata.command.id on every event").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// wrapsCanonicalEnricher returns true when a WithEnricher call receives
// event.CommandCausalityEnricher (or WithCommandCausality) directly as one
// of its arguments, meaning the enricher is NOT custom.
func wrapsCanonicalEnricher(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		sel, ok := analyzer.SelectorFromExpr(arg)
		if !ok {
			continue
		}

		if sel.Sel.Name == "CommandCausalityEnricher" || sel.Sel.Name == "WithCommandCausality" {
			return true
		}
	}

	return false
}

func containsEnricher(s string) bool {
	return strings.Contains(strings.ToLower(s), "enrich")
}

// B025: Missing state cache on repository.
// Detects decider.NewRepository calls without the WithStateCache option.
// For hot streams, incremental loads via state cache are 7.4x faster.
//
//nolint:ireturn // factory returns public interface
func NewB025Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	// Build a name → declarations index of all top-level functions in the
	// analyzed packages. Consumers commonly construct repository options inside
	// a helper (e.g. repositoryOptions[State](cfg)...) that the call-site scan
	// cannot see through. This index lets the detector trace into such helpers
	// and recognize an indirect WithStateCache wiring, eliminating a frequent
	// false positive for libraries with reusable wiring.
	funcDeclsByName := indexFuncDecls(ctx)

	return finding.NamedDetectorFunc(
		"B025-missing-state-cache",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					// Find NewRepository calls from the decider package.
					if sel.Sel.Name != "NewRepository" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "decider" {
						return true
					}

					// Check if WithStateCache is among the arguments.
					hasStateCache := false
					for _, arg := range call.Args {
						// Options are typically call expressions like
						// decider.WithStateCache(cache) or
						// decider.WithStateCache[State](cache).
						var argSel *ast.SelectorExpr

						if argCall, ok := arg.(*ast.CallExpr); ok {
							argSel, _ = analyzer.SelectorFromExpr(argCall.Fun)
						} else {
							argSel, _ = analyzer.SelectorFromExpr(arg)
						}

						if argSel == nil {
							continue
						}

						if argSel.Sel.Name == "WithStateCache" {
							hasStateCache = true
							break
						}
					}

					// Trace through an option-builder helper. When NewRepository
					// receives a variadic spread from a function call (e.g.
					// repo := decider.NewRepository(s, b, d, repositoryOptions[State](cfg)...)),
					// inspect that helper's body for a WithStateCache call. This
					// recognizes indirect wiring that the direct argument scan
					// above cannot see, which is the common pattern in libraries
					// with reusable repository-wiring helpers.
					if !hasStateCache {
						if helper := spreadHelperName(call); helper != "" {
							if funcBodyContainsCall(funcDeclsByName, helper, "WithStateCache") {
								hasStateCache = true
							}
						}
					}

					if hasStateCache {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"B025", toolName,
						"Repository created without decider.WithStateCache — "+
							"hot streams benefit from incremental loads (7.4x faster)",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceLow).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Add decider.WithStateCache(decider.NewStateCache[State](256)) to " +
							"NewRepository options for incremental event loading on hot streams").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// indexFuncDecls builds a name → declarations index of all top-level function
// declarations across the analyzed (non-test) Go files. Used by detectors that
// need to trace a helper function's body for a specific call (e.g. B025 looking
// for an indirect WithStateCache wiring).
func indexFuncDecls(ctx *analyzer.AnalysisContext) map[string][]*ast.FuncDecl {
	index := map[string][]*ast.FuncDecl{}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}

			index[fd.Name.Name] = append(index[fd.Name.Name], fd)
		}
	}

	return index
}

// spreadHelperName returns the function name being invoked when the last
// argument of call is a variadic spread from a function call. For example,
// given:
//
//	decider.NewRepository(s, b, d, repositoryOptions[State](cfg)...)
//
// it returns "repositoryOptions". Returns "" when there is no variadic spread
// or the spread source is not a function call (e.g. a bare identifier or
// composite literal that cannot be traced).
func spreadHelperName(call *ast.CallExpr) string {
	if !call.Ellipsis.IsValid() || len(call.Args) == 0 {
		return ""
	}

	helperCall, ok := call.Args[len(call.Args)-1].(*ast.CallExpr)
	if !ok {
		return ""
	}

	return callFunctionName(helperCall.Fun)
}

// callFunctionName extracts the name of the function being called, handling
// bare calls (foo()), selector calls (pkg.foo()), and generic instantiations
// (foo[T]() and pkg.foo[T]()). Returns "" when the name cannot be determined.
func callFunctionName(fun ast.Expr) string {
	switch e := fun.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.IndexExpr: // single type-parameter instantiation: foo[T]()
		return callFunctionName(e.X)
	case *ast.IndexListExpr: // multi type-parameter instantiation: foo[T, U]()
		return callFunctionName(e.X)
	default:
		return ""
	}
}

// funcBodyContainsCall reports whether any top-level function named funcName in
// the index contains a call to the named option (matched by selector or bare
// identifier, e.g. "WithStateCache"). This is a shallow textual-ish match on
// the call name; it does not resolve types or cross package boundaries, so it
// is intentionally conservative: it only suppresses a finding when the helper
// visibly constructs the option, never the reverse.
func funcBodyContainsCall(index map[string][]*ast.FuncDecl, funcName, option string) bool {
	for _, fd := range index[funcName] {
		if funcDeclCallsOption(fd, option) {
			return true
		}
	}

	return false
}

func funcDeclCallsOption(fd *ast.FuncDecl, option string) bool {
	found := false

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if sel, ok := analyzer.SelectorFromExpr(call.Fun); ok && sel.Sel.Name == option {
			found = true
			return false
		}

		if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == option {
			found = true
			return false
		}

		return true
	})

	return found
}
