package boilerplate

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects command/event dispatchers and buses with zero middleware. A panic
// in any handler without recovery middleware crashes the process.
//
// B023: command.Dispatcher with zero .Use() calls.
// B024: event.Bus / watermill bus without Recovery/EventRecovery middleware.
//
//nolint:ireturn // factory returns public interface
func NewB023Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B023-missing-command-middleware",
		func(_ context.Context) ([]finding.Finding, error) {
			return detectMissingMiddleware(ctx, "command", []string{"NewDispatcher"},
				"B023", "command dispatcher")
		},
	)
}

//nolint:ireturn // factory returns public interface
func NewB024Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B024-missing-bus-recovery",
		func(_ context.Context) ([]finding.Finding, error) {
			return detectMissingBusRecovery(ctx)
		},
	)
}

// detectMissingMiddleware checks whether a dispatcher created with the given
// constructor has any .Use() calls applied to it.
func detectMissingMiddleware(
	ctx *analyzer.AnalysisContext,
	pkgName string,
	constructors []string,
	ruleID, desc string,
) ([]finding.Finding, error) {
	var findings []finding.Finding

	// Collect dispatcher variable names and creation positions.
	type dispatcherInfo struct {
		varName string
		file    string
		line    int
		col     int
	}

	var dispatchers []dispatcherInfo
	hasAnyUse := make(map[string]bool) // var name → has .Use()

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			// Detect dispatcher creation: var d = command.NewDispatcher(...)
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			for i, lhs := range assign.Lhs {
				if i >= len(assign.Rhs) {
					break
				}

				ident, ok := lhs.(*ast.Ident)
				if !ok {
					continue
				}

				call, ok := assign.Rhs[i].(*ast.CallExpr)
				if !ok {
					continue
				}

				sel, ok := analyzer.SelectorFromExpr(call.Fun)
				if !ok {
					continue
				}

				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != pkgName {
					continue
				}

				found := false
				for _, ctor := range constructors {
					if sel.Sel.Name == ctor {
						found = true
						break
					}
				}

				if !found {
					continue
				}

				pos := ctx.Fset.Position(assign.Pos())
				dispatchers = append(dispatchers, dispatcherInfo{
					varName: ident.Name,
					file:    pos.Filename,
					line:    pos.Line,
					col:     pos.Column,
				})
			}

			return true
		})

		// Detect .Use() calls on the dispatchers.
		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Use" {
				return true
			}

			recv, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			hasAnyUse[recv.Name] = true
			return true
		})
	}

	for _, d := range dispatchers {
		if hasAnyUse[d.varName] {
			continue
		}

		// Also check common alias names.
		if hasAnyUse["cmdDisp"] || hasAnyUse["cmdDispatcher"] {
			continue
		}

		f, err := finding.NewBuilder(
			finding.RuleName(ruleID), toolName,
			desc+" has no middleware (.Use) — panics in handlers crash the process",
			finding.SeverityWarning,
			finding.Pos(finding.FilePath(d.file), d.line, d.col),
		).
			WithCategory(finding.CategoryBestPractice).
			WithConfidence(finding.ConfidenceMedium).
			WithFixStrategy(finding.FixStrategySuggest).
			WithSuggestion("Add middleware.CommandRecovery() and middleware.CommandLogging(logger) at minimum").
			WithSnippet(ctx.SourceLine(d.file, d.line)).
			Build()
		if err != nil {
			continue
		}

		findings = append(findings, f)
	}

	return findings, nil
}

func detectMissingBusRecovery(ctx *analyzer.AnalysisContext) ([]finding.Finding, error) {
	var findings []finding.Finding

	hasBusCreation := false
	var busPos struct {
		file string
		line int
		col  int
	}
	hasRecovery := false

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

			fnName := sel.Sel.Name

			// Detect event bus creation.
			if fnName == "NewEventBus" || fnName == "NewBus" {
				pkg := analyzer.SelectorPackage(sel)
				if pkg == "event" || pkg == "watermill" || pkg == "memory" {
					if !hasBusCreation {
						pos := ctx.Fset.Position(call.Pos())
						busPos.file = pos.Filename
						busPos.line = pos.Line
						busPos.col = pos.Column
						hasBusCreation = true
					}
				}
			}

			// Detect recovery middleware.
			if fnName == "EventRecovery" || fnName == "NewRecovery" || fnName == "Recovery" {
				pkg := analyzer.SelectorPackage(sel)
				if strings.Contains(pkg, "middleware") || pkg == "mw" {
					hasRecovery = true
				}
			}

			return true
		})
	}

	if hasBusCreation && !hasRecovery {
		f, err := finding.NewBuilder(
			"B024", toolName,
			"Event bus has no recovery middleware — panics in handlers crash the bus",
			finding.SeverityWarning,
			finding.Pos(finding.FilePath(busPos.file), busPos.line, busPos.col),
		).
			WithCategory(finding.CategoryBestPractice).
			WithConfidence(finding.ConfidenceMedium).
			WithFixStrategy(finding.FixStrategySuggest).
			WithSuggestion("Add middleware.EventRecovery() to the bus middleware chain").
			WithSnippet(ctx.SourceLine(busPos.file, busPos.line)).
			Build()
		if err == nil {
			findings = append(findings, f)
		}
	}

	return findings, nil
}
