package analyzer

import "go/ast"

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
		len(reg.pendingCatalogEventTypeRefs) == 0 &&
		len(reg.pendingSchemaEventTypeRefs) == 0 {
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

	for _, ref := range reg.pendingSchemaEventTypeRefs {
		val := reg.TypeConstValues[ref.constName]
		if val == "" {
			continue
		}

		reg.EventTypesInSchemaDecl[val] = EventEmission{File: ref.file, Line: ref.line}
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
