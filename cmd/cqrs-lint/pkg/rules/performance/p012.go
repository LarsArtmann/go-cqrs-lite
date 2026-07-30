package performance

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// P012: Missing WAL mode for SQLite.
// SQLite stores without WAL mode are prone to "database is locked" errors
// under concurrent access. storage.SQLiteEnableWAL enables WAL + busy_timeout.
//
//nolint:ireturn // factory returns public interface
func NewP012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P012-missing-sqlite-wal",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				usesSQLite := false
				hasWAL := false

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					callStr := analyzer.ExprString(call.Fun)

					if strings.Contains(callStr, "NewSQLiteBackend") ||
						strings.Contains(callStr, "sqlite.New") ||
						strings.Contains(callStr, "modernc.org/sqlite") ||
						strings.Contains(callStr, "sqlite3") {
						usesSQLite = true
					}

					if strings.Contains(callStr, "SQLiteEnableWAL") ||
						strings.Contains(callStr, "PRAGMA journal_mode") {
						hasWAL = true
					}

					return true
				})

				if usesSQLite && !hasWAL {
					pos := ctx.Fset.Position(gf.AST.Pos())

					f, err := finding.NewBuilder(
						"P012", toolName,
						"SQLite store without WAL mode — prone to 'database is locked' errors under concurrent access",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryPerformance).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Call storage.SQLiteEnableWAL(ctx, db) after opening the SQLite database").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)
				}
			}

			return findings, nil
		},
	)
}
