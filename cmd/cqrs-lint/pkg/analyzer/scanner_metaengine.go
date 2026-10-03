package analyzer

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// scanMetaengineQueryDecl records metaengine.Query[Q,R](collection, opts...)
// declarations: the R type name, the Volume literal, and whether declarative
// pushdown options (FilterOnField/SortOnField) are present. The utilization
// rules (F022/F023) coach Go-side sort/filter call sites over a registered R
// type whose declaration lacks them (nsfw-classifier feedback, 2026-10-03).
//
// The generic instantiation's Fun is an *ast.IndexListExpr, which the
// SelectorExpr-based call scanner cannot see — hence this dedicated walk.
func scanMetaengineQueryDecl(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	idxList, ok := call.Fun.(*ast.IndexListExpr)
	if !ok || len(idxList.Indices) != 2 {
		return
	}

	sel, ok := SelectorFromExpr(idxList.X)
	if !ok || sel.Sel.Name != "Query" {
		return
	}

	if !IsQualifierFor(gf, sel, "go-cqrs-lite/metaengine") {
		return
	}

	resultType := typeNameFromGenericArg(idxList.Indices[1])
	if resultType == "" {
		return
	}

	info := QueryDeclInfo{
		ResultType: resultType,
		File:       gf.Path,
		Line:       ctx.Fset.Position(call.Pos()).Line,
	}

	if len(call.Args) > 0 {
		if lit := StringLit(call.Args[0]); lit != "" {
			info.Collection = lit
		} else if name := ExprIdentName(call.Args[0]); name != "" {
			info.Collection = name
		}
	}

	for _, arg := range call.Args {
		opt, ok := arg.(*ast.CallExpr)
		if !ok || len(opt.Args) == 0 {
			continue
		}

		switch optionName(opt) {
		case "Volume":
			if n, ok := intLiteral(opt.Args[0]); ok {
				info.Volume = n
				info.HasVolume = true
			}
		case "FilterOnField":
			info.HasFilterOnField = true
		case "SortOnField":
			info.HasSortOnField = true
		}
	}

	ctx.Registry.MetaengineQueries = append(ctx.Registry.MetaengineQueries, info)
}

// optionName renders a Query option call's function name, unwrapping generic
// instantiation (FilterOnField[itemView](...) → "FilterOnField").
func optionName(call *ast.CallExpr) string {
	fun := call.Fun

	if idx, ok := fun.(*ast.IndexExpr); ok {
		fun = idx.X
	} else if idxList, ok := fun.(*ast.IndexListExpr); ok {
		fun = idxList.X
	}

	sel, ok := SelectorFromExpr(fun)
	if !ok {
		return ""
	}

	return sel.Sel.Name
}

// intLiteral parses a decimal integer literal expression, accepting Go's
// underscore digit separators (100_000). Non-literal expressions (constants,
// variables) are reported as absent — an unresolvable volume must not gate
// coaching on.
func intLiteral(expr ast.Expr) (int, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}

	value := strings.ReplaceAll(lit.Value, "_", "")

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	return n, true
}
