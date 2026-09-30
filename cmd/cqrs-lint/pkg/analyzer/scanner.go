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
			capturePayloadTypeFromVar(ctx, gf, node, varAssigns)
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

	if decl.Tok == token.VAR {
		scanVarAliasDecl(ctx, decl)
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
				info.Embeds = append(info.Embeds, exprStr)
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
	gf *GoFile,
	call *ast.CallExpr,
	varAssigns map[string]string,
) {
	sel, ok := SelectorFromExpr(call.Fun)
	if !ok {
		return
	}

	funcName := sel.Sel.Name
	if funcName != "New" && funcName != "NewEvent" {
		return
	}

	if !IsQualifierFor(gf, sel, "go-cqrs-lite/event") {
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

// scanConstDecl records command.Type / query.Type / event.Type constant
// declarations so that type-constant arguments passed to Register/RegisterTyped
// can be resolved to their struct names later and const-identifier fold cases
// can be resolved to their string values. Recognizes both forms:
//
//	const GetVisitQueryType query.Type = "GetVisitQuery"
//	const cmdCreate command.Type = "create"
//
// Only constants whose declared type is exactly command.Type, query.Type, or
// event.Type (a SelectorExpr ending in ".Type") are recorded — this avoids
// capturing unrelated string constants. Constants whose VALUE references
// another constant (type-inherited aliases like
// `aliasEvent = pkg.TypedConst`) are recorded as alias expressions and
// resolved in a post-pass — see ResolveEmittedEventTypeConsts. See
// browser-history feedback (E005/E007) and cqrs-htmx feedback (C040
// phantoms on alias-emitted events).
func scanConstDecl(ctx *AnalysisContext, _ *GoFile, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || len(vs.Values) == 0 {
			continue
		}

		if isCommandOrQueryType(vs.Type) {
			val := StringLit(vs.Values[0])
			if val == "" {
				continue
			}

			for _, name := range vs.Names {
				ctx.Registry.TypeConstValues[name.Name] = val
			}

			continue
		}

		if isConstReferenceExpr(vs.Values[0]) {
			for _, name := range vs.Names {
				ctx.Registry.constAliasExprs[name.Name] = vs.Values[0]
			}
		}
	}
}

// scanVarAliasDecl records package-level var declarations whose values are
// cross-package constant references. Go has no const aliases, so consumers
// re-expose domain constants as vars (`var eventUserRegistered =
// identitymodel.EventUserRegistered` — the cqrs-htmx identity-model↔usermgmt
// pattern) and emission sites then reference the var. Restricted to
// selector-shaped values: same-package var references are mutable and stay
// unrecorded, so a missed resolution is a false negative, never a false
// positive. Values may be wrapped in a string(...) conversion.
func scanVarAliasDecl(ctx *AnalysisContext, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || len(vs.Values) == 0 {
			continue
		}

		if !isCrossPackageConstReference(vs.Values[0]) {
			continue
		}

		for _, name := range vs.Names {
			ctx.Registry.constAliasExprs[name.Name] = vs.Values[0]
		}
	}
}

// unwrapStringConv returns the inner expression of a string(...) conversion,
// or expr unchanged.
func unwrapStringConv(expr ast.Expr) ast.Expr {
	if call, ok := expr.(*ast.CallExpr); ok && len(call.Args) == 1 {
		if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "string" {
			return call.Args[0]
		}
	}

	return expr
}

// isCrossPackageConstReference reports whether expr is a package-qualified
// identifier (possibly string(...)-wrapped) — the var-alias shape of a
// constant re-exported from another package.
func isCrossPackageConstReference(expr ast.Expr) bool {
	_, ok := unwrapStringConv(expr).(*ast.SelectorExpr)

	return ok
}

// isConstReferenceExpr reports whether expr is a reference to another
// constant: a bare identifier, a selector expression, a parenthesized
// reference, or a string(...) conversion of one of those. Literal values are
// handled by the typed-const path in scanConstDecl.
func isConstReferenceExpr(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return true
	case *ast.ParenExpr:
		return isConstReferenceExpr(e.X)
	case *ast.CallExpr:
		if id, ok := e.Fun.(*ast.Ident); ok && id.Name == "string" && len(e.Args) == 1 {
			return isConstReferenceExpr(e.Args[0])
		}
	}

	return false
}

