package boilerplate

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B009: Emit function boilerplate.
// Detects hand-written emit/publish helper functions that wrap event.New + bus.Publish.
func NewB009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B009-emit-function-boilerplate",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil || fn.Name == nil {
						continue
					}

					name := strings.ToLower(fn.Name.Name)
					if !strings.HasPrefix(name, "emit") && !strings.HasPrefix(name, "publish") {
						continue
					}

					hasNewEvent := false
					hasPublish := false

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						if sel.Sel.Name == "New" || sel.Sel.Name == "NewEvent" {
							pkg := analyzer.SelectorPackage(sel)
							if pkg == "event" {
								hasNewEvent = true
							}
						}

						if sel.Sel.Name == "Publish" || sel.Sel.Name == "Save" ||
							sel.Sel.Name == "AppendBatch" {
							hasPublish = true
						}

						return true
					})

					if !hasNewEvent || !hasPublish {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"B009",
						toolName,
						fmt.Sprintf(
							"Function %s wraps event creation + publish — consider code generation with cqrs-gen",
							fn.Name.Name,
						),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Use cqrs-gen to generate typed emit functions from struct tags").
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

// B010: Catalog event list boilerplate.
// Detects 3+ catalog.Event() calls in the same function — could be generated.
func NewB010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B010-catalog-event-list-boilerplate",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						return true
					}

					catalogCallCount := 0

					var firstPos token.Position

					ast.Inspect(fn.Body, func(nn ast.Node) bool {
						call, ok := nn.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						if sel.Sel.Name == "Event" && analyzer.SelectorPackage(sel) == "catalog" {
							if catalogCallCount == 0 {
								firstPos = ctx.Fset.Position(call.Pos())
							}

							catalogCallCount++
						}

						return true
					})

					if catalogCallCount < 3 {
						return true
					}

					pos := firstPos

					f, err := finding.NewBuilder(
						"B010",
						toolName,
						fmt.Sprintf(
							"%d catalog.Event calls in %s — consider cqrs-gen to auto-generate from struct tags",
							catalogCallCount,
							fn.Name.Name,
						),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Run cqrs-gen to auto-generate catalog registrations from Go struct types").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						return true
					}

					findings = append(findings, f)

					return true
				})
			}

			return findings, nil
		},
	)
}

// B012: Make-event helper.
// Detects hand-written makeEvent/newEvent helper functions that should use event.New.
func NewB012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B012-make-event-helper",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil || fn.Name == nil {
						continue
					}

					name := strings.ToLower(fn.Name.Name)
					if !strings.HasPrefix(name, "makeevent") &&
						!strings.HasPrefix(name, "newevent") &&
						!strings.HasPrefix(name, "make_event") {
						continue
					}

					hasMarshal := false

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						if sel.Sel.Name == "Marshal" {
							hasMarshal = true

							return false
						}

						return true
					})

					if !hasMarshal {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"B012", toolName,
						fmt.Sprintf("Function %s manually constructs events — event.New auto-marshals payloads", fn.Name.Name),
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Use event.New(eventType, aggID, aggType, version, payload) which handles marshaling automatically").
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
