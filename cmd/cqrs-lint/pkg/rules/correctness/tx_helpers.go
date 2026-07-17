package correctness

import (
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func returnsError(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Results == nil {
		return false
	}

	for _, field := range fn.Type.Results.List {
		if id, ok := field.Type.(*ast.Ident); ok && id.Name == "error" {
			return true
		}
	}

	return false
}

func findBeginTxVar(fn *ast.FuncDecl) string {
	var txVar string

	ast.Inspect(fn, func(n ast.Node) bool {
		if txVar != "" {
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

		if sel.Sel.Name != "BeginTx" && sel.Sel.Name != "Begin" {
			return true
		}

		if assignStmt := findContainingAssignStmt(fn, call); assignStmt != nil {
			if len(assignStmt.Lhs) > 0 {
				if id, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
					txVar = id.Name
				}
			}
		}

		return true
	})

	return txVar
}

func findContainingAssignStmt(fn *ast.FuncDecl, target ast.Node) *ast.AssignStmt {
	var result *ast.AssignStmt

	ast.Inspect(fn, func(n ast.Node) bool {
		if result != nil {
			return false
		}

		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for _, rhs := range assign.Rhs {
			if containsNode(rhs, target) {
				result = assign

				return false
			}
		}

		return true
	})

	return result
}

func containsNode(parent, target ast.Node) bool {
	if parent == target {
		return true
	}

	found := false

	ast.Inspect(parent, func(n ast.Node) bool {
		if n == target {
			found = true

			return false
		}

		return !found
	})

	return found
}

func hasCommitCall(fn *ast.FuncDecl, txVar string) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
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

		if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar && sel.Sel.Name == "Commit" {
			found = true
		}

		return true
	})

	return found
}

func hasDeferCommit(fn *ast.FuncDecl, txVar string) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}

		call := deferStmt.Call
		if call == nil {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			return true
		}

		if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar && sel.Sel.Name == "Commit" {
			found = true
		}

		return true
	})

	return found
}

func hasReturnNil(fn *ast.FuncDecl) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}

		for _, result := range ret.Results {
			if id, ok := result.(*ast.Ident); ok && id.Name == "nil" {
				found = true

				return false
			}
		}

		return true
	})

	return found
}

// txVarEscapesToArg reports whether the tx variable is passed as an argument
// to any call expression within the function body (excluding the BeginTx call
// that produced it). This covers the closure/callback transaction-helper
// pattern that C001 cannot evaluate statically:
//
//	func withTx(db *sql.DB, body func(*sql.Tx) error) error {
//	    tx, err := db.BeginTx(ctx, nil)
//	    if err != nil { return err }
//	    if err := body(tx); err != nil { return err } // tx escapes to body
//	    return nil // body contractually commits; flagging this would double-commit
//	}
//
// When tx escapes to a callback, the commit cannot be verified within the
// helper's own body, so C001 must not flag it. Method calls on tx itself
// (tx.Commit, tx.Exec, tx.Rollback) are NOT escapes — tx is the receiver of
// the SelectorExpr in call.Fun, not an argument in call.Args.
func txVarEscapesToArg(fn *ast.FuncDecl, txVar string) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		for _, arg := range call.Args {
			if exprReferencesIdent(arg, txVar) {
				found = true

				return false
			}
		}

		return true
	})

	return found
}

// exprReferencesIdent reports whether expr is, or address-of, the named ident.
// It intentionally stays narrow: only direct uses (tx) and &tx. Deeper nesting
// (tx.Field) does not count as an escape of the tx variable itself.
func exprReferencesIdent(expr ast.Expr, name string) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name == name
	case *ast.UnaryExpr:
		return e.Op == token.AND && exprReferencesIdent(e.X, name)
	}

	return false
}
