package boilerplate

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B011: must-marshal helper.
// Detects helper functions that panic on marshal errors.
func NewB011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B011-must-marshal-helper",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name == nil {
						continue
					}

					name := fn.Name.Name
					if !strings.HasPrefix(name, "must") && !strings.HasPrefix(name, "Must") {
						continue
					}

					hasMarshal := false

					if fn.Body != nil {
						ast.Inspect(fn.Body, func(n ast.Node) bool {
							call, ok := n.(*ast.CallExpr)
							if !ok {
								return true
							}

							sel, ok := call.Fun.(*ast.SelectorExpr)
							if !ok {
								return true
							}

							if sel.Sel.Name == "Marshal" || sel.Sel.Name == "NewEvent" {
								hasMarshal = true

								return false
							}

							return true
						})
					}

					if !hasMarshal {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"B011", toolName,
						fmt.Sprintf("Function %s panics on marshal error — event.New already handles this", name),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Use event.New() which returns (*ImmutableEvent, error) — handle errors explicitly").
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

// B013: Missing correlation enricher.
// Detects bus/repository setups without WithCommandCausality.
func NewB013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B013-missing-correlation-enricher",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasCausality := false
			hasRepository := false

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					if sel.Sel.Name == "WithCommandCausality" ||
						sel.Sel.Name == "WithCorrelationID" {
						hasCausality = true
					}

					if sel.Sel.Name == "NewRepository" {
						hasRepository = true
					}

					return true
				})
			}

			if hasCausality || !hasRepository {
				return nil, nil
			}

			f, err := finding.NewBuilder(
				"B013", toolName,
				"Repository created without correlation enricher — command→event traceability is lost",
				finding.SeverityWarning,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("Use event.WithCommandCausality(ctx, cmdType, cmdID) in your decide function").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

// B014: Missing OTel middleware.
// Detects bus/dispatcher setups without tracing middleware.
func NewB014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B014-missing-otel-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasOTel := false
			hasMiddleware := false

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					if sel.Sel.Name == "EventTracing" || sel.Sel.Name == "CommandTracing" ||
						sel.Sel.Name == "NewOTelBundle" {
						hasOTel = true

						return false
					}

					if sel.Sel.Name == "Use" || sel.Sel.Name == "UsePublish" {
						hasMiddleware = true
					}

					return true
				})
			}

			if hasOTel || !hasMiddleware {
				return nil, nil
			}

			f, err := finding.NewBuilder(
				"B014", toolName,
				"Event bus / command dispatcher lacks OTel tracing middleware — no distributed tracing visibility",
				finding.SeverityInfo,
				finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Add middleware.NewOTelBundle(tracer, meter) and register Event()/Command()/Query() middleware").
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
