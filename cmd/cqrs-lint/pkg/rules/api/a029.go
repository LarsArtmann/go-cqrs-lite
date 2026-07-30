package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// A029: bus.UsePublish is a stub returning nil.
// Detects a method named UsePublish whose body is just `return nil` — a
// stubbed middleware chain in a custom event.Bus implementation. The publish
// middleware chain is silently discarded, so signing, tracing, and encryption
// middleware applied via UsePublish are never executed.
//
//nolint:ireturn // factory returns public interface
func NewA029Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A029-usepublish-stub",
		func(_ context.Context) ([]finding.Finding, error) {
			if !projectImportsCQRS(ctx) {
				return nil, nil
			}

			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !lintutil.FileImportsCQRS(gf.AST) {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name == nil || fn.Name.Name != "UsePublish" {
						continue
					}

					if !isStubReturnNil(fn) {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"A029", toolName,
						"UsePublish is a stub returning nil — publish middleware chain is silently discarded",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Implement UsePublish to chain publish middleware, or use watermill.NewEventBus() which provides a working implementation").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						continue
					}

					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// isStubReturnNil returns true if the function body is exactly `return nil`.
func isStubReturnNil(fn *ast.FuncDecl) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}

	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return false
	}

	if len(ret.Results) != 1 {
		return false
	}

	ident, ok := ret.Results[0].(*ast.Ident)

	return ok && ident.Name == "nil"
}
