package boilerplate

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// B032: command without tracing metadata.
// Detects command.New / command.NewPersistedCommand calls whose option list
// carries none of the tracing options (WithCorrelationID, WithCausationID,
// WithRequestID, WithCommandMetadata). Zero tracing metadata renders as
// 26-zero IDs on audit panels — the constructor is the only place it can be
// set for BasicCommand.
//
//nolint:ireturn // factory returns public interface
func NewB032Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B032-command-without-tracing-metadata",
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

					if !isCommandConstructor(call) {
						return true
					}

					// A spread forwards the caller's option list: whether tracing
					// reaches the command is decided at those call sites, not here.
					if call.Ellipsis.IsValid() || hasTracingOption(call) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := findingTemplate.Builder(
						"B032",
						fmt.Sprintf("command built without tracing metadata — audit panels render zero Correlation/Causation/Request IDs (%s)", call.Fun),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("pass command.WithCorrelationID (and WithRequestID), or WithCommandMetadata carrying them").
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

// isCommandConstructor reports whether the call targets command.New or
// command.NewPersistedCommand. The X ident must be the canonical `command`
// package name for New; NewPersistedCommand is distinctive enough to match
// on method name alone.
func isCommandConstructor(call *ast.CallExpr) bool {
	sel, ok := analyzer.SelectorFromExpr(call.Fun)
	if !ok {
		return false
	}

	if sel.Sel.Name == "NewPersistedCommand" {
		return true
	}

	if sel.Sel.Name != "New" {
		return false
	}

	x, ok := sel.X.(*ast.Ident)

	return ok && x.Name == "command"
}

// hasTracingOption reports whether any argument selects one of the options
// that put tracing IDs on the command or its persisted record.
func hasTracingOption(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		expr := arg

		if nested, ok := arg.(*ast.CallExpr); ok {
			expr = nested.Fun
		}

		sel, ok := analyzer.SelectorFromExpr(expr)
		if !ok {
			continue
		}

		switch sel.Sel.Name {
		case "WithCorrelationID", "WithCausationID", "WithRequestID", "WithCommandMetadata":
			return true
		}
	}

	return false
}

// B033: passive audit-store read without a writer.
// system.CommandStore() / system.QueryStore() are passive sinks — the
// library ships no writer. A module that reads them (to feed a journal or
// dashboard panel) while nothing in the module writes to them renders a
// permanently empty audit surface. Writers are detected by their
// distinctive calls: SaveQuery (query store), AppendBatch / NewPersistedCommand
// (command store or an auditor middleware).
//
//nolint:ireturn // factory returns public interface
func NewB033Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B033-passive-audit-store-read",
		func(_ context.Context) ([]finding.Finding, error) {
			var (
				readFile  string
				readLine  int
				readCol   int
				readName  string
				hasRead   bool
				hasWriter bool
			)

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

					switch sel.Sel.Name {
					case "CommandStore", "QueryStore":
						if !hasRead {
							pos := ctx.Fset.Position(call.Pos())
							readFile, readLine, readCol = pos.Filename, pos.Line, pos.Column
							readName = sel.Sel.Name
							hasRead = true
						}
					case "SaveQuery", "AppendBatch", "NewPersistedCommand":
						hasWriter = true
					}

					return true
				})
			}

			if !hasRead || hasWriter {
				return nil, nil
			}

			f, err := findingTemplate.Builder(
				"B033",
				readName+"() read but nothing in this module writes to the audit store — the panel or journal stays empty forever",
				finding.SeverityWarning,
				finding.Pos(finding.FilePath(readFile), readLine, readCol),
			).
				WithCategory(finding.CategoryBestPractice).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("add an auditor middleware that persists records (SaveQuery / NewPersistedCommand + Save), or stop reading the store").
				WithSnippet(ctx.SourceLine(readFile, readLine)).
				Build()
			if err != nil {
				return nil, err
			}

			return []finding.Finding{f}, nil
		},
	)
}
