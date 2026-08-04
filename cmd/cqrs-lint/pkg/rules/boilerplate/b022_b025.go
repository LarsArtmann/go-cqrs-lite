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
// The detector traces through option-builder helpers — including those in
// OTHER packages within the analyzed workspace. When NewRepository receives
// a variadic spread from a helper call (e.g. wiring.repositoryOptions(cfg)...),
// the detector resolves the helper's package via the import graph and inspects
// its body for a WithStateCache call. This eliminates false positives for
// codebases with shared wiring packages that do not directly import CQRS.
//
//nolint:ireturn // factory returns public interface
func NewB025Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	fnIndex := buildCrossPkgFuncIndex(ctx)

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
					// repo := decider.NewRepository(s, b, d, wiring.repositoryOptions(cfg)...)),
					// resolve the helper's package and inspect its body for a
					// WithStateCache call. This recognizes indirect wiring in both
					// same-package and cross-package helpers — the common pattern
					// in codebases with reusable repository-wiring packages.
					if !hasStateCache {
						if helperName, pkgAlias := spreadHelperInfo(call); helperName != "" {
							pkgPath := ""
							if pkgAlias != "" {
								pkgPath = resolveImportPath(gf.AST, pkgAlias)
							}

							if fnIndex.containsOption(pkgPath, helperName, "WithStateCache") {
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

// funcIndex is a cross-package index of top-level function declarations.
// It enables B025 to trace through helper functions that live in a different
// package than the call site — the common pattern in consumer codebases with
// shared wiring packages (e.g. myapp/wiring.repositoryOptions).
type funcIndex struct {
	// byPkgFunc maps "pkgPath\x00funcName" to declarations — precise cross-
	// package lookup when the import alias is resolved.
	byPkgFunc map[string][]*ast.FuncDecl
	// byName maps bare funcName to declarations — fallback for same-package
	// lookups and test contexts where import resolution is unavailable.
	byName map[string][]*ast.FuncDecl
}

// buildCrossPkgFuncIndex indexes all top-level function declarations across
// every analyzed package — including non-CQRS packages that do not appear in
// ctx.GoFiles. This closes the B025 cross-package helper gap: a wiring helper
// in myapp/wiring (which does not import CQRS directly) is now visible because
// packages.Load already parsed its syntax; it just was not being indexed.
func buildCrossPkgFuncIndex(ctx *analyzer.AnalysisContext) *funcIndex {
	idx := &funcIndex{
		byPkgFunc: map[string][]*ast.FuncDecl{},
		byName:    map[string][]*ast.FuncDecl{},
	}

	seen := map[string]bool{} // dedup by file path

	// Pass 1: ctx.GoFiles (CQRS-importing packages — scanned and registered).
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if gf.Path != "" {
			seen[gf.Path] = true
		}

		pkgPath := ""
		if gf.Pkg != nil {
			pkgPath = gf.Pkg.PkgPath
		}

		idx.addFile(pkgPath, gf.AST)
	}

	// Pass 2: ctx.Packages — ALL loaded packages with syntax, including
	// non-CQRS packages whose files are not in ctx.GoFiles.
	for _, pkg := range ctx.Packages {
		if pkg == nil {
			continue
		}

		for i, syntax := range pkg.Syntax {
			if syntax == nil {
				continue
			}

			var path string
			if i < len(pkg.GoFiles) {
				path = pkg.GoFiles[i]
			}

			if path != "" && seen[path] {
				continue // already indexed from GoFiles
			}

			if path != "" {
				seen[path] = true
			}

			idx.addFile(pkg.PkgPath, syntax)
		}
	}

	return idx
}

func (idx *funcIndex) addFile(pkgPath string, file *ast.File) {
	if file == nil {
		return
	}

	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil || fd.Name == nil {
			continue
		}

		name := fd.Name.Name
		idx.byName[name] = append(idx.byName[name], fd)

		if pkgPath != "" {
			key := pkgPath + "\x00" + name
			idx.byPkgFunc[key] = append(idx.byPkgFunc[key], fd)
		}
	}
}

// containsOption reports whether any function matching (pkgPath, funcName) in
// the index contains a call to the named option (matched by selector or bare
// identifier, e.g. "WithStateCache"). When pkgPath is non-empty, it first
// tries a precise lookup; it always falls back to a bare-name search across
// all packages. This is intentionally conservative: it only suppresses a
// finding when the helper visibly constructs the option, never the reverse.
func (idx *funcIndex) containsOption(pkgPath, funcName, option string) bool {
	if pkgPath != "" {
		key := pkgPath + "\x00" + funcName
		for _, fd := range idx.byPkgFunc[key] {
			if funcDeclCallsOption(fd, option) {
				return true
			}
		}
	}

	for _, fd := range idx.byName[funcName] {
		if funcDeclCallsOption(fd, option) {
			return true
		}
	}

	return false
}

// spreadHelperInfo extracts the function name and package qualifier from a
// variadic spread argument. Given:
//
//	decider.NewRepository(s, b, d, wiring.repositoryOptions(cfg)...)
//
// it returns ("repositoryOptions", "wiring"). For a bare same-package call:
//
//	decider.NewRepository(s, b, d, repositoryOptions[State](cfg)...)
//
// it returns ("repositoryOptions", ""). Returns ("", "") when there is no
// variadic spread or the spread source is not a function call.
func spreadHelperInfo(call *ast.CallExpr) (funcName, pkgAlias string) {
	if !call.Ellipsis.IsValid() || len(call.Args) == 0 {
		return "", ""
	}

	helperCall, ok := call.Args[len(call.Args)-1].(*ast.CallExpr)
	if !ok {
		return "", ""
	}

	return callNameAndQualifier(helperCall.Fun)
}

// callNameAndQualifier extracts the function name and optional package
// qualifier from a call expression's Fun node, handling generic instantiations
// (foo[T]() and pkg.foo[T]()).
func callNameAndQualifier(fun ast.Expr) (name, qualifier string) {
	switch e := fun.(type) {
	case *ast.Ident:
		return e.Name, ""
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok {
			return e.Sel.Name, ident.Name
		}

		return e.Sel.Name, ""
	case *ast.IndexExpr: // single type-parameter: foo[T]()
		return callNameAndQualifier(e.X)
	case *ast.IndexListExpr: // multi type-parameter: foo[T, U]()
		return callNameAndQualifier(e.X)
	default:
		return "", ""
	}
}

// resolveImportPath resolves an import alias to the full package import path
// using the file's import declarations. Handles both aliased imports
// (import w "myapp/wiring") and path-derived aliases (import "myapp/wiring").
// Returns "" when the alias does not match any import.
func resolveImportPath(file *ast.File, alias string) string {
	if file == nil || alias == "" {
		return ""
	}

	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil {
			if imp.Name.Name == alias {
				return path
			}

			continue
		}

		// No explicit alias: derive from the last path segment.
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[len(parts)-1] == alias {
			return path
		}
	}

	return ""
}

// funcDeclCallsOption reports whether a function declaration's body contains a
// call to the named option (matched by selector or bare identifier). This is a
// shallow textual match; it does not resolve types.
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
