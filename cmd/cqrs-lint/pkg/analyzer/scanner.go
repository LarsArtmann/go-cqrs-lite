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
	// Scan all call expressions for event.New/NewEvent, RegisterTyped, etc.
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
		switch structType := ts.Type.(type) {
		case *ast.StructType:
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
			// Also collect as potential event payload type.
			ctx.Registry.Events = append(ctx.Registry.Events, EventInfo{
				Name:    ts.Name.Name,
				Package: gf.Pkg.PkgPath,
				File:    gf.Path,
				Pos:     pos,
			})
		}
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
		// Check for *command.BasicCommand embedding.
		if isBasicCommandEmbed(field.Type) {
			info.HasBasicCmd = true
		}
		// Check for embedded field with no name (embedded type).
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

func scanFuncDecl(ctx *AnalysisContext, gf *GoFile, fn *ast.FuncDecl) {
	pos := ctx.Fset.Position(fn.Pos())

	// Check for ID() method returning zero value.
	if isIDMethod(fn) {
		scanIDMethod(ctx, gf, fn, pos)
	}

	// Check for fold function: func(State, event.Event) (State, error)
	if foldInfo := detectFoldFunc(ctx, gf, fn, pos); foldInfo != nil {
		ctx.Registry.Folds = append(ctx.Registry.Folds, *foldInfo)
	}

	// Check for OO aggregate (has method returning uncommittedEvents or similar).
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
	// Look for the command struct this method belongs to.
	recvType := recvTypeName(fn)
	if recvType == "" {
		return
	}

	cmd := findOrCreateCommand(ctx, recvType, gf, pos)
	cmd.ManualID = true

	// Check if ID() returns zero-value composite literal.
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

func detectFoldFunc(
	ctx *AnalysisContext,
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

	// Param 2 should be event.Event or similar.
	paramTypeStr := ExprString(params.List[1].Type)
	if !strings.Contains(paramTypeStr, "event.Event") && !strings.Contains(paramTypeStr, "Event") {
		return nil
	}

	// Result 2 should be error.
	resultTypeStr := ExprString(results.List[1].Type)
	if resultTypeStr != "error" {
		return nil
	}

	stateType := ExprString(params.List[0].Type)
	if stateType == "" {
		stateType = baseTypeName(params.List[0].Type)
	}

	info := &FoldInfo{
		FuncName:  funcName(fn),
		File:      gf.Path,
		Pos:       pos,
		StateType: stateType,
	}

	// Collect event variable name(s).
	if len(params.List[1].Names) > 0 {
		for _, name := range params.List[1].Names {
			info.UnknownVars = append(info.UnknownVars, name.Name)
		}
	}

	// Analyze switch statement for default case returning nil.
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

			if cc.List == nil { // default case
				info.HasDefault = true
				// Check if default returns nil error.
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

func isOOAggregate(fn *ast.FuncDecl) bool {
	bodyStr := ""
	if fn.Body != nil {
		bodyStr = nodeString(fn.Body)
	}

	return strings.Contains(bodyStr, "uncommittedEvents") ||
		strings.Contains(bodyStr, "pendingEvents") ||
		strings.Contains(bodyStr, "UncommittedEvents")
}

func scanCallExpr(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	funcName := sel.Sel.Name
	pkgName := selectorPackage(sel)

	switch {
	case funcName == "New" && pkgName == "event":
		// event.New(type, ...) — first arg is the event type string.
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = gf.Path
			}
		}

	case funcName == "NewEvent" && pkgName == "event":
		// event.NewEvent(type, ...) — same pattern.
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = gf.Path
			}
		}

	case funcName == "RegisterTyped":
		// RegisterTyped — first arg or context determines the command type.
		ctx.Registry.CommandTypesRegistered[handlerTypeFromCall(call)] = true

	case funcName == "Event" && pkgName == "catalog":
		// catalog.Event(type, ...) — registers event type in catalog.
		if len(call.Args) > 0 {
			if eventTypeStr := stringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesInCatalog[eventTypeStr] = true
			}
		}
	}
}

func handlerTypeFromCall(call *ast.CallExpr) string {
	// Try to extract a type from the arguments.
	for _, arg := range call.Args {
		// Look for composite literal type or function call type.
		switch a := arg.(type) {
		case *ast.CompositeLit:
			if id, ok := a.Type.(*ast.Ident); ok {
				return id.Name
			}
		case *ast.CallExpr:
			// Could be reflect.TypeOf(X{}) or similar.
			return ExprString(a)
		}
	}

	return ""
}

// Helper: extract string literal value from an ast.Expr.
