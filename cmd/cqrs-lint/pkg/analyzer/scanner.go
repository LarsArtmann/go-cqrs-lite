package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
)

// scanFile analyzes a Go file for CQRS patterns and populates the registry.
func scanFile(ctx *AnalysisContext, gf *GoFile) {
	for _, decl := range gf.AST.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			scanGenDecl(ctx, gf, d)
		case *ast.FuncDecl:
			scanFuncDecl(ctx, gf, d)
		}
	}

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		scanCallExpr(ctx, gf, call)

		return true
	})
}

func scanGenDecl(ctx *AnalysisContext, gf *GoFile, decl *ast.GenDecl) {
	if decl.Tok != token.TYPE {
		return
	}

	for _, spec := range decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok || ts.Type == nil {
			continue
		}

		pos := ctx.Fset.Position(ts.Pos())
		structType, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}

		info := CommandInfo{
			Name:    ts.Name.Name,
			Package: gf.Pkg.PkgPath,
			File:    gf.Path,
			Pos:     pos,
		}
		scanStructFields(structType, &info)

		if isCommandType(&info) {
			ctx.Registry.Commands = append(ctx.Registry.Commands, info)
		}

		ctx.Registry.Events = append(ctx.Registry.Events, EventInfo{
			Name:    ts.Name.Name,
			Package: gf.Pkg.PkgPath,
			File:    gf.Path,
			Pos:     pos,
		})
	}
}

func scanStructFields(st *ast.StructType, info *CommandInfo) {
	if st.Fields == nil {
		return
	}

	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			info.Fields = append(info.Fields, name.Name)
		}

		if isBasicCommandEmbed(field.Type) {
			info.HasBasicCmd = true
		}

		if len(field.Names) == 0 {
			if exprStr := ExprString(field.Type); exprStr != "" {
				info.Fields = append(info.Fields, exprStr)
				if strings.Contains(exprStr, "BasicCommand") {
					info.HasBasicCmd = true
				}
			}
		}
	}
}

func isCommandType(info *CommandInfo) bool {
	return info.HasBasicCmd || info.ManualID || info.ManualType || info.ManualAggID
}

func isBasicCommandEmbed(expr ast.Expr) bool {
	s := ExprString(expr)

	return strings.Contains(s, "BasicCommand")
}
