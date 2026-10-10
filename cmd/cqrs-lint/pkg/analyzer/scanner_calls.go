package analyzer

import (
	"go/ast"
	"slices"
)

// scanCallExpr inspects call expressions for event.New/NewEvent,
// RegisterTyped, catalog.Event, and similar CQRS API calls.
func scanCallExpr(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	// Generic type-instantiation calls (requireCommandType[*MyCommand](cmd),
	// requireQueryType[*MyQuery](q)) have call.Fun as *ast.IndexExpr, not
	// *ast.SelectorExpr. Handle them before the SelectorExpr path. These calls
	// unambiguously identify handler bodies: a command/query struct that
	// appears as a generic type argument is being handled. This suppresses
	// E005/E007 for handlers registered via string-typed APIs
	// (dispatcher.Register with command.Type constants) whose handler→struct
	// link the analyzer cannot otherwise trace.
	scanGenericHandlerCall(ctx, call)

	sel, ok := SelectorFromExpr(call.Fun)
	if !ok {
		return
	}

	funcName := sel.Sel.Name
	pos := ctx.Fset.Position(call.Pos())

	// IsQualifierFor resolves the qualifier through type info when available:
	// aliased imports (es "…/event/v4") match by import path instead of the
	// local name, and shadowing locals stop matching. The string fallback
	// preserves syntax-only-load behavior (T20-8 convention).
	switch {
	case funcName == "New" && IsQualifierFor(gf, sel, "go-cqrs-lite/event"):
		if len(call.Args) > 0 {
			if eventTypeStr := StringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = EventEmission{
					File: gf.Path,
					Line: pos.Line,
				}
			} else if name := ExprIdentName(call.Args[0]); name != "" {
				// Constant identifier (possibly an alias chain): resolved
				// against TypeConstValues in ResolveEmittedEventTypeConsts,
				// after all const declarations have been scanned.
				ctx.Registry.pendingEmittedEventTypeRefs = append(
					ctx.Registry.pendingEmittedEventTypeRefs,
					pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
				)
			}
		}

		capturePayloadType(ctx, call)

	case funcName == "NewEvent" && IsQualifierFor(gf, sel, "go-cqrs-lite/event"):
		if len(call.Args) > 0 {
			if eventTypeStr := StringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesEmitted[eventTypeStr] = EventEmission{
					File: gf.Path,
					Line: pos.Line,
				}
			} else if name := ExprIdentName(call.Args[0]); name != "" {
				ctx.Registry.pendingEmittedEventTypeRefs = append(
					ctx.Registry.pendingEmittedEventTypeRefs,
					pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
				)
			}
		}

		capturePayloadType(ctx, call)

	case funcName == "RegisterTyped" || funcName == "RegisterQuery":
		if handlerType := handlerTypeFromCall(call); handlerType != "" {
			ctx.Registry.CommandTypesRegistered[handlerType] = true
		} else {
			// Handler type could not be extracted from the call args directly.
			// Try fallback strategies:
			//   1. Constructor-call handler (NewMyCommand(bus)): recorded in
			//      ConstructorHandlers — its text is a CALL expression, never a
			//      type name, so it must not pollute CommandTypesRegistered
			//      (T20-4).
			//   2. If the handler arg is a method value (h.handleX), record the
			//      method name for a post-pass that finds the FuncDecl and
			//      extracts the param type. Covers SEC's typed handler methods.
			//   3. Record the type-constant arg for const-value resolution.
			//      Covers consumers whose const values are struct names.
			if ctor := constructorHandlerText(call); ctor != "" {
				ctx.Registry.ConstructorHandlers[ctor] = true
			}
			if methodName := methodNameFromHandlerArg(call); methodName != "" {
				ctx.Registry.pendingHandlerMethods[methodName] = true
			}

			recordTypeConstArg(ctx, call, 1)
		}

	case funcName == "Register" && len(call.Args) == 1 &&
		!IsQualifierFor(gf, sel, "go-cqrs-lite/event") &&
		eventMetadataTypeExpr(call) != nil:
		// Event-catalog metadata registration (e.g. a catalog builder's
		// Register(EventMetadata{Type: ...})): the Type field carries the
		// event type as a literal, a constant reference, or a string(...)
		// conversion of either. Counts as a catalog declaration for the
		// provider-parity rules (C040/E018). See cqrs-htmx feedback (C040
		// phantoms on EventCatalog-declared events).
		expr := unwrapStringConv(eventMetadataTypeExpr(call))
		if eventTypeStr := StringLit(expr); eventTypeStr != "" {
			ctx.Registry.EventTypesInCatalog[eventTypeStr] = true
		} else if name := ExprIdentName(expr); name != "" {
			ctx.Registry.pendingCatalogEventTypeRefs = append(
				ctx.Registry.pendingCatalogEventTypeRefs,
				pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
			)
		}

	case funcName == "Register" && !IsQualifierFor(gf, sel, "go-cqrs-lite/event"):
		// Plain dispatcher.Register(typeConst, handler) — the string-type-based
		// command registration API. The handler type is not visible in the call
		// (it lives inside the handler body), so record the type-constant arg
		// for post-pass resolution. pkgName != "event" excludes event bus
		// Subscribe-style calls that happen to be named "Register" — though
		// such collisions are rare, the guard is cheap. See browser-history
		// feedback (E005 false positives on dispatcher.Register).
		recordTypeConstArg(ctx, call, 0)

	case funcName == "RegisterCommand" && IsQualifierFor(gf, sel, "go-cqrs-lite/system"):
		// system.RegisterCommand[MyCmd, MyState](sys, name, handler) — the
		// System composition root's typed command registration. The command
		// type is the FIRST generic type argument; the closure handler's
		// first non-context parameter is the fallback. Without this case
		// E005 flags every system-registered command as handlerless
		// (example/taskmanager: 10 false positives).
		if name := commandTypeFromSystemRegisterCommand(call); name != "" {
			ctx.Registry.CommandTypesRegistered[name] = true
		}

	case funcName == "Event" && IsQualifierFor(gf, sel, "go-cqrs-lite/catalog"):
		if len(call.Args) > 0 {
			if eventTypeStr := StringLit(call.Args[0]); eventTypeStr != "" {
				ctx.Registry.EventTypesInCatalog[eventTypeStr] = true
			} else if name := ExprIdentName(call.Args[0]); name != "" {
				ctx.Registry.pendingCatalogEventTypeRefs = append(
					ctx.Registry.pendingCatalogEventTypeRefs,
					pendingEventTypeRef{constName: name, file: gf.Path, line: pos.Line},
				)
			}
		}

	case (funcName == "Event" || funcName == "EventOf") && IsQualifierFor(gf, sel, "go-cqrs-lite/schema"):
		// schema.Event / schema.EventOf — a schema declaration: the first
		// arg is the event type (literal or constant). Feeds E021's
		// declared-set.
		recordSchemaDeclaredEvent(ctx, gf, call, pos)

	case funcName == "Event" && isSchemaBuilderEventCall(ctx, gf, sel):
		// (*system.SchemaSet).Event[T](type, version, ops...) — the
		// system.Schemas() fluent builder, chained directly or through a
		// local builder variable. SelectorFromExpr already unwrapped the
		// generic instantiation, so sel.X is the receiver.
		recordSchemaDeclaredEvent(ctx, gf, call, pos)

	case isSchemaOpName(funcName) && IsQualifierFor(gf, sel, "go-cqrs-lite/schema"):
		// schema.RenameField / AddField / RemoveField / Transform / Split —
		// migration ops declaring a rung of a ladder (type, sourceVersion).
		// Feeds E022's missing-rung check.
		recordSchemaOp(ctx, gf, call, pos)

	case funcName == "AddDataProduct" && (IsQualifierFor(gf, sel, "go-cqrs-lite/catalog") ||
		argIsCatalogDataProduct(call) ||
		// Variable-passed product: the ident argument resolves to a
		// same-file catalog.DataProduct literal (13-32 §f21).
		dataProductLiteralFor(gf, call) != nil):
		scanDataProductDeclaration(ctx, gf, call)

	case funcName == "NewProjection":
		scanProjectionRegistration(ctx, gf, call)

	case funcName == "Subscribe":
		scanProjectionSubscription(ctx, gf, call)

	case funcName == "StrictApply" && IsQualifierFor(gf, sel, "go-cqrs-lite/decider"):
		// decider.StrictApply(foldFunc, knownTypes) — record the fold function
		// name so B005 can suppress its "use decider.StrictApply" suggestion
		// when the suggestion is already implemented. See browser-history
		// feedback (B005 latent gap).
		if name := foldNameFromStrictApplyArg(call); name != "" {
			ctx.Registry.StrictApplyFolds[name] = true
		}
	}
}

