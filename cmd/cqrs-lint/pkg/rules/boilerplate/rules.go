// Package boilerplate implements repetitive-code detection rules.
package boilerplate

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

const toolName finding.ToolName = "cqrs-lint"

// B001: Single-event helper pattern.
// Detects helper functions that wrap event.New/NewEvent to create a single event slice.
func NewB001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B001-single-event-helper",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			singleEventNames := []string{
				"singleEvent",
				"makeEvent",
				"mustEvent",
				"mustNewEvent",
				"createEvent",
				"oneEvent",
				"newEventSlice",
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name == nil {
						continue
					}

					for _, name := range singleEventNames {
						if strings.EqualFold(fn.Name.Name, name) ||
							strings.Contains(strings.ToLower(fn.Name.Name), strings.ToLower(name)) {
							// Verify it calls event.New or event.NewEvent.
							callsEventNew := false

							ast.Inspect(fn.Body, func(n ast.Node) bool {
								call, ok := n.(*ast.CallExpr)
								if !ok {
									return true
								}

								sel, ok := call.Fun.(*ast.SelectorExpr)
								if !ok {
									return true
								}

								pkgIdent, ok := sel.X.(*ast.Ident)
								if !ok || pkgIdent.Name != "event" {
									return true
								}

								if sel.Sel.Name == "New" || sel.Sel.Name == "NewEvent" {
									callsEventNew = true
								}

								return true
							})

							if callsEventNew {
								pos := ctx.Fset.Position(fn.Pos())

								f, err := finding.NewBuilder(
									"B001", toolName,
									fmt.Sprintf("Single-event helper %s — use event.Single() from the library instead", fn.Name.Name),
									finding.SeverityInfo,
									finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
								).
									WithCategory(finding.CategoryDuplication).
									WithConfidence(finding.ConfidenceHigh).
									WithSuggestion("Replace with event.Single(eventType, aggID, aggType, version, payload) which returns []Event").
									Build()
								if err == nil {
									findings = append(findings, f)
								}
							}
						}
					}
				}
			}

			return findings, nil
		},
	)
}

// B002: Manual repository wiring.
// Detects manual sequence of NewEventStore + NewEventBus + NewRepository calls.
func NewB002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B002-manual-repository-wiring",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				var callSequence []string

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						return true
					}

					callSequence = callSequence[:0]

					ast.Inspect(fn.Body, func(nn ast.Node) bool {
						call, ok := nn.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						callStr := analyzer.ExprString(call.Fun)
						callSequence = append(callSequence, callStr)
						_ = sel

						return true
					})

					if hasWiringSequence(callSequence) {
						pos := ctx.Fset.Position(fn.Pos())

						f, err := finding.NewBuilder(
							"B002", toolName,
							fmt.Sprintf("Function %s manually wires event store + bus + repository — use a stack preset (stack/sqlite, stack/pebble) instead", fn.Name.Name),
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryDuplication).
							WithConfidence(finding.ConfidenceMedium).
							WithSuggestion("Use stack/sqlite.New(dsn) or stack/pebble.New(dir) for one-call wiring of all stores").
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

// B003: SubscribeAll with large switch.
// Detects bus.SubscribeAll with >5 switch cases (should be separate projections).
func NewB003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B003-subscribeall-large-switch",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						continue
					}

					hasSubscribeAll := false

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						if sel.Sel.Name == "SubscribeAll" {
							hasSubscribeAll = true
						}

						return true
					})

					if !hasSubscribeAll {
						continue
					}
					// Count switch cases.
					caseCount := 0

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						sw, ok := n.(*ast.SwitchStmt)
						if !ok {
							return true
						}

						for _, stmt := range sw.Body.List {
							if _, ok := stmt.(*ast.CaseClause); ok {
								caseCount++
							}
						}

						return true
					})

					if caseCount > 5 {
						pos := ctx.Fset.Position(fn.Pos())

						f, err := finding.NewBuilder(
							"B003", toolName,
							fmt.Sprintf("SubscribeAll handler with %d switch cases — split into separate projections registered with projectionhost", caseCount),
							finding.SeverityInfo,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryDuplication).
							WithConfidence(finding.ConfidenceMedium).
							WithSuggestion("Register each event handler as a separate projection.Projection with projectionhost.Host").
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

func hasWiringSequence(calls []string) bool {
	hasEventStore := false
	hasBus := false
	hasRepo := false

	for _, c := range calls {
		if strings.Contains(c, "NewSQLEventStore") || strings.Contains(c, "NewEventStore") ||
			strings.Contains(c, "NewStore") {
			hasEventStore = true
		}

		if strings.Contains(c, "NewEventBus") || strings.Contains(c, "NewBus") {
			hasBus = true
		}

		if strings.Contains(c, "NewRepository") || strings.Contains(c, "newRepository") {
			hasRepo = true
		}
	}

	return hasEventStore && hasBus && hasRepo
}
