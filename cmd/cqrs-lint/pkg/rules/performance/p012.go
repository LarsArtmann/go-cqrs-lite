package performance

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// P012: Missing WAL mode for SQLite.
//
// SQLite stores without WAL mode are prone to "database is locked" errors
// under concurrent access. WAL can be set via:
//   - DSN query parameter: ?_pragma=journal_mode(WAL)  (modernc.org/sqlite)
//   - DSN query parameter: ?_journal_mode=WAL          (mattn/go-sqlite3)
//   - Post-open PRAGMA:    db.Exec("PRAGMA journal_mode = WAL")
//   - Library wrapper:     storage.SQLiteEnableWAL(ctx, db)
//
// This rule performs per-call-site analysis: for each sql.Open with a SQLite
// driver, it checks whether WAL mode is configured via the DSN, a post-open
// PRAGMA, or a library wrapper call. It shares the same DSN resolution
// infrastructure as P013.
//
//nolint:ireturn // factory returns public interface
func NewP012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P012-missing-sqlite-wal",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			sites := findSQLiteOpenSites(ctx)
			if len(sites) == 0 {
				return findings, nil
			}

			constMap := buildConstStringMap(ctx)

			for _, site := range sites {
				if hasWALEvidence(ctx, site, constMap) {
					continue
				}

				pos := ctx.Fset.Position(site.call.Pos())

				f, err := finding.NewBuilder(
					"P012",
					toolName,
					"SQLite store without WAL mode — prone to 'database is locked' errors under concurrent access",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategoryPerformance).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion(
						"Add journal_mode(WAL) to the DSN " +
							"(e.g. ?_pragma=journal_mode(WAL) for modernc.org/sqlite) " +
							"or call db.Exec(\"PRAGMA journal_mode = WAL\") after opening",
					).
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

// hasWALEvidence checks whether WAL mode is configured for the given sql.Open
// call site. Mirrors hasBusyTimeoutEvidence but for journal_mode(WAL).
func hasWALEvidence(
	_ *analyzer.AnalysisContext,
	site sqliteOpenSite,
	constMap map[string]string,
) bool {
	localExprScope := buildLocalExprScope(site.funcDecl)

	// 1. Check if any resolvable string part of the DSN enables WAL.
	if site.dsnArg != nil {
		if dsnExprContainsPragma(site.dsnArg, constMap, localExprScope, nil, dsnHasWAL) {
			return true
		}

		if !hasInspectableStringParts(site.dsnArg, constMap, localExprScope, nil) {
			return true
		}
	}

	// 2. Check for post-open PRAGMA in the enclosing function.
	if funcSetsPragma(site.funcDecl, "journal_mode") {
		return true
	}

	// 3. Check for library wrapper calls in the same file.
	if site.file != nil &&
		fileHasWrapperCall(site.file, "SQLiteEnableWAL") {
		return true
	}

	return false
}
