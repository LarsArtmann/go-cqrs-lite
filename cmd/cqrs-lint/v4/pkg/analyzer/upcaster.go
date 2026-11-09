package analyzer

import (
	"go/ast"
	"sync"
)

// upcasterRangeCache memoizes the per-file list of schema.NewUpcaster closure
// bodies. IsInsideUpcasterClosure is consulted for EVERY A014/C005 candidate
// call; a full ast.Inspect of the file per query was O(file)×queries (T20-7)
// and is now O(file) per FILE plus O(closures) per query. Keyed by *GoFile —
// a file's AST never changes after load.
var upcasterRangeCache sync.Map //nolint:gochecknoglobals // per-run memo of immutable file data

// IsInsideUpcasterClosure checks if the given call expression is inside a
// function literal passed to schema.NewUpcaster. In that context,
// event.NewEvent and json.Unmarshal on event payloads are the correct APIs
// (upcasters transform raw bytes and reconstruct events).
//
// This is used to suppress A014 and C005 findings in upcaster context.
func IsInsideUpcasterClosure(gf *GoFile, call *ast.CallExpr) bool {
	for _, body := range upcasterClosures(gf) {
		if body.Pos() <= call.Pos() && call.Pos() <= body.End() {
			return true
		}
	}

	return false
}

// upcasterClosures returns the memoized bodies of the schema.NewUpcaster
// closure arguments in the file. The qualifier resolves through the type
// checker when available (IsQualifierFor), so aliased imports match and
// shadowed qualifiers do not (T20-8).
func upcasterClosures(gf *GoFile) []*ast.BlockStmt {
	if cached, ok := upcasterRangeCache.Load(gf); ok {
		ranges, isRanges := cached.([]*ast.BlockStmt)
		if isRanges {
			return ranges
		}
	}

	var closures []*ast.BlockStmt

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		upcasterCall, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := SelectorFromExpr(upcasterCall.Fun)
		if !ok || sel.Sel.Name != "NewUpcaster" {
			return true
		}

		if !IsQualifierFor(gf, sel, "go-cqrs-lite/schema") {
			return true
		}

		for _, arg := range upcasterCall.Args {
			if fnLit, ok := arg.(*ast.FuncLit); ok && fnLit.Body != nil {
				closures = append(closures, fnLit.Body)
			}
		}

		return true
	})

	upcasterRangeCache.Store(gf, closures)

	return closures
}
