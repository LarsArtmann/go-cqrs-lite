package performance

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// dsnHasBusyTimeout checks whether the resolved DSN string contains a
// busy_timeout pragma in any supported driver syntax:
//   - modernc.org/sqlite: ?_pragma=busy_timeout(5000)
//   - mattn/go-sqlite3: ?_busy_timeout=5000 or ?busy_timeout=5000
//   - URL-encoded variants: _pragma=busy_timeout%3D5000
func dsnHasBusyTimeout(dsn string) bool {
	lower := strings.ToLower(dsn)
	return strings.Contains(lower, "busy_timeout")
}

// dsnHasWAL checks whether the resolved DSN string enables WAL mode in any
// supported driver syntax:
//   - modernc.org/sqlite: ?_pragma=journal_mode(WAL)
//   - mattn/go-sqlite3: ?_journal_mode=WAL
func dsnHasWAL(dsn string) bool {
	lower := strings.ToLower(dsn)
	return strings.Contains(lower, "journal_mode(wal)") ||
		strings.Contains(lower, "journal_mode=wal")
}

// funcSetsPragma checks whether the enclosing function (or the same file)
// sets a PRAGMA via db.Exec/ExecContext for the given pragma name on the
// *sql.DB variable returned by the sql.Open call. This catches the post-open
// PRAGMA pattern:
//
//	db, _ := sql.Open("sqlite", dsn)
//	db.Exec("PRAGMA busy_timeout = 5000")
//
// dbVar is the variable name; if empty, we check any Exec call in the function.
func funcSetsPragma(fn *ast.FuncDecl, pragmaName string) bool {
	if fn == nil {
		return false
	}

	target := strings.ToLower(pragmaName)

	found := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			return true
		}

		// Check for db.Exec / db.ExecContext calls.
		if sel.Sel.Name != "Exec" && sel.Sel.Name != "ExecContext" {
			return true
		}

		// Check if any string argument contains the pragma.
		for _, arg := range call.Args {
			s := analyzer.StringLit(arg)
			if s != "" && strings.Contains(strings.ToLower(s), "pragma "+target) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

// fileHasWrapperCall checks whether the file contains a call to a known
// library wrapper function that sets the given configuration (e.g.
// SQLiteEnableWAL sets both WAL and busy_timeout). This catches patterns
// where a helper function applies pragmas to an already-opened *sql.DB.
func fileHasWrapperCall(file *ast.File, wrapperNames ...string) bool {
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		callStr := analyzer.ExprString(call.Fun)

		for _, name := range wrapperNames {
			if strings.Contains(callStr, name) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

// hasSQLiteOpenEvidence reports whether an sql.Open call site already handles
// the pragma concern (suppressing the finding): any resolvable DSN string part
// matching dsnCheck, a fully opaque DSN (no literals or resolvable identifiers
// — suppressed to avoid false positives on runtime-built DSNs), a post-open
// PRAGMA statement in the enclosing function, or a known library wrapper call
// in the same file. Shared by the p012 WAL and p013 busy_timeout evidence
// checks, which differ only in predicate, pragma, and wrapper names.
func hasSQLiteOpenEvidence(
	site sqliteOpenSite,
	constMap map[string]string,
	pragma string,
	dsnCheck func(string) bool,
	wrapperNames ...string,
) bool {
	localExprScope := buildLocalExprScope(site.funcDecl)

	// 1. Check if any resolvable string part of the DSN matches the predicate.
	if site.dsnArg != nil {
		if dsnExprContainsPragma(site.dsnArg, constMap, localExprScope, nil, dsnCheck) {
			return true
		}

		// If the DSN has no inspectable string parts at all, it's fully
		// opaque — suppress.
		if !hasInspectableStringParts(site.dsnArg, constMap, localExprScope, nil) {
			return true
		}
	}

	// 2. Check for post-open PRAGMA in the enclosing function.
	if funcSetsPragma(site.funcDecl, pragma) {
		return true
	}

	// 3. Check for library wrapper calls in the same file.
	return site.file != nil && fileHasWrapperCall(site.file, wrapperNames...)
}
