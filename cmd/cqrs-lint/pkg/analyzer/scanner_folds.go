package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
)

// scanFuncDecl analyzes a function declaration for CQRS patterns.
func scanFuncDecl(ctx *AnalysisContext, gf *GoFile, fn *ast.FuncDecl) {
	pos := ctx.Fset.Position(fn.Pos())

	if isIDMethod(fn) {
		scanIDMethod(ctx, gf, fn, pos)
	}

	if foldInfo := detectFoldFunc(ctx, gf, fn, pos); foldInfo != nil {
		ctx.Registry.Folds = append(ctx.Registry.Folds, *foldInfo)
	}

	if isOOAggregate(fn) {
		ctx.Registry.Deciders = append(ctx.Registry.Deciders, DeciderInfo{
			Package: gf.Pkg.PkgPath,
			File:    gf.Path,
			Pos:     pos,
			IsOO:    true,
		})
	}
}

func isIDMethod(fn *ast.FuncDecl) bool {
	return fn.Recv != nil && fn.Name != nil && fn.Name.Name == "ID"
}

func scanIDMethod(ctx *AnalysisContext, gf *GoFile, fn *ast.FuncDecl, pos token.Position) {
	recvType := recvTypeName(fn)
	if recvType == "" {
		return
	}

	cmd := findOrCreateCommand(ctx, recvType, gf, pos)
	cmd.ManualID = true

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}

		for _, result := range ret.Results {
			lit, ok := result.(*ast.CompositeLit)
			if ok && len(lit.Elts) == 0 {
				cmd.IDReturnsZero = true
			}
		}

		return true
	})
}

func findOrCreateCommand(
	ctx *AnalysisContext,
	name string,
	gf *GoFile,
	pos token.Position,
) *CommandInfo {
	for i := range ctx.Registry.Commands {
		if ctx.Registry.Commands[i].Name == name {
			return &ctx.Registry.Commands[i]
		}
	}

	ctx.Registry.Commands = append(ctx.Registry.Commands, CommandInfo{
		Name:    name,
		Package: gf.Pkg.PkgPath,
		File:    gf.Path,
		Pos:     pos,
	})

	return &ctx.Registry.Commands[len(ctx.Registry.Commands)-1]
}

func isOOAggregate(fn *ast.FuncDecl) bool {
	bodyStr := ""
	if fn.Body != nil {
		bodyStr = nodeString(fn.Body)
	}

	return strings.Contains(bodyStr, "uncommittedEvents") ||
		strings.Contains(bodyStr, "pendingEvents") ||
		strings.Contains(bodyStr, "UncommittedEvents")
}

func detectFoldFunc(
	_ *AnalysisContext,
	gf *GoFile,
	fn *ast.FuncDecl,
	pos token.Position,
) *FoldInfo {
	if fn.Body == nil || fn.Type == nil {
		return nil
	}

	params := fn.Type.Params

	results := fn.Type.Results
	if params == nil || results == nil {
		return nil
	}

	if len(params.List) != 2 || len(results.List) != 2 {
		return nil
	}

	paramTypeStr := ExprString(params.List[1].Type)
	if !strings.Contains(paramTypeStr, "event.Event") && !strings.Contains(paramTypeStr, "Event") {
		return nil
	}

	resultTypeStr := ExprString(results.List[1].Type)
	if resultTypeStr != "error" {
		return nil
	}

	stateType := ExprString(params.List[0].Type)
	if stateType == "" {
		stateType = BaseTypeName(params.List[0].Type)
	}

	info := &FoldInfo{
		FuncName:  funcName(fn),
		Package:   gf.Pkg.PkgPath,
		File:      gf.Path,
		Pos:       pos,
		StateType: stateType,
	}

	if len(params.List[1].Names) > 0 {
		for _, name := range params.List[1].Names {
			info.UnknownVars = append(info.UnknownVars, name.Name)
		}
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}

		info.HasSwitch = true

		for _, stmt := range sw.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}

			if cc.List == nil {
				info.HasDefault = true

				for _, bodyStmt := range cc.Body {
					ast.Inspect(bodyStmt, func(nn ast.Node) bool {
						ret, ok := nn.(*ast.ReturnStmt)
						if !ok {
							return true
						}

						if len(ret.Results) >= 2 {
							if id, ok := ret.Results[len(ret.Results)-1].(*ast.Ident); ok &&
								id.Name == "nil" {
								info.DefaultNil = true
							}
						}

						return true
					})
				}
			}
		}

		return true
	})

	return info
}
