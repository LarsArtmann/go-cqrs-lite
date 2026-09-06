package analyzer

import (
	"go/ast"
	"go/token"
)

// FoldCaseInfo is one case label of a fold function, carrying the position
// data rules need to report findings at the case clause.
type FoldCaseInfo struct {
	Value    string
	FoldName string
	File     string
	Pos      token.Position
}

// CollectFoldCasesWithPos walks all fold functions in the registry and
// extracts their switch case labels with positions. A fold function is
// identified by the scanner as a function with signature
// func(state, event) (state, error) — see detectFoldFunc in scanner_folds.go.
//
// String-literal labels are returned verbatim. Const-identifier labels
// (e.g. `case event.UserCreated:`) are resolved through the registry's
// TypeConstValues map, so rules see the same string for a literal
// "user.created" and a typed constant of that value.
//
// This is shared by rules that cross-reference emitted event types against
// handled event types (C038 typo detection, C040 dead fold case, E006
// orphaned events).
func (ctx *AnalysisContext) CollectFoldCasesWithPos() []FoldCaseInfo {
	var cases []FoldCaseInfo

	for _, fold := range ctx.Registry.Folds {
		for _, gf := range ctx.GoFiles {
			if gf.Path != fold.File || gf.IsTest {
				continue
			}

			ast.Inspect(gf.AST, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Name == nil {
					return true
				}

				if fn.Name.Name != fold.FuncName {
					return true
				}

				ast.Inspect(fn.Body, func(nn ast.Node) bool {
					sw, ok := nn.(*ast.SwitchStmt)
					if !ok {
						return true
					}

					for _, stmt := range sw.Body.List {
						cc, ok := stmt.(*ast.CaseClause)
						if !ok || cc.List == nil {
							continue
						}

						for _, expr := range cc.List {
							value, ok := ctx.caseLabelValue(expr)
							if !ok {
								continue
							}

							cases = append(cases, FoldCaseInfo{
								Value:    value,
								FoldName: fold.FuncName,
								File:     fold.File,
								Pos:      ctx.Fset.Position(expr.Pos()),
							})
						}
					}

					return true
				})

				return true
			})
		}
	}

	return cases
}

// CollectFoldCaseStrings returns just the case-label values collected by
// [AnalysisContext.CollectFoldCasesWithPos], in collection order.
func (ctx *AnalysisContext) CollectFoldCaseStrings() []string {
	infos := ctx.CollectFoldCasesWithPos()

	cases := make([]string, 0, len(infos))
	for _, info := range infos {
		cases = append(cases, info.Value)
	}

	return cases
}

// caseLabelValue resolves one case-clause expression to its event-type
// string. String literals map directly; identifiers and selector
// expressions resolve through TypeConstValues when the name is a known
// event-type constant.
func (ctx *AnalysisContext) caseLabelValue(expr ast.Expr) (string, bool) {
	if s := StringLit(expr); s != "" {
		return s, true
	}

	switch e := expr.(type) {
	case *ast.Ident:
		if v, found := ctx.Registry.TypeConstValues[e.Name]; found {
			return v, true
		}
	case *ast.SelectorExpr:
		if v, found := ctx.Registry.TypeConstValues[e.Sel.Name]; found {
			return v, true
		}
	}

	return "", false
}
