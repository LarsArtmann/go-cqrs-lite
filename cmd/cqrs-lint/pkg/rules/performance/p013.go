package performance

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// P013: Missing busy_timeout for SQLite.
// SQLite without busy_timeout throws "database is locked" errors under
// concurrent access. storage.SQLiteEnableWAL sets busy_timeout=5000ms.
// If WAL mode is already enabled (P012), busy_timeout is implicitly set.
//
//nolint:ireturn // factory returns public interface
func NewP013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P013-missing-sqlite-busy-timeout",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				// Only flag files that DIRECTLY open a SQLite connection.
				// See P012 for the full rationale on why constructor calls
				// (sqlite.New, NewSQLiteBackend, ...) are excluded.
				usesSQLite := directlyOpensSQLite(gf.AST)
				if !usesSQLite {
					continue
				}

				hasBusyTimeout := false

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					callStr := analyzer.ExprString(call.Fun)

					// SQLiteEnableWAL sets both WAL and busy_timeout.
					if strings.Contains(callStr, "SQLiteEnableWAL") ||
						strings.Contains(callStr, "busy_timeout") ||
						strings.Contains(callStr, "EnsureSQLiteDSNBusyTimeout") {
						hasBusyTimeout = true
					}

					return true
				})

				if !hasBusyTimeout {
					pos := ctx.Fset.Position(gf.AST.Pos())

					f, err := finding.NewBuilder(
						"P013", toolName,
						"SQLite store without busy_timeout — 'database is locked' errors under concurrent access",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryPerformance).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Call storage.SQLiteEnableWAL(ctx, db) or storage.EnsureSQLiteDSNBusyTimeout(dsn) after opening the database").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)
				}
			}

			return findings, nil
		},
	)
}