// isCommandOrQueryType reports whether expr is a typed event/command/query
// type: "command.Type", "query.Type", or "event.Type" (a SelectorExpr whose
// Sel is "Type" and whose qualifier names one of those packages).
func isCommandOrQueryType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}

	if sel.Sel.Name != "Type" {
		return false
	}

	pkg := SelectorPackage(sel)

	return pkg == "command" || pkg == "query" || pkg == "event"
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

// ResolveEmittedEventTypeConsts resolves event-type arguments that were
// recorded as constant references at event.New / event.NewEvent /
// catalog.Event call sites, and expands alias const chains into
// TypeConstValues. Must run AFTER all files are scanned: the referenced const
// may live in another file or package, and alias chains resolve only once
// every declaration is in the table. Resolved entries land in
// EventTypesEmitted / EventTypesInCatalog exactly as if the call site had
// passed a string literal, so C038/C040/E006 and the catalog-parity rules see
// constant-emitted events. See cqrs-htmx feedback (C040 phantoms).
func ResolveEmittedEventTypeConsts(reg *CQRSRegistry) {
	if len(reg.pendingEmittedEventTypeRefs) == 0 &&
		len(reg.pendingCatalogEventTypeRefs) == 0 {
		return
	}

	reg.expandConstAliases()

	for _, ref := range reg.pendingEmittedEventTypeRefs {
		val, ok := reg.TypeConstValues[ref.constName]
		if !ok || val == "" {
			continue
		}

		reg.EventTypesEmitted[val] = EventEmission{File: ref.file, Line: ref.line}
	}

	for _, ref := range reg.pendingCatalogEventTypeRefs {
		val := reg.TypeConstValues[ref.constName]
		if val == "" {
			continue
		}

		reg.EventTypesInCatalog[val] = true
	}
}

// maxConstAliasDepth bounds alias-chain resolution (a = b, b = c, ...).
const maxConstAliasDepth = 16

// expandConstAliases resolves every recorded alias const into
// TypeConstValues. Resolution is a memoized fixpoint: chains resolve
// regardless of declaration order, and cycles are cut by the seen-set.
func (r *CQRSRegistry) expandConstAliases() {
	if len(r.constAliasExprs) == 0 {
		return
	}

	for name := range r.constAliasExprs {
		r.resolveConstAlias(name, make(map[string]bool), 0)
	}
}

// resolveConstAlias returns the string value of the named constant, following
// alias references through constAliasExprs when the constant has no literal
// value of its own. Missing names resolve to "".
func (r *CQRSRegistry) resolveConstAlias(name string, seen map[string]bool, depth int) string {
	if val, ok := r.TypeConstValues[name]; ok {
		return val
	}

	expr, ok := r.constAliasExprs[name]
	if !ok || depth >= maxConstAliasDepth || seen[name] {
		return ""
	}

	seen[name] = true

	if val := r.evalConstExpr(expr, seen, depth+1); val != "" {
		r.TypeConstValues[name] = val

		return val
	}

	return ""
}

// evalConstExpr evaluates a const value expression to its string: a string
// literal, an identifier/selector reference to another constant (matched by
// bare name — cross-package collisions only risk a false negative, the same
// tradeoff as recordTypeConstArg), or a string(...) conversion of either.
func (r *CQRSRegistry) evalConstExpr(expr ast.Expr, seen map[string]bool, depth int) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return StringLit(e)
	case *ast.ParenExpr:
		return r.evalConstExpr(e.X, seen, depth)
	case *ast.CallExpr:
		if id, ok := e.Fun.(*ast.Ident); ok && id.Name == "string" && len(e.Args) == 1 {
			return r.evalConstExpr(e.Args[0], seen, depth)
		}
	case *ast.Ident:
		return r.resolveConstAlias(e.Name, seen, depth)
	case *ast.SelectorExpr:
		return r.resolveConstAlias(e.Sel.Name, seen, depth)
	}

	return ""
}
