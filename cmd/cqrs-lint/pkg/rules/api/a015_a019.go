package api

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A015: Global mutable state.
// Detects package-level var declarations that are mutated at runtime.
// Only flags globals that are actually written to after initialization
// (not read-only lookup tables initialized at package load).
func NewA015Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A015-global-mutable-state",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			candidates := collectGlobalMutables(ctx)

			for _, name := range candidates {
				if !isGlobalWrittenAfterInit(ctx, name.origName) {
					continue
				}

				f, err := finding.NewBuilder(
					"A015", toolName,
					fmt.Sprintf("Global mutable variable %s — may cause race conditions in concurrent handlers", name.origName),
					finding.SeverityError,
					finding.Pos(finding.FilePath(name.file), name.line, name.col),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion("Inject dependencies via constructor parameters or use sync.OnceValue for lazy init").
					WithSnippet(ctx.SourceLine(name.file, name.line)).
					Build()
				if err != nil {
					continue
				}

				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

type globalCandidate struct {
	origName string
	file     string
	line     int
	col      int
}

// collectGlobalMutables finds package-level var declarations whose names
// contain "cache", "registry", or "instance" (excluding Err/Sentinel errors).
func collectGlobalMutables(ctx *analyzer.AnalysisContext) []globalCandidate {
	var candidates []globalCandidate

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok.String() != "var" {
				continue
			}

			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) == 0 {
					continue
				}

				name := vs.Names[0].Name
				lower := strings.ToLower(name)

				if strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "Sentinel") {
					continue
				}

				if !strings.Contains(lower, "cache") &&
					!strings.Contains(lower, "registry") &&
					!strings.Contains(lower, "instance") {
					continue
				}

				pos := ctx.Fset.Position(vs.Pos())
				candidates = append(candidates, globalCandidate{
					origName: name,
					file:     pos.Filename,
					line:     pos.Line,
					col:      pos.Column,
				})
			}
		}
	}

	return candidates
}

// isGlobalWrittenAfterInit checks if a package-level variable is assigned to
// anywhere in the codebase inside a function body (excluding the declaration).
// Read-only globals (lookup tables initialized at package load and never
// written again) return false.
func isGlobalWrittenAfterInit(ctx *analyzer.AnalysisContext, varName string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var found bool

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			for _, lhs := range assign.Lhs {
				// Direct assignment: globalVar = newVal
				if id, ok := lhs.(*ast.Ident); ok && id.Name == varName {
					found = true
					return false
				}

				// Index assignment: globalVar[key] = val
				if idx, ok := lhs.(*ast.IndexExpr); ok {
					if ident, ok := idx.X.(*ast.Ident); ok && ident.Name == varName {
						found = true
						return false
					}
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

// A016: Missing idempotency middleware.
// Detects command dispatchers without idempotency middleware.
// Only flags when dispatchers are actually used for Dispatch() calls —
// read-only dispatchers (e.g., dashboards that never dispatch commands)
// are not at risk of duplicate execution.
func NewA016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A016-missing-idempotency-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasIdempotency := false
			hasDispatch := false

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

					if sel.Sel.Name == "CommandIdempotency" || sel.Sel.Name == "EventIdempotency" {
						hasIdempotency = true

						return false
					}

					if sel.Sel.Name == "Dispatch" {
						hasDispatch = true
					}

					return true
				})
			}

			if hasIdempotency || !hasDispatch {
				return nil, nil
			}

			hasDispatcher := false
			dispFile := ""
			dispLine := 0

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

					if sel.Sel.Name == "NewDispatcher" || sel.Sel.Name == "Use" {
						hasDispatcher = true
						p := ctx.Fset.Position(call.Pos())
						dispFile = p.Filename
						dispLine = p.Line

						return false
					}

					return true
				})
			}

			if !hasDispatcher {
				return nil, nil
			}

			f, err := finding.NewBuilder(
				"A016", toolName,
				"Command dispatcher lacks idempotency middleware — duplicate commands may execute twice",
				finding.SeverityWarning,
				finding.Pos(finding.FilePath(dispFile), dispLine, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Add middleware.CommandIdempotency(store, ttl, nil) to your dispatcher").
				WithSnippet(ctx.SourceLine(dispFile, dispLine)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A018: No actual event sourcing.
// Detects projects that import event/ but never call store.Save or bus.Publish.
func NewA018Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A018-no-actual-event-sourcing",
		func(_ context.Context) ([]finding.Finding, error) {
			hasSaveOrPublish := false

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

					if sel.Sel.Name == "Save" || sel.Sel.Name == "Publish" ||
						sel.Sel.Name == "AppendBatch" {
						hasSaveOrPublish = true

						return false
					}

					return true
				})
			}

			if hasSaveOrPublish || len(ctx.Registry.Folds) > 0 {
				return nil, nil
			}

			var findings []finding.Finding

			f, err := finding.NewBuilder(
				"A018",
				toolName,
				"Project imports go-cqrs-lite but never calls Save/Publish — possible dead import or missing wiring",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceHigh).
				WithSuggestion("Wire up an event store and bus, or remove the unused import").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// A019: Vendored cqrs.
// Detects vendored copies of go-cqrs-lite instead of proper module imports.
func NewA019Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A019-vendored-cqrs",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, pkg := range ctx.Packages {
				for _, imp := range pkg.Imports {
					if imp == nil {
						continue
					}

					if strings.Contains(imp.PkgPath, "vendor/") &&
						strings.Contains(imp.PkgPath, "cqrs") {
						f, err := finding.NewBuilder(
							"A019", toolName,
							"Vendored copy of go-cqrs-lite detected — update lag and missing bug fixes",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Remove vendor/ and use proper go.mod dependency for automatic updates").
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}
				}
			}

			return findings, nil
		},
	)
}