// handlerTypeFromClosure extracts the handler type from a function literal's
// parameter list. It finds the first parameter that is a pointer to a named
// type (e.g., *MyCommand) or a named type (e.g., MyQuery), skipping
// context.Context parameters. Handles both bare identifiers (*MyCommand) and
// package-qualified types (*pkg.MyCommand → "MyCommand").
func handlerTypeFromClosure(fn *ast.FuncLit) string {
	if fn.Type == nil || fn.Type.Params == nil {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		switch t := param.Type.(type) {
		case *ast.StarExpr:
			return lastIdentSegment(t.X)
		case *ast.Ident:
			// Skip context.Context-like params that are just Idents
			// (context.Context itself is a SelectorExpr, so it won't match here)
			return t.Name
		}
	}

	return ""
}

// commandTypeFromSystemRegisterCommand extracts the command struct name from
// a system.RegisterCommand[Cmd, State](sys, name, handler) call. The FIRST
// generic type argument names the command type (CreateTaskCmd above). When
// the call carries no resolvable generic argument, fall back to the closure
// handler's first non-context parameter.
func commandTypeFromSystemRegisterCommand(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.IndexExpr:
		if name := typeNameFromGenericArg(fn.Index); name != "" {
			return name
		}
	case *ast.IndexListExpr:
		if len(fn.Indices) > 0 {
			if name := typeNameFromGenericArg(fn.Indices[0]); name != "" {
				return name
			}
		}
	}

	return handlerTypeFromCall(call)
}

