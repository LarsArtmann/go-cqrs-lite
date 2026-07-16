package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
)

// scanFile analyzes a Go file for CQRS patterns and populates the registry.
func scanFile(ctx *AnalysisContext, gf *GoFile) {
	varAssigns := map[string]string{}

	for _, decl := range gf.AST.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			scanGenDecl(ctx, gf, d)
		case *ast.FuncDecl:
			scanFuncDecl(ctx, gf, d)
		}
	}

	ast.Inspect(gf.AST, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			trackVarAssignments(node, varAssigns)
		case *ast.CallExpr:
			scanCallExpr(ctx, gf, node)
			capturePayloadTypeFromVar(ctx, node, varAssigns)
		}
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

// trackVarAssignments records variable → type mappings from short variable
// declarations (:=) and assignments where the RHS is a composite literal.
// This lets capturePayloadType resolve variable references to their actual types.
func trackVarAssignments(stmt *ast.AssignStmt, varAssigns map[string]string) {
	for i, lhs := range stmt.Lhs {
		if i >= len(stmt.Rhs) {
			break
		}

		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}

		if cl, ok := stmt.Rhs[i].(*ast.CompositeLit); ok {
			if typeIdent, ok := cl.Type.(*ast.Ident); ok {
				varAssigns[ident.Name] = typeIdent.Name
			}
		}
	}
}

// capturePayloadTypeFromVar resolves variable payload references using
// the varAssigns map. If event.New's payload arg is a variable name that
// was assigned a composite literal earlier, register the actual type.
func capturePayloadTypeFromVar(
	ctx *AnalysisContext,
	call *ast.CallExpr,
	varAssigns map[string]string,
) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	funcName := sel.Sel.Name
	pkgName := SelectorPackage(sel)

	if funcName != "New" && funcName != "NewEvent" {
		return
	}

	if pkgName != "event" {
		return
	}

	if len(call.Args) < 5 {
		return
	}

	payloadArg := call.Args[4]
	ident, ok := payloadArg.(*ast.Ident)
	if !ok {
		return
	}

	if typeName, ok := varAssigns[ident.Name]; ok {
		ctx.Registry.EventPayloadTypes[typeName] = true
	}
}
