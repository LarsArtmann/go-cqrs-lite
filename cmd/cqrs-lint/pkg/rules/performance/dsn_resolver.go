package performance

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// sqliteOpenSite records a sql.Open call with a SQLite driver and the
// metadata needed to check whether busy_timeout / WAL is configured.
type sqliteOpenSite struct {
	call     *ast.CallExpr
	dsnArg   ast.Expr
	funcDecl *ast.FuncDecl
	file     *ast.File
}

// findSQLiteOpenSites walks every non-test Go file and returns one entry per
// sql.Open call whose driver argument is a string literal containing "sqlite".
// For sql.OpenDB (no string driver name), the file must import a known SQLite
// driver path — the site is reported but dsnArg may be nil.
func findSQLiteOpenSites(ctx *analyzer.AnalysisContext) []sqliteOpenSite {
	var sites []sqliteOpenSite

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

			pkg := analyzer.SelectorPackage(sel)
			if pkg != "sql" {
				return true
			}

			switch sel.Sel.Name {
			case "Open":
				if len(call.Args) < 2 {
					return true
				}

				driver := analyzer.StringLit(call.Args[0])
				if !strings.Contains(strings.ToLower(driver), "sqlite") {
					return true
				}

				sites = append(sites, sqliteOpenSite{
					call:     call,
					dsnArg:   call.Args[1],
					funcDecl: enclosingFunc(ctx, call.Pos()),
					file:     gf.AST,
				})

			case "OpenDB":
				if !fileImportsSQLiteDriver(gf.AST) {
					return true
				}

				sites = append(sites, sqliteOpenSite{
					call:     call,
					dsnArg:   nil,
					funcDecl: enclosingFunc(ctx, call.Pos()),
					file:     gf.AST,
				})
			}

			return true
		})
	}

	return sites
}

// fileImportsSQLiteDriver checks whether the file imports a known SQLite
// driver package.
func fileImportsSQLiteDriver(file *ast.File) bool {
	knownDrivers := []string{
		"modernc.org/sqlite",
		"github.com/mattn/go-sqlite3",
		"database/sql", // re-exported by some wrappers
	}

	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}

		p := strings.Trim(imp.Path.Value, `"`)

		if slices.Contains(knownDrivers, p) {
			return true
		}
	}

	return false
}

// enclosingFunc finds the FuncDecl whose body contains the given position.
func enclosingFunc(ctx *analyzer.AnalysisContext, pos token.Pos) *ast.FuncDecl {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			if pos >= fn.Body.Pos() && pos <= fn.Body.End() {
				return fn
			}
		}
	}

	return nil
}

// resolveStringExpr resolves an AST expression to a best-effort string value.
// It handles:
//   - String literals: "value" → value
//   - String concatenation: a + b → resolveStringExpr(a) + resolveStringExpr(b)
//   - Identifiers referencing package-level constants
//   - Identifiers referencing local variables (within the same function)
//   - Parenthesized expressions: (expr) → resolveStringExpr(expr)
//
// Returns "" when the expression cannot be statically resolved (e.g. function
// call results, field access, etc.). Callers must treat "" as "unknown.".
func resolveStringExpr(
	expr ast.Expr,
	constMap map[string]string,
	localScope map[string]string,
) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			return unquoteGoString(e.Value)
		}

		return ""

	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return ""
		}

		left := resolveStringExpr(e.X, constMap, localScope)
		right := resolveStringExpr(e.Y, constMap, localScope)

		// If either side is unresolvable, the whole concatenation is unknown.
		// Exception: empty string literal concatenation (rare, but valid Go).
		// We only return a value if BOTH sides resolved.
		if left == "" && !isStringLiteral(e.X) {
			return ""
		}

		if right == "" && !isStringLiteral(e.Y) {
			return ""
		}

		return left + right

	case *ast.Ident:
		// Check local scope first (shadows package-level).
		if val, ok := localScope[e.Name]; ok {
			return val
		}

		// Check package-level constants.
		if val, ok := constMap[e.Name]; ok {
			return val
		}

		return ""

	case *ast.ParenExpr:
		return resolveStringExpr(e.X, constMap, localScope)

	default:
		return ""
	}
}

// isStringLiteral returns true if expr is a string literal (BasicLit with
// STRING kind). Used to distinguish "unresolvable because opaque" from
// "unresolvable because empty string literal".
func isStringLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

// unquoteGoString removes the surrounding quotes from a Go string literal
// value (as produced by go/ast BasicLit.Value). Handles both interpreted
// ("...") and raw (`...`) string literals. Does NOT process escape sequences
// — the raw bytes between quotes are returned as-is, which is sufficient for
// DSN analysis (DSNs don't contain escape sequences in practice).
func unquoteGoString(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}

	return s
}

// buildConstStringMap scans all non-test Go files for package-level constant
// declarations (const Name = "value" or const Name = "a" + "b") and returns
// a map of constant name → resolved string value. Only string constants are
// included.
func buildConstStringMap(ctx *analyzer.AnalysisContext) map[string]string {
	result := make(map[string]string)

	// First pass: collect raw const expressions.
	rawValues := make(map[string]ast.Expr)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}

			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for i, name := range valSpec.Names {
					if i < len(valSpec.Values) {
						rawValues[name.Name] = valSpec.Values[i]
					}
				}
			}
		}
	}

	// Second pass: resolve each const, allowing consts to reference other
	// consts in the map (handles const chains).
	for name, expr := range rawValues {
		if resolved := resolveStringExpr(
			expr,
			result,
			nil,
		); resolved != "" ||
			isStringLiteral(expr) {
			result[name] = resolved
		}
	}

	// Third pass: re-resolve to pick up any consts that reference consts
	// resolved in the second pass (handles ordering).
	for name, expr := range rawValues {
		if resolved := resolveStringExpr(
			expr,
			result,
			nil,
		); resolved != "" ||
			isStringLiteral(expr) {
			result[name] = resolved
		}
	}

	return result
}

// buildLocalExprScope scans a function body for short variable declarations
// (name := expr) and assignments (name = expr), returning a map of variable
// name → RHS expression. Unlike buildLocalScope (which tries to resolve to a
// string), this preserves the raw expression so that partially-resolvable
// assignments (e.g. path + pragmas) can be inspected for individual string
// parts.
func buildLocalExprScope(fn *ast.FuncDecl) map[string]ast.Expr {
	if fn == nil || fn.Body == nil {
		return nil
	}

	scope := make(map[string]ast.Expr)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if stmt, ok := n.(*ast.AssignStmt); ok {
			for i, lhs := range stmt.Lhs {
				if i >= len(stmt.Rhs) {
					break
				}

				if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
					scope[ident.Name] = stmt.Rhs[i]
				}
			}
		}

		return true
	})

	return scope
}

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
