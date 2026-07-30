package correctness

import (
	"context"
	"go/ast"
	"slices"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// Detects memory.NewMemoryStore() used as a silent fallback when a store
// doesn't implement event.Journal. Projections replay from an empty journal
// with NO error or warning, silently losing all historical events.
//
// C018: Silent journal fallback to empty store.
//
//nolint:ireturn // factory returns public interface
func NewC018Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C018-silent-journal-fallback",
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

					if !funcHasJournalTypeAssert(fn) {
						continue
					}

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if !ok {
							return true
						}

						if sel.Sel.Name != "NewMemoryStore" {
							return true
						}

						pkg := analyzer.SelectorPackage(sel)
						if pkg != "memory" {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"C018", toolName,
							"memory.NewMemoryStore() used as journal fallback — "+
								"projections replay from empty journal with no error",
							finding.SeverityError,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceHigh).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Return an error when the store doesn't implement event.Journal, " +
								"or use a store that implements it (SQLite/Postgres/Pebble)").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)

						return true
					})
				}
			}

			return findings, nil
		},
	)
}

// funcHasJournalTypeAssert returns true if the function body contains a type
// assertion or type switch involving a "Journal" interface. This is the signal
// that the function is adapting a store to a journal interface.
func funcHasJournalTypeAssert(fn *ast.FuncDecl) bool {
	found := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}

		// Type switch: switch x := y.(type) { case event.Journal: ... }
		if sw, ok := n.(*ast.TypeSwitchStmt); ok {
			for _, s := range sw.Body.List {
				if cc, ok := s.(*ast.CaseClause); ok {
					if slices.ContainsFunc(cc.List, mentionsJournal) {
						found = true
						return false
					}
				}
			}
		}

		// Comma-ok type assertion: j, ok := store.(event.Journal)
		if as, ok := n.(*ast.TypeAssertExpr); ok {
			if mentionsJournal(as.Type) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

// mentionsJournal returns true if the type expression references a Journal
// interface (e.g., event.Journal, event.SeekableJournal, Journal).
func mentionsJournal(expr ast.Expr) bool {
	name := analyzer.ExprString(expr)
	return name == "event.Journal" ||
		name == "event.SeekableJournal" ||
		name == "Journal" ||
		name == "SeekableJournal"
}
