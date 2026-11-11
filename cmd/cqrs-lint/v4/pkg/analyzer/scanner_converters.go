package analyzer

import "go/ast"

// Local id-CONVERTER functions. The wrapper-replay paths
// (scanEmitHelperFunc / scanCatalogHelperFunc) cover helpers that forward a
// PARAMETER into catalog.Event / event.New; the blind spot left is the
// mirror shape — a local single-return func whose body is a pure conversion
// of its parameter, CALLED at the id position:
//
//	func messageID(t string) catalog.MessageID { return catalog.MessageID(t) }
//
//	catalog.Event[P](messageID(string(EventCreated)), catalog.Sends)
//
// The id is unresolvable at the call site without inlining through the
// converter (issue #121: timesheets suppressed all of E004 on a complete,
// test-pinned catalog). Converters are recorded during scanning; id
// arguments that are still call expressions at scan time are deferred and
// re-resolved in ResolveConverterRoutedEventIDs, so declaration order never
// matters.

// scanConverterFunc records a local single-return converter: a function
// whose entire body is `return Conv(param)` — Conv a builtin or
// package-qualified conversion, param one of the function's own parameters.
// Name-matched like emitHelperParams (cross-package collisions are a
// documented limitation).
func scanConverterFunc(ctx *AnalysisContext, fn *ast.FuncDecl) {
	if fn.Name == nil || fn.Body == nil || fn.Recv != nil || fn.Type == nil || fn.Type.Params == nil {
		return
	}

	if len(fn.Body.List) != 1 {
		return
	}

	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return
	}

	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return
	}

	switch call.Fun.(type) {
	case *ast.Ident, *ast.SelectorExpr:
	default:
		return
	}

	arg, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return
	}

	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == arg.Name {
				ctx.Registry.converterFuncParams[fn.Name.Name] = true

				return
			}
		}
	}
}

// maxConverterInlineDepth bounds inlining (messageID(toMessageID(x)): each
// pass strips one converter level; real code bottoms out in two).
const maxConverterInlineDepth = 8

// inlineConverterCalls rewrites an id expression routed through recorded
// converter funcs down to its argument — messageID(string(Const)) → Const —
// unwrapping builtin string(...) conversions alongside. Unrecorded call
// shapes return unchanged (today's behavior: unresolved, a false negative
// never a false positive).
func inlineConverterCalls(reg *CQRSRegistry, expr ast.Expr) ast.Expr {
	for range maxConverterInlineDepth {
		unwrapped := unwrapStringConv(expr)

		call, ok := unwrapped.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return unwrapped
		}

		id, ok := call.Fun.(*ast.Ident)
		if !ok || !reg.converterFuncParams[id.Name] {
			return unwrapped
		}

		expr = call.Args[0]
	}

	return unwrapStringConv(expr)
}

// eventIDDest names where a deferred id argument was seen; resolution feeds
// the same destinations the direct paths write.
type eventIDDest uint8

const (
	destCatalog eventIDDest = iota
	destEmitted
	destSchema
)

// pendingConverterEventID is one id argument that was a call expression at
// scan time (unresolvable before every converter declaration is known).
type pendingConverterEventID struct {
	expr ast.Expr
	file string
	line int
	dest eventIDDest
}

// deferConverterEventID queues an unresolved call-shaped id argument for
// the post-pass. Only call expressions can benefit from converter inlining;
// anything else is dead weight.
func deferConverterEventID(
	ctx *AnalysisContext,
	expr ast.Expr,
	gf *GoFile,
	line int,
	dest eventIDDest,
) {
	if _, ok := expr.(*ast.CallExpr); ok {
		ctx.Registry.pendingConverterEventIDs = append(
			ctx.Registry.pendingConverterEventIDs,
			pendingConverterEventID{expr: expr, file: gf.Path, line: line, dest: dest},
		)
	}
}

// ResolveConverterRoutedEventIDs re-resolves deferred call-shaped id
// arguments with converter inlining. Must run AFTER all files are scanned
// (converters may be declared in any file) and BEFORE
// ResolveEmittedEventTypeConsts, which drains the pending refs this pass
// appends to.
func ResolveConverterRoutedEventIDs(ctx *AnalysisContext) {
	if len(ctx.Registry.pendingConverterEventIDs) == 0 {
		return
	}

	for _, pending := range ctx.Registry.pendingConverterEventIDs {
		expr := inlineConverterCalls(ctx.Registry, pending.expr)

		if eventTypeStr := StringLit(expr); eventTypeStr != "" {
			switch pending.dest {
			case destCatalog:
				ctx.Registry.EventTypesInCatalog[eventTypeStr] = true
			case destEmitted:
				ctx.Registry.EventTypesEmitted[eventTypeStr] = EventEmission{
					File: pending.file,
					Line: pending.line,
				}
			case destSchema:
				ctx.Registry.EventTypesInSchemaDecl[eventTypeStr] = EventEmission{
					File: pending.file,
					Line: pending.line,
				}
			}

			continue
		}

		if name := ExprIdentName(expr); name != "" {
			ref := pendingEventTypeRef{constName: name, file: pending.file, line: pending.line}

			switch pending.dest {
			case destCatalog:
				ctx.Registry.pendingCatalogEventTypeRefs = append(
					ctx.Registry.pendingCatalogEventTypeRefs, ref,
				)
			case destEmitted:
				ctx.Registry.pendingEmittedEventTypeRefs = append(
					ctx.Registry.pendingEmittedEventTypeRefs, ref,
				)
			case destSchema:
				ctx.Registry.pendingSchemaEventTypeRefs = append(
					ctx.Registry.pendingSchemaEventTypeRefs, ref,
				)
			}
		}
	}

	ctx.Registry.pendingConverterEventIDs = nil
}
