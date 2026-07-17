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

		if sel.Sel.Name != "BeginTx" && sel.Sel.Name != "Begin" && sel.Sel.Name != "Beginx" {
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

// txAnalysis collects every tx-related signal C001 needs from a single AST
// walk of the function body, instead of one walk per signal. Filled by
// analyzeTxUsage.
type txAnalysis struct {
	commitCalled bool // tx.Commit() appears as a direct call
	deferCommit  bool // defer tx.Commit() appears
	returnsNil   bool // a bare `return nil` success path exists
	escapesToArg bool // tx is passed as a call argument (callback-helper pattern)
	txUsed       bool // tx.<Method>() called where Method is not Commit/Rollback
}

// analyzeTxUsage walks the function body once and collects every tx signal C001
// consults. This replaces five separate ast.Inspect walks (hasCommitCall,
// hasDeferCommit, hasReturnNil, txVarEscapesToArg, txIsUsed) with a single
// traversal, eliminating the O(functions × 5 × AST-size) cost flagged in the
// round-2 self-critique (§d-1).
func analyzeTxUsage(fn *ast.FuncDecl, txVar string) txAnalysis {
	a := txAnalysis{}

	ast.Inspect(fn, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.DeferStmt:
			if node.Call == nil {
				return true
			}
			if sel, ok := analyzer.SelectorFromExpr(node.Call.Fun); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar &&
					sel.Sel.Name == "Commit" {
					a.deferCommit = true
				}
			}

		case *ast.ReturnStmt:
			if !a.returnsNil {
				for _, result := range node.Results {
					if id, ok := result.(*ast.Ident); ok && id.Name == "nil" {
						a.returnsNil = true
					}
				}
			}

		case *ast.CallExpr:
			// Direct tx.Commit() call.
			if sel, ok := analyzer.SelectorFromExpr(node.Fun); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar &&
					sel.Sel.Name == "Commit" {
					a.commitCalled = true
				}
			}
			// tx passed as an argument to any call (callback-helper escape).
			for _, arg := range node.Args {
				if exprReferencesIdent(arg, txVar) {
					a.escapesToArg = true
				}
			}

		case *ast.SelectorExpr:
			// tx.<Method> where Method is not a lifecycle method = real use.
			id, ok := node.X.(*ast.Ident)
			if !ok || id.Name != txVar {
				return true
			}
			switch node.Sel.Name {
			case "Commit", "Rollback":
				// lifecycle — not a "use"
			default:
				a.txUsed = true
			}
		}

		return true
	})

	return a
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

// txIsUsed reports whether the transaction variable has any method called on
// it other than the lifecycle methods Commit and Rollback. This covers
// tx.Exec/ExecContext, tx.Query/QueryContext/QueryRow/QueryRowContext,
// tx.Prepare/PrepareContext, tx.Stmt/StmtContext, and sqlx extras
// (NamedExec, Get, Select, MustExec) by shape rather than by enumerating
// every name: any `tx.<Method>(...)` selector that isn't Commit/Rollback
// counts as a real use.
//
// C001 treats tx usage as a stronger bug signal than a bare `return nil`:
// if the tx is used at all and never committed (and doesn't escape to a
// callback that owns the commit), the work is silently lost regardless of the
// function's return shape. See item f-7 in the DiscordSync feedback triage.