// capturePayloadType records the struct type name used as the event payload.
// In event.New/NewEvent, the payload is always the 5th argument (index 4):
// event.New(type, streamID, streamType, version, payload, opts...).
func capturePayloadType(ctx *AnalysisContext, call *ast.CallExpr) {
	for i := 4; i < len(call.Args); i++ {
		arg := call.Args[i]

		switch a := arg.(type) {
		case *ast.CompositeLit:
			if id, ok := a.Type.(*ast.Ident); ok {
				ctx.Registry.EventPayloadTypes[id.Name] = true
				return
			}
		case *ast.UnaryExpr:
			// &T{...} — pointer composite literal, the dominant payload form
			// (events are appended by pointer). Without this case the
			// registry missed exactly the payloads real code emits.
			if lit, ok := a.X.(*ast.CompositeLit); ok {
				if id, ok := lit.Type.(*ast.Ident); ok {
					ctx.Registry.EventPayloadTypes[id.Name] = true
					return
				}
			}
		case *ast.CallExpr:
			// T(x) — conversion payload: the domain wraps a wider view into
			// the payload type at the emit site (nsfw-classifier history:
			// event.New(..., MediaCaptured(cmd.View), ...) — the conversion
			// blinded the registry and made that package the lone E003
			// flag while identical siblings routed through emit helpers).
			// A conversion names the exact payload type, so this case is
			// precise. Deliberately NOT handled here: selector payloads
			// (cmd.Field) — the field name need not equal the payload type
			// name, and revealing them today would flip consumers to new
			// E003 findings before E003's construct counting understands
			// emit helpers.
			if id, ok := a.Fun.(*ast.Ident); ok && len(a.Args) == 1 {
				ctx.Registry.EventPayloadTypes[id.Name] = true
				return
			}
		case *ast.Ident:
			ctx.Registry.EventPayloadTypes[a.Name] = true
			return
		}
	}
}

func scanProjectionRegistration(ctx *AnalysisContext, gf *GoFile, call *ast.CallExpr) {
	pos := ctx.Fset.Position(call.Pos())
	info := ProjectionInfo{
		Package: gf.Pkg.PkgPath,
		File:    gf.Path,
		Pos:     pos,
	}

	if len(call.Args) > 0 {
		info.Name = StringLit(call.Args[0])
	}

	for _, arg := range call.Args {
		if cl, ok := arg.(*ast.CompositeLit); ok {
			for _, elt := range cl.Elts {
				if eventTypeStr := StringLit(elt); eventTypeStr != "" {
					info.EventTypes = append(info.EventTypes, eventTypeStr)
				}
			}
		}

		// Check if any handler argument is a function literal that launches goroutines.
		if fn := extractHandlerFuncLit(arg); fn != nil {
			info.HasAsync = hasAsyncInBody(fn.Body)
		}
	}

	ctx.Registry.Projections = append(ctx.Registry.Projections, info)
}

func scanProjectionSubscription(
	ctx *AnalysisContext,
	gf *GoFile,
	call *ast.CallExpr,
) {
	eventTypeStr := ""
	if len(call.Args) > 0 {
		eventTypeStr = StringLit(call.Args[0])
	}

	if eventTypeStr == "" {
		return
	}

	found := false

	for i := range ctx.Registry.Projections {
		if slices.Contains(ctx.Registry.Projections[i].EventTypes, eventTypeStr) {
			found = true
		}
	}

	if !found {
		pos := ctx.Fset.Position(call.Pos())
		ctx.Registry.Projections = append(ctx.Registry.Projections, ProjectionInfo{
			Package:    gf.Pkg.PkgPath,
			File:       gf.Path,
			Pos:        pos,
			EventTypes: []string{eventTypeStr},
		})
	}
}
