package performance

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// P013: Missing busy_timeout for SQLite.
//
// SQLite without busy_timeout throws "database is locked" errors under
// concurrent access. busy_timeout can be set via:
//   - DSN query parameter: ?_pragma=busy_timeout(5000)  (modernc.org/sqlite)
//   - DSN query parameter: ?_busy_timeout=5000          (mattn/go-sqlite3)
//   - Post-open PRAGMA:    db.Exec("PRAGMA busy_timeout = 5000")
//   - Library wrapper:     storage.SQLiteEnableWAL(ctx, db)
//
// This rule performs per-call-site analysis: for each sql.Open with a SQLite
// driver, it resolves the DSN argument (handling literals, concatenation, and
// constant references) and checks whether busy_timeout is present. If the DSN
// is resolvable and lacks busy_timeout, it checks for a post-open PRAGMA in
// the same function and for library wrapper calls in the same file before
// emitting a finding.
//
//nolint:ireturn // factory returns public interface
func NewP013Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P013-missing-sqlite-busy-timeout",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			sites := findSQLiteOpenSites(ctx)
			if len(sites) == 0 {
				return findings, nil
			}

			constMap := buildConstStringMap(ctx)

			for _, site := range sites {
				if hasBusyTimeoutEvidence(ctx, site, constMap) {
					continue
				}

				pos := ctx.Fset.Position(site.call.Pos())

				f, err := finding.NewBuilder(
					"P013", toolName,
					"SQLite connection opened without busy_timeout — "+
						"'database is locked' errors under concurrent access",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategoryPerformance).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion(
						"Add busy_timeout to the DSN "+
							"(e.g. ?_pragma=busy_timeout(5000) for modernc.org/sqlite) "+
							"or call db.Exec(\"PRAGMA busy_timeout = 5000\") after opening",
					).
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

// hasBusyTimeoutEvidence checks whether there is sufficient evidence that
// busy_timeout is configured for the given sql.Open call site. Returns true
// (suppress the finding) when any of these hold:
//
//  1. The resolved DSN string contains "busy_timeout" (any driver syntax).
//  2. The enclosing function calls db.Exec("PRAGMA busy_timeout ...").
//  3. The file calls a known library wrapper (SQLiteEnableWAL, etc.).
//  4. The DSN argument is opaque (cannot be statically resolved) — we
//     suppress to avoid false positives on dynamically constructed DSNs
//     that may set busy_timeout via runtime logic we cannot see.
func hasBusyTimeoutEvidence(
	ctx *analyzer.AnalysisContext,
	site sqliteOpenSite,
	constMap map[string]string,
) bool {
	// If the DSN argument is present and resolvable, check its content.
	if site.dsnArg != nil {
		localScope := buildLocalScope(site.funcDecl, constMap)
		resolved := resolveStringExpr(site.dsnArg, constMap, localScope)

		if resolved != "" {
			// We successfully resolved the DSN — check it directly.
			return dsnHasBusyTimeout(resolved)
		}

		// DSN is a literal but empty string, or a concatenation that includes
		// at least one string literal. If it's purely opaque (function call,
		// field access, etc.), we can't resolve it.
		if !containsStringLiteral(site.dsnArg) {
			// Opaque DSN — suppress to avoid false positives.
			return true
		}
	}

	// DSN is resolvable but empty, or absent (sql.OpenDB). Check for
	// post-open PRAGMA in the same function.
	if funcSetsPragma(site.funcDecl, "busy_timeout") {
		return true
	}

	// Check for library wrapper calls in the same file.
	if site.file != nil &&
		fileHasWrapperCall(site.file, "SQLiteEnableWAL", "EnsureSQLiteDSNBusyTimeout") {
		return true
	}

	return false
}

// containsStringLiteral reports whether the expression tree contains at least
// one string literal node. Used to distinguish "opaque DSN" (no literals at
// all, e.g. a function call result) from "DSN with literals but unresolvable
// parts" (e.g. path + someFunc()). When a DSN contains literals but can't be
// fully resolved, we still flag it because the literal parts are visible and
// should contain busy_timeout if it's set.
func containsStringLiteral(expr ast.Expr) bool {
	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			found = true
			return false
		}

		return true
	})

	return found
}
