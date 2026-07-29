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
//
//nolint:ireturn // factory returns public interface
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
					hasPanic := false

					if fn.Body != nil {
						ast.Inspect(fn.Body, func(n ast.Node) bool {
							call, ok := n.(*ast.CallExpr)
							if !ok {
								return true
							}

							sel, ok := analyzer.SelectorFromExpr(call.Fun)
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

					// Confirm the helper actually panics; a must*-prefixed function that
					// returns the error instead of panicking is not the anti-pattern.
					if fn.Body != nil {
						ast.Inspect(fn.Body, func(n ast.Node) bool {
							call, ok := n.(*ast.CallExpr)
							if ok {
								if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "panic" {
									hasPanic = true
								}
							}

							return true
						})
					}

					if !hasPanic {
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

// B013: Missing correlation enricher.
// Detects bus/repository setups without WithCommandCausality.
//
//nolint:ireturn // factory returns public interface
func NewB013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B013-missing-correlation-enricher",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			hasCausality := false
			hasRepository := false
			repoFile := ""
			repoLine := 0

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

					if sel.Sel.Name == "WithCommandCausality" ||
						sel.Sel.Name == "WithCorrelationID" {
						hasCausality = true
					}

					if sel.Sel.Name == "NewRepository" {
						hasRepository = true
						p := ctx.Fset.Position(call.Pos())
						repoFile = p.Filename
						repoLine = p.Line
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
				finding.Pos(finding.FilePath(repoFile), repoLine, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("Use event.WithCommandCausality(ctx, cmdType, cmdID) in your decide function").
				WithSnippet(ctx.SourceLine(repoFile, repoLine)).
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
// Suppressed for local-only systems (no server) — distributed tracing adds
// overhead without value for single-user CLI tools.
//
//nolint:ireturn // factory returns public interface
func NewB014Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B014-missing-otel-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			if !ctx.FeatureProfile.HasServer {
				return nil, nil
			}

			var findings []finding.Finding

			hasOTel := false
			hasMiddleware := false
			mwFile := ""
			mwLine := 0

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

					if sel.Sel.Name == "EventTracing" || sel.Sel.Name == "CommandTracing" ||
						sel.Sel.Name == "NewOTelBundle" {
						hasOTel = true

						return false
					}

					if sel.Sel.Name == "Use" || sel.Sel.Name == "UsePublish" {
						hasMiddleware = true
						p := ctx.Fset.Position(call.Pos())
						mwFile = p.Filename
						mwLine = p.Line
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
				finding.Pos(finding.FilePath(mwFile), mwLine, 1),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion("Add middleware.NewOTelBundle(tracer, meter) and register Event()/Command()/Query() middleware").
				WithSnippet(ctx.SourceLine(mwFile, mwLine)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}
