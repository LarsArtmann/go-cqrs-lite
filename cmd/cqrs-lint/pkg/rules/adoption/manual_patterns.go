package adoption

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"
)

// loopBody extracts the body BlockStmt from a for, range, or range-style loop.
// Returns nil if n is not a loop or has no body.
func loopBody(n ast.Node) *ast.BlockStmt {
	switch loop := n.(type) {
	case *ast.ForStmt:
		return loop.Body
	case *ast.RangeStmt:
		return loop.Body
	default:
		return nil
	}
}

// filterAppendInBlock reports whether a block contains the pattern:
// if-statement whose body calls append().
func filterAppendInBlock(body *ast.BlockStmt) bool {
	for _, stmt := range body.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}

		if containsAppendCall(ifStmt.Body) {
			return true
		}
	}

	return false
}

// containsAppendCall reports whether a block contains an append() call.
func containsAppendCall(body *ast.BlockStmt) bool {
	found := false

	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name == "append" {
			found = true
			return false
		}

		return true
	})

	return found
}

// hasAggregationStmt reports whether a loop body contains an increment (++)
// or compound assignment (+=) statement — the "manual count/sum" idiom.
// Searches recursively because the increment is often inside an if-guard
// (for _, x := range items { if x.Active { count++ } }).
func hasAggregationStmt(body *ast.BlockStmt) bool {
	found := false

	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}

		switch s := n.(type) {
		case *ast.IncDecStmt:
			found = true
			return false
		case *ast.AssignStmt:
			if s.Tok == token.ADD_ASSIGN || s.Tok == token.SUB_ASSIGN {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

// paginationVarNames are identifier names that signal manual pagination.
var paginationVarNames = []string{ //nolint:gochecknoglobals // static lookup table for pagination identifier detection
	"offset", "limit", "start", "end", "page", "size", "skip", "take", "from",
}

// sliceHasPaginationVar reports whether an index expression references a
// pagination-like variable name (case-insensitive).
func sliceHasPaginationVar(expr ast.Expr) bool {
	if expr == nil {
		return false
	}

	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		name := strings.ToLower(ident.Name)
		if slices.Contains(paginationVarNames, name) {
			found = true
			return false
		}

		return true
	})

	return found
}
