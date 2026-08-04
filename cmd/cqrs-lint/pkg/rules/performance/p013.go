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
						"Add busy_timeout to the DSN " +
							"(e.g. ?_pragma=busy_timeout(5000) for modernc.org/sqlite) " +
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
//  1. Any resolvable string part of the DSN expression contains "busy_timeout"
//     (checks literals, resolved consts, resolved local vars).
//  2. The enclosing function calls db.Exec("PRAGMA busy_timeout ...").
//  3. The file calls a known library wrapper (SQLiteEnableWAL, etc.).
//  4. The DSN argument is fully opaque (no string literals or resolvable
//     identifiers) — suppress to avoid false positives on dynamically
//     constructed DSNs that may set busy_timeout via runtime logic we cannot see.
func hasBusyTimeoutEvidence(
	_ *analyzer.AnalysisContext,
	site sqliteOpenSite,
	constMap map[string]string,
) bool {
	localExprScope := buildLocalExprScope(site.funcDecl)

	// 1. Check if any resolvable string part of the DSN contains busy_timeout.
	if site.dsnArg != nil {
		if dsnExprContainsPragma(site.dsnArg, constMap, localExprScope, nil, dsnHasBusyTimeout) {
			return true
		}

		// If the DSN has no inspectable string parts at all (no literals, no
		// resolvable consts/vars), it's fully opaque — suppress.
		if !hasInspectableStringParts(site.dsnArg, constMap, localExprScope, nil) {
			return true
		}
	}

	// 2. Check for post-open PRAGMA in the enclosing function.
	if funcSetsPragma(site.funcDecl, "busy_timeout") {
		return true
	}

	// 3. Check for library wrapper calls in the same file.
	if site.file != nil &&
		fileHasWrapperCall(site.file, "SQLiteEnableWAL", "EnsureSQLiteDSNBusyTimeout") {
		return true
	}

	return false
}

// dsnExprContainsPragma walks the DSN expression tree and checks every
// resolvable string value (literals, resolved package-level consts, resolved
// local variable expressions) against the provided predicate. Returns true if
// any value matches.
//
// The visited set prevents infinite recursion through self-referential
// variable assignments (a := a + b).
func dsnExprContainsPragma(
	expr ast.Expr,
	constMap map[string]string,
	localExprScope map[string]ast.Expr,
	visited map[string]bool,
	pred func(string) bool,
) bool {
	if visited == nil {
		visited = make(map[string]bool)
	}

	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		switch e := n.(type) {
		case *ast.BasicLit:
			if e.Kind == token.STRING && pred(unquoteGoString(e.Value)) {
				found = true
				return false
			}

		case *ast.Ident:
			// Check package-level constants.
			if val, ok := constMap[e.Name]; ok && pred(val) {
				found = true
				return false
			}

			// Recursively check local variable assignments.
			if rhs, ok := localExprScope[e.Name]; ok && !visited[e.Name] {
				visited[e.Name] = true

				if dsnExprContainsPragma(rhs, constMap, localExprScope, visited, pred) {
					found = true
					return false
				}
			}
		}

		return true
	})

	return found
}

// hasInspectableStringParts returns true if the expression contains at least
// one string literal or an identifier that resolves to a string (const or
// local var). When false, the DSN is fully opaque and we suppress the finding.
func hasInspectableStringParts(
	expr ast.Expr,
	constMap map[string]string,
	localExprScope map[string]ast.Expr,
	visited map[string]bool,
) bool {
	if visited == nil {
		visited = make(map[string]bool)
	}

	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		switch e := n.(type) {
		case *ast.BasicLit:
			if e.Kind == token.STRING {
				found = true
				return false
			}

		case *ast.Ident:
			if _, ok := constMap[e.Name]; ok {
				found = true
				return false
			}

			// Recursively check local variable assignments.
			if rhs, ok := localExprScope[e.Name]; ok && !visited[e.Name] {
				visited[e.Name] = true

				if hasInspectableStringParts(rhs, constMap, localExprScope, visited) {
					found = true
					return false
				}
			}
		}

		return true
	})

	return found
}
