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
		case *ast.TypeAssertExpr:
			scanTypeAssertion(ctx, node)
		}
		return true
	})
}

func scanGenDecl(ctx *AnalysisContext, gf *GoFile, decl *ast.GenDecl) {
	if decl.Tok == token.CONST {
		scanConstDecl(ctx, gf, decl)
		return
	}

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
	return info.HasBasicCmd || info.ManualID
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
	sel, ok := SelectorFromExpr(call.Fun)
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

// scanConstDecl records command.Type / query.Type constant declarations so
// that type-constant arguments passed to Register/RegisterTyped can be
// resolved to their struct names later. Recognizes both forms:
//
//	const GetVisitQueryType query.Type = "GetVisitQuery"
//	const cmdCreate command.Type = "create"
//
// Only constants whose declared type is exactly command.Type or query.Type
// (a SelectorExpr ending in ".Type") are recorded — this avoids capturing
// unrelated string constants. See browser-history feedback (E005/E007).
func scanConstDecl(ctx *AnalysisContext, _ *GoFile, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || len(vs.Values) == 0 {
			continue
		}

		if !isCommandOrQueryType(vs.Type) {
			continue
		}

		val := stringLit(vs.Values[0])
		if val == "" {
			continue
		}

		for _, name := range vs.Names {
			ctx.Registry.TypeConstValues[name.Name] = val
		}
	}
}

// isCommandOrQueryType reports whether expr is "command.Type" or "query.Type"
// (a SelectorExpr whose Sel is "Type" and whose qualifier contains "command"
// or "query").
func isCommandOrQueryType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}

	if sel.Sel.Name != "Type" {
		return false
	}

	pkg := SelectorPackage(sel)

	return pkg == "command" || pkg == "query"
}

// ResolveRegisteredTypeConsts resolves type-constant arguments recorded during
// scanning to their target command/query struct names and marks those structs
// as registered. Must run AFTER all files have been scanned, because the const
// declaration and the Register call may live in different files or packages.
//
// For each recorded const name, looks up its string value in TypeConstValues.
// If the value matches a known command or query struct name, marks that struct
// registered. This is a no-op when no type constants were recorded.
func ResolveRegisteredTypeConsts(reg *CQRSRegistry) {
	if len(reg.registeredTypeConsts) == 0 || len(reg.TypeConstValues) == 0 {
		return
	}

	known := make(map[string]bool, len(reg.Commands)+len(reg.Events))
	for _, cmd := range reg.Commands {
		known[cmd.Name] = true
	}

	// Query types are not in a separate registry slice — they are detected by
	// the E007 rule via the "Query" suffix at the call site. To support
	// suppressing E007, we accept any const value that resembles a struct name
	// (Capitalized identifier). The IsCommandRegistered check is shared by both
	// E005 (commands) and E007 (queries), so marking a value here suppresses
	// both.
	for _, constName := range reg.registeredTypeConsts {
		val, ok := reg.TypeConstValues[constName]
		if !ok || val == "" {
			continue
		}

		// Mark the value as registered. For commands this directly suppresses
		// E005. For queries, E007 consults IsCommandRegistered (shared map),
		// so this suppresses E007 as well.
		if known[val] || looksLikeStructName(val) {
			reg.CommandTypesRegistered[val] = true
		}
	}
}

