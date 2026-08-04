package analyzer

import "go/ast"

// CollectFoldCaseStrings walks all fold functions in the registry and extracts
// the string-literal case labels from their switch statements. A fold function
// is identified by the scanner as a function with signature
// func(state, event) (state, error) — see detectFoldFunc in scanner_folds.go.
//
// This is shared by rules that cross-reference emitted event types against
// handled event types (C038 typo detection, E006 orphaned events).
func (ctx *AnalysisContext) CollectFoldCaseStrings() []string {
	var cases []string

	for _, fold := range ctx.Registry.Folds {
		for _, gf := range ctx.GoFiles {
			if gf.Path != fold.File {
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
							if s := StringLit(expr); s != "" {
								cases = append(cases, s)
							}
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
